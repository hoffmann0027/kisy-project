//go:build integration

package admin_test

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/admin"
	"kisy-backend/internal/platform/testdb"
)

// Audit A-03: a deactivated, demoted or reset account lost its sessions, but
// its open WebSockets kept receiving and sending. Each of these must also end
// the target's live connections — after the commit, so a reconnect cannot get
// in before the revocation is visible.

type userKicks struct {
	mu    sync.Mutex
	users []uuid.UUID
}

func (k *userKicks) KickSession(userID, _ uuid.UUID) { k.KickUser(userID, uuid.Nil) }

func (k *userKicks) KickUser(userID, keep uuid.UUID) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if keep == uuid.Nil {
		k.users = append(k.users, userID)
	}
}

func (k *userKicks) count(id uuid.UUID) int {
	k.mu.Lock()
	defer k.mu.Unlock()
	n := 0
	for _, u := range k.users {
		if u == id {
			n++
		}
	}
	return n
}

func TestAdminRevocationsEndTargetSockets(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	kick := &userKicks{}
	e.svc.SetSessionKicker(kick)

	ceo := testdb.SeedUser(t, e.pool, "ceo_kick_it", 1)
	actor := admin.ActorMeta{UserID: ceo, SessionID: openSession(t, e, ceo)}

	cases := []struct {
		name string
		run  func(target uuid.UUID) error
	}{
		{"deactivation", func(target uuid.UUID) error { return e.svc.SetActive(ctx, target, false, actor) }},
		{"role change", func(target uuid.UUID) error { return e.svc.ChangeRole(ctx, target, 9, actor) }},
		{"password reset", func(target uuid.UUID) error {
			return e.svc.ResetPassword(ctx, target, "a-new-long-password-1", actor)
		}},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := testdb.SeedUser(t, e.pool, "kick_target_"+string(rune('a'+i)), 5)
			openSession(t, e, target)
			if err := tc.run(target); err != nil {
				t.Fatal(err)
			}
			if kick.count(target) != 1 {
				t.Fatalf("%s did not end the target's sockets (kicks: %v)", tc.name, kick.users)
			}
		})
	}

	t.Run("reactivation kicks nobody", func(t *testing.T) {
		target := testdb.SeedUser(t, e.pool, "kick_target_z", 5)
		if err := e.svc.SetActive(ctx, target, true, actor); err != nil {
			t.Fatal(err)
		}
		if kick.count(target) != 0 {
			t.Fatal("activating an account must not end its sockets")
		}
	})
	if slices.Contains(kick.users, ceo) {
		t.Fatal("the acting CEO was kicked")
	}
}
