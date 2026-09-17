//go:build integration

package ratelimit

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Run against a real Redis:
//
//	TEST_REDIS_URL=redis://localhost:6379/0 go test -tags integration ./internal/platform/ratelimit/

func liveLimiter(t *testing.T) (*Limiter, *redis.Client) {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set; skipping integration test")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(opts)
	t.Cleanup(func() { _ = rdb.Close() })
	return NewLimiter(rdb, slog.New(slog.NewTextHandler(io.Discard, nil))), rdb
}

func hit(t *testing.T, mw func(http.Handler) http.Handler, remote string) *httptest.ResponseRecorder {
	t.Helper()
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = remote
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// A refusal says when to come back: 429 with Retry-After in whole seconds,
// never zero and never beyond the window.
func TestRefusalCarriesRetryAfter(t *testing.T) {
	l, _ := liveLimiter(t)
	mw := l.Limit("t-retry-"+uuid.NewString(), 2, 30*time.Second)
	for i := 0; i < 2; i++ {
		if rec := hit(t, mw, "203.0.113.20:1"); rec.Code != http.StatusOK {
			t.Fatalf("request %d: %d", i+1, rec.Code)
		}
	}
	rec := hit(t, mw, "203.0.113.20:1")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("over the limit: %d, want 429", rec.Code)
	}
	secs, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	if err != nil || secs < 1 || secs > 30 {
		t.Fatalf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
}

// Audit B-28: a counter without a TTL never resets, and its owner is refused
// forever. Whatever left a key without one — a crash between INCR and EXPIRE
// under the old code, an operator's SET — the next hit must give it a TTL.
func TestCounterAlwaysExpires(t *testing.T) {
	l, rdb := liveLimiter(t)
	ctx := context.Background()
	scope := "t-ttl-" + uuid.NewString()
	key := "rl:" + scope + ":user"
	// The state the non-atomic INCR-then-EXPIRE could leave behind.
	if err := rdb.Set(ctx, key, 99, 0).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rdb.Del(ctx, key) })

	d := l.Take(ctx, scope, "user", 5, time.Minute)
	if d.Within {
		t.Fatal("a counter above the limit must still refuse")
	}
	if ttl := rdb.PTTL(ctx, key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("TTL after a hit = %v; the key would refuse forever", ttl)
	}
	if d.RetryAfter <= 0 || d.RetryAfter > time.Minute {
		t.Fatalf("RetryAfter = %v", d.RetryAfter)
	}
}

// A fresh counter gets its TTL in the same step as its first increment.
func TestFirstHitSetsTheWindow(t *testing.T) {
	l, rdb := liveLimiter(t)
	ctx := context.Background()
	scope := "t-first-" + uuid.NewString()
	d := l.Take(ctx, scope, "k", 3, 10*time.Second)
	if !d.Within || !d.Available {
		t.Fatalf("first hit: %+v", d)
	}
	if ttl := rdb.PTTL(ctx, "rl:"+scope+":k").Val(); ttl <= 0 || ttl > 10*time.Second {
		t.Fatalf("TTL = %v", ttl)
	}
}

// Audit A-23: an IPv6 subscriber gets a whole /64 and can rotate through it
// freely; per-address buckets would give them 2^64 fresh allowances.
func TestIPv6AddressesShareTheirSlash64(t *testing.T) {
	l, _ := liveLimiter(t)
	mw := l.Limit("t-v6-"+uuid.NewString(), 2, time.Minute)
	for i, addr := range []string{"[2001:db8:1:2::1]:1", "[2001:db8:1:2:ffff::9]:1"} {
		if rec := hit(t, mw, addr); rec.Code != http.StatusOK {
			t.Fatalf("request %d: %d", i+1, rec.Code)
		}
	}
	if rec := hit(t, mw, "[2001:db8:1:2:abcd::77]:1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("third address in the same /64: %d, want 429", rec.Code)
	}
	// A different /64 is a different subscriber.
	if rec := hit(t, mw, "[2001:db8:1:3::1]:1"); rec.Code != http.StatusOK {
		t.Fatalf("other /64: %d, want 200", rec.Code)
	}
	// IPv4 stays per address.
	if rec := hit(t, mw, "203.0.113.30:1"); rec.Code != http.StatusOK {
		t.Fatalf("ipv4: %d", rec.Code)
	}
}
