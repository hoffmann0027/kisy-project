package main

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/reports"
)

// What the queue can see of a reported thing, and who wrote it.
//
// The rule the code has to keep: a private message is encrypted, and reporting
// it does not decrypt it. The loader says "there is content here, we cannot
// read it" rather than returning an empty string that would read as "they sent
// nothing" (audit E-02, confirmed with the owner).
func wireReports(svc *reports.Service, pool *pgxpool.Pool) {
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
