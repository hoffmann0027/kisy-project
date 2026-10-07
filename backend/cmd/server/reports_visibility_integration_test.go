//go:build integration

package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/posts"
	"kisy-backend/internal/quarantine"
	"kisy-backend/internal/reports"
	"kisy-backend/internal/users"
)

// Two holes in reporting, found while opening reports on posts, people and
// communities to the app:
//
//   - the server accepted a report on anything whose id it was given — a post
//     in a closed community, a message in somebody else's chat — so the
//     endpoint answered questions about things the reporter could not see, and
//     strangers could vote a post they were never shown out of the feed;
//   - every report counted toward hiding a post, including those of accounts
//     registered minutes ago, which the new-account quarantine stops from
//     posting but not from reporting: five fresh sign-ups hid any public post.

func (f reportFixture) try(reporter uuid.UUID, kind string, id uuid.UUID) error {
	_, err := f.svc.Create(f.ctx, reports.ActorMeta{UserID: reporter}, reports.Input{
		TargetKind: kind, TargetID: id, Reason: "spam",
	})
	return err
}

func TestOnlyWhatYouCanSeeCanBeReported(t *testing.T) {
	f := newReportFixture(t)
	suffix := uuid.NewString()[:8]
	outsider, member := f.readers[0], f.readers[1]

	// A closed community, and a post in it.
	closed, err := f.groups.Create(f.ctx, groups.CreateInput{
		Name: "Closed " + suffix, Kind: groups.KindCommunity, IsPublic: false,
	}, groups.ActorMeta{UserID: f.author, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := f.posts.Create(f.ctx, posts.CreateInput{CommunityID: closed.ID, Text: "только для своих"},
		posts.ActorMeta{UserID: f.author, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)`, closed.ID, member); err != nil {
		t.Fatal(err)
	}

	if err := f.try(outsider, reports.TargetPost, hidden.ID); !errors.Is(err, reports.ErrNotFound) {
		t.Fatalf("a post in a closed community was reportable from outside: %v", err)
	}
	if err := f.try(outsider, reports.TargetCommunity, closed.ID); !errors.Is(err, reports.ErrNotFound) {
		t.Fatalf("a closed community was reportable from outside: %v", err)
	}
	if err := f.try(member, reports.TargetPost, hidden.ID); err != nil {
		t.Fatalf("a member could not report a post they can read: %v", err)
	}

	// A community above the reader's clearance is invisible to them too.
	high, err := f.groups.Create(f.ctx, groups.CreateInput{
		Name: "Leadership " + suffix, Kind: groups.KindCommunity, IsPublic: true,
	}, groups.ActorMeta{UserID: f.author, RoleLevel: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(f.ctx, `UPDATE groups SET min_role_level = 3 WHERE id = $1`, high.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.try(outsider, reports.TargetCommunity, high.ID); !errors.Is(err, reports.ErrNotFound) {
		t.Fatalf("a community above the reader's level was reportable: %v", err)
	}

	// Somebody else's private conversation.
	var chatID, messageID uuid.UUID
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO private_chats (user_a_id, user_b_id, initiated_by)
		VALUES ($1, $2, $1) RETURNING id`, f.author, member).Scan(&chatID); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO messages (chat_type, chat_id, sender_id, ciphertext, alg, epoch)
		VALUES ('private', $1, $2, decode('00', 'hex'), 1, 1) RETURNING id`, chatID, f.author).Scan(&messageID); err != nil {
		t.Fatal(err)
	}
	if err := f.try(outsider, reports.TargetMessage, messageID); !errors.Is(err, reports.ErrNotFound) {
		t.Fatalf("a message from somebody else's chat was reportable: %v", err)
	}
	if err := f.try(member, reports.TargetMessage, messageID); err != nil {
		t.Fatalf("a participant could not report a message they received: %v", err)
	}

	// Nothing there at all answers the same as "not yours to see".
	if err := f.try(outsider, reports.TargetPost, uuid.New()); !errors.Is(err, reports.ErrNotFound) {
		t.Fatalf("a made-up post id: %v", err)
	}

	// And what is public stays reportable by anyone.
	if err := f.try(outsider, reports.TargetPost, f.post.ID); err != nil {
		t.Fatalf("a public post could not be reported: %v", err)
	}
	if err := f.try(outsider, reports.TargetUser, f.author); err != nil {
		t.Fatalf("a person could not be reported: %v", err)
	}
}

func TestFreshAccountsCannotHideAPostTogether(t *testing.T) {
	f := newReportFixture(t)

	// The readers become basic accounts registered just now: inside the
	// quarantine, like a batch of sign-ups made to silence someone.
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE users SET account_kind = 'basic', role_id = NULL, created_at = now()
		WHERE id = ANY($1)`, f.readers); err != nil {
		t.Fatal(err)
	}
	repo := users.NewPostgresRepository()
	checker := quarantine.New(quarantine.Policy{Duration: 24 * time.Hour},
		func(ctx context.Context, id uuid.UUID) (quarantine.Account, bool, error) {
			u, err := repo.GetByID(ctx, f.pool, id)
			if errors.Is(err, users.ErrNotFound) {
				return quarantine.Account{}, false, nil
			}
			if err != nil {
				return quarantine.Account{}, false, err
			}
			return quarantine.Account{Invited: u.AccountKind == users.KindInvited, CreatedAt: u.CreatedAt}, true, nil
		})
	wireReportWeight(f.svc, checker)

	for i := 0; i < reports.AutoHideThreshold; i++ {
		f.report(t, f.readers[i], reports.TargetPost, f.post.ID)
	}

	// The reports reached the queue...
	queue, err := f.svc.List(f.ctx, reports.StatusOpen, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(queue) != reports.AutoHideThreshold {
		t.Fatalf("queue = %d, want every report kept", len(queue))
	}
	// ...but five accounts minutes old did not take the post out of the feed.
	observer := testdb.SeedUser(t, f.pool, "observer_"+uuid.NewString()[:8], 5)
	if !f.inFeedOf(t, observer) {
		t.Fatal("five quarantined accounts hid a post")
	}

	// Once those accounts are past the quarantine, their next report counts.
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE users SET created_at = now() - interval '2 days' WHERE id = ANY($1)`, f.readers); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < reports.AutoHideThreshold; i++ {
		f.report(t, f.readers[i], reports.TargetPost, f.post.ID)
	}
	if f.inFeedOf(t, observer) {
		t.Fatal("five established accounts no longer hide a post: the threshold itself broke")
	}
}
