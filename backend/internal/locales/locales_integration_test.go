//go:build integration

package locales_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/i18n"
	"kisy-backend/internal/locales"
	"kisy-backend/internal/platform/testdb"
)

func setup(t *testing.T) (*locales.Recorder, func(context.Context, uuid.UUID) i18n.Lang, uuid.UUID) {
	t.Helper()
	pool := testdb.New(t)
	opts, err := redis.ParseURL(testdb.RedisURL(t))
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(opts)
	t.Cleanup(func() { _ = rdb.Close() })
	user := testdb.SeedUser(t, pool, "anna", 3)
	t.Cleanup(func() { _ = rdb.Del(context.Background(), "kisy:lang:"+user.String()).Err() })
	return locales.NewRecorder(pool, rdb, slog.New(slog.NewTextHandler(io.Discard, nil))), locales.Of(pool), user
}

func through(rec *locales.Recorder, user uuid.UUID, header string) {
	h := rec.Middleware(func(*http.Request) (uuid.UUID, bool) { return user, true })(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	r := httptest.NewRequest("GET", "/", nil)
	if header != "" {
		r.Header.Set(i18n.Header, header)
	}
	h.ServeHTTP(httptest.NewRecorder(), r)
}

// An account is Russian until its app says otherwise; then pushes follow the
// app — there and back again.
func TestTheAppsLanguageIsWhatPushesAreWrittenIn(t *testing.T) {
	rec, of, user := setup(t)
	ctx := context.Background()

	if got := of(ctx, user); got != "ru" {
		t.Fatalf("a new account speaks %q, want ru", got)
	}
	through(rec, user, "")
	through(rec, user, "klingon")
	if got := of(ctx, user); got != "ru" {
		t.Fatalf("no header or an unknown one changed the language to %q", got)
	}

	through(rec, user, "en")
	if got := of(ctx, user); got != "en" {
		t.Fatalf("after the app switched: %q, want en", got)
	}
	through(rec, user, "ru")
	if got := of(ctx, user); got != "ru" {
		t.Fatalf("after switching back: %q, want ru", got)
	}
}

func TestAnUnknownAccountGetsTheDefault(t *testing.T) {
	_, of, _ := setup(t)
	if got := of(context.Background(), uuid.New()); got != i18n.Default {
		t.Fatalf("got %q", got)
	}
}
