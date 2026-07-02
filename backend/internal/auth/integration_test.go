//go:build integration

package auth_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth"
	"kisy-backend/internal/auth/password"
	"kisy-backend/internal/auth/token"
	"kisy-backend/internal/invitations"
	"kisy-backend/internal/platform/postgres"
	"kisy-backend/internal/users"
)

// The integration suite exercises the real SQL against a disposable
// database. Run with:
//
//	TEST_DATABASE_URL=postgres://kisy:<pass>@localhost:5432/kisy \
//	go test -tags integration ./internal/auth/
//
// The URL must point at the maintenance database; the suite creates and
// drops its own kisy_auth_test database.
const testDBName = "kisy_auth_test"

type env struct {
	pool     *pgxpool.Pool
	svc      *auth.Service
	invites  *invitations.Service
	users    users.Repository
	sessions auth.SessionRepository
	tokens   *token.Manager
}

func setup(t *testing.T) *env {
	t.Helper()

	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()

	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("connect admin db: %v", err)
	}
	t.Cleanup(admin.Close)

	if _, err := admin.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, testDBName)); err != nil {
		t.Fatalf("drop test db: %v", err)
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE DATABASE %s`, testDBName)); err != nil {
		t.Fatalf("create test db: %v", err)
	}

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	u.Path = "/" + testDBName
	testURL := u.String()

	if err := postgres.Migrate(testURL, "../../migrations"); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	pool, err := pgxpool.New(ctx, testURL)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	t.Cleanup(pool.Close)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := audit.NewPostgresRecorder(log)
	usersRepo := users.NewPostgresRepository()
	sessionsRepo := auth.NewPostgresSessionRepository()
	invitesRepo := invitations.NewPostgresRepository()
	tokens := token.NewManager("integration-test-secret-32-chars-min", 15*time.Minute)

	svc, err := auth.NewService(pool, usersRepo, sessionsRepo, invitesRepo, rec, tokens, 24*time.Hour)
	if err != nil {
		t.Fatalf("auth service: %v", err)
	}
	invitesSvc := invitations.NewService(pool, invitesRepo, rec, 120*time.Second)

	return &env{pool: pool, svc: svc, invites: invitesSvc, users: usersRepo, sessions: sessionsRepo, tokens: tokens}
}

func (e *env) createUser(t *testing.T, username, plain string, level int) *users.User {
	t.Helper()
	hash, err := password.Hash(plain)
	if err != nil {
		t.Fatal(err)
	}
	u := &users.User{Username: username, DisplayName: username, PasswordHash: hash, RoleID: level}
	if err := e.users.Create(context.Background(), e.pool, u); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

var testMeta = auth.ClientMeta{IPHash: "test-ip-hash", UserAgent: "go-test", RequestID: "req-test"}

func TestLoginSuccessAndSessionLifecycle(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.createUser(t, "ceo_user", "ceo-password-42", 1)

	res, err := e.svc.Login(ctx, "ceo_user", "ceo-password-42", testMeta)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.User.RoleID != 1 {
		t.Fatalf("role = %d, want 1", res.User.RoleID)
	}

	claims, err := e.tokens.ParseAccess(res.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("parse issued access token: %v", err)
	}
	if claims.UserID != res.User.ID || claims.SessionID != res.Tokens.SessionID {
		t.Fatal("claims do not match session")
	}

	// Logout revokes the session.
	if err := e.svc.Logout(ctx, res.User.ID, res.Tokens.SessionID, testMeta); err != nil {
		t.Fatalf("logout: %v", err)
	}
	sess, err := e.sessions.GetByID(ctx, e.pool, res.Tokens.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Active(time.Now().UTC()) {
		t.Fatal("session still active after logout")
	}
}

func TestLoginLockoutAfterRepeatedFailures(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.createUser(t, "victim", "correct-password-1", 5)

	var lastErr error
	for i := 0; i < auth.MaxLoginAttempts; i++ {
		_, lastErr = e.svc.Login(ctx, "victim", "wrong-password-xx1", testMeta)
	}
	if !errors.Is(lastErr, auth.ErrAccountLocked) {
		t.Fatalf("attempt %d error = %v, want ErrAccountLocked", auth.MaxLoginAttempts, lastErr)
	}

	// Correct password is also rejected while locked.
	if _, err := e.svc.Login(ctx, "victim", "correct-password-1", testMeta); !errors.Is(err, auth.ErrAccountLocked) {
		t.Fatalf("locked login error = %v, want ErrAccountLocked", err)
	}
}

func TestLoginUnknownUser(t *testing.T) {
	e := setup(t)
	if _, err := e.svc.Login(context.Background(), "who_is_this", "any-password-11", testMeta); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestInviteRegisterFlow(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	ceo := e.createUser(t, "the_ceo", "ceo-password-42", 1)

	created, err := e.invites.Create(ctx, invitations.CreatorMeta{ActorID: ceo.ID, SessionID: ceo.ID, IPHash: "x", RequestID: "r"})
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if time.Until(created.ExpiresAt) > 121*time.Second {
		t.Fatalf("invite TTL too long: %v", created.ExpiresAt)
	}

	res, err := e.svc.Register(ctx, created.Token, "new_employee", "employee-pass-99", testMeta)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if res.User.RoleID != auth.DefaultRegisteredRoleLevel {
		t.Fatalf("new user level = %d, want %d", res.User.RoleID, auth.DefaultRegisteredRoleLevel)
	}

	// Single use: the same token must be rejected now.
	if _, err := e.svc.Register(ctx, created.Token, "second_try", "employee-pass-99", testMeta); !errors.Is(err, auth.ErrInvalidInvite) {
		t.Fatalf("reuse error = %v, want ErrInvalidInvite", err)
	}

	// Unknown token.
	if _, err := e.svc.Register(ctx, "made-up-token", "third_try", "employee-pass-99", testMeta); !errors.Is(err, auth.ErrInvalidInvite) {
		t.Fatalf("unknown token error = %v, want ErrInvalidInvite", err)
	}

	// New user can log in.
	if _, err := e.svc.Login(ctx, "new_employee", "employee-pass-99", testMeta); err != nil {
		t.Fatalf("new user login: %v", err)
	}
}

func TestExpiredInviteRejected(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	ceo := e.createUser(t, "boss", "ceo-password-42", 1)

	// Insert an invitation whose 120s window is already in the past.
	plain, digest, err := token.NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	created := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Microsecond)
	repo := invitations.NewPostgresRepository()
	inv := &invitations.Invitation{
		TokenHash: digest,
		CreatedBy: ceo.ID,
		CreatedAt: created,
		ExpiresAt: created.Add(120 * time.Second),
	}
	if err := repo.Create(ctx, e.pool, inv); err != nil {
		t.Fatalf("insert expired invite: %v", err)
	}

	if _, err := e.svc.Register(ctx, plain, "late_user", "employee-pass-99", testMeta); !errors.Is(err, auth.ErrInvalidInvite) {
		t.Fatalf("expired invite error = %v, want ErrInvalidInvite", err)
	}
}

func TestRefreshRotationAndReuseDetection(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.createUser(t, "rotator", "rotate-pass-777", 4)

	login, err := e.svc.Login(ctx, "rotator", "rotate-pass-777", testMeta)
	if err != nil {
		t.Fatal(err)
	}
	sid := login.Tokens.SessionID
	firstRefresh := refreshPlain(t, login.Tokens.RefreshCookie)

	// First rotation succeeds.
	rotated, err := e.svc.Refresh(ctx, sid, firstRefresh, testMeta)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	secondRefresh := refreshPlain(t, rotated.Tokens.RefreshCookie)
	if secondRefresh == firstRefresh {
		t.Fatal("refresh token was not rotated")
	}

	// Replaying the first (already rotated) token must revoke the session.
	if _, err := e.svc.Refresh(ctx, sid, firstRefresh, testMeta); !errors.Is(err, auth.ErrInvalidRefresh) {
		t.Fatalf("replay error = %v, want ErrInvalidRefresh", err)
	}
	sess, err := e.sessions.GetByID(ctx, e.pool, sid)
	if err != nil {
		t.Fatal(err)
	}
	if sess.RevokedAt == nil {
		t.Fatal("session not revoked after refresh token reuse")
	}

	// The now-current token also dies with the session.
	if _, err := e.svc.Refresh(ctx, sid, secondRefresh, testMeta); !errors.Is(err, auth.ErrInvalidRefresh) {
		t.Fatalf("post-revocation refresh error = %v, want ErrInvalidRefresh", err)
	}
}

func TestChangePasswordRevokesOtherSessions(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.createUser(t, "changer", "original-pass-11", 6)

	first, err := e.svc.Login(ctx, "changer", "original-pass-11", testMeta)
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.svc.Login(ctx, "changer", "original-pass-11", testMeta)
	if err != nil {
		t.Fatal(err)
	}

	// Change password from the second session; wrong current is rejected.
	if err := e.svc.ChangePassword(ctx, second.User.ID, second.Tokens.SessionID, "wrong-current-1", "brand-new-pass-22", testMeta); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("wrong current error = %v, want ErrInvalidCredentials", err)
	}
	if err := e.svc.ChangePassword(ctx, second.User.ID, second.Tokens.SessionID, "original-pass-11", "brand-new-pass-22", testMeta); err != nil {
		t.Fatalf("change password: %v", err)
	}

	// First session is revoked, second survives.
	s1, err := e.sessions.GetByID(ctx, e.pool, first.Tokens.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Active(time.Now().UTC()) {
		t.Fatal("other session still active after password change")
	}
	s2, err := e.sessions.GetByID(ctx, e.pool, second.Tokens.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.Active(time.Now().UTC()) {
		t.Fatal("current session was wrongly revoked")
	}

	// Old password no longer works, new one does.
	if _, err := e.svc.Login(ctx, "changer", "original-pass-11", testMeta); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("old password error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := e.svc.Login(ctx, "changer", "brand-new-pass-22", testMeta); err != nil {
		t.Fatalf("new password login: %v", err)
	}
}

// refreshPlain extracts the token part of the "<sid>.<token>" cookie value.
func refreshPlain(t *testing.T, cookie string) string {
	t.Helper()
	for i := 0; i < len(cookie); i++ {
		if cookie[i] == '.' {
			return cookie[i+1:]
		}
	}
	t.Fatalf("malformed refresh cookie: %s", cookie)
	return ""
}
