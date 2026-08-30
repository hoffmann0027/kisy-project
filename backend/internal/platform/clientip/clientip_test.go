package clientip_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"

	"kisy-backend/internal/platform/clientip"
)

// resolve runs a request through the given middleware chain (the same one
// cmd/server/main.go installs) and reports what the app would treat as the
// client address.
func resolve(mws []func(http.Handler) http.Handler, r *http.Request) string {
	var got string
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = clientip.From(r)
	})
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	h.ServeHTTP(httptest.NewRecorder(), r)
	return got
}

func request(remoteAddr string, headers map[string]string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.RemoteAddr = remoteAddr
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

// Without a configured proxy nothing in the request may move the client
// address: this is the deployment where the process faces the internet, so
// every header is attacker-supplied. Before the switch away from
// middleware.RealIP, True-Client-IP alone decided this value.
func TestNoTrustedProxyIgnoresHeaders(t *testing.T) {
	chain := []func(http.Handler) http.Handler{middleware.ClientIPFromRemoteAddr}

	headers := []map[string]string{
		{"True-Client-IP": "198.51.100.1"},
		{"X-Real-IP": "198.51.100.2"},
		{"X-Forwarded-For": "198.51.100.3"},
		{"True-Client-IP": "198.51.100.4", "X-Forwarded-For": "198.51.100.5, 198.51.100.6"},
	}
	for _, h := range headers {
		if got := resolve(chain, request("203.0.113.7:41234", h)); got != "203.0.113.7" {
			t.Errorf("headers %v steered the client IP to %q, want the TCP peer 203.0.113.7", h, got)
		}
	}
}

// Behind our own proxy the forwarded address is what we want — that is the
// whole point of running one — but only the entry the proxy itself appended.
func TestTrustedProxyUsesForwardedAddress(t *testing.T) {
	chain := []func(http.Handler) http.Handler{
		middleware.ClientIPFromRemoteAddr,
		middleware.ClientIPFromXFF("172.16.0.0/12"),
	}

	// nginx (172.23.0.1) overwrites XFF with the peer it saw.
	got := resolve(chain, request("172.23.0.5:5555", map[string]string{
		"X-Forwarded-For": "203.0.113.7",
	}))
	if got != "203.0.113.7" {
		t.Fatalf("client IP = %q, want the address the proxy forwarded", got)
	}
}

// A caller that prepends its own hops must not be able to hide behind them:
// chi walks right-to-left and stops at the first address outside the trusted
// networks, so the entry our proxy appended wins over anything left of it.
func TestTrustedProxyIgnoresPrependedHops(t *testing.T) {
	chain := []func(http.Handler) http.Handler{
		middleware.ClientIPFromRemoteAddr,
		middleware.ClientIPFromXFF("172.16.0.0/12"),
	}

	got := resolve(chain, request("172.23.0.5:5555", map[string]string{
		// "1.2.3.4" is the forgery; "203.0.113.7" is what nginx appended.
		"X-Forwarded-For": "1.2.3.4, 203.0.113.7",
	}))
	if got != "203.0.113.7" {
		t.Fatalf("client IP = %q, want 203.0.113.7 — a forged leading hop was believed", got)
	}
}

// True-Client-IP is not consulted in either mode. It is the header chi's
// deprecated RealIP preferred above all others, and no proxy of ours sets it.
func TestTrueClientIPNeverWins(t *testing.T) {
	chain := []func(http.Handler) http.Handler{
		middleware.ClientIPFromRemoteAddr,
		middleware.ClientIPFromXFF("172.16.0.0/12"),
	}

	got := resolve(chain, request("172.23.0.5:5555", map[string]string{
		"True-Client-IP":  "9.9.9.9",
		"X-Forwarded-For": "203.0.113.7",
	}))
	if got != "203.0.113.7" {
		t.Fatalf("client IP = %q, want 203.0.113.7", got)
	}
}

// An unusable chain must fall back to the peer rather than to whatever
// parsed: everyone behind the proxy then shares one rate-limit bucket, which
// throttles honest traffic but never trusts a forged value.
func TestUnparseableForwardedChainFallsBackToPeer(t *testing.T) {
	chain := []func(http.Handler) http.Handler{
		middleware.ClientIPFromRemoteAddr,
		middleware.ClientIPFromXFF("172.16.0.0/12"),
	}

	got := resolve(chain, request("172.23.0.5:5555", map[string]string{
		"X-Forwarded-For": "not-an-ip",
	}))
	if got != "172.23.0.5" {
		t.Fatalf("client IP = %q, want the peer 172.23.0.5", got)
	}
}

// Non-HTTP entry points and tests run without the middleware; the helper
// still has to answer with something usable.
func TestFallsBackToRemoteAddrWithoutMiddleware(t *testing.T) {
	if got := clientip.From(request("203.0.113.11:80", nil)); got != "203.0.113.11" {
		t.Fatalf("client IP = %q, want 203.0.113.11", got)
	}
	if got := clientip.From(request("203.0.113.12", nil)); got != "203.0.113.12" {
		t.Fatalf("bare address: client IP = %q, want 203.0.113.12", got)
	}
}
