package security

import (
	"net/http"
	"strings"
)

// allowedRequestHeaders are the headers the mobile app may send. Preflight
// responses must list them explicitly: a browser refuses any header it did not
// get permission for, and the app uploads carry attachment metadata this way.
var allowedRequestHeaders = strings.Join([]string{
	"Content-Type",
	"Authorization",
	"X-Kisy-Client",
	"X-File-Name",
	"X-Note-Text",
	"X-Attachment-Kind",
	"X-Attachment-Duration-Ms",
	"X-Attachment-Waveform",
	"X-Attachment-Width",
	"X-Attachment-Height",
}, ", ")

// CORS answers cross-origin requests from the packaged mobile apps.
//
// The web app is same-origin and needs none of this; the WebView shell is not:
// it runs on its own origin (https://localhost), so every API call is
// cross-origin and the browser first sends an OPTIONS preflight. Without a
// reply carrying Access-Control-* the WebView blocks the request outright —
// which looks to the user like a failed login rather than a network error.
//
// Only origins on the allowlist are echoed back, never "*", and credentials
// are deliberately NOT allowed: native clients authenticate with a Bearer
// token, so no cookie should ever ride along on a cross-site request.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" || !originInList(origin, allowedOrigins) {
				// Same-origin (web) or an origin we do not serve: leave the
				// response untouched so nothing is widened by accident.
				if r.Method == http.MethodOptions && origin != "" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			// The response varies per origin, so caches must not share it.
			h.Add("Vary", "Origin")

			if r.Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", allowedRequestHeaders)
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func originInList(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a != "" && strings.EqualFold(origin, a) {
			return true
		}
	}
	return false
}
