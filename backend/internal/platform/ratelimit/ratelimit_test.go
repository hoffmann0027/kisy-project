package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
