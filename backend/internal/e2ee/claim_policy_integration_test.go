//go:build integration

package e2ee_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/e2ee"
)

const claimLimitPerPair = 3

// Audit A-09: any account could claim — and so exhaust — any other user's
// one-time key packages, with no shared chat and no limit. An exhausted pool
// is how a private chat was pushed into plaintext; with sending now fail-closed
// it would be a denial of service instead. Claims now require a private chat
// with the target and are rate-limited per pair.

func uploadPackages(t *testing.T, h harness, owner uuid.UUID, n int) uuid.UUID {
	t.Helper()
	device := registerDevice(t, h, owner)
	pkgs := make([][]byte, n)
	for i := range pkgs {
		pkgs[i] = []byte{byte(i + 1)}
	}
	if err := h.svc.UploadKeyPackages(h.ctx, e2ee.Actor{UserID: owner}, device, pkgs); err != nil {
		t.Fatal(err)
	}
	return device
}

func TestKeyPackagesAreClaimableOnlyByChatPartners(t *testing.T) {
	h := setup(t)
	deviceA := uploadPackages(t, h, h.a, 5)
	stranger := h.seed("stranger", 0)

	if _, err := h.svc.ClaimKeyPackages(h.ctx, e2ee.Actor{UserID: stranger}, h.a, uuid.Nil); !errors.Is(err, e2ee.ErrNotFound) {
		t.Fatalf("claim by someone with no chat: want ErrNotFound, got %v", err)
	}
	if n, _ := h.svc.CountKeyPackages(h.ctx, e2ee.Actor{UserID: h.a}, deviceA); n != 5 {
		t.Fatalf("a refused claim consumed packages: %d left, want 5", n)
	}

	// Bob shares the private chat with alice.
	if got, err := h.svc.ClaimKeyPackages(h.ctx, e2ee.Actor{UserID: h.b}, h.a, uuid.Nil); err != nil || len(got) != 1 {
		t.Fatalf("claim by the chat partner: %v, %d packages", err, len(got))
	}
}

func TestKeyPackageClaimsAreRateLimitedPerPair(t *testing.T) {
	h := setup(t)
	uploadPackages(t, h, h.a, 10)
	for i := 0; i < claimLimitPerPair; i++ {
		if _, err := h.svc.ClaimKeyPackages(h.ctx, e2ee.Actor{UserID: h.b}, h.a, uuid.Nil); err != nil {
			t.Fatalf("claim %d within the limit: %v", i, err)
		}
	}
	if _, err := h.svc.ClaimKeyPackages(h.ctx, e2ee.Actor{UserID: h.b}, h.a, uuid.Nil); !errors.Is(err, e2ee.ErrRateLimited) {
		t.Fatalf("claim past the per-pair limit: want ErrRateLimited, got %v", err)
	}
}

func TestOwnOtherDevicesAreAlwaysClaimable(t *testing.T) {
	h := setup(t)
	uploadPackages(t, h, h.a, 2)
	ownNew := registerDevice(t, h, h.a)
	// A user adds their own other devices to a chat; no chat "with yourself" is needed.
	if got, err := h.svc.ClaimKeyPackages(h.ctx, e2ee.Actor{UserID: h.a}, h.a, ownNew); err != nil || len(got) != 1 {
		t.Fatalf("claiming own other devices: %v, %d packages", err, len(got))
	}
}
