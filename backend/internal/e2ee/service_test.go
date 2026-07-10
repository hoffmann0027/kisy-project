package e2ee

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/platform/db"
)

// fakeRepo implements the subset of Repository the service tests exercise.
type fakeRepo struct {
	Repository
	devices map[uuid.UUID]*Device
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{devices: make(map[uuid.UUID]*Device)}
}

func (f *fakeRepo) UpsertDevice(_ context.Context, _ db.DBTX, d *Device) error {
	d.CreatedAt = time.Now()
	f.devices[d.ID] = d
	return nil
}

func (f *fakeRepo) GetDevice(_ context.Context, _ db.DBTX, id uuid.UUID) (*Device, error) {
	d, ok := f.devices[id]
	if !ok {
		return nil, ErrNotFound
	}
	return d, nil
}

func newTestService(repo Repository) *Service {
	return NewService(nil, repo, Authorizer{})
}

func TestRegisterDeviceValidation(t *testing.T) {
	svc := newTestService(newFakeRepo())
	actor := Actor{UserID: uuid.New()}

	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.RegisterDevice(context.Background(), actor, RegisterDeviceInput{
		DeviceID: uuid.New(), Ed25519Pub: pub[:16],
	}); err != ErrValidation {
		t.Fatalf("short key: want ErrValidation, got %v", err)
	}

	if _, err := svc.RegisterDevice(context.Background(), actor, RegisterDeviceInput{
		DeviceID: uuid.Nil, Ed25519Pub: pub,
	}); err != ErrValidation {
		t.Fatalf("nil device id: want ErrValidation, got %v", err)
	}

	d, err := svc.RegisterDevice(context.Background(), actor, RegisterDeviceInput{
		DeviceID: uuid.New(), Name: "laptop", Ed25519Pub: pub,
	})
	if err != nil {
		t.Fatalf("valid device: %v", err)
	}
	if d.UserID != actor.UserID {
		t.Fatalf("device bound to wrong user")
	}
}

func TestRegisterDeviceVouchChain(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	actor := Actor{UserID: uuid.New()}

	oldPub, oldPriv, _ := ed25519.GenerateKey(nil)
	newPub, _, _ := ed25519.GenerateKey(nil)

	oldDevice, err := svc.RegisterDevice(context.Background(), actor, RegisterDeviceInput{
		DeviceID: uuid.New(), Ed25519Pub: oldPub,
	})
	if err != nil {
		t.Fatal(err)
	}

	// The vouch signs "KISY-device-vouch-v1" || newDevicePublicKey — the same
	// format the frontend crypto core produces (identity.ts).
	msg := append(append([]byte{}, vouchContext...), newPub...)
	goodSig := ed25519.Sign(oldPriv, msg)

	if _, err := svc.RegisterDevice(context.Background(), actor, RegisterDeviceInput{
		DeviceID: uuid.New(), Ed25519Pub: newPub, SignedBy: &oldDevice.ID, Signature: goodSig,
	}); err != nil {
		t.Fatalf("valid vouch rejected: %v", err)
	}

	// A signature over a different key must be rejected.
	otherPub, _, _ := ed25519.GenerateKey(nil)
	if _, err := svc.RegisterDevice(context.Background(), actor, RegisterDeviceInput{
		DeviceID: uuid.New(), Ed25519Pub: otherPub, SignedBy: &oldDevice.ID, Signature: goodSig,
	}); err != ErrValidation {
		t.Fatalf("forged vouch: want ErrValidation, got %v", err)
	}

	// A signer belonging to another user must be rejected.
	stranger := Actor{UserID: uuid.New()}
	if _, err := svc.RegisterDevice(context.Background(), stranger, RegisterDeviceInput{
		DeviceID: uuid.New(), Ed25519Pub: newPub, SignedBy: &oldDevice.ID, Signature: goodSig,
	}); err != ErrForbidden {
		t.Fatalf("cross-user vouch: want ErrForbidden, got %v", err)
	}
}

func TestPublishHandshakeValidation(t *testing.T) {
	svc := newTestService(newFakeRepo())
	actor := Actor{UserID: uuid.New()}

	// Oversized payload.
	err := svc.PublishHandshake(context.Background(), actor, PublishHandshakeInput{
		ChatType: "group", ChatID: uuid.New(), Kind: KindCommit,
		SenderDevice: uuid.New(), Payload: make([]byte, MaxHandshakeBytes+1),
	})
	if err != ErrValidation {
		t.Fatalf("oversized payload: want ErrValidation, got %v", err)
	}

	// Welcome without recipients.
	err = svc.PublishHandshake(context.Background(), actor, PublishHandshakeInput{
		ChatType: "group", ChatID: uuid.New(), Kind: KindWelcome,
		SenderDevice: uuid.New(), Payload: []byte{1},
	})
	if err != ErrValidation {
		t.Fatalf("welcome without recipients: want ErrValidation, got %v", err)
	}

	// Unknown kind.
	err = svc.PublishHandshake(context.Background(), actor, PublishHandshakeInput{
		ChatType: "group", ChatID: uuid.New(), Kind: 9,
		SenderDevice: uuid.New(), Payload: []byte{1},
	})
	if err != ErrValidation {
		t.Fatalf("bad kind: want ErrValidation, got %v", err)
	}
}
