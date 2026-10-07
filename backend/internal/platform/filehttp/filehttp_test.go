package filehttp

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Audit A-41: an uploaded x.html came back as text/html; charset=utf-8.
func TestActiveContentGoesOutAsAnOpaqueDownload(t *testing.T) {
	for _, mime := range []string{"text/html; charset=utf-8", "text/xml; charset=utf-8", "image/svg+xml", "application/pdf", "text/plain; charset=utf-8", ""} {
		rec := httptest.NewRecorder()
		Write(rec, mime, "x.html", []byte("<script>alert(1)</script>"), "private")
		h := rec.Header()
		if got := h.Get("Content-Type"); got != "application/octet-stream" {
			t.Errorf("%q served as %q", mime, got)
		}
		if got := h.Get("Content-Disposition"); !strings.HasPrefix(got, "attachment;") {
			t.Errorf("%q served with disposition %q", mime, got)
		}
		assertSandboxed(t, mime, h.Get("Content-Security-Policy"), h.Get("X-Content-Type-Options"))
	}
}

func TestMediaStillRendersInPlace(t *testing.T) {
	for _, mime := range []string{"image/png", "image/jpeg", "audio/ogg", "video/webm"} {
		rec := httptest.NewRecorder()
		Write(rec, mime, "отчёт.png", []byte{1, 2, 3}, "private, max-age=60")
		h := rec.Header()
		if h.Get("Content-Type") != mime || !strings.HasPrefix(h.Get("Content-Disposition"), "inline;") {
			t.Errorf("%q: %q %q", mime, h.Get("Content-Type"), h.Get("Content-Disposition"))
		}
		if h.Get("Content-Length") != "3" || h.Get("Cache-Control") != "private, max-age=60" || rec.Body.Len() != 3 {
			t.Errorf("%q: body or headers lost: %v", mime, h)
		}
		assertSandboxed(t, mime, h.Get("Content-Security-Policy"), h.Get("X-Content-Type-Options"))
	}
}

func assertSandboxed(t *testing.T, mime, csp, nosniff string) {
	t.Helper()
	if !strings.HasPrefix(csp, "sandbox;") || !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("%q: CSP %q does not sandbox the file", mime, csp)
	}
	if strings.Contains(csp, "script-src") || strings.Contains(csp, "allow-scripts") {
		t.Errorf("%q: CSP %q lets something run", mime, csp)
	}
	if nosniff != "nosniff" {
		t.Errorf("%q: nosniff missing", mime)
	}
}
