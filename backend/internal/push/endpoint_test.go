package push

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// Audit A-17: the endpoint of a browser push subscription comes from the
// client and the server POSTs to it. It was stored as given and called with a
// default http.Client — a blind SSRF into the server's own network, and a
// goroutine per push held forever by a slow host.

func TestOnlyBrowserPushServicesAreAccepted(t *testing.T) {
	accepted := []string{
		"https://fcm.googleapis.com/fcm/send/abc:def",
		"https://updates.push.services.mozilla.com/wpush/v2/gAAAAA",
		"https://wns2-db5p.notify.windows.com/w/?token=BQYAAA",
		"https://web.push.apple.com/QGuQyavXutnMH",
		"https://FCM.googleapis.com/fcm/send/x", // host case does not matter
		"https://fcm.googleapis.com:443/fcm/send/x",
	}
	for _, raw := range accepted {
		if err := ValidateEndpoint(raw); err != nil {
			t.Errorf("a real push service was refused: %s", raw)
		}
	}

	refused := []string{
		"http://fcm.googleapis.com/fcm/send/x",      // not https
		"https://169.254.169.254/latest/meta-data/", // cloud metadata
		"https://127.0.0.1:8080/api/v1/admin",       // loopback
		"https://[::1]/x",                           // loopback, v6
		"https://localhost/x",                       // not a push service
		"https://fcm.googleapis.com.evil.example/x", // suffix trick
		"https://evilnotify.windows.com/x",          // missing dot
		"https://fcm.googleapis.com:8443/x",         // another port
		"https://user:pass@fcm.googleapis.com/x",    // credentials in the URL
		"file:///etc/passwd",                        // another scheme
		"https://internal.kisy.example/push",        // anything else
		"",
	}
	for _, raw := range refused {
		if err := ValidateEndpoint(raw); !errors.Is(err, ErrBadEndpoint) {
			t.Errorf("accepted as a push endpoint: %q", raw)
		}
	}
}

func TestSubscribingRefusesAForeignEndpointBeforeStoringIt(t *testing.T) {
	// No pool: a refused endpoint must not get as far as the database.
	svc := &Service{}
	err := svc.Subscribe(context.Background(), uuid.New(), Subscription{
		Endpoint: "http://169.254.169.254/latest/meta-data/", P256dh: "k", Auth: "a",
	})
	if !errors.Is(err, ErrBadEndpoint) {
		t.Fatalf("Subscribe = %v, want ErrBadEndpoint", err)
	}
}

func TestPushClientNeverReachesThePrivateNetwork(t *testing.T) {
	reached := false
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
	}))
	defer internal.Close()

	client := newSendClient()
	if client.Timeout == 0 {
		t.Fatal("the push client has no timeout: a slow host would hold the goroutine forever")
	}
	resp, err := client.Post(internal.URL, "application/octet-stream", nil)
	if err == nil {
		resp.Body.Close()
		t.Fatal("the push client connected to a loopback address")
	}
	if reached {
		t.Fatal("the request reached the internal server")
	}

	// A push service answers; it never sends us elsewhere.
	if err := client.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirects are followed: %v", err)
	}
}
