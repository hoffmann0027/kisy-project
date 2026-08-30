// Package clientip resolves the address a request came from. It is the
// single definition of "client IP" for the whole service, because two
// security controls key off it — the per-IP rate limits and the salted IP
// digest recorded on sessions and audit events — and they must never drift
// apart on what they consider trustworthy.
package clientip

import (
	"net"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// From returns the client address for a request.
//
// The value comes from whichever ClientIPFrom* middleware the router
// installed (see cmd/server/main.go): the TCP peer by default, or the
// X-Forwarded-For entry beyond the configured trusted proxies when
// TRUSTED_PROXY_CIDRS is set. Both are set by us, never taken on the
// client's word — chi's own middleware.RealIP is deliberately not used
// because it honours True-Client-IP and X-Real-IP from any caller, which
// lets a request pick its own rate-limit bucket (GHSA-3fxj-6jh8-hvhx).
//
// The RemoteAddr fallback keeps callers correct when no middleware ran
// (unit tests, non-HTTP entry points) and when the XFF chain was
// unparseable — in that case everyone behind the proxy shares one bucket,
// which throttles honest traffic but never trusts a forged value.
func From(r *http.Request) string {
	if ip := middleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
