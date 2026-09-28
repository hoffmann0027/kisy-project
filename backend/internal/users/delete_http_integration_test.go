//go:build integration

package users_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/internal/users"
)

// DELETE /users/me over HTTP: the password is asked for again, the word is
// typed, and the browser's session ends with the account.

type deleteAPI struct {
	srv          *httptest.Server
	fixture      deletionFixture
	passwordUsed string
}

func newDeleteAPI(t *testing.T, correctPassword string) *deleteAPI {
	t.Helper()
	f := newDeletionFixture(t)
	api := &deleteAPI{fixture: f}
	h := users.NewHandler(f.svc, nil,
		func(*http.Request) (users.Identity, bool) { return users.Identity{UserID: f.me}, true },
		func(*http.Request) users.ActorMeta { return users.ActorMeta{} })
	h.SetAccountDeletion(
		func(_ context.Context, _ uuid.UUID, plaintext string) (bool, error) {
			api.passwordUsed = plaintext
			return plaintext == correctPassword, nil
		},
		func(w http.ResponseWriter) {
			http.SetCookie(w, &http.Cookie{Name: "kisy_access", Value: "", Path: "/", MaxAge: -1})
		},
	)
	r := chi.NewRouter()
	r.Route("/users", h.Routes)
	api.srv = httptest.NewServer(r)
	t.Cleanup(api.srv.Close)
	return api
}

func (a *deleteAPI) del(t *testing.T, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, a.srv.URL+"/users/me", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func (a *deleteAPI) stillThere(t *testing.T) bool {
	t.Helper()
	var deleted *string
	if err := a.fixture.pool.QueryRow(a.fixture.ctx,
		`SELECT deleted_at::text FROM users WHERE id = $1`, a.fixture.me).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	return deleted == nil
}

func TestDeleteMeNeedsThePasswordAndTheWord(t *testing.T) {
	api := newDeleteAPI(t, "right-password-1")

	if res := api.del(t, `{"password":"right-password-1"}`); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("without the confirmation word: %d, want 400", res.StatusCode)
	}
	if res := api.del(t, `{"password":"wrong","confirm":"УДАЛИТЬ"}`); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("with a wrong password: %d, want 401", res.StatusCode)
	}
	if !api.stillThere(t) {
		t.Fatal("a refused request deleted the account anyway")
	}

	res := api.del(t, `{"password":"right-password-1","confirm":"УДАЛИТЬ"}`)
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("with both: %d (%s)", res.StatusCode, body)
	}
	var envelope struct {
		Data struct {
			Deleted bool `json:"deleted"`
		} `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&envelope)
	if !envelope.Data.Deleted {
		t.Fatal("the response does not say the account is gone")
	}
	if api.stillThere(t) {
		t.Fatal("the account survived its own deletion")
	}
	// The session this was asked from ends with it.
	var expired bool
	for _, c := range res.Cookies() {
		if c.Name == "kisy_access" && c.MaxAge < 0 {
			expired = true
		}
	}
	if !expired {
		t.Fatal("the browser kept its session cookie after the account was deleted")
	}
}

// Nothing wired to re-check the password: refuse rather than delete on the
// strength of a session cookie alone.
func TestDeleteMeRefusesWhenItCannotReAuthenticate(t *testing.T) {
	f := newDeletionFixture(t)
	h := users.NewHandler(f.svc, nil,
		func(*http.Request) (users.Identity, bool) { return users.Identity{UserID: f.me}, true },
		func(*http.Request) users.ActorMeta { return users.ActorMeta{} })
	r := chi.NewRouter()
	r.Route("/users", h.Routes)
	srv := httptest.NewServer(r)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/users/me", strings.NewReader(`{"password":"x","confirm":"УДАЛИТЬ"}`))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		t.Fatal("an unconfigured handler deleted the account")
	}
}
