package spa

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The SPA handler turns a URL into a filesystem path, which is the shape of
// every path-traversal bug. It is reached through chi, which does not
// normalise the path for us, so a request arrives with "../" intact and
// percent-decoded — exactly as an attacker wrote it.

func webRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html>app"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestResolveStaysInsideTheWebRoot(t *testing.T) {
	dir := filepath.Join(string(filepath.Separator), "srv", "web")
	// Percent-encoding is not represented here on purpose: net/url decodes the
	// path before the handler sees it, so "%2e%2e%2f" arrives as "../".
	outside := []string{
		"/../../../../etc/passwd",
		"/assets/../../../../etc/passwd",
		"/./../.././etc/shadow",
		"//....//....//etc/passwd",
		"/assets/..",
		"/..",
		"/",
	}
	prefix := dir + string(filepath.Separator)
	for _, p := range outside {
		got := resolve(dir, p)
		if got != dir && !strings.HasPrefix(got, prefix) {
			t.Fatalf("resolve(%q) escaped the web root: %q", p, got)
		}
		// Element-wise, not substring: "...." is an ordinary directory name
		// that happens to contain "..", and rejecting it would be wrong.
		for _, elem := range strings.Split(got, string(filepath.Separator)) {
			if elem == ".." {
				t.Fatalf("resolve(%q) left a traversal in place: %q", p, got)
			}
		}
	}
}

func TestTraversalDoesNotServeAFileOutsideTheRoot(t *testing.T) {
	dir := webRoot(t)
	// A real file next to the web root, the thing a traversal is after.
	secret := filepath.Join(filepath.Dir(dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("TOP SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(secret) })

	h := Handler(dir)
	for _, p := range []string{"/../secret.txt", "/assets/../../secret.txt", "/./.././secret.txt"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if strings.Contains(rec.Body.String(), "TOP SECRET") {
			t.Fatalf("GET %s leaked a file from outside the web root", p)
		}
	}
}

func TestServesAssetsAndFallsBackToIndex(t *testing.T) {
	dir := webRoot(t)
	h := Handler(dir)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "console.log") {
		t.Fatalf("asset not served: %d %q", rec.Code, rec.Body.String())
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("assets must be cached immutably, got %q", cc)
	}

	// A client-side route is not a file; the app itself answers.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/chats/42", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "doctype") {
		t.Fatalf("history route did not fall back to index.html: %d", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
		t.Fatalf("index.html must not be cached, got %q", cc)
	}
}
