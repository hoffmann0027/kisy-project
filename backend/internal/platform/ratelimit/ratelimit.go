// Package ratelimit implements a fixed-window per-IP rate limiter backed
// by Redis, applied to authentication endpoints per
// docs/spec/06-security.md ("Rate limiting", "brute-force detection").
package ratelimit

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/platform/clientip"
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
// client IP for the named scope. Redis outages fail open: for these scopes
// availability is worth more than a strict ceiling, and the event is logged
// for alerting. Credential endpoints want the opposite trade — see
// LimitStrict.
func (l *Limiter) Limit(scope string, max int, window time.Duration) func(http.Handler) http.Handler {
	return l.middleware(scope, max, window, false)
}

// LimitStrict is Limit for scopes where the ceiling is the security control
// itself — password guessing on login and account creation on register. If
// the limiter cannot answer, the request is refused rather than waved
// through: a Redis outage must not silently hand an attacker unlimited
// guesses. The refusal is 503, not 429, because the fault is ours, not the
// caller's; per-account lockout still applies underneath either way.
func (l *Limiter) LimitStrict(scope string, max int, window time.Duration) func(http.Handler) http.Handler {
	return l.middleware(scope, max, window, true)
}

func (l *Limiter) middleware(scope string, max int, window time.Duration, failClosed bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			within, available := l.count(r.Context(), scope, clientIP(r), max, window)
			if !available && failClosed {
				httpresponse.Fail(w, r, http.StatusServiceUnavailable, httpresponse.ErrInternal,
					"service temporarily unavailable, try again shortly")
				return
			}
			if !within {
				httpresponse.Fail(w, r, http.StatusTooManyRequests, httpresponse.ErrRateLimited, "too many requests, slow down")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Allow reports whether an action identified by (scope, key) is still within
// max occurrences per fixed window. It backs non-HTTP callers such as the
// WebSocket call-signaling path (keyed by user id). Redis outages fail open
// (return true) — availability is preferred over strict limiting, and the
// event is logged for alerting.
func (l *Limiter) Allow(ctx context.Context, scope, key string, max int, window time.Duration) bool {
	within, _ := l.count(ctx, scope, key, max, window)
	return within
}

// count increments the window counter and reports whether the caller is
// within budget, and whether the limiter could answer at all. Separating the
// two lets each caller choose what an outage means: Allow and Limit wave the
// request through, LimitStrict refuses it.
func (l *Limiter) count(ctx context.Context, scope, key string, max int, window time.Duration) (within, available bool) {
	rkey := "rl:" + scope + ":" + key
	c, err := l.rdb.Incr(ctx, rkey).Result()
	if err != nil {
		l.log.Warn("rate limiter unavailable", "scope", scope, "error", err)
		return true, false
	}
	if c == 1 {
		if err := l.rdb.Expire(ctx, rkey, window).Err(); err != nil {
			l.log.Warn("rate limiter expire failed", "scope", scope, "error", err)
		}
	}
	return c <= int64(max), true
}

func clientIP(r *http.Request) string { return clientip.From(r) }
