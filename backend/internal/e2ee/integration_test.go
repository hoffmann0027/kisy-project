//go:build integration

package e2ee_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/e2ee"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/testdb"
)

type harness struct {
	svc  *e2ee.Service
	msgs *messages.Service
	chat uuid.UUID
	a, b uuid.UUID
	ctx  context.Context
}

func setup(t *testing.T) harness {
	pool := testdb.New(t)
	ctx := context.Background()

	a := testdb.SeedUser(t, pool, "alice", 3)
	b := testdb.SeedUser(t, pool, "bob", 8)
	levels := map[uuid.UUID]int{a: 3, b: 8}
	lookup := func(_ context.Context, id uuid.UUID) (int, bool) {
		lvl, ok := levels[id]
		return lvl, ok
	}

	chatsSvc := chats.NewService(pool, chats.NewPostgresRepository(), lookup)
	chat, err := chatsSvc.OpenPrivateChat(ctx, b, chats.ActorMeta{UserID: a, RoleLevel: 3})
	if err != nil {
		t.Fatalf("open chat: %v", err)
	}

	privateAuthz := func(ctx context.Context, chatID, actorID uuid.UUID) (bool, error) {
		return chatsSvc.IsParticipant(ctx, chatID, actorID)
	}

	svc := e2ee.NewService(pool, e2ee.NewPostgresRepository(), e2ee.Authorizer{
		Private: func(ctx context.Context, chatID, actorID uuid.UUID) error {
			ok, err := privateAuthz(ctx, chatID, actorID)
			if err != nil {
				return err
			}
			if !ok {
				return e2ee.ErrNotFound
			}
			return nil
		},
		Group: func(context.Context, uuid.UUID, uuid.UUID, int) error { return e2ee.ErrNotFound },
	})

	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	msgs := messages.NewService(pool, messages.NewPostgresRepository(), rec, messages.Authorizer{
		Private: func(ctx context.Context, chatID, actorID uuid.UUID) error {
			ok, err := privateAuthz(ctx, chatID, actorID)
			if err != nil {
				return err
			}
			if !ok {
				return messages.ErrNotFound
			}
			return nil
		},
		Group: func(context.Context, uuid.UUID, uuid.UUID, int) error { return messages.ErrNotFound },
	})

	return harness{svc: svc, msgs: msgs, chat: chat.ID, a: a, b: b, ctx: ctx}
}

func registerDevice(t *testing.T, h harness, user uuid.UUID) uuid.UUID {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	d, err := h.svc.RegisterDevice(h.ctx, e2ee.Actor{UserID: user}, e2ee.RegisterDeviceInput{
		DeviceID: uuid.New(), Name: "test-device", Ed25519Pub: pub,
	})
	if err != nil {
		t.Fatalf("register device: %v", err)
	}
	return d.ID
}

func TestKeyPackageLifecycle(t *testing.T) {
	h := setup(t)
	alice := e2ee.Actor{UserID: h.a}
	deviceA := registerDevice(t, h, h.a)

	packages := [][]byte{[]byte("kp-1"), []byte("kp-2"), []byte("kp-3")}
	if err := h.svc.UploadKeyPackages(h.ctx, alice, deviceA, packages); err != nil {
		t.Fatalf("upload: %v", err)
	}
	n, err := h.svc.CountKeyPackages(h.ctx, alice, deviceA)
	if err != nil || n != 3 {
		t.Fatalf("count = %d (%v), want 3", n, err)
	}

	// Bob claims one package per alice device; a second claim gets the next one.
	claimed, err := h.svc.ClaimKeyPackages(h.ctx, h.a, uuid.Nil)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim: %v, %d packages", err, len(claimed))
	}
	if !bytes.Equal(claimed[0].KeyPackage, []byte("kp-1")) {
		t.Fatalf("expected oldest package first, got %q", claimed[0].KeyPackage)
	}
	if n, _ = h.svc.CountKeyPackages(h.ctx, alice, deviceA); n != 2 {
		t.Fatalf("after claim count = %d, want 2", n)
	}

	// Uploading to someone else's device is forbidden.
	bob := e2ee.Actor{UserID: h.b}
	if err := h.svc.UploadKeyPackages(h.ctx, bob, deviceA, packages); err != e2ee.ErrForbidden {
		t.Fatalf("cross-user upload: want ErrForbidden, got %v", err)
	}
}

