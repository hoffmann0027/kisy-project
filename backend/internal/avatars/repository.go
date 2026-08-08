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

// Image is a stored avatar's bytes and metadata. Bytes is empty when the image
// lives in object storage; StoragePath then names the object.
type Image struct {
	ContentType string
	Bytes       []byte
	StoragePath string
	UpdatedAt   time.Time
}

// Repository is the persistence port for avatars.
type Repository interface {
	// Upsert stores an avatar. Exactly one of bytes/storagePath carries the
	// image (the schema enforces it) — bytes for the in-database path,
	// storagePath when the image was written to object storage.
	Upsert(ctx context.Context, q db.DBTX, ownerType string, ownerID uuid.UUID, contentType string, bytes []byte, storagePath string) (time.Time, error)
	Get(ctx context.Context, q db.DBTX, ownerType string, ownerID uuid.UUID) (Image, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Upsert(ctx context.Context, q db.DBTX, ownerType string, ownerID uuid.UUID, contentType string, bytes []byte, storagePath string) (time.Time, error) {
	var updatedAt time.Time
	err := q.QueryRow(ctx, `
		INSERT INTO avatars (owner_type, owner_id, content_type, bytes, storage_path, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (owner_type, owner_id)
		DO UPDATE SET content_type = EXCLUDED.content_type, bytes = EXCLUDED.bytes,
		              storage_path = EXCLUDED.storage_path, updated_at = now()
		RETURNING updated_at`,
		ownerType, ownerID, contentType, bytes, storagePath).Scan(&updatedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("avatars: upsert: %w", err)
	}
	return updatedAt, nil
}

func (r *PostgresRepository) Get(ctx context.Context, q db.DBTX, ownerType string, ownerID uuid.UUID) (Image, error) {
	var img Image
	err := q.QueryRow(ctx, `
		SELECT content_type, bytes, storage_path, updated_at
		FROM avatars WHERE owner_type = $1 AND owner_id = $2`,
		ownerType, ownerID).Scan(&img.ContentType, &img.Bytes, &img.StoragePath, &img.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Image{}, ErrNotFound
	}
	if err != nil {
		return Image{}, fmt.Errorf("avatars: get: %w", err)
	}
	return img, nil
}
