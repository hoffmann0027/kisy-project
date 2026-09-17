package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeSiteverify answers like Cloudflare: "good-token" passes on hostname
// kisy.onrender.com, "localhost-token" on localhost, anything else fails.
func fakeSiteverify(t *testing.T, seen *map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if seen != nil {
			*seen = map[string]string{"secret": r.Form.Get("secret"), "response": r.Form.Get("response"), "remoteip": r.Form.Get("remoteip")}
		}
		res := map[string]any{"success": false, "error-codes": []string{"invalid-input-response"}}
		switch {
		case r.Form.Get("secret") != "s3cret":
			res["error-codes"] = []string{"invalid-input-secret"}
		case r.Form.Get("response") == "good-token":
			res = map[string]any{"success": true, "hostname": "kisy.onrender.com"}
		case r.Form.Get("response") == "localhost-token":
			res = map[string]any{"success": true, "hostname": "localhost"}
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
}

func TestVerify(t *testing.T) {
	var seen map[string]string
	srv := fakeSiteverify(t, &seen)
	defer srv.Close()
	c := NewClient("s3cret", srv.URL, time.Second, []string{"kisy.onrender.com", "localhost"})
	ctx := context.Background()

	if err := c.Verify(ctx, "", "1.2.3.4"); !errors.Is(err, ErrMissing) {
		t.Fatalf("no token: %v", err)
	}
	if err := c.Verify(ctx, "forged", "1.2.3.4"); !errors.Is(err, ErrRejected) {
		t.Fatalf("invalid token: %v", err)
	}
	if err := c.Verify(ctx, "good-token", "1.2.3.4"); err != nil {
		t.Fatalf("valid token: %v", err)
	}
	if seen["secret"] != "s3cret" || seen["remoteip"] != "1.2.3.4" {
		t.Fatalf("siteverify got %v", seen)
	}
	// The Android app's WebView serves the bundle from https://localhost.
	if err := c.Verify(ctx, "localhost-token", ""); err != nil {
		t.Fatalf("token issued in the app: %v", err)
	}
}

func TestVerifyRefusesATokenIssuedOnAnotherHostname(t *testing.T) {
	srv := fakeSiteverify(t, nil)
	defer srv.Close()
	c := NewClient("s3cret", srv.URL, time.Second, []string{"kisy.onrender.com"})
	if err := c.Verify(context.Background(), "localhost-token", ""); !errors.Is(err, ErrRejected) {
		t.Fatalf("foreign hostname: %v", err)
	}
}

func TestVerifyFailsClosed(t *testing.T) {
	srv := fakeSiteverify(t, nil)
	defer srv.Close()
	ctx := context.Background()

	// Our own wrong secret is a 503, not "you are a bot".
	if err := NewClient("wrong", srv.URL, time.Second, nil).Verify(ctx, "good-token", ""); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("bad secret: %v", err)
	}
	// Cloudflare unreachable.
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	down.Close()
	if err := NewClient("s3cret", down.URL, time.Second, nil).Verify(ctx, "good-token", ""); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unreachable: %v", err)
	}
	// Hanging: the timeout ends it.
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer slow.Close()
	if err := NewClient("s3cret", slow.URL, 50*time.Millisecond, nil).Verify(ctx, "good-token", ""); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("hung: %v", err)
	}
	if err := (Unconfigured{}).Verify(ctx, "good-token", ""); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unconfigured: %v", err)
	}
}

func TestSelect(t *testing.T) {
	ctx := context.Background()
	if v, key := Select(false, "k", "s", nil); key != "" || v.Verify(ctx, "", "") != nil {
		t.Fatalf("disabled: key %q", key)
	}
	// On, but a key is missing: sign-up closes instead of opening.
	for _, keys := range [][2]string{{"", ""}, {"k", ""}, {"", "s"}} {
		v, key := Select(true, keys[0], keys[1], nil)
		if key != "" || !errors.Is(v.Verify(ctx, "token", ""), ErrUnavailable) {
			t.Fatalf("keys %v: key %q", keys, key)
		}
	}
	if v, key := Select(true, "k", "s", nil); key != "k" {
		t.Fatalf("configured: key %q", key)
	} else if _, ok := v.(*Client); !ok {
		t.Fatalf("configured: %T", v)
	}
}
