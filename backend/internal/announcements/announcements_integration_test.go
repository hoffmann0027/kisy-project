//go:build integration

package announcements_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sort"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/announcements"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/testdb"
)

type live struct {
	mu      sync.Mutex
	created map[uuid.UUID]int
	revoked map[uuid.UUID]int
}

func (l *live) PublishNotification(id uuid.UUID, _ any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.created[id]++
}

func (l *live) PublishNotificationRevoked(id uuid.UUID, _ any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.revoked[id]++
}

type env struct {
	pool *pgxpool.Pool
	svc  *announcements.Service
	live *live
	// One account per level 1-5, plus two basic accounts.
	ceo, exec, director, senior, manager uuid.UUID
	basic1, basic2                       uuid.UUID
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := testdb.New(t)
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	e := &env{pool: pool, live: &live{created: map[uuid.UUID]int{}, revoked: map[uuid.UUID]int{}}}
	e.svc = announcements.NewService(pool, rec)
	e.svc.SetPublisher(e.live)
	e.ceo = testdb.SeedUser(t, pool, "the_ceo", 1)
	e.exec = testdb.SeedUser(t, pool, "exec", 2)
	e.director = testdb.SeedUser(t, pool, "director", 3)
	e.senior = testdb.SeedUser(t, pool, "senior", 4)
	e.manager = testdb.SeedUser(t, pool, "manager", 5)
	e.basic1 = testdb.SeedUser(t, pool, "basic_one", 0)
	e.basic2 = testdb.SeedUser(t, pool, "basic_two", 0)
	return e
}

func actor(id uuid.UUID) announcements.ActorMeta {
	return announcements.ActorMeta{UserID: id, SessionID: uuid.New(), RequestID: "test"}
}

// recipientsOf lists who holds a notification for announcement id.
func (e *env) recipientsOf(t *testing.T, id uuid.UUID) []uuid.UUID {
	t.Helper()
	rows, err := e.pool.Query(context.Background(),
		`SELECT user_id FROM notifications WHERE announcement_id = $1 AND type = 'announcement'`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var u uuid.UUID
		if err := rows.Scan(&u); err != nil {
			t.Fatal(err)
		}
		out = append(out, u)
	}
	return out
}

func sameSet(t *testing.T, got, want []uuid.UUID) {
	t.Helper()
	key := func(s []uuid.UUID) []string {
		out := make([]string, len(s))
		for i, u := range s {
			out[i] = u.String()
		}
		sort.Strings(out)
		return out
	}
	g, w := key(got), key(want)
	if len(g) != len(w) {
		t.Fatalf("recipients: got %d, want %d", len(g), len(w))
	}
	for i := range g {
		if g[i] != w[i] {
			t.Fatalf("recipients differ:\n got  %v\n want %v", g, w)
		}
	}
}

func send(t *testing.T, e *env, from uuid.UUID, in announcements.Input) (*announcements.Announcement, error) {
	t.Helper()
	if in.Title == "" {
		in.Title = "Заголовок"
	}
	if in.Body == "" {
		in.Body = "Текст объявления"
	}
	return e.svc.Send(context.Background(), actor(from), in)
}

func TestOnlyLevelsOneToThreeMaySend(t *testing.T) {
	e := setup(t)
	for _, who := range []uuid.UUID{e.senior, e.manager, e.basic1} {
		if _, err := send(t, e, who, announcements.Input{Audience: announcements.AudienceAll}); !errors.Is(err, announcements.ErrNotAuthor) {
			t.Fatalf("got %v, want ErrNotAuthor", err)
		}
	}
	for _, who := range []uuid.UUID{e.ceo, e.exec, e.director} {
		if _, err := send(t, e, who, announcements.Input{Audience: announcements.AudienceAll}); err != nil {
			t.Fatalf("level 1-3 author refused: %v", err)
		}
	}
}

// The token is not trusted for the level: a director demoted a minute ago
// cannot keep sending until their access token expires.
func TestALiveDemotionTakesEffectAtOnce(t *testing.T) {
	e := setup(t)
	if _, err := e.pool.Exec(context.Background(), `UPDATE users SET role_id = 6 WHERE id = $1`, e.director); err != nil {
		t.Fatal(err)
	}
	if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceAll}); !errors.Is(err, announcements.ErrNotAuthor) {
		t.Fatalf("got %v, want ErrNotAuthor", err)
	}
}

// "Everyone" means everyone at or below the author: the CEO reaches the whole
// company, a director everyone from level 3 down plus basic accounts. The
// author never notifies themselves.
func TestEveryoneMeansEveryoneAtOrBelowTheAuthor(t *testing.T) {
	e := setup(t)
	a, err := send(t, e, e.ceo, announcements.Input{Audience: announcements.AudienceAll})
	if err != nil {
		t.Fatal(err)
	}
	sameSet(t, e.recipientsOf(t, a.ID), []uuid.UUID{e.exec, e.director, e.senior, e.manager, e.basic1, e.basic2})
	if a.RecipientCount != 6 {
		t.Fatalf("RecipientCount = %d", a.RecipientCount)
	}

	a, err = send(t, e, e.director, announcements.Input{Audience: announcements.AudienceAll})
	if err != nil {
		t.Fatal(err)
	}
	sameSet(t, e.recipientsOf(t, a.ID), []uuid.UUID{e.senior, e.manager, e.basic1, e.basic2})
}

