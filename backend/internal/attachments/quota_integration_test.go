//go:build integration

package attachments_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/attachments"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/quota"
)

// Audit A-07: with open registration and bytes kept in Postgres, per-file
// ceilings were the only bound — one free account could upload until the
// database was full. These tests pin the per-account quota on every path that
// stores attachment bytes.

const (
	basicQuota   = 64 << 10  // 64 KiB
	invitedQuota = 256 << 10 // 256 KiB
	basicPerFile = 48 << 10
)

func quotaSetup(t *testing.T) (*attachments.Service, *pgxpool.Pool, context.Context) {
	t.Helper()
	pool := testdb.New(t)
	svc := attachments.NewService(pool, attachments.NewPostgresRepository(), attachments.Limits{
		MaxBytesLeadership: leaderLimit,
		MaxBytesStaff:      staffLimit,
		MaxBytesBasic:      basicPerFile,
		LeadershipMaxLevel: 3,
		ChunkBytes:         testChunk,
		SessionTTL:         time.Hour,
	})
	svc.SetQuota(quota.New(quota.Policy{UserBytesBasic: basicQuota, UserBytesInvited: invitedQuota}))
	return svc, pool, context.Background()
}

// A PNG header keeps inspect() happy without being an executable.
func payload(size int) []byte {
	b := pattern(size)
	copy(b, []byte("\x89PNG\r\n\x1a\n"))
	return b
}

func TestUploadsStopAtTheAccountQuota(t *testing.T) {
	svc, pool, ctx := quotaSetup(t)
	basic := testdb.SeedUser(t, pool, "basic", 0)

	if _, err := svc.Upload(ctx, "a.png", payload(40<<10), basic, 0, attachments.Meta{}); err != nil {
		t.Fatalf("first upload within quota: %v", err)
	}
	if _, err := svc.Upload(ctx, "b.png", payload(40<<10), basic, 0, attachments.Meta{}); !errors.Is(err, quota.ErrUserStorage) {
		t.Fatalf("upload past the account quota: want ErrUserStorage, got %v", err)
	}
	// A chunked session reserves its declared size up front — otherwise its
	// chunks alone (stored in the database) would bypass the quota.
	if _, err := svc.InitUpload(ctx, basic, 0, "c.bin", 40<<10, attachments.Meta{}); !errors.Is(err, quota.ErrUserStorage) {
		t.Fatalf("chunked session past the quota: want ErrUserStorage, got %v", err)
	}
}

func TestOpenSessionsCountTowardsTheQuota(t *testing.T) {
	svc, pool, ctx := quotaSetup(t)
	basic := testdb.SeedUser(t, pool, "basic", 0)
	if _, err := svc.InitUpload(ctx, basic, 0, "big.bin", 40<<10, attachments.Meta{}); err != nil {
		t.Fatalf("first session: %v", err)
	}
	if _, err := svc.InitUpload(ctx, basic, 0, "big2.bin", 40<<10, attachments.Meta{}); !errors.Is(err, quota.ErrUserStorage) {
		t.Fatalf("second session past the quota: want ErrUserStorage, got %v", err)
	}
}

func TestInvitedAccountsAndTheCEOHaveLargerOrNoQuota(t *testing.T) {
	svc, pool, ctx := quotaSetup(t)
	staff := testdb.SeedUser(t, pool, "staff", staffLevel)
	ceo := testdb.SeedUser(t, pool, "chief", 1)

	for i := 0; i < 3; i++ { // 3 × 80 KiB = 240 KiB < 256 KiB
		if _, err := svc.Upload(ctx, "s.png", payload(80<<10), staff, staffLevel, attachments.Meta{}); err != nil {
			t.Fatalf("invited upload %d within its quota: %v", i, err)
		}
	}
	if _, err := svc.Upload(ctx, "s.png", payload(80<<10), staff, staffLevel, attachments.Meta{}); !errors.Is(err, quota.ErrUserStorage) {
		t.Fatalf("invited upload past its quota: want ErrUserStorage, got %v", err)
	}
	for i := 0; i < 5; i++ { // 5 × 80 KiB, far past any quota
		if _, err := svc.Upload(ctx, "c.png", payload(80<<10), ceo, 1, attachments.Meta{}); err != nil {
			t.Fatalf("CEO upload %d: %v", i, err)
		}
	}
}

func TestBasicAccountsHaveASmallerPerFileCeiling(t *testing.T) {
	svc, pool, ctx := quotaSetup(t)
	basic := testdb.SeedUser(t, pool, "basic", 0)
	if max, _ := svc.Limits(0); max != basicPerFile {
		t.Fatalf("advertised limit for a basic account: want %d, got %d", basicPerFile, max)
	}
	if _, err := svc.Upload(ctx, "big.png", payload(basicPerFile+1), basic, 0, attachments.Meta{}); !errors.Is(err, attachments.ErrTooLarge) {
		t.Fatalf("basic upload over its per-file ceiling: want ErrTooLarge, got %v", err)
	}
}

func TestParallelUploadsCannotOvershootTheQuota(t *testing.T) {
	svc, pool, ctx := quotaSetup(t)
	basic := testdb.SeedUser(t, pool, "basic", 0)

	const n, size = 8, 20 << 10 // 8 × 20 KiB against a 64 KiB quota → at most 3 fit
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.Upload(ctx, "p.png", payload(size), basic, 0, attachments.Meta{}); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	used, err := quota.UserBytes(ctx, pool, basic)
	if err != nil {
		t.Fatal(err)
	}
	if used > basicQuota {
		t.Fatalf("%d parallel uploads succeeded, storing %d bytes past a %d-byte quota", ok, used, basicQuota)
	}
}

func TestForwardedCopiesCountAgainstTheForwarder(t *testing.T) {
	svc, pool, ctx := quotaSetup(t)
	owner := testdb.SeedUser(t, pool, "owner", staffLevel)
	forwarder := testdb.SeedUser(t, pool, "forwarder", 0)

	att, err := svc.Upload(ctx, "f.png", payload(40<<10), owner, staffLevel, attachments.Meta{})
	if err != nil {
		t.Fatal(err)
	}
	message := func(sender uuid.UUID) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `
			INSERT INTO messages (chat_type, chat_id, sender_id, text)
			VALUES ('group', gen_random_uuid(), $1, 'with a file') RETURNING id`, sender).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	msgID := message(owner)
	if err := svc.Link(ctx, pool, []uuid.UUID{att.ID}, msgID, owner); err != nil {
		t.Fatal(err)
	}

	// 40 KiB fits the forwarder's 64 KiB once — and a forward duplicates the
	// bytes, so the second must not.
	if err := svc.CheckCopyQuota(ctx, forwarder, []uuid.UUID{msgID}); err != nil {
		t.Fatalf("first forward within quota: %v", err)
	}
	if _, err := svc.CopyToMessage(ctx, msgID, message(forwarder), forwarder); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if err := svc.CheckCopyQuota(ctx, forwarder, []uuid.UUID{msgID}); !errors.Is(err, quota.ErrUserStorage) {
		t.Fatalf("forward past the quota: want ErrUserStorage, got %v", err)
	}
}
