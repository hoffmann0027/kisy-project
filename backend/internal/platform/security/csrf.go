package security

import (
	"net/http"
	"net/url"
	"strings"

	"kisy-backend/pkg/httpresponse"
)

// CSRF mitigates cross-site request forgery for cookie-authenticated,
// state-changing requests by verifying the Origin (or Referer) header.
//
// Cookies are already SameSite=Strict, which by itself blocks cross-site
// cookie delivery; this is defense-in-depth. Browsers always send an
// Origin header on non-GET fetch/XHR, so a cross-site forgery is rejected
// here. Non-browser API clients (which cannot be driven by CSRF) omit
// Origin and are allowed through.
func CSRF(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = originOf(r.Header.Get("Referer"))
			}
			if origin == "" {
				// No browser context: not a CSRF vector.
				next.ServeHTTP(w, r)
				return
			}

			if OriginAllowed(origin, r, allowedOrigin) {
				next.ServeHTTP(w, r)
				return
			}

			httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "cross-origin request rejected")
		})
	}
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func originOf(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// OriginAllowed reports whether a browser-supplied Origin may talk to this
// server: either it equals the explicitly configured allowed origin, or it is
// same-origin with the request (the Origin host matches the request Host —
// behind the edge proxy the Host header carries the public host). It is shared
// by the CSRF middleware and the WebSocket handshake check so both enforce the
// same fail-closed policy.
func OriginAllowed(origin string, r *http.Request, allowedOrigin string) bool {
	if allowedOrigin != "" && strings.EqualFold(origin, allowedOrigin) {
		return true
	}
	if u, err := url.Parse(origin); err == nil && strings.EqualFold(u.Host, r.Host) {
		return true
	}
	return false
}
