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
func CSRF(allowedOrigin string, extraAllowed ...string) func(http.Handler) http.Handler {
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

			if OriginAllowed(origin, r, allowedOrigin, extraAllowed...) {
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
// extraAllowed lists additional exact origins (the mobile shell's WebView
// origin, e.g. https://localhost) that may talk to this API. Those clients
// authenticate with a Bearer token rather than cookies, so admitting their
// origin does not widen the CSRF surface: a cross-site page cannot make the
// browser attach a token it does not hold, and the cookies stay SameSite=Strict.
func OriginAllowed(origin string, r *http.Request, allowedOrigin string, extraAllowed ...string) bool {
	if allowedOrigin != "" && strings.EqualFold(origin, allowedOrigin) {
		return true
	}
	for _, a := range extraAllowed {
		if a != "" && strings.EqualFold(origin, a) {
			return true
		}
	}
	if u, err := url.Parse(origin); err == nil && strings.EqualFold(u.Host, r.Host) {
		return true
	}
	return false
}
