//go:build integration

package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"kisy-backend/internal/auth"
	"kisy-backend/internal/platform/turnstile"
)

// Sign-up requires a Cloudflare Turnstile token checked server-side: a script
// that posts the form directly, skipping the widget, gets 403 and no account.
// siteverify is replaced by a local server that answers like Cloudflare.

func captchaServer(t *testing.T, verifier turnstile.Verifier) (*env, *httptest.Server) {
	t.Helper()
	e := setup(t)
	h := auth.NewHandler(e.svc, auth.NewMiddleware(e.tokens, e.sessions, e.pool), "salt", false)
	h.SetCaptcha(verifier, "site-key-123")
	r := chi.NewRouter()
	r.Route("/api/v1/auth", h.Routes)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return e, srv
}

func fakeCloudflare(t *testing.T) *httptest.Server {
	t.Helper()
	cf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		ok := r.Form.Get("secret") == "cf-secret" && r.Form.Get("response") == "human-token"
		_ = json.NewEncoder(w).Encode(map[string]any{"success": ok, "hostname": "localhost"})
	}))
	t.Cleanup(cf.Close)
	return cf
}

func registerBody(token string) string {
	body := map[string]any{"username": "captcha_user", "displayName": "Проверка Капчи", "password": "captcha-pass-12"}
	if token != "" {
		body["turnstileToken"] = token
	}
	raw, _ := json.Marshal(body)
	return string(raw)
}

func accountExists(t *testing.T, e *env) bool {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(t.Context(), `SELECT count(*) FROM users WHERE username = 'captcha_user'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func TestRegistrationWithoutAValidTurnstileTokenIsRefused(t *testing.T) {
	cf := fakeCloudflare(t)
	e, srv := captchaServer(t, turnstile.NewClient("cf-secret", cf.URL, time.Second, nil))

	if got := post(t, srv, "/register", "", false, registerBody(""), nil); got.status != http.StatusForbidden {
		t.Fatalf("no token: %d, want 403", got.status)
	}
	if got := post(t, srv, "/register", "", false, registerBody("forged"), nil); got.status != http.StatusForbidden {
		t.Fatalf("invalid token: %d, want 403", got.status)
	}
	if accountExists(t, e) {
		t.Fatal("a refused sign-up must not leave an account behind")
	}
	if got := post(t, srv, "/register", "", false, registerBody("human-token"), nil); got.status != http.StatusCreated {
		t.Fatalf("valid token: %d, want 201", got.status)
	}
	if !accountExists(t, e) {
		t.Fatal("the verified sign-up must create the account")
	}
}

func TestRegistrationFailsClosedWhenTurnstileCannotBeChecked(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	down.Close()
	e, srv := captchaServer(t, turnstile.NewClient("cf-secret", down.URL, time.Second, nil))
	if got := post(t, srv, "/register", "", false, registerBody("human-token"), nil); got.status != http.StatusServiceUnavailable {
		t.Fatalf("siteverify unreachable: %d, want 503", got.status)
	}
	if accountExists(t, e) {
		t.Fatal("an unverifiable sign-up must not create an account")
	}
}

// A handler nobody configured must refuse, not open registration to scripts.
func TestRegistrationIsClosedByDefault(t *testing.T) {
	e := setup(t)
	h := auth.NewHandler(e.svc, auth.NewMiddleware(e.tokens, e.sessions, e.pool), "salt", false)
	r := chi.NewRouter()
	r.Route("/api/v1/auth", h.Routes)
	srv := httptest.NewServer(r)
	defer srv.Close()
	if got := post(t, srv, "/register", "", false, registerBody("anything"), nil); got.status == http.StatusCreated {
		t.Fatal("an unconfigured captcha must not let a sign-up through")
	}
}

func TestRegistrationPolicyTellsTheScreenWhichSiteKeyToUse(t *testing.T) {
	cf := fakeCloudflare(t)
	_, srv := captchaServer(t, turnstile.NewClient("cf-secret", cf.URL, time.Second, nil))
	res, err := http.Get(srv.URL + "/api/v1/auth/registration")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body struct {
		Data struct {
			TurnstileSiteKey string `json:"turnstileSiteKey"`
		} `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&body)
	if body.Data.TurnstileSiteKey != "site-key-123" {
		t.Fatalf("site key = %q", body.Data.TurnstileSiteKey)
	}
}
