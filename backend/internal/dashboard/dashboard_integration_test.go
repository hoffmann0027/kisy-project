//go:build integration

package dashboard_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/dashboard"
	"kisy-backend/internal/platform/testdb"
)

// The overview counts what is really there; nothing on it is invented.
func TestOverviewCountsRealData(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	opts, err := redis.ParseURL(testdb.RedisURL(t))
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(opts)
	t.Cleanup(func() { _ = rdb.Close() })

	ceo := testdb.SeedUser(t, pool, "the_ceo", 1)
	a := testdb.SeedUser(t, pool, "anna", 5)
	b := testdb.SeedUser(t, pool, "boris", 0)
	old := testdb.SeedUser(t, pool, "old_timer", 8)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	// One account joined 10 days ago; one is active today.
	exec(`UPDATE users SET created_at = now() - interval '10 days' WHERE id = $1`, old)
	exec(`UPDATE users SET last_seen_at = now() WHERE id = $1`, a)
	exec(`INSERT INTO feedback (author_id, body) VALUES ($1, 'идея')`, b)
	exec(`INSERT INTO reports (reporter_id, target_kind, target_id, reason) VALUES ($1, 'user', $2, 'fraud')`, a, b)

	svc := dashboard.NewService(pool, rdb, dashboard.Settings{
		Version: "abc1234", StartedAt: time.Now().Add(-time.Hour),
		DBLimitBytes: 512 << 20, RedisLimitBytes: 256 << 20,
	})
	o, err := svc.Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}

	k := o.KPI
	if k.UsersTotal != 4 || k.UsersNew24h != 3 || k.UsersNew7d != 3 || k.Active24h != 1 {
		t.Fatalf("kpi: %+v", k)
	}
	if o.Inbox.OpenReports != 1 || o.Inbox.UnansweredFeedback != 1 {
		t.Fatalf("inbox: %+v", o.Inbox)
	}
	if len(o.Reports) != 1 || o.Reports[0].Severity != "high" {
		t.Fatalf("reports: %+v", o.Reports)
	}

	// Every one of the last 90 days is on the chart, ending at the total.
	if len(o.Growth) != dashboard.GrowthDays {
		t.Fatalf("growth: %d days", len(o.Growth))
	}
	last := o.Growth[len(o.Growth)-1]
	if last.Total != 4 || last.Registrations != 3 {
		t.Fatalf("today: %+v", last)
	}
	if o.Growth[len(o.Growth)-11].Registrations != 1 {
		t.Fatalf("ten days ago: %+v", o.Growth[len(o.Growth)-11])
	}

	if o.Limits.Database.UsedBytes <= 0 || o.Limits.Database.LimitBytes != 512<<20 {
		t.Fatalf("database usage: %+v", o.Limits.Database)
	}
	if o.Limits.Redis == nil || o.Limits.Redis.UsedBytes <= 0 {
		t.Fatalf("redis usage: %+v", o.Limits.Redis)
	}

	if o.System.Version != "abc1234" {
		t.Fatalf("version %q", o.System.Version)
	}
	states := map[string]string{}
	for _, c := range o.System.Checks {
		states[c.Name] = c.State
	}
	if states["database"] != "ok" || states["redis"] != "ok" {
		t.Fatalf("live checks: %v", states)
	}
	// Nothing configured is shown as off, not as fine.
	for _, name := range []string{"turn", "push_android", "push_web", "captcha"} {
		if states[name] != "off" {
			t.Fatalf("%s: %q, want off", name, states[name])
		}
	}
	_ = ceo
}
