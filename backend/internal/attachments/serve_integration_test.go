//go:build integration

package attachments_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"kisy-backend/internal/attachments"
)

// Audit A-41: an uploaded x.html came back as text/html; charset=utf-8 from
// the API origin — one lost header away from stored XSS.
func TestAnUploadedPageIsServedAsAnOpaqueDownload(t *testing.T) {
	svc, uploader, ctx := setup(t)
	page := []byte("<!doctype html><html><body><script>alert(document.cookie)</script></body></html>")
	session, err := svc.InitUpload(ctx, uploader, staffLevel, "x.html", int64(len(page)), attachments.Meta{})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.PutChunk(ctx, uploader, session.ID, 0, page); err != nil {
		t.Fatal(err)
	}
	dto, err := svc.CompleteUpload(ctx, uploader, session.ID)
	if err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	attachments.NewHandler(svc, func(*http.Request) (attachments.Actor, bool) {
		return attachments.Actor{UserID: uploader, RoleLevel: staffLevel}, true
	}).Routes(router)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/attachments/"+dto.ID.String(), nil))

	if rec.Code != http.StatusOK || rec.Body.String() != string(page) {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body.String())
	}
	h := rec.Header()
	if got := h.Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("an HTML upload was served as %q", got)
	}
	if !strings.HasPrefix(h.Get("Content-Disposition"), "attachment;") {
		t.Fatalf("disposition %q", h.Get("Content-Disposition"))
	}
	if !strings.HasPrefix(h.Get("Content-Security-Policy"), "sandbox;") || h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("not sandboxed: CSP %q, nosniff %q", h.Get("Content-Security-Policy"), h.Get("X-Content-Type-Options"))
	}
}