func TestBasicAudienceReachesOnlyBasicAccounts(t *testing.T) {
	e := setup(t)
	a, err := send(t, e, e.exec, announcements.Input{Audience: announcements.AudienceBasic})
	if err != nil {
		t.Fatal(err)
	}
	sameSet(t, e.recipientsOf(t, a.ID), []uuid.UUID{e.basic1, e.basic2})
}

func TestLevelsAudienceIsDownwardsOnly(t *testing.T) {
	e := setup(t)
	a, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceLevels, Levels: []int{3, 5, 5}})
	if err != nil {
		t.Fatal(err)
	}
	sameSet(t, e.recipientsOf(t, a.ID), []uuid.UUID{e.manager}) // level 3 is the director alone
	if len(a.Levels) != 2 {
		t.Fatalf("levels not de-duplicated: %v", a.Levels)
	}

	if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceLevels, Levels: []int{2, 5}}); !errors.Is(err, announcements.ErrAboveYou) {
		t.Fatalf("director → level 2: got %v, want ErrAboveYou", err)
	}
	if _, err := send(t, e, e.ceo, announcements.Input{Audience: announcements.AudienceLevels, Levels: []int{2}}); err != nil {
		t.Fatalf("CEO → level 2: %v", err)
	}
}

func TestPersonalAnnouncementFollowsTheHierarchyAndBlocks(t *testing.T) {
	e := setup(t)
	a, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceUser, UserID: e.manager})
	if err != nil {
		t.Fatal(err)
	}
	sameSet(t, e.recipientsOf(t, a.ID), []uuid.UUID{e.manager})

	for name, target := range map[string]uuid.UUID{"above": e.exec, "the CEO": e.ceo, "self": e.director, "nobody": uuid.New()} {
		if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceUser, UserID: target}); !errors.Is(err, announcements.ErrUnreachable) {
			t.Fatalf("%s: got %v, want ErrUnreachable", name, err)
		}
	}

	// A block either way: the person is unreachable, and the refusal does not
	// say who blocked whom.
	ctx := context.Background()
	if _, err := e.pool.Exec(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1, $2)`, e.basic1, e.director); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1, $2)`, e.director, e.basic2); err != nil {
		t.Fatal(err)
	}
	for _, target := range []uuid.UUID{e.basic1, e.basic2} {
		if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceUser, UserID: target}); !errors.Is(err, announcements.ErrUnreachable) {
			t.Fatalf("blocked: got %v, want ErrUnreachable", err)
		}
	}
	// And a broadcast passes them by.
	a, err = send(t, e, e.director, announcements.Input{Audience: announcements.AudienceBasic})
	if err != nil {
		t.Fatal(err)
	}
	if got := e.recipientsOf(t, a.ID); len(got) != 0 {
		t.Fatalf("broadcast reached a blocked pair: %v", got)
	}
}

func TestInactiveAccountsAreNotNotified(t *testing.T) {
	e := setup(t)
	if _, err := e.pool.Exec(context.Background(), `UPDATE users SET is_active = false WHERE id = $1`, e.basic2); err != nil {
		t.Fatal(err)
	}
	a, err := send(t, e, e.exec, announcements.Input{Audience: announcements.AudienceBasic})
	if err != nil {
		t.Fatal(err)
	}
	sameSet(t, e.recipientsOf(t, a.ID), []uuid.UUID{e.basic1})
}

func TestDailyLimitForLevelsTwoAndThreeButNotTheCEO(t *testing.T) {
	e := setup(t)
	for i := 0; i < announcements.BroadcastsPerDay; i++ {
		if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceBasic}); err != nil {
			t.Fatalf("broadcast %d: %v", i+1, err)
		}
	}
	if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceBasic}); !errors.Is(err, announcements.ErrQuota) {
		t.Fatalf("over the broadcast limit: got %v, want ErrQuota", err)
	}
	// Personal notes are counted on their own.
	if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceUser, UserID: e.manager}); err != nil {
		t.Fatalf("personal after broadcasts: %v", err)
	}
	// Revoking does not give the quota back.
	list, err := e.svc.List(context.Background(), actor(e.director), 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Revoke(context.Background(), actor(e.director), list[len(list)-1].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceBasic}); !errors.Is(err, announcements.ErrQuota) {
		t.Fatalf("revoke freed the quota: %v", err)
	}
	// A day later the window has moved on.
	if _, err := e.pool.Exec(context.Background(),
		`UPDATE announcements SET created_at = created_at - interval '25 hours' WHERE author_id = $1`, e.director); err != nil {
		t.Fatal(err)
	}
	if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceBasic}); err != nil {
		t.Fatalf("next day: %v", err)
	}

	for i := 0; i < announcements.BroadcastsPerDay+2; i++ {
		if _, err := send(t, e, e.ceo, announcements.Input{Audience: announcements.AudienceBasic}); err != nil {
			t.Fatalf("CEO broadcast %d: %v", i+1, err)
		}
	}
}

