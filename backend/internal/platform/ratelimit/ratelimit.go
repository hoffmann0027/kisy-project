// Package ratelimit implements a fixed-window per-IP rate limiter backed
// by Redis, applied to authentication endpoints per
// docs/spec/06-security.md ("Rate limiting", "brute-force detection").
package ratelimit

import (
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
			key := "rl:" + scope + ":" + clientIP(r)

			count, err := l.rdb.Incr(r.Context(), key).Result()
			if err != nil {
				l.log.Warn("rate limiter unavailable, failing open", "scope", scope, "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				if err := l.rdb.Expire(r.Context(), key, window).Err(); err != nil {
					l.log.Warn("rate limiter expire failed", "scope", scope, "error", err)
				}
			}
			if count > int64(max) {
				httpresponse.Fail(w, r, http.StatusTooManyRequests, httpresponse.ErrRateLimited, "too many requests, slow down")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
