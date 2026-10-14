package quickbooks

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const (
	DefaultBaseURL      = "https://quickbooks.api.intuit.com/v3/company"
	SandboxBaseURL      = "https://sandbox-quickbooks.api.intuit.com/v3/company"
	DefaultAuthURL      = "https://appcenter.intuit.com/connect/oauth2"
	DefaultTokenURL     = "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer"
	DefaultMinorVersion = 75

	// DefaultTransferTimeout bounds file uploads and streaming downloads. These
	// use a copy of the HTTP client with no http.Client.Timeout, so large
	// transfers are limited by this context deadline instead of the (typically
	// short) client-wide timeout suited to JSON API calls.
	DefaultTransferTimeout = 5 * time.Minute

	RequestIDMaxLength = 50

	AccountingScope = "com.intuit.quickbooks.accounting"
	OpenIDProfile   = "profile"
	OpenIDEmail     = "email"
	OpenIDPhone     = "phone"
	OpenIDAddress   = "address"

	// IncludeInvoiceLink requests population of the Invoice.InvoiceLink field.
	// Only applies to invoices with online payment enabled and a valid customer
	// email address. Pass via WithInclude on Get, Query, or List calls.
	IncludeInvoiceLink = "invoiceLink"

	// IncludeUpdateAccountOnTxns propagates account changes on a Customer or
	// Vendor update to all existing transactions for that entity.
	IncludeUpdateAccountOnTxns = "updateaccountontxns"

	// IncludeDoNotUpdateAccountOnTxns prevents account changes on an Item
	// update from propagating to existing transactions.
	IncludeDoNotUpdateAccountOnTxns = "donotupdateaccountontxns"
)

// Option configures a Client at construction time. Pass one or more Option
// values to NewClient. All options have safe defaults; only supply the ones
// you need to override.
type Option interface {
	apply(*clientConfig) error
}

type optionFunc func(*clientConfig) error

func (f optionFunc) apply(cfg *clientConfig) error {
	return f(cfg)
}

type clientConfig struct {
	httpClient      *http.Client
	baseURL         string
	minorVersion    int
	oauthConfig     *oauth2.Config
	transferTimeout time.Duration

	rateLimiterDisabled bool
	rateLimitConfig     RateLimitConfig
	rateLimiter         RateLimiter
}

// Client is a QuickBooks Online API client scoped to a single application.
// It holds shared transport configuration (HTTP client, base URL, OAuth
// config, rate limiter) and is safe for concurrent use.
//
// Use NewClient to construct a Client, then call Company to obtain a
// CompanyClient bound to a specific realm and token source.
type Client struct {
	httpClient   *http.Client
	baseURL      *url.URL
	minorVersion int
	oauthConfig  *oauth2.Config

	// transferClient is a copy of httpClient with no Timeout, used for file
	// uploads and streaming downloads so large transfers are bounded by
	// transferTimeout (via context) rather than the client-wide Timeout.
	transferClient  *http.Client
	transferTimeout time.Duration

	rateLimiter          RateLimiter
	rateLimitWaitTimeout time.Duration

	common service
}

// RequestOption adjusts a single API request. Unlike Option (which configures
// the Client globally), RequestOption is passed per-call. All service methods
// accept a variadic ...RequestOption as their last parameter.
type RequestOption interface {
	applyRequest(*requestConfig)
}

type requestOptionFunc func(*requestConfig)

func (f requestOptionFunc) applyRequest(cfg *requestConfig) {
	f(cfg)
}

type requestConfig struct {
	requestID *string
	include   []string
}

// WithRequestID manually sets the requestID for the API request. The requestID can also be omitted from the request by passing "" as the id string. When a custom requestID is not passed, requests default to using a uuid.
func WithRequestID(id string) RequestOption {
	return requestOptionFunc(func(cfg *requestConfig) {
		cfg.requestID = &id
	})
}

// WithInclude appends one or more values to the include query parameter of the
// request. Use the predefined Include* constants for known values:
//
//   - IncludeInvoiceLink — populate Invoice.InvoiceLink on read/query
//   - IncludeUpdateAccountOnTxns — propagate account changes to transactions (Customer, Vendor update)
//   - IncludeDoNotUpdateAccountOnTxns — prevent account propagation (Item update)
func WithInclude(values ...string) RequestOption {
	return requestOptionFunc(func(cfg *requestConfig) {
		cfg.include = append(cfg.include, values...)
	})
}

