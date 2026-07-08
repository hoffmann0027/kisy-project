// Package ratelimit implements a fixed-window per-IP rate limiter backed
// by Redis, applied to authentication endpoints per
// docs/spec/06-security.md ("Rate limiting", "brute-force detection").
package ratelimit

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"kisy-backend/pkg/httpresponse"
)

type Limiter struct {
	rdb *redis.Client
	log *slog.Logger
}

func NewLimiter(rdb *redis.Client, log *slog.Logger) *Limiter {
	return &Limiter{rdb: rdb, log: log}
}

// Limit returns middleware allowing at most max requests per window per
// client IP for the named scope. Redis outages fail open: availability of
// login is preferred over strict limiting, and the event is logged for
// alerting.
func (l *Limiter) Limit(scope string, max int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !l.Allow(r.Context(), scope, clientIP(r), max, window) {
				httpresponse.Fail(w, r, http.StatusTooManyRequests, httpresponse.ErrRateLimited, "too many requests, slow down")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Allow reports whether an action identified by (scope, key) is still within
// max occurrences per fixed window. It backs both the HTTP middleware (keyed by
// client IP) and non-HTTP callers such as the WebSocket call-signaling path
// (keyed by user id). Redis outages fail open (return true) — availability is
// preferred over strict limiting, and the event is logged for alerting.
func (l *Limiter) Allow(ctx context.Context, scope, key string, max int, window time.Duration) bool {
	rkey := "rl:" + scope + ":" + key
	count, err := l.rdb.Incr(ctx, rkey).Result()
	if err != nil {
		l.log.Warn("rate limiter unavailable, failing open", "scope", scope, "error", err)
		return true
	}
	if count == 1 {
		if err := l.rdb.Expire(ctx, rkey, window).Err(); err != nil {
			l.log.Warn("rate limiter expire failed", "scope", scope, "error", err)
		}
	}
	return count <= int64(max)
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
