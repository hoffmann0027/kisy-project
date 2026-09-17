package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"kisy-backend/pkg/httpresponse"
)

// expensiveDeadline bounds the requests whose cost is not bounded by their
// input: search, the feed and link previews (a full-text query, a ranking scan,
// an outbound fetch). Without it one slow query holds a database connection
// for the full 30-second router timeout, and a handful of them in parallel
// drain the pool for everyone.
const expensiveDeadline = 2 * time.Second

// withDeadline gives the handler d to answer. Whatever it writes after the
// deadline has passed — its own error for the cancelled query, or nothing at
// all — goes out as 503 with Retry-After, so the client knows it was load, not
// a bad request. A success that made it in time is left alone.
func withDeadline(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			dw := &deadlineWriter{ResponseWriter: w, r: r, ctx: ctx}
			next.ServeHTTP(dw, r.WithContext(ctx))
			if !dw.wroteHeader && errors.Is(ctx.Err(), context.DeadlineExceeded) {
				dw.overloaded()
			}
		})
	}
}

type deadlineWriter struct {
	http.ResponseWriter
	r           *http.Request
	ctx         context.Context
	wroteHeader bool
	// swallowed: the handler's own response was replaced; its body is dropped.
	swallowed bool
}

func (dw *deadlineWriter) WriteHeader(status int) {
	if dw.wroteHeader {
		return
	}
	if status >= http.StatusBadRequest && errors.Is(dw.ctx.Err(), context.DeadlineExceeded) {
		dw.overloaded()
		dw.swallowed = true
		return
	}
	dw.wroteHeader = true
	dw.ResponseWriter.WriteHeader(status)
}

func (dw *deadlineWriter) Write(b []byte) (int, error) {
	if !dw.wroteHeader {
		dw.WriteHeader(http.StatusOK)
	}
	if dw.swallowed {
		return len(b), nil
	}
	return dw.ResponseWriter.Write(b)
}

func (dw *deadlineWriter) overloaded() {
	dw.wroteHeader = true
	h := dw.ResponseWriter.Header()
	h.Del("Content-Length")
	h.Set("Retry-After", "1")
	httpresponse.Fail(dw.ResponseWriter, dw.r, http.StatusServiceUnavailable, httpresponse.ErrInternal,
		"service is busy, try again shortly")
}
