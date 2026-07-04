// Package avatars stores and serves user and group avatar images from the
// database, so they survive redeploys on platforms with ephemeral disks.
package avatars

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
)

// ErrNotFound is returned when no avatar exists for an owner.
var ErrNotFound = errors.New("avatars: not found")

// Image is a stored avatar's bytes and metadata.
type Image struct {
	ContentType string
	Bytes       []byte
	UpdatedAt   time.Time
}

// Repository is the persistence port for avatars.
type Repository interface {
	Upsert(ctx context.Context, q db.DBTX, ownerType string, ownerID uuid.UUID, contentType string, bytes []byte) (time.Time, error)
	Get(ctx context.Context, q db.DBTX, ownerType string, ownerID uuid.UUID) (Image, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Upsert(ctx context.Context, q db.DBTX, ownerType string, ownerID uuid.UUID, contentType string, bytes []byte) (time.Time, error) {
	var updatedAt time.Time
	err := q.QueryRow(ctx, `
		INSERT INTO avatars (owner_type, owner_id, content_type, bytes, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (owner_type, owner_id)
		DO UPDATE SET content_type = EXCLUDED.content_type, bytes = EXCLUDED.bytes, updated_at = now()
		RETURNING updated_at`,
		ownerType, ownerID, contentType, bytes).Scan(&updatedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("avatars: upsert: %w", err)
	}
	return updatedAt, nil
}

func (r *PostgresRepository) Get(ctx context.Context, q db.DBTX, ownerType string, ownerID uuid.UUID) (Image, error) {
	var img Image
	err := q.QueryRow(ctx, `
		SELECT content_type, bytes, updated_at
		FROM avatars WHERE owner_type = $1 AND owner_id = $2`,
		ownerType, ownerID).Scan(&img.ContentType, &img.Bytes, &img.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Image{}, ErrNotFound
	}
	if err != nil {
		return Image{}, fmt.Errorf("avatars: get: %w", err)
	}
	return img, nil
}
