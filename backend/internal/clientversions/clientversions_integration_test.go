//go:build integration

package clientversions_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/clientversions"
	"kisy-backend/internal/platform/testdb"
)

func TestBuildsAreRecordedAndCounted(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	opts, err := redis.ParseURL(testdb.RedisURL(t))
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(opts)
	t.Cleanup(func() { _ = rdb.Close() })
	stale, _ := rdb.Keys(ctx, "kisy:appver:*").Result()
	if len(stale) > 0 {
		_ = rdb.Del(ctx, stale...).Err()
	}

	rec := clientversions.NewRecorder(pool, rdb, slog.New(slog.NewTextHandler(io.Discard, nil)))
	a := testdb.SeedUser(t, pool, "anna", 5)
	b := testdb.SeedUser(t, pool, "boris", 0)
	c := testdb.SeedUser(t, pool, "vera", 8)

	who := map[string]uuid.UUID{"a": a, "b": b, "c": c}
	h := rec.Middleware(func(r *http.Request) (uuid.UUID, bool) {
		id, ok := who[r.Header.Get("X-Test-User")]
		return id, ok
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	hit := func(user, version string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if user != "" {
			req.Header.Set("X-Test-User", user)
		}
		if version != "" {
			req.Header.Set(clientversions.Header, version)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("the request was not passed on: %d", rec.Code)
		}
	}

	hit("a", "1.0.0 (100)")
	hit("a", "1.0.0 (100)") // throttled, still one row
	hit("b", "1.0.0 (100)")
	hit("c", "1.1.0 (200)")
	hit("c", "garbage")     // ignored
	hit("", "1.2.0 (300)")  // not signed in: ignored
	hit("a", "1.1.0 (200)") // anna updated

	got, err := clientversions.Recent(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Build != 200 || got[0].Users != 2 || got[1].Build != 100 || got[1].Users != 1 {
		t.Fatalf("recent builds: %+v", got)
	}
}
