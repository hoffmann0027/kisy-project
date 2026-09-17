package main

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"kisy-backend/pkg/httpresponse"
)

// serverTimeouts bound how long one connection may hold the server without
// progress (audit B-10). Only ReadHeaderTimeout existed before: a client could
// open a request and dribble its body for as long as it liked, or keep idle
// keep-alive connections open indefinitely, and each such connection costs a
// goroutine and a file descriptor.
type serverTimeouts struct {
	// ReadHeader: headers must arrive promptly — slowloris.
	ReadHeader time.Duration
	// Read: the whole request, body included. Generous enough for the largest
	// upload chunk over a slow mobile uplink. A WebSocket sets its own read
	// deadline right after the upgrade, so this does not cut sockets.
	Read time.Duration
	// Idle: a keep-alive connection with no request in flight.
	Idle time.Duration
}

var defaultServerTimeouts = serverTimeouts{
	ReadHeader: 5 * time.Second,
	Read:       60 * time.Second,
	Idle:       120 * time.Second,
}

func newHTTPServer(port int, handler http.Handler, t serverTimeouts) *http.Server {
	return &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           handler,
		ReadHeaderTimeout: t.ReadHeader,
		ReadTimeout:       t.Read,
		IdleTimeout:       t.Idle,
	}
}

// maxJSONBody caps every request body except the upload routes below. JSON
// decoding already stopped at 1 MiB (pkg/httpjson), but only for handlers that
// decode; this refuses an oversized body up front, before authentication, for
// every route.
const maxJSONBody = 1 << 20

// uploadRoutes are the endpoints that legitimately take more than
// maxJSONBody. Each caps its own body by the account's upload limits and
// storage quota (internal/quota), so they are exempt here — and only they.
var uploadRoutes = []struct {
	method string
	path   *regexp.Regexp
}{
	{http.MethodPost, regexp.MustCompile(`^/api/v1/attachments$`)},
	{http.MethodPut, regexp.MustCompile(`^/api/v1/attachments/[^/]+/chunk$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/notes/file$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/posts/[^/]+/media$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/users/me/avatar$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/groups/[^/]+/avatar$`)},
}

func isUploadRoute(r *http.Request) bool {
	for _, u := range uploadRoutes {
		if r.Method == u.method && u.path.MatchString(r.URL.Path) {
			return true
		}
	}
	return false
}

// limitBody refuses a declared oversized body with 413 straight away and caps
// an undeclared (chunked) one while it is read, so no handler ever holds more
// than max bytes of a non-upload request.
func limitBody(max int64, exempt func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil || r.Body == http.NoBody || exempt(r) {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > max {
				// The rest of the body is not worth reading.
				w.Header().Set("Connection", "close")
				httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed,
					"request body too large")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, max)
			next.ServeHTTP(w, r)
		})
	}
}
