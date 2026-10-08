//go:build integration

package announcements_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/internal/announcements"
)

// The HTTP surface: what the screen sends is decoded as written, and every
// refusal comes back with the status and code the client acts on.
func TestHTTPSurface(t *testing.T) {
	e := setup(t)
	caller := e.director
	r := chi.NewRouter()
	announcements.NewHandler(e.svc, func(*http.Request) (announcements.ActorMeta, bool) {
		return actor(caller), true
	}).Routes(r)

	do := func(method, path, body string) (int, map[string]any) {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		var env map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &env)
		return rec.Code, env
	}
	code := func(env map[string]any) string {
		e, _ := env["error"].(map[string]any)
		s, _ := e["code"].(string)
		return s
	}

	status, env := do(http.MethodPost, "/", `{"audience":"levels","levels":[4,5],"title":"Отчёты","body":"До пятницы"}`)
	if status != http.StatusCreated {
		t.Fatalf("send: %d %v", status, env)
	}
	a := env["data"].(map[string]any)["announcement"].(map[string]any)
	if a["recipientCount"].(float64) != 2 || a["audience"] != "levels" || a["title"] != "Отчёты" {
		t.Fatalf("announcement: %v", a)
	}
	id := a["id"].(string)

	if status, env = do(http.MethodPost, "/", `{"audience":"levels","levels":[2],"title":"t","body":"b"}`); status != http.StatusForbidden || code(env) != "ACCESS_DENIED" {
		t.Fatalf("above: %d %v", status, env)
	}
	if status, env = do(http.MethodPost, "/", `{"audience":"user","userId":"`+e.ceo.String()+`","title":"t","body":"b"}`); status != http.StatusForbidden {
		t.Fatalf("to the CEO: %d %v", status, env)
	}
	if status, env = do(http.MethodPost, "/", `{"audience":"user","userId":"nope","title":"t","body":"b"}`); status != http.StatusBadRequest {
		t.Fatalf("bad id: %d %v", status, env)
	}
	if status, env = do(http.MethodPost, "/", `{"audience":"all","title":"","body":"b"}`); status != http.StatusBadRequest || code(env) != "VALIDATION_FAILED" {
		t.Fatalf("blank title: %d %v", status, env)
	}

	for i := 1; i < announcements.BroadcastsPerDay; i++ {
		if status, env = do(http.MethodPost, "/", `{"audience":"basic","title":"t","body":"b"}`); status != http.StatusCreated {
			t.Fatalf("broadcast %d: %d %v", i, status, env)
		}
	}
	if status, env = do(http.MethodPost, "/", `{"audience":"basic","title":"t","body":"b"}`); status != http.StatusTooManyRequests || code(env) != "QUOTA_EXCEEDED" {
		t.Fatalf("over the limit: %d %v", status, env)
	}

	if status, env = do(http.MethodGet, "/", ""); status != http.StatusOK || len(env["data"].(map[string]any)["announcements"].([]any)) != announcements.BroadcastsPerDay {
		t.Fatalf("list: %d %v", status, env)
	}

	caller = e.manager
	if status, env = do(http.MethodDelete, "/"+id, ""); status != http.StatusNotFound {
		t.Fatalf("revoke by a recipient: %d %v", status, env)
	}
	caller = e.director
	if status, env = do(http.MethodDelete, "/"+id, ""); status != http.StatusOK {
		t.Fatalf("revoke: %d %v", status, env)
	}
	if status, _ = do(http.MethodDelete, "/"+uuid.NewString(), ""); status != http.StatusNotFound {
		t.Fatalf("revoke unknown: %d", status)
	}
}
