package quickbooks

import (
	"context"
	"testing"
	"time"
)

func TestRealmRateLimiterDelay(t *testing.T) {
	m, err := NewRealmRateLimiter(RateLimitConfig{BatchPerMinute: 60, BatchBurst: 2})
	if err != nil {
		t.Fatal(err)
	}
	batch := RequestScope{RealmID: "realm-1", Endpoint: "batch", IsBatch: true}
	query := RequestScope{RealmID: "realm-1", Endpoint: "query"}

	if d := m.Delay(batch); d != 0 {
		t.Fatalf("a fresh realm should have no delay, got %v", d)
	}
	for range 2 {
		release, err := m.Acquire(context.Background(), batch)
		if err != nil {
			t.Fatal(err)
		}
		release()
	}

	// The batch burst is spent; at 60 a minute the next token is about a
	// second away. General requests are unaffected.
	if d := m.Delay(batch); d < 900*time.Millisecond || d > time.Second {
		t.Fatalf("want about a second until the next batch call, got %v", d)
	}
	if d := m.Delay(query); d != 0 {
		t.Fatalf("general requests should not wait, got %v", d)
	}
	// Delay takes no token: asking again gives the same answer.
	if d := m.Delay(batch); d < 800*time.Millisecond {
		t.Fatalf("Delay should not take a token, got %v", d)
	}
	if d := m.Delay(RequestScope{RealmID: "realm-2", IsBatch: true}); d != 0 {
		t.Fatalf("another realm has its own allowance, got %v", d)
	}
}