// CompanyClient is a Client bound to a specific QBO realm (company) and an
// oauth2.TokenSource. It is lightweight and can be created per-request if
// needed, but is also safe to reuse across goroutines.
//
// The TokenSource is responsible for refreshing the access token when it
// expires. If the refresh fails, the underlying request will return an error
// that wraps the TokenSource error.
//
// Obtain a CompanyClient via Client.Company.
type CompanyClient struct {
	client      *Client
	realmID     string
	tokenSource oauth2.TokenSource
}

// NewClient constructs a Client from the supplied options. All options are
// optional; defaults are:
//
//   - HTTP client: http.DefaultClient
//   - Base URL: DefaultBaseURL (production)
//   - Minor version: DefaultMinorVersion (75)
//   - Rate limiting: enabled with DefaultRateLimitConfig
//
// Supply WithOAuthConfig if you intend to use the OAuth helpers
// (AuthCodeURL, Exchange, TokenSource). API calls via Company do not require
// it — they use whatever oauth2.TokenSource the caller provides.
func NewClient(opts ...Option) (*Client, error) {
	cfg := clientConfig{
		baseURL:         DefaultBaseURL,
		minorVersion:    DefaultMinorVersion,
		transferTimeout: DefaultTransferTimeout,

		rateLimitConfig: DefaultRateLimitConfig(),
	}

	for _, opt := range opts {
		if err := opt.apply(&cfg); err != nil {
			return nil, err
		}
	}

	if cfg.minorVersion < DefaultMinorVersion {
		return nil, fmt.Errorf("minor version must be at least 75")
	}

	baseURL, err := url.Parse(cfg.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	var limiter RateLimiter

	if !cfg.rateLimiterDisabled {
		limiter = cfg.rateLimiter
		if limiter == nil {
			limiter, err = NewRealmRateLimiter(cfg.rateLimitConfig)
			if err != nil {
				return nil, err
			}
		}
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// transferClient shares httpClient's Transport (and redirect/cookie
	// policies) but drops the client-wide Timeout so large uploads/downloads
	// are bounded by transferTimeout via context instead. When httpClient has
	// no Timeout, reuse it directly.
	transferClient := httpClient
	if httpClient.Timeout != 0 {
		tc := *httpClient
		tc.Timeout = 0
		transferClient = &tc
	}

	client := &Client{
		httpClient:      httpClient,
		transferClient:  transferClient,
		transferTimeout: cfg.transferTimeout,
		baseURL:         baseURL,
		minorVersion:    cfg.minorVersion,
		oauthConfig:     cloneOAuthConfig(cfg.oauthConfig),

		rateLimiter:          limiter,
		rateLimitWaitTimeout: cfg.rateLimitConfig.WaitTimeout,
	}
	client.common.client = client

	return client, nil
}

// WithHTTPClient overrides the HTTP client used for all requests. Useful for
// testing with httptest.Server.Client() or for adding custom transports.
func WithHTTPClient(httpClient *http.Client) Option {
	return optionFunc(func(cfg *clientConfig) error {
		cfg.httpClient = httpClient
		return nil
	})
}

// WithTransferTimeout overrides the deadline applied to file uploads and
// streaming downloads. These run on a copy of the HTTP client with no
// http.Client.Timeout, so this context-based deadline governs them instead of
// the (typically short) client-wide timeout. A non-positive value disables the
// deadline, relying solely on the caller's context. Defaults to
// DefaultTransferTimeout.
func WithTransferTimeout(d time.Duration) Option {
	return optionFunc(func(cfg *clientConfig) error {
		cfg.transferTimeout = d
		return nil
	})
}

// WithBaseURL overrides the API base URL. The default is DefaultBaseURL
// (production). For sandbox use SandboxBaseURL, or supply a custom URL for
// testing.
func WithBaseURL(baseURL string) Option {
	return optionFunc(func(cfg *clientConfig) error {
		cfg.baseURL = baseURL
		return nil
	})
}

// WithMinorVersion sets the QBO API minor version. Must be at least 75 (the
// minimum supported by this SDK). Defaults to DefaultMinorVersion.
func WithMinorVersion(minorVersion int) Option {
	return optionFunc(func(cfg *clientConfig) error {
		cfg.minorVersion = minorVersion
		return nil
	})
}

// WithOAuthConfig configures the OAuth2 client used by AuthCodeURL, Exchange,
// and TokenSource. It is not required for making API calls — those use whatever
// oauth2.TokenSource is passed to Company.
//
// Pass empty strings for authURL and tokenURL to use the Intuit production
// defaults (DefaultAuthURL and DefaultTokenURL). Use the endpoints returned by
// Discovery to point at sandbox or custom endpoints instead.
//
// If any of the OpenID scopes (OpenIDProfile, OpenIDEmail, OpenIDPhone,
// OpenIDAddress) are included, the "openid" scope is automatically appended.
func WithOAuthConfig(clientID, clientSecret, redirectURL, authURL, tokenURL string, scopes ...string) Option {
	if authURL == "" {
		authURL = DefaultAuthURL
	}
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}

	openid := false
	for _, scope := range scopes {
		switch scope {
		case OpenIDProfile, OpenIDEmail, OpenIDPhone, OpenIDAddress:
			openid = true
		}
	}
	if openid {
		scopes = append(scopes, "openid")
	}

	return optionFunc(func(cfg *clientConfig) error {
		cfg.oauthConfig = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       append([]string(nil), scopes...),
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
		}
		return nil
	})
}

