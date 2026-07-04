package avatars

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // register JPEG decoder
	_ "image/png"  // register PNG decoder
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// MaxBytes caps stored avatar size. Clients crop/resize before upload, so
	// this is a generous ceiling against abuse, not the expected size.
	MaxBytes = 512 * 1024
	maxDim   = 1024
	minDim   = 16
)

// OwnerUser and OwnerGroup are the valid owner types.
const (
	OwnerUser  = "user"
	OwnerGroup = "group"
)

var (
	// ErrTooLarge, ErrUnsupported and ErrNotSquare are validation failures the
	// handler maps to 400s.
	ErrTooLarge    = errors.New("avatars: image exceeds size limit")
	ErrUnsupported = errors.New("avatars: unsupported image type (jpeg or png only)")
	ErrNotSquare   = errors.New("avatars: image must be square")
	ErrBadImage    = errors.New("avatars: could not decode image")
)

// Service validates and stores avatar images and reports back a versioned URL
// the owner's avatar_url column should point at.
type Service struct {
	pool *pgxpool.Pool
	repo Repository
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

// Store validates raw image bytes and persists them for the owner, returning
// a versioned URL (cache-busted by the update time) for the avatar_url column.
// The content type is sniffed from the bytes, never trusted from the client.
func (s *Service) Store(ctx context.Context, ownerType string, ownerID uuid.UUID, raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", ErrBadImage
	}
	if len(raw) > MaxBytes {
		return "", ErrTooLarge
	}

	ct := http.DetectContentType(raw)
	if ct != "image/jpeg" && ct != "image/png" {
		return "", ErrUnsupported
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return "", ErrBadImage
	}
	if cfg.Width < minDim || cfg.Height < minDim || cfg.Width > maxDim || cfg.Height > maxDim {
		return "", ErrBadImage
	}
	if cfg.Width != cfg.Height {
		return "", ErrNotSquare
	}
	// Fully decode to reject truncated/corrupt payloads that pass the header.
	if _, _, err := image.Decode(bytes.NewReader(raw)); err != nil {
		return "", ErrBadImage
	}

	updatedAt, err := s.repo.Upsert(ctx, s.pool, ownerType, ownerID, ct, raw)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/api/v1/avatars/%s/%s?v=%d", ownerType, ownerID, updatedAt.Unix()), nil
}

// Load returns an owner's stored avatar, or ErrNotFound.
func (s *Service) Load(ctx context.Context, ownerType string, ownerID uuid.UUID) (Image, error) {
	return s.repo.Get(ctx, s.pool, ownerType, ownerID)
}
