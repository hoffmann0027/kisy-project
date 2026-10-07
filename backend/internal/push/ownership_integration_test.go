//go:build integration

package push_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/push"
)

// Audit A-20: unsubscribing deleted any row with the given endpoint or token,
// so anyone who knew someone's push endpoint or FCM token could silently
// switch off their notifications.
func TestYouCanOnlySwitchOffYourOwnNotifications(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	alice := testdb.SeedUser(t, pool, "alice_push", 5)
	mallory := testdb.SeedUser(t, pool, "mallory_push", 5)
	svc := push.NewService(pool, push.NewPostgresRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)), "pub", "priv", "mailto:x@example.com")

	const endpoint = "https://fcm.googleapis.com/fcm/send/alice-browser"
	if err := svc.Subscribe(ctx, alice, push.Subscription{Endpoint: endpoint, P256dh: "k", Auth: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.RegisterDevice(ctx, alice, push.Device{Token: "alice-phone-token", Platform: "android"}); err != nil {
		t.Fatal(err)
	}

	count := func(query string, arg string) int {
		var n int
		if err := pool.QueryRow(ctx, query, arg).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	subs := func() int { return count(`SELECT count(*) FROM push_subscriptions WHERE endpoint = $1`, endpoint) }
	devices := func() int { return count(`SELECT count(*) FROM device_tokens WHERE token = $1`, "alice-phone-token") }

	// Mallory knows both and tries to switch them off.
	if err := svc.Unsubscribe(ctx, mallory, endpoint); err != nil {
		t.Fatal(err)
	}
	if err := svc.UnregisterDevice(ctx, mallory, "alice-phone-token"); err != nil {
		t.Fatal(err)
	}
	if subs() != 1 || devices() != 1 {
		t.Fatalf("someone else switched off alice's notifications: subscriptions=%d devices=%d", subs(), devices())
	}

	// Alice can.
	if err := svc.Unsubscribe(ctx, alice, endpoint); err != nil {
		t.Fatal(err)
	}
	if err := svc.UnregisterDevice(ctx, alice, "alice-phone-token"); err != nil {
		t.Fatal(err)
	}
	if subs() != 0 || devices() != 0 {
		t.Fatalf("alice could not switch off her own: subscriptions=%d devices=%d", subs(), devices())
	}
}