// WithRateLimitConfig replaces the default rate-limit configuration. Zero
// values in cfg are filled with defaults from DefaultRateLimitConfig.
// Has no effect when rate limiting is disabled via WithNoRateLimiting.
func WithRateLimitConfig(cfg RateLimitConfig) Option {
	return optionFunc(func(c *clientConfig) error {
		c.rateLimitConfig = cfg
		return nil
	})
}

// WithRateLimiting explicitly enables rate limiting (it is on by default).
// Useful to re-enable after WithNoRateLimiting in a config chain.
func WithRateLimiting() Option {
	return optionFunc(func(cfg *clientConfig) error {
		cfg.rateLimiterDisabled = false
		return nil
	})
}

// WithNoRateLimiting disables the built-in rate limiter entirely. Useful in
// tests or when the caller manages throttling externally. Be careful in
// production — the QBO API enforces its own limits and will return 429s.
func WithNoRateLimiting() Option {
	return optionFunc(func(cfg *clientConfig) error {
		cfg.rateLimiterDisabled = true
		return nil
	})
}

// WithRateLimiter supplies a custom RateLimiter, replacing the default
// RealmRateLimiter. Use this to integrate with an external limiter or for
// deterministic testing.
func WithRateLimiter(limiter RateLimiter) Option {
	return optionFunc(func(cfg *clientConfig) error {
		cfg.rateLimiter = limiter
		return nil
	})
}

// OAuthConfig returns a copy of the OAuth2 config set via WithOAuthConfig, or
// nil if OAuth was not configured.
func (c *Client) OAuthConfig() *oauth2.Config {
	return cloneOAuthConfig(c.oauthConfig)
}

// AuthCodeURL returns the Intuit OAuth2 authorization URL. The user should be
// redirected to this URL to grant access. Requires WithOAuthConfig.
func (c *Client) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) (string, error) {
	if c.oauthConfig == nil {
		return "", ErrOAuthConfigNotSet
	}
	return c.oauthConfig.AuthCodeURL(state, opts...), nil
}

// AuthCodeURLWithRedirect is like AuthCodeURL but overrides the redirect URL
// for this call only, without modifying the Client's OAuth config.
func (c *Client) AuthCodeURLWithRedirect(state string, redirectURL string, opts ...oauth2.AuthCodeOption) (string, error) {
	if c.oauthConfig == nil {
		return "", ErrOAuthConfigNotSet
	}

	cfg := cloneOAuthConfig(c.oauthConfig)

	cfg.RedirectURL = redirectURL

	return cfg.AuthCodeURL(state, opts...), nil
}

// Exchange converts an authorization code (received at the redirect URL) into
// an oauth2.Token. Requires WithOAuthConfig.
func (c *Client) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	if c.oauthConfig == nil {
		return nil, ErrOAuthConfigNotSet
	}
	return c.oauthConfig.Exchange(ctx, code, opts...)
}

// ExchangeWithRedirect is like Exchange but overrides the redirect URL for
// this call only.
func (c *Client) ExchangeWithRedirect(ctx context.Context, code string, redirectURL string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	if c.oauthConfig == nil {
		return nil, ErrOAuthConfigNotSet
	}

	cfg := cloneOAuthConfig(c.oauthConfig)
	cfg.RedirectURL = redirectURL

	return cfg.Exchange(ctx, code, opts...)
}

