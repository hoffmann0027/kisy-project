// Package ratelimit implements a fixed-window rate limiter backed by Redis:
// per client IP for the HTTP middleware, per any key (an account, a socket)
// through Take and Allow. Per docs/spec/06-security.md ("Rate limiting",
// "brute-force detection").
package ratelimit

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strconv"
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

// Decision is the outcome of one counted hit.
type Decision struct {
	// Within: the hit is inside the budget.
	Within bool
	// Available: the limiter could answer at all. When false, Within is true
	// and it is the caller's choice what an outage means (see LimitStrict).
	Available bool
	// RetryAfter is how long until the window resets; set when refused.
	RetryAfter time.Duration
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
			d := l.Take(r.Context(), scope, Bucket(clientIP(r)), max, window)
			if !d.Available && failClosed {
				httpresponse.Fail(w, r, http.StatusServiceUnavailable, httpresponse.ErrInternal,
					"service temporarily unavailable, try again shortly")
				return
			}
			if !d.Within {
				Refuse(w, r, d.RetryAfter)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Refuse writes the 429 every limit answers with. Retry-After is in whole
// seconds, rounded up and at least 1, so a client that honours it is never
// refused again for the same window.
func Refuse(w http.ResponseWriter, r *http.Request, retryAfter time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(RetryAfterSeconds(retryAfter)))
	httpresponse.Fail(w, r, http.StatusTooManyRequests, httpresponse.ErrRateLimited, "too many requests, slow down")
}

// RetryAfterSeconds rounds a wait up to whole seconds, minimum 1.
func RetryAfterSeconds(d time.Duration) int {
	s := int((d + time.Second - 1) / time.Second)
	if s < 1 {
		return 1
	}
	return s
}

// Allow reports whether an action identified by (scope, key) is still within
// max occurrences per fixed window. Redis outages fail open (return true) —
// availability is preferred over strict limiting, and the event is logged.
func (l *Limiter) Allow(ctx context.Context, scope, key string, max int, window time.Duration) bool {
	return l.Take(ctx, scope, key, max, window).Within
}

// hitScript counts a hit and guarantees the counter expires, in one atomic
// step (audit B-28). The old INCR-then-EXPIRE left a key without a TTL when
// the second call failed, and that key refused its owner forever; here the
// TTL is (re)set whenever it is missing, which also heals such a key.
var hitScript = redis.NewScript(`
local c = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
if ttl < 0 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
  ttl = tonumber(ARGV[1])
end
return {c, ttl}
`)

// Take counts one hit against (scope, key) and reports the decision.
func (l *Limiter) Take(ctx context.Context, scope, key string, max int, window time.Duration) Decision {
	rkey := "rl:" + scope + ":" + key
	res, err := hitScript.Run(ctx, l.rdb, []string{rkey}, window.Milliseconds()).Int64Slice()
	if err != nil || len(res) != 2 {
		l.log.Warn("rate limiter unavailable", "scope", scope, "error", err)
		return Decision{Within: true, Available: false}
	}
	d := Decision{Within: res[0] <= int64(max), Available: true}
	if !d.Within {
		d.RetryAfter = time.Duration(res[1]) * time.Millisecond
	}
	return d
}

// Bucket is the rate-limit key for a client address. IPv4 is per address.
// IPv6 is per /64: that is what one subscriber is normally given, and they can
// pick any address inside it — per-address buckets would hand them 2^64 fresh
// allowances (audit A-23).
func Bucket(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() != nil {
		return ip
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

func clientIP(r *http.Request) string { return clientip.From(r) }
