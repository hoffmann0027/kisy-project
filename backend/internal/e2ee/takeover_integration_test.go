//go:build integration

package e2ee_test

import (
	"crypto/ed25519"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/e2ee"
)

// Audit A-21, two halves.
//
// Registering a device with somebody else's device id renamed their device
// and answered 201 — the rename showed up in the victim's device list ("PWNED
// by A" in the audit's run).
//
// A Welcome's recipient map comes from the client, deviceID → userID, and the
// user ids decided who got a real-time event. Nothing tied them to the devices
// or to the chat, and the map had no size limit.

func TestSomeoneElsesDeviceCannotBeTakenOver(t *testing.T) {
	h := setup(t)
	pub, _, _ := ed25519.GenerateKey(nil)
	deviceID := uuid.New()
	if _, err := h.svc.RegisterDevice(h.ctx, e2ee.Actor{UserID: h.b}, e2ee.RegisterDeviceInput{
		DeviceID: deviceID, Name: "телефон Боба", Ed25519Pub: pub,
	}); err != nil {
		t.Fatal(err)
	}

	// Alice reuses Bob's device id — with her key, and with his.
	otherPub, _, _ := ed25519.GenerateKey(nil)
	for _, key := range [][]byte{otherPub, pub} {
		_, err := h.svc.RegisterDevice(h.ctx, e2ee.Actor{UserID: h.a}, e2ee.RegisterDeviceInput{
			DeviceID: deviceID, Name: "PWNED", Ed25519Pub: key,
		})
		if !errors.Is(err, e2ee.ErrDeviceTaken) {
			t.Fatalf("someone else's device id was accepted: %v", err)
		}
	}
	// Bob himself with another key: the id is not a free slot either.
	if _, err := h.svc.RegisterDevice(h.ctx, e2ee.Actor{UserID: h.b}, e2ee.RegisterDeviceInput{
		DeviceID: deviceID, Name: "подмена ключа", Ed25519Pub: otherPub,
	}); !errors.Is(err, e2ee.ErrDeviceTaken) {
		t.Fatalf("a device's key was swapped by re-registering it: %v", err)
	}

	devices, err := h.svc.ListDevices(h.ctx, h.b)
	if err != nil || len(devices) != 1 {
		t.Fatalf("bob's devices: %v %d", err, len(devices))
	}
	if devices[0].Name != "телефон Боба" {
		t.Fatalf("bob's device was renamed to %q", devices[0].Name)
	}

	// The owner, with the same key, may still rename it — that is what
	// re-registering is for.
	if _, err := h.svc.RegisterDevice(h.ctx, e2ee.Actor{UserID: h.b}, e2ee.RegisterDeviceInput{
		DeviceID: deviceID, Name: "новое имя", Ed25519Pub: pub,
	}); err != nil {
		t.Fatalf("the owner could not rename their own device: %v", err)
	}
}

func TestWelcomesReachOnlyDevicesInTheChat(t *testing.T) {
	h := setup(t)
	aliceDevice := registerDevice(t, h, h.a)
	bobDevice := registerDevice(t, h, h.b)
	carol := h.seed("carol", 5)
	carolDevice := registerDevice(t, h, carol)

	welcome := func(recipients map[uuid.UUID]uuid.UUID) error {
		return h.svc.PublishHandshake(h.ctx, e2ee.Actor{UserID: h.a, RoleLevel: 3}, e2ee.PublishHandshakeInput{
			ChatType: "private", ChatID: h.chat, Kind: e2ee.KindWelcome,
			SenderDevice: aliceDevice, Payload: []byte{1, 2, 3}, Recipients: recipients,
		})
	}

	if err := welcome(map[uuid.UUID]uuid.UUID{carolDevice: carol}); !errors.Is(err, e2ee.ErrValidation) {
		t.Fatalf("a welcome went to someone outside the chat: %v", err)
	}
	if err := welcome(map[uuid.UUID]uuid.UUID{bobDevice: carol}); !errors.Is(err, e2ee.ErrValidation) {
		t.Fatalf("a welcome's event was aimed at a user who does not own the device: %v", err)
	}
	if err := welcome(map[uuid.UUID]uuid.UUID{uuid.New(): h.b}); !errors.Is(err, e2ee.ErrValidation) {
		t.Fatalf("a welcome went to a device that does not exist: %v", err)
	}
	flood := map[uuid.UUID]uuid.UUID{}
	for i := 0; i <= e2ee.MaxWelcomeRecipients; i++ {
		flood[uuid.New()] = h.b
	}
	if err := welcome(flood); !errors.Is(err, e2ee.ErrValidation) {
		t.Fatalf("an unbounded recipient map was accepted: %v", err)
	}

	// The real thing still goes through.
	if err := welcome(map[uuid.UUID]uuid.UUID{bobDevice: h.b}); err != nil {
		t.Fatalf("a welcome to the other participant was refused: %v", err)
	}
}
