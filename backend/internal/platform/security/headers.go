// Package security provides HTTP hardening middleware: response security
// headers and CSRF protection (docs/spec/06-security.md "Application
// Security"). These are defense-in-depth: the Nginx edge sets the same
// headers, but the backend must be safe even if fronted differently.
package security

import "net/http"

// Content-Security-Policy for the SPA. Scripts are same-origin only (Vite
// emits external bundles, no inline scripts). Inline styles are permitted
// because React renders style attributes; everything else is locked to
// 'self'. connect-src includes ws/wss for the WebSocket gateway.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; " +
	"font-src 'self'; " +
	"connect-src 'self' ws: wss:; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

// Headers sets response security headers on every response.
func Headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
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