// TokenSource returns a reusing, auto-refreshing oauth2.TokenSource from a
// previously obtained token. Pass the result to Company for ongoing API calls.
// Requires WithOAuthConfig.
func (c *Client) TokenSource(ctx context.Context, token *oauth2.Token) (oauth2.TokenSource, error) {
	if c.oauthConfig == nil {
		return nil, ErrOAuthConfigNotSet
	}
	return c.oauthConfig.TokenSource(ctx, token), nil
}

// Company returns a CompanyClient bound to the given realm and token source.
// The returned CompanyClient does not retain any mutable state beyond those
// two values, so it is safe to create one per request if preferred.
func (c *Client) Company(realmID string, tokenSource oauth2.TokenSource) *CompanyClient {
	return &CompanyClient{
		client:      c,
		realmID:     realmID,
		tokenSource: tokenSource,
	}
}

// Client returns the underlying Client from which this CompanyClient was created.
func (c *CompanyClient) Client() *Client {
	return c.client
}

// HTTPClient returns the underlying *http.Client used by this Client. Callers
// can use it to make requests outside the QBO API (e.g. fetching a file from
// a pre-signed S3 URL returned by Attachable.Download) while reusing the same
// transport configuration.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// RealmID returns the QBO realm (company) ID bound to this CompanyClient.
func (c *CompanyClient) RealmID() string {
	return c.realmID
}

// TokenSource returns the oauth2.TokenSource bound to this CompanyClient.
func (c *CompanyClient) TokenSource() oauth2.TokenSource {
	return c.tokenSource
}

func (c *CompanyClient) doJSON(ctx context.Context, method, path string, query url.Values, payload any, out any, opts ...RequestOption) error {
	return c.client.doJSON(ctx, method, c.realmID, path, c.tokenSource, query, payload, out, opts...)
}

func (c *CompanyClient) doBytes(ctx context.Context, method, path string, query url.Values, contentType string, opts ...RequestOption) ([]byte, error) {
	return c.client.doBytes(ctx, method, c.realmID, path, c.tokenSource, query, contentType, opts...)
}

func (c *CompanyClient) doStream(ctx context.Context, method, path string, query url.Values, contentType string, opts ...RequestOption) (io.ReadCloser, error) {
	return c.client.doStream(ctx, method, c.realmID, path, c.tokenSource, query, contentType, opts...)
}

func (c *CompanyClient) doMultipart(ctx context.Context, path string, query url.Values, body io.Reader, contentType string, opts ...RequestOption) ([]byte, error) {
	return c.client.doMultipart(ctx, c.realmID, path, c.tokenSource, query, body, contentType, opts...)
}

// prepareRequest validates credentials, resolves the request ID, acquires the
// rate limiter, and returns a ready-to-execute *http.Request together with a
// release function that must be called when the request is complete.
//
// body and contentType describe the request body. Pass nil body and an empty
// contentType for requests with no body (e.g. GET). The caller is responsible
// for marshalling the payload before calling prepareRequest.
func (c *Client) prepareRequest(ctx context.Context, method, realmID, requestPath string, tokenSource oauth2.TokenSource, query url.Values, body io.Reader, contentType, accept string, opts ...RequestOption) (*http.Request, string, func(), error) {
	if strings.TrimSpace(realmID) == "" {
		return nil, "", nil, ErrRealmIDRequired
	}
	if tokenSource == nil {
		return nil, "", nil, ErrTokenSourceNil
	}

	isBatch := strings.Trim(requestPath, "/") == "batch"

	var cfg requestConfig
	for _, opt := range opts {
		opt.applyRequest(&cfg)
	}

	rpt, err := validateRequestID(isBatch, cfg.requestID)
	if err != nil {
		return nil, "", nil, err
	}

	requestID := applyRequestID(method, rpt)

	// Merge caller-supplied include values into the query before building URL.
	if len(cfg.include) > 0 {
		merged := cloneValues(query)
		merged.Set("include", strings.Join(cfg.include, ","))
		query = merged
	}

	requestURL := c.requestURL(realmID, requestPath, query, requestID)

	token, err := tokenSource.Token()
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to fetch access token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	scope := RequestScope{
		RealmID:  realmID,
		Endpoint: strings.Trim(requestPath, "/"),
		IsBatch:  isBatch,
	}

	limiterCtx := ctx
	var cancel context.CancelFunc
	if c.rateLimitWaitTimeout > 0 {
		limiterCtx, cancel = context.WithTimeout(ctx, c.rateLimitWaitTimeout)
	}

	release := func() {
		if cancel != nil {
			cancel()
		}
	}

	if c.rateLimiter != nil {
		rl, err := c.rateLimiter.Acquire(limiterCtx, scope)
		if err != nil {
			release()
			return nil, "", nil, err
		}
		inner := release
		release = func() {
			rl()
			inner()
		}
	}

	return req, requestID, release, nil
}

