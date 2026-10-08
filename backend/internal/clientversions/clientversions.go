// Package clientversions records which build of the mobile app each account
// runs, so the admin overview can say how many people are still on an old
// one — the number that decides when an old app's behaviour can be retired.
//
// The app sends X-Kisy-App-Version: "<versionName> (<versionCode>)" on every
// request. Writing that down per request would be a database write per
// request, so a Redis key per account and build lets it through at most once
// every Throttle. Old builds send nothing and are simply not counted.
package clientversions

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Header is what the app sends.
const Header = "X-Kisy-App-Version"

// Throttle: how often one account's build is written down at most.
const Throttle = 6 * time.Hour

// Window: how recent a report must be to count as someone still using it.
const Window = 30 * 24 * time.Hour

// "1.4.0 (29312345)" — a version name of plain characters and a numeric build.
var headerRe = regexp.MustCompile(`^([0-9A-Za-z._+\-]{1,64}) \(([0-9]{1,15})\)$`)

// Parse reads the header; ok is false for anything else.
func Parse(v string) (version string, build int64, ok bool) {
	m := headerRe.FindStringSubmatch(v)
	if m == nil {
		return "", 0, false
	}
	b, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return m[1], b, true
}

type Recorder struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
	log  *slog.Logger
}

func NewRecorder(pool *pgxpool.Pool, rdb *redis.Client, log *slog.Logger) *Recorder {
	return &Recorder{pool: pool, rdb: rdb, log: log}
}

// Record writes down that userID runs build, unless it did within Throttle.
func (r *Recorder) Record(ctx context.Context, userID uuid.UUID, version string, build int64) error {
	key := fmt.Sprintf("kisy:appver:%s:%d", userID, build)
	fresh, err := r.rdb.SetNX(ctx, key, 1, Throttle).Result()
	if err != nil {
		return fmt.Errorf("clientversions: throttle: %w", err)
	}
	if !fresh {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO client_versions (user_id, platform, version, build, last_seen_at)
		VALUES ($1, 'android', $2, $3, now())
		ON CONFLICT (user_id, platform) DO UPDATE
		SET version = EXCLUDED.version, build = EXCLUDED.build, last_seen_at = now()`,
		userID, version, build); err != nil {
		// Let the next request try again.
		_ = r.rdb.Del(ctx, key).Err()
		return fmt.Errorf("clientversions: record: %w", err)
	}
	return nil
}

// Middleware records the version of signed-in requests that carry one. It
// never fails a request: a lost record costs a number on a dashboard.
func (r *Recorder) Middleware(userID func(*http.Request) (uuid.UUID, bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if raw := req.Header.Get(Header); raw != "" {
				if version, build, ok := Parse(raw); ok {
					if id, signedIn := userID(req); signedIn {
						if err := r.Record(req.Context(), id, version, build); err != nil {
							r.log.Warn("clientversions: record", "error", err)
						}
					}
				}
			}
			next.ServeHTTP(w, req)
		})
	}
}

// Count is how many accounts used one build lately.
type Count struct {
	Version string `json:"version"`
	Build   int64  `json:"build"`
	Users   int    `json:"users"`
}

// Recent lists the Android builds used within Window, newest build first.
func Recent(ctx context.Context, pool *pgxpool.Pool) ([]Count, error) {
	rows, err := pool.Query(ctx, `
		SELECT version, build, count(*)
		FROM client_versions
		WHERE platform = 'android' AND last_seen_at > now() - make_interval(secs => $1)
		GROUP BY version, build
		ORDER BY build DESC
		LIMIT 20`, Window.Seconds())
	if err != nil {
		return nil, fmt.Errorf("clientversions: recent: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Count, error) {
		var c Count
		return c, r.Scan(&c.Version, &c.Build, &c.Users)
	})
	if err != nil {
		return nil, fmt.Errorf("clientversions: recent: %w", err)
	}
	if out == nil {
		out = []Count{}
	}
	return out, nil
}