func TestRevokeRemovesItFromEveryList(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceAll})
	if err != nil {
		t.Fatal(err)
	}
	if len(e.recipientsOf(t, a.ID)) != 4 {
		t.Fatal("not delivered")
	}
	// Someone else's announcement does not exist for an outsider.
	if err := e.svc.Revoke(ctx, actor(e.exec), a.ID); !errors.Is(err, announcements.ErrNotFound) {
		t.Fatalf("level 2 revoking a director's: got %v, want ErrNotFound", err)
	}
	if err := e.svc.Revoke(ctx, actor(e.manager), a.ID); !errors.Is(err, announcements.ErrNotFound) {
		t.Fatalf("recipient revoking: got %v, want ErrNotFound", err)
	}
	if err := e.svc.Revoke(ctx, actor(e.director), a.ID); err != nil {
		t.Fatalf("author revoking: %v", err)
	}
	if got := e.recipientsOf(t, a.ID); len(got) != 0 {
		t.Fatalf("still held by %d people", len(got))
	}
	if e.live.revoked[e.manager] != 1 {
		t.Fatal("open clients were not told")
	}
	// Twice is harmless.
	if err := e.svc.Revoke(ctx, actor(e.director), a.ID); err != nil {
		t.Fatal(err)
	}

	// The CEO can take back anyone's.
	b, err := send(t, e, e.exec, announcements.Input{Audience: announcements.AudienceBasic})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Revoke(ctx, actor(e.ceo), b.ID); err != nil {
		t.Fatalf("CEO revoking: %v", err)
	}
	list, err := e.svc.List(ctx, actor(e.ceo), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].RevokedAt == nil || list[1].RevokedAt == nil {
		t.Fatalf("CEO's list: %+v", list)
	}
	// An author sees only their own.
	mine, err := e.svc.List(ctx, actor(e.exec), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 1 || mine[0].ID != b.ID {
		t.Fatalf("exec's list: %+v", mine)
	}
}

func TestRecipientsSeeWhoWroteIt(t *testing.T) {
	e := setup(t)
	if _, err := send(t, e, e.director, announcements.Input{Audience: announcements.AudienceUser, UserID: e.manager, Title: "  Собрание  ", Body: "Завтра в 10:00"}); err != nil {
		t.Fatal(err)
	}
	var title, author string
	var level int
	if err := e.pool.QueryRow(context.Background(), `
		SELECT payload->>'title', payload->'author'->>'displayName', (payload->'author'->>'roleLevel')::int
		FROM notifications WHERE user_id = $1 AND type = 'announcement'`, e.manager).Scan(&title, &author, &level); err != nil {
		t.Fatal(err)
	}
	if title != "Собрание" || author != testdb.SeedDisplayName("director") || level != 3 {
		t.Fatalf("payload: title %q, author %q, level %d", title, author, level)
	}
	if e.live.created[e.manager] != 1 {
		t.Fatal("no live event")
	}
	var action string
	if err := e.pool.QueryRow(context.Background(),
		`SELECT action FROM audit_logs WHERE actor_id = $1 ORDER BY created_at DESC LIMIT 1`, e.director).Scan(&action); err != nil {
		t.Fatal(err)
	}
	if action != announcements.ActionSent {
		t.Fatalf("audit action %q", action)
	}
}

func TestMalformedAnnouncementsAreRefused(t *testing.T) {
	e := setup(t)
	long := make([]rune, announcements.MaxTitle+1)
	for i := range long {
		long[i] = 'я'
	}
	for name, in := range map[string]announcements.Input{
		"unknown audience": {Audience: "everyone", Title: "t", Body: "b"},
		"blank title":      {Audience: announcements.AudienceAll, Title: "   ", Body: "b"},
		"long title":       {Audience: announcements.AudienceAll, Title: string(long), Body: "b"},
		"no levels":        {Audience: announcements.AudienceLevels, Title: "t", Body: "b"},
		"level 11":         {Audience: announcements.AudienceLevels, Levels: []int{11}, Title: "t", Body: "b"},
		"user without id":  {Audience: announcements.AudienceUser, Title: "t", Body: "b"},
		"all with levels":  {Audience: announcements.AudienceAll, Levels: []int{5}, Title: "t", Body: "b"},
	} {
		if _, err := e.svc.Send(context.Background(), actor(e.ceo), in); !errors.Is(err, announcements.ErrValidation) {
			t.Fatalf("%s: got %v, want ErrValidation", name, err)
		}
	}
}
