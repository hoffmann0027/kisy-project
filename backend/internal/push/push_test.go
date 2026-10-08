package push

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/i18n"
)

// fakeRepo keeps devices in memory. The pool argument is unused: these tests
// exercise the service's fan-out and pruning, not SQL.
type fakeRepo struct {
	devices []Device
	deleted []string
}

func (f *fakeRepo) Upsert(context.Context, *pgxpool.Pool, uuid.UUID, Subscription) error  { return nil }
func (f *fakeRepo) Delete(context.Context, *pgxpool.Pool, string) error                   { return nil }
func (f *fakeRepo) DeleteForUser(context.Context, *pgxpool.Pool, uuid.UUID, string) error { return nil }
func (f *fakeRepo) DeleteDeviceForUser(context.Context, *pgxpool.Pool, uuid.UUID, string) error {
	return nil
}
func (f *fakeRepo) ListForUser(context.Context, *pgxpool.Pool, uuid.UUID) ([]Subscription, error) {
	return nil, nil
}
func (f *fakeRepo) UpsertDevice(_ context.Context, _ *pgxpool.Pool, _ uuid.UUID, d Device) error {
	f.devices = append(f.devices, d)
	return nil
}
func (f *fakeRepo) DeleteDevice(_ context.Context, _ *pgxpool.Pool, token string) error {
	f.deleted = append(f.deleted, token)
	for i, d := range f.devices {
		if d.Token == token {
			f.devices = append(f.devices[:i], f.devices[i+1:]...)
			break
		}
	}
	return nil
}
func (f *fakeRepo) ListDevicesForUser(context.Context, *pgxpool.Pool, uuid.UUID) ([]Device, error) {
	return f.devices, nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNotifyWithoutTransportsDoesNothing(t *testing.T) {
	repo := &fakeRepo{devices: []Device{{Token: "d1", Platform: "android"}}}
	// No VAPID keys and no FCM: the service must stay inert rather than panic
	// on a nil sender.
	svc := NewService(nil, repo, quietLogger(), "", "", "")
	if svc.Enabled() || svc.MobileEnabled() {
		t.Fatal("service reports a transport it does not have")
	}
	svc.Notify(context.Background(), uuid.New(), i18n.Raw("t"), i18n.Raw("b"), "/chats/1")
	if len(repo.deleted) != 0 {
		t.Fatalf("deleted %v with no transport configured", repo.deleted)
	}
}

func TestNotifyDeliversToEveryDeviceWithoutWebPush(t *testing.T) {
	srv := newFCMServer(t)
	repo := &fakeRepo{devices: []Device{
		{Token: "phone-1", Platform: "android"},
		{Token: "phone-2", Platform: "android"},
	}}
	// VAPID intentionally empty: mobile push must work on its own.
	svc := NewService(nil, repo, quietLogger(), "", "", "")
	svc.SetFCM(newTestFCM(t, srv))

	svc.Notify(context.Background(), uuid.New(), i18n.Raw("Иван"), i18n.Raw("Привет"), "/chats/7")

	if got := srv.sendCalls.Load(); got != 2 {
		t.Fatalf("sends = %d, want 2", got)
	}
	if len(repo.deleted) != 0 {
		t.Fatalf("healthy devices pruned: %v", repo.deleted)
	}
}

func TestNotifyPrunesUnregisteredDevices(t *testing.T) {
	srv := newFCMServer(t)
	srv.sendStatus.Store(int32(http.StatusNotFound))
	srv.sendBody.Store(`{"error":{"status":"NOT_FOUND","details":[{"errorCode":"UNREGISTERED"}]}}`)

	repo := &fakeRepo{devices: []Device{{Token: "stale", Platform: "android"}}}
	svc := NewService(nil, repo, quietLogger(), "", "", "")
	svc.SetFCM(newTestFCM(t, srv))

	svc.Notify(context.Background(), uuid.New(), i18n.Raw("t"), i18n.Raw("b"), "")

	if len(repo.deleted) != 1 || repo.deleted[0] != "stale" {
		t.Fatalf("deleted = %v, want [stale]", repo.deleted)
	}
	if len(repo.devices) != 0 {
		t.Fatalf("devices = %v, want empty", repo.devices)
	}
}

func TestNotifyKeepsDevicesOnTransientFailure(t *testing.T) {
	srv := newFCMServer(t)
	srv.sendStatus.Store(int32(http.StatusServiceUnavailable))
	srv.sendBody.Store(`{"error":{"status":"UNAVAILABLE"}}`)

	repo := &fakeRepo{devices: []Device{{Token: "phone-1", Platform: "android"}}}
	svc := NewService(nil, repo, quietLogger(), "", "", "")
	svc.SetFCM(newTestFCM(t, srv))

	svc.Notify(context.Background(), uuid.New(), i18n.Raw("t"), i18n.Raw("b"), "")

	if len(repo.deleted) != 0 {
		t.Fatalf("an outage cost the user a registration: %v", repo.deleted)
	}
}

// A push is written in its recipient's language, not the sender's: whoever
// mentions you in Russian, your English phone says "New message".
func TestNotifySpeaksTheRecipientsLanguage(t *testing.T) {
	srv := newFCMServer(t)
	repo := &fakeRepo{devices: []Device{{Token: "phone", Platform: "android"}}}
	svc := NewService(nil, repo, quietLogger(), "", "", "")
	svc.SetFCM(newTestFCM(t, srv))
	recipient := uuid.New()
	svc.SetLocaleOf(func(_ context.Context, id uuid.UUID) i18n.Lang {
		if id != recipient {
			t.Errorf("language looked up for %v, want the recipient", id)
		}
		return "en"
	})

	svc.Notify(context.Background(), recipient, i18n.Raw("KISY"), i18n.M("push.newMessage"), "/")

	body, _ := srv.lastBody.Load().(json.RawMessage)
	if !strings.Contains(string(body), "New message") {
		t.Fatalf("push body = %s, want it in English", body)
	}
}

// A push that may be taken back carries its own tag — and the retraction
// names that tag, so the phone removes exactly it and nothing that arrived
// after it.
func TestRetractTakesDownTheTaggedPush(t *testing.T) {
	srv := newFCMServer(t)
	repo := &fakeRepo{devices: []Device{{Token: "phone", Platform: "android"}}}
	svc := NewService(nil, repo, quietLogger(), "", "", "")
	svc.SetFCM(newTestFCM(t, srv))
	user := uuid.New()

	svc.NotifyTagged(context.Background(), user, "announcement-1", i18n.Raw("Собрание"), i18n.Raw("В пять"), "/")
	var shown struct {
		Message struct {
			Android struct {
				Notification struct {
					Tag string `json:"tag"`
				} `json:"notification"`
			} `json:"android"`
		} `json:"message"`
	}
	if err := json.Unmarshal(srv.lastBody.Load().(json.RawMessage), &shown); err != nil {
		t.Fatalf("decode push: %v", err)
	}
	if shown.Message.Android.Notification.Tag != "announcement-1" {
		t.Fatalf("tag = %q, want announcement-1", shown.Message.Android.Notification.Tag)
	}

	svc.Retract(context.Background(), user, "announcement-1")
	var retract struct {
		Message struct {
			Notification *json.RawMessage  `json:"notification"`
			Data         map[string]string `json:"data"`
			Android      struct {
				TTL string `json:"ttl"`
			} `json:"android"`
		} `json:"message"`
	}
	if err := json.Unmarshal(srv.lastBody.Load().(json.RawMessage), &retract); err != nil {
		t.Fatalf("decode retraction: %v", err)
	}
	if retract.Message.Notification != nil {
		t.Errorf("the retraction would itself be drawn: %s", *retract.Message.Notification)
	}
	if retract.Message.Data["type"] != RetractType || retract.Message.Data["tag"] != "announcement-1" {
		t.Errorf("data = %v, want type %s and the push's tag", retract.Message.Data, RetractType)
	}
	if retract.Message.Android.TTL != "2419200s" {
		t.Errorf("ttl = %q, want as long as the push itself is kept (2419200s)", retract.Message.Android.TTL)
	}
}

// Everything else keeps sharing one tag, so a new message still replaces the
// previous one in the shade instead of stacking.
func TestNotifyKeepsTheSharedTag(t *testing.T) {
	srv := newFCMServer(t)
	repo := &fakeRepo{devices: []Device{{Token: "phone", Platform: "android"}}}
	svc := NewService(nil, repo, quietLogger(), "", "", "")
	svc.SetFCM(newTestFCM(t, srv))

	svc.Notify(context.Background(), uuid.New(), i18n.Raw("Иван"), i18n.Raw("Привет"), "/chat/1")
	if body := string(srv.lastBody.Load().(json.RawMessage)); !strings.Contains(body, `"tag":"kisy"`) {
		t.Fatalf("push = %s, want the shared tag", body)
	}
}
