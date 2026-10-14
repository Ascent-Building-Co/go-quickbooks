package quickbooks

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RequestScope describes a single outbound API call and is passed to
// RateLimiter.Acquire so the limiter can apply per-realm and per-endpoint
// (batch vs general) limits independently.
type RequestScope struct {
	RealmID  string
	Endpoint string
	IsBatch  bool
}

// ConcurrentRelease is a function returned by RateLimiter.Acquire. The caller
// must invoke it exactly once when the request is complete to release the
// concurrency slot.
type ConcurrentRelease func()

// RateLimiter is implemented by any value that can throttle outbound API
// requests. Acquire blocks until the request is allowed to proceed and returns
// a ConcurrentRelease that must be called when the request completes.
// The default implementation is RealmRateLimiter.
type RateLimiter interface {
	Acquire(ctx context.Context, scope RequestScope) (ConcurrentRelease, error)
}

// RateLimitConfig controls the token-bucket rate limiter. Zero values are
// replaced with defaults from DefaultRateLimitConfig when passed to
// WithRateLimitConfig or NewRealmRateLimiter.
//
//   - Concurrent: maximum in-flight requests per realm (default 10).
//   - GeneralPerMinute / GeneralBurst: steady-state rate and burst for normal
//     endpoints (default 500/min, burst 10).
//   - BatchPerMinute / BatchBurst: rate and burst for the batch endpoint
//     (default 40/min, burst 5).
//   - WaitTimeout: if non-zero, Acquire returns an error after this duration
//     instead of blocking indefinitely (default 0 = block forever).
type RateLimitConfig struct {
	Concurrent       uint
	GeneralPerMinute float64
	GeneralBurst     uint
	BatchPerMinute   float64
	BatchBurst       uint
	WaitTimeout      time.Duration
}

func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Concurrent:       10,
		GeneralPerMinute: 500,
		GeneralBurst:     10,
		BatchPerMinute:   40,
		BatchBurst:       5,
	}
}

func NormalizeRateLimitConfig(cfg RateLimitConfig) (RateLimitConfig, error) {
	defaults := DefaultRateLimitConfig()

	if cfg.Concurrent == 0 {
		cfg.Concurrent = defaults.Concurrent
	}
	if cfg.GeneralPerMinute == 0 {
		cfg.GeneralPerMinute = defaults.GeneralPerMinute
	}
	if cfg.GeneralBurst == 0 {
		cfg.GeneralBurst = defaults.GeneralBurst
	}
	if cfg.BatchPerMinute == 0 {
		cfg.BatchPerMinute = defaults.BatchPerMinute
	}
	if cfg.BatchBurst == 0 {
		cfg.BatchBurst = defaults.BatchBurst
	}

	if cfg.GeneralPerMinute < 0 {
		return RateLimitConfig{}, fmt.Errorf("general rate cannot be negative")
	}
	if cfg.BatchPerMinute < 0 {
		return RateLimitConfig{}, fmt.Errorf("batch rate cannot be negative")
	}
	if cfg.WaitTimeout < 0 {
		return RateLimitConfig{}, fmt.Errorf("rate limit wait timeout cannot be negative")
	}

	return cfg, nil
}

type realmRateLimiter struct {
	general    *rate.Limiter
	batch      *rate.Limiter
	concurrent chan struct{}
}

// RealmRateLimiter is the default RateLimiter. It maintains independent
// token-bucket limiters per realm, with separate buckets for general and batch
// endpoints, plus a concurrency cap. Limiter state is created lazily on the
// first request for each realm. RealmRateLimiter is safe for concurrent use.
type RealmRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*realmRateLimiter
	config   RateLimitConfig
}

func NewRealmRateLimiter(cfg RateLimitConfig) (*RealmRateLimiter, error) {
	normalized, err := NormalizeRateLimitConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &RealmRateLimiter{
		limiters: make(map[string]*realmRateLimiter),
		config:   normalized,
	}, nil
}

func (m *RealmRateLimiter) Acquire(ctx context.Context, scope RequestScope) (ConcurrentRelease, error) {
	limiter, err := m.getRealmLimiter(scope.RealmID)
	if err != nil {
		return nil, fmt.Errorf("unable to get realm limiter: %w", err)
	}

	if err := limiter.general.Wait(ctx); err != nil {
		return nil, fmt.Errorf("realm general rate limiter wait error: %w", err)
	}
	if scope.IsBatch {
		if err := limiter.batch.Wait(ctx); err != nil {
			return nil, fmt.Errorf("realm batch rate limiter wait error: %w", err)
		}
	}

	select {
	case limiter.concurrent <- struct{}{}:
		var once sync.Once
		return func() {
			once.Do(func() {
				<-limiter.concurrent
			})
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Delay returns how long until Acquire for scope would pass the realm's rate
// limits, without taking a token: zero when a request could start now. It
// ignores the concurrency cap. A caller that batches its requests can use it
// to hold them while the realm's allowance is spent, sending more in one call
// rather than queueing calls behind the limiter.
func (m *RealmRateLimiter) Delay(scope RequestScope) time.Duration {
	limiter, err := m.getRealmLimiter(scope.RealmID)
	if err != nil {
		return 0
	}
	now := time.Now()
	delay := tokenDelay(limiter.general, now)
	if scope.IsBatch {
		delay = max(delay, tokenDelay(limiter.batch, now))
	}
	return delay
}

// tokenDelay returns how long until l has a whole token at now.
func tokenDelay(l *rate.Limiter, now time.Time) time.Duration {
	limit := l.Limit()
	tokens := l.TokensAt(now)
	if limit == rate.Inf || tokens >= 1 {
		return 0
	}
	if limit <= 0 {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration((1 - tokens) / float64(limit) * float64(time.Second))
}

func (m *RealmRateLimiter) getRealmLimiter(realmID string) (*realmRateLimiter, error) {
	if realmID == "" {
		return nil, fmt.Errorf("realmID is \"\"")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if limiter, ok := m.limiters[realmID]; ok {
		return limiter, nil
	}

	limiter := &realmRateLimiter{
		general:    rate.NewLimiter(rate.Limit(m.config.GeneralPerMinute/60.0), int(m.config.GeneralBurst)),
		batch:      rate.NewLimiter(rate.Limit(m.config.BatchPerMinute/60.0), int(m.config.BatchBurst)),
		concurrent: make(chan struct{}, int(m.config.Concurrent)),
	}
	m.limiters[realmID] = limiter
	return limiter, nil
}
