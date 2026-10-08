//go:build integration

package e2ee_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/e2ee"
)

// A device that appears after a chat exists has to be added to the chat's
// group by a member. The owner's tablet sat outside its own chat for good:
// its one announcement went out at registration, before it had a single key
// package to be added with, and nothing ever asked again.

func announcing(h harness) *recordingPublisher {
	rec := &recordingPublisher{}
	h.svc.SetPublisher(rec)
	h.svc.SetChatsOfUser(func(_ context.Context, user uuid.UUID) ([]e2ee.ChatPeer, error) {
		switch user {
		case h.a:
			return []e2ee.ChatPeer{{ChatID: h.chat, PeerID: h.b}}, nil
		case h.b:
			return []e2ee.ChatPeer{{ChatID: h.chat, PeerID: h.a}}, nil
		}
		return nil, nil
	})
	return rec
}

func TestADeviceIsAnnouncedAgainOnceItHasKeyPackages(t *testing.T) {
	h := setup(t)
	device := registerDevice(t, h, h.a)
	rec := announcing(h)

	if err := h.svc.UploadKeyPackages(h.ctx, e2ee.Actor{UserID: h.a}, device, [][]byte{[]byte("kp")}); err != nil {
		t.Fatal(err)
	}
	if len(rec.devices) != 1 || rec.devices[0].data["deviceId"] != device {
		t.Fatalf("announcements after the upload = %+v, want one for %v", rec.devices, device)
	}
}

func TestAClaimCanTargetTheOneDeviceMissingFromAGroup(t *testing.T) {
	h := setup(t)
	member := uploadPackages(t, h, h.a, 3)
	newcomer := uploadPackages(t, h, h.a, 3)

	got, err := h.svc.ClaimKeyPackages(h.ctx, e2ee.Actor{UserID: h.b}, h.a, uuid.Nil, newcomer)
	if err != nil || len(got) != 1 || got[0].DeviceID != newcomer {
		t.Fatalf("claim for the newcomer: %v, %+v", err, got)
	}
	// The device already in the group keeps every package: adding someone
	// else must not drain it.
	if n, _ := h.svc.CountKeyPackages(h.ctx, e2ee.Actor{UserID: h.a}, member); n != 3 {
		t.Fatalf("the member's pool = %d, want 3 untouched", n)
	}
}

func TestADeviceOutsideTheGroupAsksTheChatToAddIt(t *testing.T) {
	h := setup(t)
	device := registerDevice(t, h, h.a)
	rec := announcing(h)

	if err := h.svc.RequestJoin(h.ctx, e2ee.Actor{UserID: h.a}, h.chat, device); err != nil {
		t.Fatalf("join request: %v", err)
	}
	if len(rec.devices) != 1 {
		t.Fatalf("announcements = %d, want 1", len(rec.devices))
	}
	got := rec.devices[0]
	if got.data["chatId"] != h.chat || got.data["deviceId"] != device {
		t.Fatalf("announced %+v, want chat %v device %v", got.data, h.chat, device)
	}
	if len(got.users) != 2 {
		t.Fatalf("told %d users, want the peer and the owner", len(got.users))
	}
}

func TestOnlyTheChatsMembersMayAskForTheirOwnDevices(t *testing.T) {
	h := setup(t)
	device := registerDevice(t, h, h.a)
	rec := announcing(h)
	stranger := h.seed("stranger", 0)

	if err := h.svc.RequestJoin(h.ctx, e2ee.Actor{UserID: stranger}, h.chat, device); !errors.Is(err, e2ee.ErrNotFound) {
		t.Fatalf("a stranger: want ErrNotFound, got %v", err)
	}
	if err := h.svc.RequestJoin(h.ctx, e2ee.Actor{UserID: h.b}, h.chat, device); !errors.Is(err, e2ee.ErrForbidden) {
		t.Fatalf("the peer, for someone else's device: want ErrForbidden, got %v", err)
	}
	if len(rec.devices) != 0 {
		t.Fatalf("refused requests announced %d times", len(rec.devices))
	}
}

func TestJoinRequestsAreLimited(t *testing.T) {
	h := setup(t)
	device := registerDevice(t, h, h.a)
	rec := announcing(h)
	h.svc.SetJoinLimit(func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil })

	if err := h.svc.RequestJoin(h.ctx, e2ee.Actor{UserID: h.a}, h.chat, device); !errors.Is(err, e2ee.ErrRateLimited) {
		t.Fatalf("past the limit: want ErrRateLimited, got %v", err)
	}
	if len(rec.devices) != 0 {
		t.Fatalf("a limited request announced %d times", len(rec.devices))
	}
}
