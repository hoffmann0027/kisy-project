//go:build integration

package notes_test

import (
	"context"
	"errors"
	"testing"

	"kisy-backend/internal/notes"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/quota"
)

// Audit A-07: note files are stored bytes too — they count towards the same
// per-account quota, and their per-file ceiling comes from configuration.
func TestNoteFilesRespectTheQuotaAndTheConfiguredCeiling(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	svc := notes.NewService(pool, notes.NewPostgresRepository())
	svc.SetQuota(quota.New(quota.Policy{UserBytesBasic: 64 << 10}))
	svc.SetMaxFileBytes(50 << 10)
	basic := testdb.SeedUser(t, pool, "basic", 0)

	if _, err := svc.CreateFile(ctx, basic, "a.bin", "", make([]byte, 51<<10)); !errors.Is(err, notes.ErrTooLarge) {
		t.Fatalf("file over the configured ceiling: want ErrTooLarge, got %v", err)
	}
	if _, err := svc.CreateFile(ctx, basic, "a.bin", "", make([]byte, 40<<10)); err != nil {
		t.Fatalf("file within quota: %v", err)
	}
	if _, err := svc.CreateFile(ctx, basic, "b.bin", "", make([]byte, 40<<10)); !errors.Is(err, quota.ErrUserStorage) {
		t.Fatalf("file past the account quota: want ErrUserStorage, got %v", err)
	}
}
