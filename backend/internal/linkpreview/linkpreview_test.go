package linkpreview

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", // loopback
		"10.0.0.5", "192.168.1.1", "172.16.0.1", // RFC1918
		"169.254.169.254", // cloud metadata (link-local)
		"fd00::1",         // ULA
		"fe80::1",         // link-local v6
		"0.0.0.0", "::",   // unspecified
		"100.64.0.1",       // CGNAT
		"224.0.0.1",        // multicast
		"::ffff:127.0.0.1", // v4-mapped loopback
		"::ffff:10.0.0.1",  // v4-mapped private
	}
	for _, s := range blocked {
		if !blockedIP(net.ParseIP(s)) {
			t.Errorf("%s must be blocked", s)
		}
	}
	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946"}
	for _, s := range allowed {
		if blockedIP(net.ParseIP(s)) {
			t.Errorf("%s must be allowed", s)
		}
	}
	if !blockedIP(nil) {
		t.Errorf("nil ip must be blocked")
	}
}

func TestValidateURL(t *testing.T) {
	bad := []string{
		"ftp://example.com", "file:///etc/passwd", "gopher://x",
		"http://127.0.0.1/", "https://10.0.0.1/", "http://169.254.169.254/latest/meta-data/",
		"http://[::1]/", "javascript:alert(1)", "http://",
	}
	for _, s := range bad {
		u, _ := url.Parse(s)
		if u == nil {
			continue
		}
		if err := validateURL(u); err == nil {
			t.Errorf("%s must be rejected", s)
		}
	}
	good := []string{"http://example.com/", "https://sub.example.org/path?q=1"}
	for _, s := range good {
		u, _ := url.Parse(s)
		if err := validateURL(u); err != nil {
			t.Errorf("%s must be allowed: %v", s, err)
		}
	}
}

func TestGuardedClientBlocksLoopbackServer(t *testing.T) {
	// A real local server: the guarded dialer must refuse to connect to it
	// because it resolves to loopback (defends against DNS rebinding too).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("<html><head><title>secret</title></head></html>"))
	}))
	defer srv.Close()

	f := newFetcher()
	if _, err := f.fetchPreview(context.Background(), srv.URL); err != ErrBlockedURL {
		t.Fatalf("loopback fetch: want ErrBlockedURL, got %v", err)
	}
}

func TestFetchPreviewParsesOG(t *testing.T) {
	// Bind the guard-bypassing raw fetch path by pointing the client at a
	// public-looking server via a custom client whose dialer rewrites to the
	// test server. Simplest: exercise parse() directly on representative HTML.
	html := []byte(`
		<html><head>
		<meta property="og:title" content="Пример &amp; заголовок">
		<meta name="og:description" content="Описание страницы">
		<meta property="og:image" content="/img/cover.png">
		<meta property="og:site_name" content="Example">
		<title>fallback</title>
		</head><body>ignored</body></html>`)
	base, _ := url.Parse("https://example.com/article")
	p := parse(html, base)
	if p.Title != "Пример & заголовок" {
		t.Fatalf("title: %q", p.Title)
	}
	if p.Description != "Описание страницы" {
		t.Fatalf("description: %q", p.Description)
	}
	if p.ImageURL != "https://example.com/img/cover.png" {
		t.Fatalf("image resolved wrong: %q", p.ImageURL)
	}
	if p.SiteName != "Example" {
		t.Fatalf("site name: %q", p.SiteName)
	}
}

func TestParseFallsBackToTitleAndHost(t *testing.T) {
	html := []byte(`<html><head><title>Just a title</title></head></html>`)
	base, _ := url.Parse("https://news.example.org/x")
	p := parse(html, base)
	if p.Title != "Just a title" {
		t.Fatalf("title fallback: %q", p.Title)
	}
	if p.SiteName != "news.example.org" {
		t.Fatalf("site name host fallback: %q", p.SiteName)
	}
	if p.ImageURL != "" {
		t.Fatalf("no image expected, got %q", p.ImageURL)
	}
}

func TestAbsoluteRejectsNonHTTP(t *testing.T) {
	base, _ := url.Parse("https://example.com/")
	if got := absolute(base, "data:image/png;base64,AAAA"); got != "" {
		t.Fatalf("data URI must be dropped, got %q", got)
	}
	if got := absolute(base, "//cdn.example.com/a.png"); got != "https://cdn.example.com/a.png" {
		t.Fatalf("protocol-relative resolve: %q", got)
	}
}
