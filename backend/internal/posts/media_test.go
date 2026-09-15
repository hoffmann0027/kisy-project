package posts

import (
	"net/url"
	"strings"
	"testing"
)

// A post attachment's name arrives in a header the uploader controls, is
// stored, and comes back in a Content-Disposition. These tests are what the
// #nosec on that line rests on: without them the suppression would be a claim
// rather than a checked invariant.

func TestFileNameCannotStartASecondHeader(t *testing.T) {
	// The classic response-splitting payload.
	got := safeFileName("report.pdf\r\nSet-Cookie: admin=1")
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("a newline survived sanitising: %q", got)
	}
	// And after escaping there is nothing left that could end the value early.
	escaped := url.PathEscape(got)
	if strings.ContainsAny(escaped, "\r\n\";") {
		t.Fatalf("escaped name still carries header syntax: %q", escaped)
	}
}

func TestFileNameIsANameAndNotAPath(t *testing.T) {
	for _, raw := range []string{"../../etc/passwd", `..\..\windows\system32`, "/etc/shadow"} {
		got := safeFileName(raw)
		if strings.ContainsAny(got, `/\`) {
			t.Fatalf("separators survived in %q -> %q", raw, got)
		}
		if strings.HasPrefix(got, ".") {
			t.Fatalf("leading dot survived in %q -> %q", raw, got)
		}
	}
}

func TestEmptyOrUselessNameGetsAFallback(t *testing.T) {
	for _, raw := range []string{"", "   ", "...", "\x00\x01"} {
		if got := safeFileName(raw); got == "" {
			t.Fatalf("no fallback for %q", raw)
		}
	}
}

func TestOrdinaryNamesSurviveIntact(t *testing.T) {
	// Sanitising must not mangle the common case, Russian names included.
	for _, raw := range []string{"отчёт за квартал.pdf", "photo_2026.jpeg", "Screenshot 1.png"} {
		if got := safeFileName(raw); got != raw {
			t.Fatalf("mangled an ordinary name: %q -> %q", raw, got)
		}
	}
}

func TestMediaKindFollowsTheSniffedType(t *testing.T) {
	// The kind drives which player the client renders, so it must follow the
	// bytes, like the content type does.
	cases := map[string]string{
		"image/png":       MediaImage,
		"audio/ogg":       MediaAudio,
		"video/mp4":       MediaVideo,
		"application/pdf": MediaFile,
		"text/html":       MediaFile,
	}
	for mime, want := range cases {
		if got := mediaKindFor(mime); got != want {
			t.Fatalf("mediaKindFor(%q) = %q, want %q", mime, got, want)
		}
	}
}
