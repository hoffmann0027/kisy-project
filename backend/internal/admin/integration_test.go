//go:build integration

package admin_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/admin"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/users"
)

// Run with:
//
//	TEST_DATABASE_URL=postgres://kisy:<pass>@localhost:5432/kisy \
//	go test -tags integration ./internal/admin/

type env struct {
	pool     *pgxpool.Pool
	svc      *admin.Service
	sessions auth.SessionRepository
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := testdb.New(t)
	sessions := auth.NewPostgresSessionRepository()
	svc := admin.NewService(pool, users.NewPostgresRepository(), sessions,
		audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil))))
	return &env{pool: pool, svc: svc, sessions: sessions}
}

// openSession gives the target a live session, as a login would.
func openSession(t *testing.T, e *env, userID uuid.UUID) uuid.UUID {
	t.Helper()
	s := &auth.Session{
		UserID:           userID,
		RefreshTokenHash: uuid.NewString(),
		IPHash:           "test-ip-hash",
		ExpiresAt:        time.Now().UTC().Add(24 * time.Hour),
	}
	if err := e.sessions.Create(context.Background(), e.pool, s); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return s.ID
}

// A demoted user must lose access immediately. The access token carries the
// clearance in its `lvl` claim and RequireClearance trusts that claim, so
// without revocation the old level stays usable until the token expires —
// up to JWT_ACCESS_TTL (15 minutes) of admin/invites/rating access that the
// user is no longer entitled to.
func TestChangeRoleRevokesTargetSessions(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	ceo := testdb.SeedUser(t, e.pool, "ceo_admin_it", 1)
	target := testdb.SeedUser(t, e.pool, "demoted_admin_it", 2)

	first := openSession(t, e, target)
	second := openSession(t, e, target)

	if err := e.svc.ChangeRole(ctx, target, 8, admin.ActorMeta{
		UserID:    ceo,
		SessionID: openSession(t, e, ceo),
		IPHash:    "ceo-ip-hash",
	}); err != nil {
		t.Fatalf("ChangeRole: %v", err)
	}

	for name, id := range map[string]uuid.UUID{"first": first, "second": second} {
		s, err := e.sessions.GetByID(ctx, e.pool, id)
		if err != nil {
			t.Fatalf("get %s session: %v", name, err)
		}
		if s.RevokedAt == nil {
			t.Errorf("%s session survived the demotion: the old clearance stays "+
				"usable until the access token expires", name)
		}
		if s.Active(time.Now().UTC()) {
			t.Errorf("%s session still reports Active after a role change", name)
		}
	}
}

// The acting CEO must keep working: revocation targets the demoted user, not
// the administrator performing it.
func TestChangeRoleKeepsActorSession(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	ceo := testdb.SeedUser(t, e.pool, "ceo_keeps_it", 1)
	target := testdb.SeedUser(t, e.pool, "target_keeps_it", 4)

	actorSession := openSession(t, e, ceo)

	if err := e.svc.ChangeRole(ctx, target, 6, admin.ActorMeta{
		UserID:    ceo,
		SessionID: actorSession,
		IPHash:    "ceo-ip-hash",
	}); err != nil {
		t.Fatalf("ChangeRole: %v", err)
	}

	s, err := e.sessions.GetByID(ctx, e.pool, actorSession)
	if err != nil {
		t.Fatalf("get actor session: %v", err)
	}
	if s.RevokedAt != nil {
		t.Error("the acting CEO was logged out by their own role change")
	}
}
