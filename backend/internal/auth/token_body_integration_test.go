//go:build integration

package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"kisy-backend/internal/auth"
)

// Audit A-06: "X-Kisy-Client: native" alone made /auth/refresh answer with raw
// tokens. A script on the web app (XSS) set the header, the browser attached
// the HttpOnly refresh cookie by itself, and the response carried a readable
// 30-day refresh token. Tokens now go in a body only to an app origin, and on
// refresh only when the token itself came in the body.

const (
	appOrigin = "https://localhost"
	webOrigin = "https://kisy.example"
)

type authCall struct {
	status  int
	tokens  map[string]any
	cookies []*http.Cookie
}

func newAuthServer(t *testing.T) (*env, *httptest.Server) {
	t.Helper()
	e := setup(t)
	h := auth.NewHandler(e.svc, auth.NewMiddleware(e.tokens, e.sessions, e.pool), "salt", false)
	h.SetNativeOrigins([]string{appOrigin, "capacitor://localhost"})
	r := chi.NewRouter()
	r.Route("/api/v1/auth", h.Routes)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return e, srv
}

func post(t *testing.T, srv *httptest.Server, path, origin string, native bool, body string, cookies []*http.Cookie) authCall {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/auth"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if native {
		req.Header.Set("X-Kisy-Client", "native")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var env struct {
		Data map[string]any `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&env)
	tokens, _ := env.Data["tokens"].(map[string]any)
	return authCall{status: res.StatusCode, tokens: tokens, cookies: res.Cookies()}
}

const loginBody = `{"username":"tokenbody","password":"token-body-pass-1"}`

func TestScriptOnTheWebCannotLaunderTheRefreshCookieIntoTokens(t *testing.T) {
	e, srv := newAuthServer(t)
	e.createUser(t, "tokenbody", "token-body-pass-1", 5)

	// A browser session: cookies only.
	login := post(t, srv, "/login", webOrigin, false, loginBody, nil)
	if login.status != 200 || login.tokens != nil || len(login.cookies) == 0 {
		t.Fatalf("browser login: status %d tokens %v cookies %d", login.status, login.tokens, len(login.cookies))
	}

	// XSS: same-site Origin, the header, the cookie the browser attaches.
	stolen := post(t, srv, "/refresh", webOrigin, true, "", login.cookies)
	if stolen.status != 200 {
		t.Fatalf("refresh itself must still work for the browser: %d", stolen.status)
	}
	if stolen.tokens != nil {
		t.Fatalf("the web origin got raw tokens in the body: %v", stolen.tokens)
	}

	// Even an app origin never gets a cookie-borne token back readable.
	fromCookie := post(t, srv, "/refresh", appOrigin, true, "", stolen.cookies)
	if fromCookie.tokens != nil {
		t.Fatalf("a refresh token taken from the cookie came back in the body: %v", fromCookie.tokens)
	}

	// Login with the header from the web origin: no tokens either.
	if l := post(t, srv, "/login", webOrigin, true, loginBody, nil); l.tokens != nil {
		t.Fatalf("login from the web origin got tokens in the body: %v", l.tokens)
	}
}

func TestInstalledAppStillGetsItsTokens(t *testing.T) {
	e, srv := newAuthServer(t)
	e.createUser(t, "tokenbody", "token-body-pass-1", 5)

	login := post(t, srv, "/login", appOrigin, true, loginBody, nil)
	if login.status != 200 || login.tokens == nil {
		t.Fatalf("app login must return tokens: status %d tokens %v", login.status, login.tokens)
	}
	refresh, _ := login.tokens["refreshToken"].(string)

	rotated := post(t, srv, "/refresh", appOrigin, true, `{"refreshToken":"`+refresh+`"}`, nil)
	if rotated.status != 200 || rotated.tokens == nil {
		t.Fatalf("app refresh with the token in the body must return tokens: status %d tokens %v", rotated.status, rotated.tokens)
	}
	if next, _ := rotated.tokens["refreshToken"].(string); next == "" || next == refresh {
		t.Fatalf("refresh token not rotated: %q", next)
	}
}
