package main

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/groups"
	"kisy-backend/internal/quarantine"
	"kisy-backend/internal/reports"
)

// What the queue can see of a reported thing, and who wrote it.
//
// The rule the code has to keep: a private message is encrypted, and reporting
// it does not decrypt it. The loader says "there is content here, we cannot
// read it" rather than returning an empty string that would read as "they sent
// nothing" (audit E-02, confirmed with the owner).
// canSeeGroup is the reader's view of a group or community: gone for everyone
// once deleted or archived, above someone's clearance invisible to them
// (groups.MemberCanSeeSQL), and — when closed — visible to members only.
// $1 is the reader, g the group row, u the reader's users row.
var canSeeGroup = `(` + groups.MemberCanSeeSQL("g", "u") + `
	AND (g.is_public
	     OR EXISTS (SELECT 1 FROM group_members gm WHERE gm.group_id = g.id AND gm.user_id = $1)))`

// reportVisibility answers "can this person see what they are reporting?" —
// one query per kind of target, each the same rule the screens apply.
var reportVisibility = map[string]string{
	// A person: exists and has not deleted the account. ($1 is unused here but
	// typed, since every query gets the same two arguments and Postgres
	// refuses a parameter whose type it cannot infer.)
	reports.TargetUser: `
		SELECT $1::uuid IS NOT NULL
		   AND EXISTS (SELECT 1 FROM users WHERE id = $2 AND deleted_at IS NULL)`,
	// A post: not deleted, in a community the reader can see.
	reports.TargetPost: `
		SELECT EXISTS (
			SELECT 1 FROM posts p
			JOIN groups g ON g.id = p.community_id
			JOIN users u ON u.id = $1
			WHERE p.id = $2 AND p.deleted_at IS NULL AND ` + canSeeGroup + `)`,
	// A community or a group the reader can see.
	reports.TargetCommunity: `
		SELECT EXISTS (
			SELECT 1 FROM groups g
			JOIN users u ON u.id = $1
			WHERE g.id = $2 AND ` + canSeeGroup + `)`,
	// A message: in a private chat the reader is part of, or in a group the
	// reader is a member of and can still see.
	reports.TargetMessage: `
		SELECT EXISTS (
			SELECT 1 FROM messages m
			JOIN private_chats pc ON m.chat_type = 'private' AND pc.id = m.chat_id
			WHERE m.id = $2 AND $1 IN (pc.user_a_id, pc.user_b_id)
		) OR EXISTS (
			SELECT 1 FROM messages m
			JOIN groups g ON m.chat_type = 'group' AND g.id = m.chat_id
			JOIN group_members gm ON gm.group_id = g.id AND gm.user_id = $1
			JOIN users u ON u.id = $1
			WHERE m.id = $2 AND ` + groups.MemberCanSeeSQL("g", "u") + `)`,
}

// wireReportWeight: a report from an account still in its new-account
// quarantine reaches the queue but does not count toward hiding a post — or
// five accounts registered a minute ago could hide anyone's post.
func wireReportWeight(svc *reports.Service, checker *quarantine.Checker) {
	svc.SetWeight(func(ctx context.Context, reporterID uuid.UUID) (bool, error) {
		left, err := checker.Remaining(ctx, reporterID)
		if err != nil {
			return false, err
		}
		return left <= 0, nil
	})
}

func wireReports(svc *reports.Service, pool *pgxpool.Pool) {
	svc.SetVisibility(func(ctx context.Context, reporterID uuid.UUID, kind string, id uuid.UUID) (bool, error) {
		query, ok := reportVisibility[kind]
		if !ok {
			return false, nil
		}
		var visible bool
		if err := pool.QueryRow(ctx, query, reporterID, id).Scan(&visible); err != nil {
			return false, err
		}
		return visible, nil
	})

	svc.SetOwnerResolver(func(ctx context.Context, kind string, id uuid.UUID) (uuid.UUID, bool, error) {
		var query string
		switch kind {
		case reports.TargetMessage:
			query = `SELECT sender_id FROM messages WHERE id = $1`
		case reports.TargetPost:
			query = `SELECT author_id FROM posts WHERE id = $1`
		case reports.TargetCommunity:
			query = `SELECT created_by FROM groups WHERE id = $1`
		default:
			return uuid.Nil, false, nil
		}
		var owner uuid.UUID
		err := pool.QueryRow(ctx, query, id).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, false, nil
		}
		if err != nil {
			return uuid.Nil, false, err
		}
		return owner, true, nil
	})

	svc.SetContentLoader(func(ctx context.Context, kind string, id uuid.UUID) (string, bool, error) {
		switch kind {
		case reports.TargetPost:
			var text string
			err := pool.QueryRow(ctx, `SELECT COALESCE(text, '') FROM posts WHERE id = $1`, id).Scan(&text)
			if errors.Is(err, pgx.ErrNoRows) {
				return "", false, nil
			}
			return text, err == nil, err

		case reports.TargetMessage:
			var chatType string
			var text *string
			err := pool.QueryRow(ctx,
				`SELECT chat_type, text FROM messages WHERE id = $1`, id).Scan(&chatType, &text)
			if errors.Is(err, pgx.ErrNoRows) {
				return "", false, nil
			}
			if err != nil {
				return "", false, err
			}
			// A private message is end-to-end encrypted: the server holds
			// ciphertext and no key. A group message it can read.
			if chatType != "group" || text == nil {
				return "", false, nil
			}
			return *text, true, nil

		case reports.TargetCommunity:
			var name string
			err := pool.QueryRow(ctx, `SELECT name FROM groups WHERE id = $1`, id).Scan(&name)
			if errors.Is(err, pgx.ErrNoRows) {
				return "", false, nil
			}
			return name, err == nil, err

		default:
			return "", false, nil
		}
	})
}
