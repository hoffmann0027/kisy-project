package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/auth/token"
	"kisy-backend/internal/platform/db"
)

// stubSessions answers GetByID with whatever the test needs; the other
// methods are never called by the middleware.
type stubSessions struct {
	session *Session
	err     error
}

func (s stubSessions) Create(context.Context, db.DBTX, *Session) error { return nil }
func (s stubSessions) GetByID(context.Context, db.DBTX, uuid.UUID) (*Session, error) {
	return s.session, s.err
}
func (s stubSessions) Rotate(context.Context, db.DBTX, uuid.UUID, string, time.Time, time.Time) error {
	return nil
}
func (s stubSessions) Revoke(context.Context, db.DBTX, uuid.UUID, time.Time) error { return nil }
func (s stubSessions) RevokeAllForUser(context.Context, db.DBTX, uuid.UUID, time.Time) (int64, error) {
	return 0, nil
}
func (s stubSessions) RevokeAllForUserExcept(context.Context, db.DBTX, uuid.UUID, uuid.UUID, time.Time) (int64, error) {
	return 0, nil
}

// TestRequireAuthSeparatesDeadSessionFromDeadDatabase pins audit B-09: a
// database outage must not look like a revoked session. Before the fix every
// error from the session store became 401, so a 20-second Neon hiccup signed
// every web user out and the error-rate alert never fired.
func TestRequireAuthSeparatesDeadSessionFromDeadDatabase(t *testing.T) {
	userID, sessionID := uuid.New(), uuid.New()
	tokens := token.NewManager("test-secret-value-for-middleware-tests", time.Minute)
	access, _, err := tokens.IssueAccess(userID, sessionID, 1, "ceo")
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	live := &Session{ID: sessionID, UserID: userID, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	revoked := time.Now().UTC().Add(-time.Minute)

	cases := []struct {
		name string
		repo stubSessions
		want int
	}{
		{"database unavailable", stubSessions{err: errors.New("dial tcp: connection refused")}, http.StatusServiceUnavailable},
		{"session deleted", stubSessions{err: ErrSessionNotFound}, http.StatusUnauthorized},
		{"session revoked", stubSessions{session: &Session{ID: sessionID, UserID: userID, ExpiresAt: live.ExpiresAt, RevokedAt: &revoked}}, http.StatusUnauthorized},
		{"session live", stubSessions{session: live}, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMiddleware(tokens, tc.repo, nil)
			handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			req.Header.Set("Authorization", "Bearer "+access)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

// TestRefreshKeepsCookiesWhenTheStoreIsDown pins the second half of B-09: the
// refresh endpoint used to expire the auth cookies on any error, so a database
// hiccup left the browser with no refresh token to retry with — a forced
// logout dressed up as a 500.
func TestRefreshKeepsCookiesWhenTheStoreIsDown(t *testing.T) {
	sessionID := uuid.New()
	tokens := token.NewManager("test-secret-value-for-middleware-tests", time.Minute)

	call := func(repo stubSessions) *httptest.ResponseRecorder {
		h := NewHandler(&Service{sessions: repo, tokens: tokens, refreshTTL: time.Hour}, nil, "salt", false)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: RefreshCookieName, Value: sessionID.String() + "." + "secret-part"})
		rec := httptest.NewRecorder()
		h.refresh(rec, req)
		return rec
	}

	// clears reports whether the response expires the refresh cookie.
	clears := func(rec *httptest.ResponseRecorder) bool {
		for _, c := range rec.Result().Cookies() {
			if c.Name == RefreshCookieName && c.MaxAge < 0 {
				return true
			}
		}
		return false
	}

	down := call(stubSessions{err: errors.New("dial tcp: connection refused")})
	if down.Code != http.StatusInternalServerError {
		t.Fatalf("store down: status = %d, want 500 (body %s)", down.Code, down.Body.String())
	}
	if clears(down) {
		t.Fatal("store down: the refresh cookie was cleared, so the retry has nothing to send")
	}

	gone := call(stubSessions{err: ErrSessionNotFound})
	if gone.Code != http.StatusUnauthorized {
		t.Fatalf("session gone: status = %d, want 401", gone.Code)
	}
	if !clears(gone) {
		t.Fatal("session gone: the dead refresh cookie must be cleared")
	}
}

// TestSeededPasswordUnlocksNothingButItsOwnChange pins audit A-16: the
// must_change_password flag was enforced only by the client's screens, so a
// temporary password set by the CEO — or the bootstrap one — kept working for
// every API call and for /admin, which skipped that screen.
func TestSeededPasswordUnlocksNothingButItsOwnChange(t *testing.T) {
	userID, sessionID := uuid.New(), uuid.New()
	tokens := token.NewManager("test-secret-value-for-middleware-tests", time.Minute)
	access, _, err := tokens.IssueAccess(userID, sessionID, 1, "ceo")
	if err != nil {
		t.Fatal(err)
	}
	seeded := stubSessions{session: &Session{
		ID: sessionID, UserID: userID, ExpiresAt: time.Now().UTC().Add(time.Hour), MustChangePassword: true,
	}}
	m := NewMiddleware(tokens, seeded, nil)
	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+access)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	for _, refused := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/chats"},
		{http.MethodPost, "/api/v1/messages"},
		{http.MethodPatch, "/api/v1/users/me"},
		{http.MethodGet, "/api/v1/admin/users"},
	} {
		if got := call(refused.method, refused.path); got != http.StatusForbidden {
			t.Errorf("%s %s = %d with a seeded password, want 403", refused.method, refused.path, got)
		}
	}
	for _, allowed := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/users/me"},
		{http.MethodPost, "/api/v1/auth/password"},
		{http.MethodPost, "/api/v1/auth/logout"},
		{http.MethodDelete, "/api/v1/users/me"},
	} {
		if got := call(allowed.method, allowed.path); got != http.StatusOK {
			t.Errorf("%s %s = %d with a seeded password, want it allowed", allowed.method, allowed.path, got)
		}
	}

	// The WebSocket is the whole app; it does not open either.
	req := httptest.NewRequest(http.MethodGet, "/ws?access_token="+access, nil)
	if m.Authenticate(req) != nil {
		t.Fatal("the WebSocket authenticated an account that must change its password")
	}

	// Once changed, everything opens.
	seeded.session.MustChangePassword = false
	if got := call(http.MethodGet, "/api/v1/chats"); got != http.StatusOK {
		t.Fatalf("after the change: %d", got)
	}
}