// transferContext derives a context bounded by the client's transfer timeout,
// or returns the parent context unchanged when the timeout is non-positive.
// Used for uploads and streaming downloads.
func (c *Client) transferContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.transferTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.transferTimeout)
}

// doRequest executes a prepared request, reads the full response body, checks
// for API errors, and returns the raw bytes. It is the shared foundation for
// doJSON, doBytes, and doMultipart.
func (c *Client) doRequest(ctx context.Context, method, realmID, requestPath string, tokenSource oauth2.TokenSource, query url.Values, body io.Reader, contentType, accept string, opts ...RequestOption) (string, []byte, error) {
	return c.doRequestWithClient(ctx, c.httpClient, method, realmID, requestPath, tokenSource, query, body, contentType, accept, opts...)
}

// doRequestWithClient is doRequest parameterised by the HTTP client to use, so
// uploads can run on the transfer client (no client-wide Timeout).
func (c *Client) doRequestWithClient(ctx context.Context, httpClient *http.Client, method, realmID, requestPath string, tokenSource oauth2.TokenSource, query url.Values, body io.Reader, contentType, accept string, opts ...RequestOption) (string, []byte, error) {
	req, requestID, release, err := c.prepareRequest(ctx, method, realmID, requestPath, tokenSource, query, body, contentType, accept, opts...)
	if err != nil {
		return "", nil, err
	}
	defer release()

	resp, err := httpClient.Do(req)
	if err != nil {
		return requestID, nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := readResponseBody(resp)
	if err != nil {
		return requestID, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	apiErr := newAPIError(resp.StatusCode, resp.Header, rawBody)

	if resp.StatusCode == http.StatusTooManyRequests {
		return requestID, nil, RateLimitError{
			APIError:   *apiErr,
			RealmID:    realmID,
			RequestID:  requestID,
			RetryAfter: resp.Header.Get("Retry-After"),
		}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return requestID, nil, apiErr
	}
	// QBO answers some rejected requests with a 200 whose body is a fault;
	// those are errors. The faults of individual batch items are not: the
	// other items succeeded, and the caller reads each item's fault from the
	// BatchResponse.
	if apiErr.Failure != nil && parseBodyFault(rawBody) != nil {
		return requestID, nil, apiErr
	}

	return requestID, rawBody, nil
}

func (c *Client) doJSON(ctx context.Context, method, realmID, requestPath string, tokenSource oauth2.TokenSource, query url.Values, payload any, out any, opts ...RequestOption) error {
	var body io.Reader
	var contentType string
	if payload != nil {
		marshaled, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewReader(marshaled)
		contentType = "application/json"
	}
	_, rawBody, err := c.doRequest(ctx, method, realmID, requestPath, tokenSource, query, body, contentType, "application/json", opts...)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(rawBody, out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

// doBytes executes a request and returns the full response body as a byte
// slice. accept is sent as the Accept header and should match the expected
// response format (e.g. "application/pdf").
func (c *Client) doBytes(ctx context.Context, method, realmID, requestPath string, tokenSource oauth2.TokenSource, query url.Values, accept string, opts ...RequestOption) ([]byte, error) {
	_, raw, err := c.doRequest(ctx, method, realmID, requestPath, tokenSource, query, nil, "", accept, opts...)
	return raw, err
}

// doStream executes a request and returns the response body as an
// io.ReadCloser for streaming. accept is sent as the Accept header.
// The caller is responsible for closing the returned ReadCloser, which also
// releases the rate-limiter slot.
// On error the body is fully consumed and closed before returning.
func (c *Client) doStream(ctx context.Context, method, realmID, requestPath string, tokenSource oauth2.TokenSource, query url.Values, accept string, opts ...RequestOption) (io.ReadCloser, error) {
	ctx, cancel := c.transferContext(ctx)

	req, requestID, release, err := c.prepareRequest(ctx, method, realmID, requestPath, tokenSource, query, nil, "", accept, opts...)
	if err != nil {
		cancel()
		return nil, err
	}

	// Release the transfer deadline together with the rate-limiter slot —
	// either on an error path below or when the returned stream is closed.
	baseRelease := release
	release = func() {
		baseRelease()
		cancel()
	}

	resp, err := c.transferClient.Do(req)
	if err != nil {
		release()
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// On non-2xx responses read the body to build a useful error, then close
	// everything before returning.
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		rawBody, _ := readResponseBody(resp)
		resp.Body.Close()
		release()
		apiErr := newAPIError(resp.StatusCode, resp.Header, rawBody)
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, RateLimitError{
				APIError:   *apiErr,
				RealmID:    realmID,
				RequestID:  requestID,
				RetryAfter: resp.Header.Get("Retry-After"),
			}
		}
		return nil, apiErr
	}

	// Wrap in a gzip reader if the server compressed the response. Chain the
	// rate-limiter release so the caller's Close() handles everything.
	reader, err := responseReader(resp)
	if err != nil {
		resp.Body.Close()
		release()
		return nil, fmt.Errorf("failed to create response reader: %w", err)
	}

	return &streamReadCloser{ReadCloser: reader, release: release}, nil
}

// doMultipart executes a multipart/form-data POST request and returns the
// full response body as bytes. The caller is responsible for constructing
// body (e.g. via mime/multipart) and providing the full Content-Type header
// value including the multipart boundary. The response is expected to be JSON.
func (c *Client) doMultipart(ctx context.Context, realmID, requestPath string, tokenSource oauth2.TokenSource, query url.Values, body io.Reader, contentType string, opts ...RequestOption) ([]byte, error) {
	ctx, cancel := c.transferContext(ctx)
	defer cancel()
	_, raw, err := c.doRequestWithClient(ctx, c.transferClient, http.MethodPost, realmID, requestPath, tokenSource, query, body, contentType, "application/json", opts...)
	return raw, err
}

// streamReadCloser wraps an io.ReadCloser and calls a release function (e.g.
// to free the rate-limiter slot) exactly once when the stream is closed.
type streamReadCloser struct {
	io.ReadCloser
	release func()
	once    sync.Once
}

func (s *streamReadCloser) Close() error {
	err := s.ReadCloser.Close()
	s.once.Do(s.release)
	return err
}

func (c *Client) requestURL(realmID, requestPath string, query url.Values, requestID string) *url.URL {
	requestURL := *c.baseURL
	requestURL.Path = path.Join(strings.TrimSuffix(c.baseURL.Path, "/"), realmID, strings.TrimPrefix(requestPath, "/"))

	values := cloneValues(query)
	values.Set("minorversion", strconv.Itoa(c.minorVersion))
	if requestID != "" {
		values.Set("requestid", requestID)
	}

	requestURL.RawQuery = values.Encode()
	return &requestURL
}

// validateRequestID validates a caller-supplied request ID pointer (as stored
// in requestConfig) against the length limits for the given request type. It
// returns the validated pointer unchanged, or an error if the ID is too long.
func validateRequestID(isBatch bool, requestID *string) (*string, error) {
	if requestID == nil {
		return nil, nil // nothing set — caller wants auto-generation or default
	}
	id := *requestID
	if id == "" {
		return &id, nil // explicit opt-out
	}
	maxLen := RequestIDMaxLength
	if isBatch {
		maxLen = BatchRequestIDMaxLength
	}
	if len(id) > maxLen {
		return nil, fmt.Errorf("request id %q exceeds maximum length of %d characters", id, maxLen)
	}
	return &id, nil
}

func applyRequestID(method string, intent *string) string {
	if intent != nil {
		return *intent // "" (opt-out) or caller-supplied ID
	}
	if !isMutationMethod(method) {
		return ""
	}
	return uuid.NewString() // generated exactly once, right before use
}

func isMutationMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func cloneValues(src url.Values) url.Values {
	if src == nil {
		return url.Values{}
	}
	dst := url.Values{}
	for k, values := range src {
		dst[k] = append([]string(nil), values...)
	}
	return dst
}

func cloneOAuthConfig(cfg *oauth2.Config) *oauth2.Config {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.Scopes = append([]string(nil), cfg.Scopes...)
	return &clone
}

func responseReader(resp *http.Response) (io.ReadCloser, error) {
	if resp.Header.Get("Content-Encoding") != "gzip" {
		return resp.Body, nil
	}

	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	return reader, nil
}
