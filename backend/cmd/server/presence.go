package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/ws"
)

// chatPartnersOnly is the presence rule: you may follow the online status of
// the people you share a private chat with — exactly the list the app
// subscribes to — and nobody else (audit A-19). One query for the whole list.
func chatPartnersOnly(pool *pgxpool.Pool) ws.PresenceFilter {
	return func(ctx context.Context, subscriber uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, error) {
		rows, err := pool.Query(ctx, `
			SELECT DISTINCT CASE WHEN user_a_id = $1 THEN user_b_id ELSE user_a_id END
			FROM private_chats
			WHERE (user_a_id = $1 AND user_b_id = ANY($2))
			   OR (user_b_id = $1 AND user_a_id = ANY($2))`, subscriber, targets)
		if err != nil {
			return nil, err
		}
		return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	}
}
