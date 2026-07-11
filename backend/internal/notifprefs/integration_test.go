//go:build integration

package notifprefs_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/notifprefs"
	"kisy-backend/internal/platform/testdb"
)

func TestMuteLifecycle(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	svc := notifprefs.NewService(pool, notifprefs.NewPostgresRepository())

	alice := testdb.SeedUser(t, pool, "alice", 3)
	bob := testdb.SeedUser(t, pool, "bob", 8)
	chat := uuid.New()

	// Not muted initially.
	muted, err := svc.MutedUsers(ctx, "group", chat, []uuid.UUID{alice, bob})
	if err != nil || len(muted) != 0 {
		t.Fatalf("initial: %v, %v", muted, err)
	}

	// Alice mutes forever.
	if err := svc.Mute(ctx, alice, "group", chat, nil); err != nil {
		t.Fatalf("mute forever: %v", err)
	}
	muted, _ = svc.MutedUsers(ctx, "group", chat, []uuid.UUID{alice, bob})
	if _, ok := muted[alice]; !ok {
		t.Fatalf("alice should be muted")
	}
	if _, ok := muted[bob]; ok {
		t.Fatalf("bob should not be muted")
	}

	// Timed mute in the past is treated as not muted; future mute counts.
	future := time.Now().Add(time.Hour)
	if err := svc.Mute(ctx, bob, "group", chat, &future); err != nil {
		t.Fatalf("timed mute: %v", err)
	}
	muted, _ = svc.MutedUsers(ctx, "group", chat, []uuid.UUID{alice, bob})
	if len(muted) != 2 {
		t.Fatalf("both should be muted, got %v", muted)
	}

	// A past mute is rejected by validation.
	past := time.Now().Add(-time.Hour)
	if err := svc.Mute(ctx, bob, "group", chat, &past); err != notifprefs.ErrValidation {
		t.Fatalf("past mute: want ErrValidation, got %v", err)
	}

	// Unmute alice.
	if err := svc.Unmute(ctx, alice, "group", chat); err != nil {
		t.Fatalf("unmute: %v", err)
	}
	muted, _ = svc.MutedUsers(ctx, "group", chat, []uuid.UUID{alice, bob})
	if _, ok := muted[alice]; ok {
		t.Fatalf("alice unmuted")
	}

	// ListMutes returns bob's future mute.
	list, err := svc.ListMutes(ctx, bob)
	if err != nil || len(list) != 1 || list[0].ChatID != chat {
		t.Fatalf("list mutes: %v, %v", list, err)
	}
}

func TestSettingsRoundtrip(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	svc := notifprefs.NewService(pool, notifprefs.NewPostgresRepository())
	alice := testdb.SeedUser(t, pool, "alice", 3)

	// Defaults when absent.
	s, err := svc.GetSettings(ctx, alice)
	if err != nil || !s.Sound || !s.Preview || s.GroupMode != notifprefs.GroupAll {
		t.Fatalf("defaults: %+v %v", s, err)
	}

	// Update and read back.
	updated, err := svc.UpdateSettings(ctx, alice, notifprefs.Settings{Sound: false, Preview: false, GroupMode: notifprefs.GroupMentionsOnly})
	if err != nil || updated.GroupMode != notifprefs.GroupMentionsOnly {
		t.Fatalf("update: %+v %v", updated, err)
	}
	s, _ = svc.GetSettings(ctx, alice)
	if s.Sound || s.Preview || s.GroupMode != notifprefs.GroupMentionsOnly {
		t.Fatalf("read back: %+v", s)
	}

	// Batch load reflects the update; an unknown user gets defaults.
	stranger := uuid.New()
	all, err := svc.SettingsFor(ctx, []uuid.UUID{alice, stranger})
	if err != nil {
		t.Fatal(err)
	}
	if all[alice].GroupMode != notifprefs.GroupMentionsOnly {
		t.Fatalf("batch alice: %+v", all[alice])
	}
	if all[stranger].GroupMode != notifprefs.GroupAll {
		t.Fatalf("batch stranger default: %+v", all[stranger])
	}

	// Invalid mode rejected.
	if _, err := svc.UpdateSettings(ctx, alice, notifprefs.Settings{GroupMode: "weird"}); err != notifprefs.ErrValidation {
		t.Fatalf("invalid mode: want ErrValidation, got %v", err)
	}
}
