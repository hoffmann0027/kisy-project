// Package locales remembers the language each account's app shows, so what
// the server writes to someone later — a push — is in their language rather
// than the sender's.
//
// The app names its language on every request (i18n.Header). Writing that down
// per request would be a database write per request, so Redis remembers the
// last language seen per account for Throttle and only a change is written.
// An app too old to send the header changes nothing: its account stays on the
// default, Russian, which is what it shows.
package locales

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/i18n"
)

// Throttle: how long a language seen for an account is trusted without
// looking at the database again.
const Throttle = 6 * time.Hour

type Recorder struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
	log  *slog.Logger
}

func NewRecorder(pool *pgxpool.Pool, rdb *redis.Client, log *slog.Logger) *Recorder {
	return &Recorder{pool: pool, rdb: rdb, log: log}
}

func cacheKey(userID uuid.UUID) string { return "kisy:lang:" + userID.String() }

// Record saves that userID's app shows lang, unless that is already known.
func (r *Recorder) Record(ctx context.Context, userID uuid.UUID, lang i18n.Lang) error {
	key := cacheKey(userID)
	seen, err := r.rdb.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("locales: cache: %w", err)
	}
	if seen == string(lang) {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `UPDATE users SET locale = $2 WHERE id = $1 AND locale <> $2`, userID, string(lang)); err != nil {
		return fmt.Errorf("locales: record: %w", err)
	}
	if err := r.rdb.Set(ctx, key, string(lang), Throttle).Err(); err != nil {
		return fmt.Errorf("locales: cache: %w", err)
	}
	return nil
}

// Middleware records the language of signed-in requests that name one the
// server speaks. It never fails a request: a lost record costs one push in the
// previous language.
func (r *Recorder) Middleware(userID func(*http.Request) (uuid.UUID, bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if raw := req.Header.Get(i18n.Header); i18n.IsSupported(raw) {
				if id, signedIn := userID(req); signedIn {
					if err := r.Record(req.Context(), id, i18n.Lang(raw)); err != nil {
						r.log.Warn("locales: record", "error", err)
					}
				}
			}
			next.ServeHTTP(w, req)
		})
	}
}

// Of is the saved language of an account; i18n.Default when it has none or
// the lookup fails — a push in Russian beats no push.
func Of(pool *pgxpool.Pool) func(context.Context, uuid.UUID) i18n.Lang {
	return func(ctx context.Context, userID uuid.UUID) i18n.Lang {
		var code string
		err := pool.QueryRow(ctx, `SELECT locale FROM users WHERE id = $1`, userID).Scan(&code)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				slog.WarnContext(ctx, "locales: lookup", "error", err)
			}
			return i18n.Default
		}
		return i18n.Parse(code)
	}
}
