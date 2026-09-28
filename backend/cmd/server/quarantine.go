package main

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/attachments"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/config"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/notes"
	"kisy-backend/internal/posts"
	"kisy-backend/internal/quarantine"
	"kisy-backend/internal/users"
)

// heldBack is everything the new-account quarantine applies to. Gathered in
// one place so the hold cannot be wired into three services and forgotten in
// the fourth. Any field may be nil (tests wire what they exercise).
type heldBack struct {
	posts       *posts.Service
	groups      *groups.Service
	attachments *attachments.Service
	notes       *notes.Service
	chats       *chats.Service
	users       *users.Handler
}

// wireQuarantine builds the hold on freshly self-registered accounts and puts
// it into every service that has to honour it.
func wireQuarantine(c config.QuarantineConfig, pool *pgxpool.Pool, repo users.Repository, h heldBack) *quarantine.Checker {
	checker := quarantine.New(quarantine.Policy{
		Duration:       time.Duration(c.Hours) * time.Hour,
		NewChatsPerDay: c.NewChatsPerDay,
		MaxUploadBytes: c.MaxUploadBytes,
	}, func(ctx context.Context, id uuid.UUID) (quarantine.Account, bool, error) {
		// The account as it is now: an access token could be minutes stale,
		// and the hold is measured in hours.
		u, err := repo.GetByID(ctx, pool, id)
		if errors.Is(err, users.ErrNotFound) {
			return quarantine.Account{}, false, nil
		}
		if err != nil {
			return quarantine.Account{}, false, err
		}
		return quarantine.Account{Invited: u.AccountKind == users.KindInvited, CreatedAt: u.CreatedAt}, true, nil
	})

	if h.posts != nil {
		h.posts.SetQuarantine(checker)
	}
	if h.groups != nil {
		h.groups.SetQuarantine(checker)
	}
	if h.attachments != nil {
		h.attachments.SetQuarantine(checker)
	}
	if h.notes != nil {
		h.notes.SetQuarantine(checker)
	}
	if h.users != nil {
		h.users.SetQuarantine(checker.StatusFor)
	}
	if h.chats != nil {
		checker.SetNewChatsToday(h.chats.StartedSince)
		// A new conversation passes both gates: the quarantine's daily
		// allowance first (its message explains the hold), then the ordinary
		// per-account rate.
		rate := h.chats.NewChatGate()
		h.chats.SetNewChatGate(func(ctx context.Context, actor chats.ActorMeta) error {
			if err := checker.AllowNewChat(ctx, actor.UserID); err != nil {
				return err
			}
			return rate(ctx, actor)
		})
	}
	return checker
}
