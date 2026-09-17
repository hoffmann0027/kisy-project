//go:build integration

package auth_test

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"
)

// Audit A-03: revoking a session stopped new requests but left the sockets
// opened with it running. Every path that revokes sessions must also end
// their live connections.

type kickCall struct {
	user, session, keep uuid.UUID
	all                 bool
}

type recordingKicker struct {
	mu    sync.Mutex
	calls []kickCall
}

func (k *recordingKicker) KickSession(userID, sessionID uuid.UUID) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.calls = append(k.calls, kickCall{user: userID, session: sessionID})
}

func (k *recordingKicker) KickUser(userID, keep uuid.UUID) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.calls = append(k.calls, kickCall{user: userID, keep: keep, all: true})
}

func (k *recordingKicker) saw(c kickCall) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return slices.Contains(k.calls, c)
}

func TestSessionRevocationsEndLiveSockets(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	kick := &recordingKicker{}
	e.svc.SetSessionKicker(kick)
	u := e.createUser(t, "kicked", "kicked-pass-123", 5)

	login := func() (uuid.UUID, string) {
		t.Helper()
		res, err := e.svc.Login(ctx, "kicked", "kicked-pass-123", testMeta)
		if err != nil {
			t.Fatal(err)
		}
		return res.Tokens.SessionID, refreshPlain(t, res.Tokens.RefreshCookie)
	}

	t.Run("logout ends that session's sockets", func(t *testing.T) {
		sid, _ := login()
		if err := e.svc.Logout(ctx, u.ID, sid, testMeta); err != nil {
			t.Fatal(err)
		}
		if !kick.saw(kickCall{user: u.ID, session: sid}) {
			t.Fatalf("logout did not kick session %s: %+v", sid, kick.calls)
		}
	})

	t.Run("logout everywhere ends every socket", func(t *testing.T) {
		sid, _ := login()
		if _, err := e.svc.LogoutAll(ctx, u.ID, sid, testMeta); err != nil {
			t.Fatal(err)
		}
		if !kick.saw(kickCall{user: u.ID, all: true}) {
			t.Fatalf("logout-all did not kick the user: %+v", kick.calls)
		}
	})

	t.Run("password change ends the other sessions' sockets", func(t *testing.T) {
		current, _ := login()
		if err := e.svc.ChangePassword(ctx, u.ID, current, "kicked-pass-123", "kicked-pass-456", testMeta); err != nil {
			t.Fatal(err)
		}
		if !kick.saw(kickCall{user: u.ID, keep: current, all: true}) {
			t.Fatalf("password change did not kick the other sessions: %+v", kick.calls)
		}
		if err := e.svc.ChangePassword(ctx, u.ID, current, "kicked-pass-456", "kicked-pass-123", testMeta); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("refresh token reuse ends the stolen session's sockets", func(t *testing.T) {
		sid, first := login()
		if _, err := e.svc.Refresh(ctx, sid, first, testMeta); err != nil {
			t.Fatal(err)
		}
		_, _ = e.svc.Refresh(ctx, sid, first, testMeta) // replay
		if !kick.saw(kickCall{user: u.ID, session: sid}) {
			t.Fatalf("refresh reuse did not kick the session: %+v", kick.calls)
		}
	})
}