func TestHandshakeMailbox(t *testing.T) {
	h := setup(t)
	alice := e2ee.Actor{UserID: h.a}
	bob := e2ee.Actor{UserID: h.b}
	deviceA := registerDevice(t, h, h.a)
	deviceB := registerDevice(t, h, h.b)

	// Alice publishes a welcome for bob's device and a commit for the chat.
	err := h.svc.PublishHandshake(h.ctx, alice, e2ee.PublishHandshakeInput{
		ChatType: "private", ChatID: h.chat, Kind: e2ee.KindWelcome,
		SenderDevice: deviceA, Payload: []byte("welcome-bytes"),
		Recipients: map[uuid.UUID]uuid.UUID{deviceB: h.b},
	})
	if err != nil {
		t.Fatalf("publish welcome: %v", err)
	}
	epoch := int64(1)
	err = h.svc.PublishHandshake(h.ctx, alice, e2ee.PublishHandshakeInput{
		ChatType: "private", ChatID: h.chat, Kind: e2ee.KindCommit,
		SenderDevice: deviceA, Payload: []byte("commit-bytes"), Epoch: &epoch,
	})
	if err != nil {
		t.Fatalf("publish commit: %v", err)
	}

	// Bob's device sees the welcome; after ack it is gone.
	welcomes, err := h.svc.ListWelcomes(h.ctx, bob, deviceB)
	if err != nil || len(welcomes) != 1 {
		t.Fatalf("welcomes: %v, %d items", err, len(welcomes))
	}
	if !bytes.Equal(welcomes[0].Payload, []byte("welcome-bytes")) {
		t.Fatalf("welcome payload mismatch")
	}
	if err := h.svc.AckWelcome(h.ctx, bob, deviceB, welcomes[0].ID); err != nil {
		t.Fatalf("ack: %v", err)
	}
	if welcomes, _ = h.svc.ListWelcomes(h.ctx, bob, deviceB); len(welcomes) != 0 {
		t.Fatalf("welcome not consumed")
	}

	// Both members see the commit; welcomes are not in the chat feed.
	frames, err := h.svc.ListChatHandshake(h.ctx, bob, "private", h.chat, uuid.Nil, 0)
	if err != nil || len(frames) != 1 || frames[0].Kind != e2ee.KindCommit {
		t.Fatalf("chat handshake: %v, %+v", err, frames)
	}

	// A stranger cannot read the chat's handshake feed.
	stranger := e2ee.Actor{UserID: uuid.New()}
	if _, err := h.svc.ListChatHandshake(h.ctx, stranger, "private", h.chat, uuid.Nil, 0); err == nil {
		t.Fatalf("stranger read handshake: want error, got nil")
	}
}

func TestEncryptedMessageRoundtrip(t *testing.T) {
	h := setup(t)

	alg := int16(1)
	epoch := int64(3)
	kind := int16(1)
	ciphertext := []byte{0x01, 0x02, 0xfe, 0xff, 0x00, 0x42}

	dto, err := h.msgs.Send(h.ctx, messages.SendInput{
		ChatType: "private", ChatID: h.chat,
		Ciphertext: ciphertext, Alg: &alg, Epoch: &epoch, ContentKind: &kind,
	}, messages.ActorMeta{UserID: h.a, RoleLevel: 3})
	if err != nil {
		t.Fatalf("send ciphertext: %v", err)
	}
	if dto.Text != nil {
		t.Fatalf("encrypted message must have nil text")
	}

	// Both text and ciphertext is rejected.
	if _, err := h.msgs.Send(h.ctx, messages.SendInput{
		ChatType: "private", ChatID: h.chat, Text: "plain",
		Ciphertext: ciphertext, Alg: &alg,
	}, messages.ActorMeta{UserID: h.a, RoleLevel: 3}); err != messages.ErrEmptyContent {
		t.Fatalf("text+ciphertext: want ErrEmptyContent, got %v", err)
	}
	// Ciphertext without alg is rejected.
	if _, err := h.msgs.Send(h.ctx, messages.SendInput{
		ChatType: "private", ChatID: h.chat, Ciphertext: ciphertext,
	}, messages.ActorMeta{UserID: h.a, RoleLevel: 3}); err != messages.ErrEmptyContent {
		t.Fatalf("ciphertext without alg: want ErrEmptyContent, got %v", err)
	}

	// The list path returns the ciphertext intact.
	page, err := h.msgs.List(h.ctx, "private", h.chat, "", 50, messages.ActorMeta{UserID: h.b, RoleLevel: 8})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := page.Items[0]
	if !bytes.Equal(got.Ciphertext, ciphertext) || got.Alg == nil || *got.Alg != 1 || *got.Epoch != 3 {
		t.Fatalf("ciphertext roundtrip mismatch: %+v", got)
	}

	// JSON wire shape: ciphertext travels base64, text stays null.
	raw, _ := json.Marshal(got)
	var wire map[string]any
	_ = json.Unmarshal(raw, &wire)
	if wire["text"] != nil {
		t.Fatalf("wire text must be null")
	}
	if _, ok := wire["ciphertext"].(string); !ok {
		t.Fatalf("wire ciphertext must be a base64 string")
	}
}
