// Package security provides HTTP hardening middleware: response security
// headers and CSRF protection (docs/spec/06-security.md "Application
// Security"). These are defense-in-depth: the Nginx edge sets the same
// headers, but the backend must be safe even if fronted differently.
package security

import (
	"net/http"
	"strings"
)

// Content-Security-Policy for the SPA. Scripts are same-origin only (Vite
// emits external bundles, no inline scripts). Inline styles are permitted
// because React renders style attributes; everything else is locked to
// 'self'. challenges.cloudflare.com is the Turnstile widget on the sign-up
// screen: its script, and the iframe the challenge runs in.
//
// 'wasm-unsafe-eval' lets WebAssembly compile and nothing else — no eval, no
// new Function. libsodium, which every E2EE key goes through, is WebAssembly
// and has no fallback: without it sodium.ready rejects and a browser cannot
// start an encrypted session at all, so a private chat refuses to send.
//
// connect-src is 'self' plus the WebSocket gateway on this very host. It used
// to be "ws: wss:" — a socket to any host on earth, which is the exfiltration
// channel an XSS needs (audit A-45). Spelling the host out rather than
// trusting 'self' to cover ws/wss keeps older WebKit working.
//
// deploy/nginx must match, with $host for the host (TestNginxCSPMatchesBackend).
const (
	cspHead = "default-src 'self'; " +
		"script-src 'self' 'wasm-unsafe-eval' https://challenges.cloudflare.com; " +
		"frame-src https://challenges.cloudflare.com; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: blob:; " +
		"font-src 'self'; "
	cspTail = "object-src 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"frame-ancestors 'none'"
)

// contentSecurityPolicy is the policy for a page served as host. A Host
// header that is not a plain host[:port] gets no WebSocket source at all —
// it cannot be spliced into the policy as a directive of its own.
func contentSecurityPolicy(host string) string {
	connect := "connect-src 'self'; "
	if plainHost(host) {
		connect = "connect-src 'self' ws://" + host + " wss://" + host + "; "
	}
	return cspHead + connect + cspTail
}

// plainHost accepts a DNS name or IP literal with an optional port.
func plainHost(host string) bool {
	if host == "" || len(host) > 255 {
		return false
	}
	for _, c := range host {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.ContainsRune(".-:[]", c):
		default:
			return false
		}
	}
	return true
}

// Headers sets response security headers on every response.
func Headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy(r.Host))
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// microphone=(self): required for WebRTC audio calls (getUserMedia) on
		// the same origin. Camera/geolocation/payment stay fully disabled.
		h.Set("Permissions-Policy", "geolocation=(), microphone=(self), camera=(), payment=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		// HSTS is only honored over HTTPS; harmless over plain HTTP and
		// correct once TLS terminates at the edge.
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// NoStore keeps API responses out of every cache — the browser's, a
// proxy's, a shared device's (audit A-45): they carry messages, profiles and
// tokens. A handler serving a file sets its own Cache-Control, which wins.
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
