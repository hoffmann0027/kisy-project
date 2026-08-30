package ratelimit

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// The limiter buckets by client IP, so whatever clientIP() returns is the
// security boundary: if a caller can steer it, every per-IP limit (login,
// register, refresh, link-preview, calls) is bypassed by varying one header.
// clientIP() must therefore read the transport-level peer address only and
// ignore identity headers, which are attacker-controlled on the wire.
func TestClientIPIgnoresSpoofableHeaders(t *testing.T) {
	const peer = "203.0.113.7:41234"
	const want = "203.0.113.7"

	headers := []struct {
		name, value string
	}{
		{"True-Client-IP", "198.51.100.1"},
		{"X-Real-IP", "198.51.100.2"},
		{"X-Forwarded-For", "198.51.100.3, 198.51.100.4"},
		{"Forwarded", "for=198.51.100.5"},
	}

	for _, h := range headers {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		r.RemoteAddr = peer
		r.Header.Set(h.name, h.value)

		if got := clientIP(r); got != want {
			t.Errorf("clientIP() with %s=%q = %q, want %q — header steers the rate-limit bucket",
				h.name, h.value, got, want)
		}
	}
}

// Two requests from one peer must share a bucket even when the attacker
// varies the spoofable header on every request — that is the brute-force
// case the limiter exists to stop.
func TestClientIPSameBucketAcrossForgedHeaders(t *testing.T) {
	const peer = "203.0.113.9:5555"

	first := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	first.RemoteAddr = peer
	first.Header.Set("True-Client-IP", "10.0.0.1")

	second := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	second.RemoteAddr = peer
	second.Header.Set("True-Client-IP", "10.0.0.2")

	if a, b := clientIP(first), clientIP(second); a != b {
		t.Fatalf("forged headers split the bucket: %q vs %q", a, b)
	}
}

// A bare address (no port) still yields a usable key rather than an empty
// one — an empty key would collapse every caller into a single bucket.
func TestClientIPWithoutPort(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.11"

	if got := clientIP(r); got != "203.0.113.11" {
		t.Fatalf("clientIP() = %q, want %q", got, "203.0.113.11")
	}
}

// deadLimiter points the limiter at a closed port so every Redis call
// errors — the outage path, exercised without mocks.
func deadLimiter(t *testing.T) *Limiter {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
		MaxRetries:   -1,
	})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewLimiter(rdb, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func probe(t *testing.T, mw func(http.Handler) http.Handler) (status int, reached bool) {
	t.Helper()
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.7:41234"
	h.ServeHTTP(rec, req)
	return rec.Code, reached
}

// Credential endpoints must not lose their brute-force ceiling because Redis
// blinked: with the limiter down an attacker would otherwise get unlimited
// guesses. Availability of login is worth less than the guarantee.
func TestLimitStrictFailsClosedOnOutage(t *testing.T) {
	status, reached := probe(t, deadLimiter(t).LimitStrict("auth-login", 10, time.Minute))

	if reached {
		t.Error("handler ran while the limiter was unavailable: brute-force ceiling is gone")
	}
	// 503, not 429: the client did nothing wrong, our dependency is down.
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", status, http.StatusServiceUnavailable)
	}
}

// Everything else keeps preferring availability — an outage must not take
// link previews, calls or e2ee key exchange down with it.
func TestLimitFailsOpenOnOutage(t *testing.T) {
	status, reached := probe(t, deadLimiter(t).Limit("link-preview", 30, time.Minute))

	if !reached {
		t.Error("handler was blocked by a limiter outage on a fail-open scope")
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want %d", status, http.StatusOK)
	}
}

// Allow backs non-HTTP callers (WS call signaling); its fail-open contract
// is unchanged by the strict HTTP variant.
func TestAllowStaysFailOpenOnOutage(t *testing.T) {
	if !deadLimiter(t).Allow(context.Background(), "call.invite", "user-id", 10, time.Minute) {
		t.Error("Allow() denied while Redis was down; non-HTTP callers must fail open")
	}
}
