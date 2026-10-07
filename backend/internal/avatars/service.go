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

	"kisy-backend/internal/platform/blobstore"
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

// ImageError is a problem with the uploaded image itself. Its message is the
// one thing about a failed upload that may be shown to the uploader: the
// handlers used to send err.Error() for every failure, so a database error
// (with its pgx text) or an object-storage one (with the endpoint and bucket)
// went straight to the client (audit A-42). Callers recognise it with
// errors.As against interface{ UserMessage() string }, without importing this
// package.
type ImageError struct{ msg string }

func (e ImageError) Error() string       { return "avatars: " + e.msg }
func (e ImageError) UserMessage() string { return e.msg }

var (
	// Validation failures, each answered as a 400 with its own message.
	ErrTooLarge    = ImageError{"image exceeds size limit"}
	ErrUnsupported = ImageError{"unsupported image type (jpeg or png only)"}
	ErrNotSquare   = ImageError{"image must be square"}
	ErrBadImage    = ImageError{"could not decode image"}
)

// Service validates and stores avatar images and reports back a versioned URL
// the owner's avatar_url column should point at.
type Service struct {
	pool  *pgxpool.Pool
	repo  Repository
	blobs blobstore.Store
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

// SetBlobStore routes new avatars to object storage. Reads stay dual-path, so
// avatars written before the switch keep working.
func (s *Service) SetBlobStore(b blobstore.Store) { s.blobs = b }

// objectKey is deterministic: one avatar per owner, and an upsert overwrites
// the same object instead of leaking a new one on every change.
func objectKey(ownerType string, ownerID uuid.UUID) string {
	return "avatars/" + ownerType + "/" + ownerID.String()
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

	// Bytes to the object store first (if configured); the row then holds only
	// the key, keeping images out of the database and its dumps.
	data, path := raw, ""
	if s.blobs != nil {
		path = objectKey(ownerType, ownerID)
		if err := s.blobs.Put(ctx, path, raw, ct); err != nil {
			return "", err
		}
		data = nil
	}

	updatedAt, err := s.repo.Upsert(ctx, s.pool, ownerType, ownerID, ct, data, path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/api/v1/avatars/%s/%s?v=%d", ownerType, ownerID, updatedAt.Unix()), nil
}

// Load returns an owner's stored avatar, or ErrNotFound. Images are served
// through this backend, never by a public bucket URL, so the existing access
// rules keep applying.
func (s *Service) Load(ctx context.Context, ownerType string, ownerID uuid.UUID) (Image, error) {
	img, err := s.repo.Get(ctx, s.pool, ownerType, ownerID)
	if err != nil || img.StoragePath == "" {
		return img, err
	}
	if s.blobs == nil {
		return Image{}, fmt.Errorf("avatars: object storage is not configured on this instance")
	}
	data, err := s.blobs.Get(ctx, img.StoragePath)
	if err != nil {
		if errors.Is(err, blobstore.ErrNotFound) {
			return Image{}, ErrNotFound
		}
		return Image{}, err
	}
	img.Bytes = data
	return img, nil
}
