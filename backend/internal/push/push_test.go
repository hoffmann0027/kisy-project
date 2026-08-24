package push

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// fakeRepo keeps devices in memory. The pool argument is unused: these tests
// exercise the service's fan-out and pruning, not SQL.
type fakeRepo struct {
	devices []Device
	deleted []string
}

func (f *fakeRepo) Upsert(context.Context, *pgxpool.Pool, uuid.UUID, Subscription) error { return nil }
func (f *fakeRepo) Delete(context.Context, *pgxpool.Pool, string) error                  { return nil }
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
	svc.Notify(context.Background(), uuid.New(), "t", "b", "/chats/1")
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

	svc.Notify(context.Background(), uuid.New(), "Иван", "Привет", "/chats/7")

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

	svc.Notify(context.Background(), uuid.New(), "t", "b", "")

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

	svc.Notify(context.Background(), uuid.New(), "t", "b", "")

	if len(repo.deleted) != 0 {
		t.Fatalf("an outage cost the user a registration: %v", repo.deleted)
	}
}
