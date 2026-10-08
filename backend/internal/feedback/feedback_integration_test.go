//go:build integration

package feedback_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/feedback"
	"kisy-backend/internal/notifications"
	"kisy-backend/internal/platform/testdb"
)

type notes struct {
	mu   sync.Mutex
	sent []notifications.Announcement
	to   []uuid.UUID
}

func (n *notes) Announce(_ context.Context, to []uuid.UUID, a notifications.Announcement) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, a)
	n.to = append(n.to, to...)
	return nil
}

type env struct {
	pool                              *pgxpool.Pool
	svc                               *feedback.Service
	notes                             *notes
	ceo, director, manager, basic, b2 uuid.UUID
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := testdb.New(t)
	e := &env{pool: pool, notes: &notes{}}
	e.svc = feedback.NewService(pool, feedback.NewPostgresRepository())
	e.svc.SetNotifier(e.notes)
	e.ceo = testdb.SeedUser(t, pool, "the_ceo", 1)
	e.director = testdb.SeedUser(t, pool, "director", 3)
	e.manager = testdb.SeedUser(t, pool, "manager", 5)
	e.basic = testdb.SeedUser(t, pool, "basic_one", 0)
	e.b2 = testdb.SeedUser(t, pool, "basic_two", 0)
	return e
}

func me(id uuid.UUID) feedback.Actor { return feedback.Actor{UserID: id} }

func (e *env) post(t *testing.T, author uuid.UUID, body string) feedback.DTO {
	t.Helper()
	d, err := e.svc.Create(context.Background(), author, body)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return d
}

func ids(items []feedback.DTO) map[uuid.UUID]bool {
	out := map[uuid.UUID]bool{}
	for _, d := range items {
		out[d.ID] = true
	}
	return out
}

// Feedback used to be a public board. What someone tells the leadership is
// theirs and the leadership's alone.
func TestAnAuthorSeesOnlyTheirOwn(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	mine := e.post(t, e.basic, "моё предложение")
	theirs := e.post(t, e.b2, "чужое предложение")

	page, err := e.svc.List(ctx, me(e.basic), "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(page.Items); len(got) != 1 || !got[mine.ID] || got[theirs.ID] {
		t.Fatalf("basic sees %v", page.Items)
	}
	// Asking for the inbox does not widen it.
	if _, err := e.svc.List(ctx, me(e.basic), feedback.ScopeInbox, "", 0); !errors.Is(err, feedback.ErrForbidden) {
		t.Fatalf("basic inbox: got %v, want ErrForbidden", err)
	}
	if _, err := e.svc.List(ctx, me(e.manager), feedback.ScopeInbox, "", 0); !errors.Is(err, feedback.ErrForbidden) {
		t.Fatalf("level 5 inbox: got %v, want ErrForbidden", err)
	}
}

func TestOneEntryPerDay(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.post(t, e.basic, "первое")
	_, err := e.svc.Create(ctx, e.basic, "второе")
	var limit *feedback.DailyLimitError
	if !errors.As(err, &limit) || limit.RetryAfter <= 23*time.Hour || limit.RetryAfter > 24*time.Hour {
		t.Fatalf("second within a day: %v", err)
	}
	// Someone else is not affected.
	e.post(t, e.b2, "другой автор")
	// A day later the author may write again.
	if _, err := e.pool.Exec(ctx, `UPDATE feedback SET created_at = created_at - interval '25 hours' WHERE author_id = $1`, e.basic); err != nil {
		t.Fatal(err)
	}
	e.post(t, e.basic, "на следующий день")
}

// Levels 1-3 see everyone's unanswered entries; answering one takes it out of
// the inbox, shows the reply to its author and tells them.
func TestLeadershipAnswersAndTheEntryLeavesTheInbox(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a := e.post(t, e.basic, "сделайте тёмную тему светлее")
	b := e.post(t, e.manager, "нужен экспорт отчётов")

	inbox, err := e.svc.List(ctx, me(e.director), feedback.ScopeInbox, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(inbox.Items); len(got) != 2 || !got[a.ID] || !got[b.ID] {
		t.Fatalf("director inbox: %v", inbox.Items)
	}

	if err := e.svc.Reply(ctx, me(e.director), a.ID, "  Сделаем в следующем обновлении  "); err != nil {
		t.Fatal(err)
	}
	inbox, err = e.svc.List(ctx, me(e.ceo), feedback.ScopeInbox, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(inbox.Items); len(got) != 1 || !got[b.ID] {
		t.Fatalf("inbox after the answer: %v", inbox.Items)
	}

	own, err := e.svc.List(ctx, me(e.basic), "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	r := own.Items[0].Reply
	if r == nil || r.Body != "Сделаем в следующем обновлении" || r.By == nil || r.By.ID != e.director {
		t.Fatalf("author's view of the reply: %+v", r)
	}
	if len(e.notes.to) != 1 || e.notes.to[0] != e.basic || e.notes.sent[0].Type != feedback.NotificationType {
		t.Fatalf("author not notified: %+v", e.notes)
	}

	// Once only.
	if err := e.svc.Reply(ctx, me(e.ceo), a.ID, "ещё раз"); !errors.Is(err, feedback.ErrAlreadyReplied) {
		t.Fatalf("second answer: got %v, want ErrAlreadyReplied", err)
	}
}

func TestOnlyLevelsOneToThreeAnswer(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a := e.post(t, e.basic, "вопрос")
	for _, who := range []uuid.UUID{e.manager, e.b2} {
		if err := e.svc.Reply(ctx, me(who), a.ID, "ответ"); !errors.Is(err, feedback.ErrForbidden) {
			t.Fatalf("got %v, want ErrForbidden", err)
		}
	}
	// A demotion takes effect at once, whatever the token still says.
	if _, err := e.pool.Exec(ctx, `UPDATE users SET role_id = 6 WHERE id = $1`, e.director); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Reply(ctx, feedback.Actor{UserID: e.director, RoleLevel: 3}, a.ID, "ответ"); !errors.Is(err, feedback.ErrForbidden) {
		t.Fatalf("demoted director: got %v, want ErrForbidden", err)
	}
	if err := e.svc.Reply(ctx, me(e.ceo), uuid.New(), "ответ"); !errors.Is(err, feedback.ErrNotFound) {
		t.Fatalf("unknown entry: got %v, want ErrNotFound", err)
	}
	if err := e.svc.Reply(ctx, me(e.ceo), a.ID, "   "); !errors.Is(err, feedback.ErrEmpty) {
		t.Fatalf("blank answer: got %v, want ErrEmpty", err)
	}
}

func TestOnlyTheCEODeletes(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a := e.post(t, e.basic, "удалить")
	if err := e.svc.Delete(ctx, a.ID, feedback.Actor{UserID: e.director, RoleLevel: 1}); !errors.Is(err, feedback.ErrForbidden) {
		t.Fatalf("director with a forged level: got %v", err)
	}
	if err := e.svc.Delete(ctx, a.ID, me(e.ceo)); err != nil {
		t.Fatal(err)
	}
}
