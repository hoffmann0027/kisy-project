// Package linkpreview fetches OpenGraph metadata for a URL on the user's
// behalf. Because the server makes the outbound request, SSRF is the primary
// threat (docs/spec/06-security.md): a hostile URL must not let a caller
// reach internal services, cloud metadata endpoints or the loopback
// interface. Every connection is gated on the RESOLVED IP right before the
// socket is dialed, which also defeats DNS-rebinding.
package linkpreview

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

var (
	ErrBlockedURL  = errors.New("linkpreview: url is not allowed")
	ErrFetchFailed = errors.New("linkpreview: fetch failed")
	ErrNotHTML     = errors.New("linkpreview: response is not html")
	ErrTooLarge    = errors.New("linkpreview: response too large")
)

const (
	fetchTimeout = 6 * time.Second
	maxRedirects = 3
	// Real-world pages put OG meta tags far into a script-heavy <head>
	// (YouTube's og:title sits ~640 KiB in), so the cap must be generous;
	// memory stays bounded by the LimitReader and the 30/min rate limit.
	maxHTMLBytes   = 2 << 20 // parse at most 2 MiB of HTML
	maxImageBytes  = 5 << 20 // proxy at most 5 MiB of image
	userAgent      = "KISY-LinkPreview/1.0 (+https://kisy.local)"
	acceptLanguage = "en,ru;q=0.8"
)

// blockedIP reports whether an IP must never be dialed for an outbound
// preview fetch. Covers loopback, private (RFC1918 + ULA), link-local
// (incl. 169.254.169.254 cloud metadata), CGNAT, multicast and unspecified.
func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast() {
		return true
	}
	// Carrier-grade NAT 100.64.0.0/10 and "this network" 0.0.0.0/8.
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 0 {
			return true
		}
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return true
		}
	}
	return false
}

// guardedControl is the dialer hook: it runs after DNS resolution with the
// concrete address about to be connected, so a hostname that resolves to a
// private IP (including via rebinding between checks) is still rejected.
func guardedControl(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrBlockedURL
	}
	ip := net.ParseIP(host)
	if blockedIP(ip) {
		return ErrBlockedURL
	}
	return nil
}

// newGuardedClient builds an http.Client whose dialer blocks private targets
// and whose redirects are re-validated (scheme + host) at every hop.
func newGuardedClient() *http.Client {
	dialer := &net.Dialer{Timeout: fetchTimeout, Control: guardedControl}
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   fetchTimeout,
		ResponseHeaderTimeout: fetchTimeout,
		DisableKeepAlives:     true,
		MaxIdleConns:          1,
	}
	return &http.Client{
		Timeout:   fetchTimeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return ErrBlockedURL
			}
			if err := validateURL(req.URL); err != nil {
				return err
			}
			return nil
		},
	}
}

// validateURL enforces the scheme allowlist and a literal-IP check. Hostname
// resolution is re-checked at dial time by guardedControl.
func validateURL(u *url.URL) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrBlockedURL
	}
	host := u.Hostname()
	if host == "" {
		return ErrBlockedURL
	}
	// A URL that is itself a literal private IP is rejected up-front.
	if ip := net.ParseIP(host); ip != nil && blockedIP(ip) {
		return ErrBlockedURL
	}
	return nil
}

// fetchResult is the raw fetched content plus its final URL (after redirects).
type fetchResult struct {
	body     []byte
	mimeType string
	finalURL *url.URL
}

// fetch retrieves up to limit bytes of a URL through the guarded client,
// returning an error for blocked targets, oversized bodies or transport
// failures. wantHTML restricts the accepted content type.
func fetch(ctx context.Context, client *http.Client, rawURL string, limit int64, wantHTML bool) (*fetchResult, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, ErrBlockedURL
	}
	if err := validateURL(u); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, ErrBlockedURL
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", acceptLanguage)

	resp, err := client.Do(req)
	if err != nil {
		// A blocked dial/redirect surfaces as a wrapped ErrBlockedURL.
		if errors.Is(err, ErrBlockedURL) {
			return nil, ErrBlockedURL
		}
		return nil, ErrFetchFailed
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, ErrFetchFailed
	}
	ct := resp.Header.Get("Content-Type")
	mime := ct
	if i := strings.IndexByte(mime, ';'); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	if wantHTML && !strings.HasPrefix(mime, "text/html") {
		return nil, ErrNotHTML
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, ErrFetchFailed
	}
	if int64(len(body)) > limit {
		return nil, ErrTooLarge
	}
	return &fetchResult{body: body, mimeType: mime, finalURL: resp.Request.URL}, nil
}

func fmtCacheKey(rawURL string) string {
	return fmt.Sprintf("linkpreview:%s", rawURL)
}

func parseURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, ErrBlockedURL
	}
	return u, nil
}

// fetcher performs the guarded outbound requests. Split from Service so tests
// can stub it without touching Redis.
type fetcher struct {
	client *http.Client
}

func newFetcher() *fetcher {
	return &fetcher{client: newGuardedClient()}
}

func (f *fetcher) fetchPreview(ctx context.Context, rawURL string) (*Preview, error) {
	res, err := fetch(ctx, f.client, rawURL, maxHTMLBytes, true)
	if err != nil {
		return nil, err
	}
	p := parse(res.body, res.finalURL)
	if p.Title == "" && p.Description == "" {
		// Nothing worth showing; treat as a failure so it is negatively cached.
		return nil, ErrFetchFailed
	}
	return &p, nil
}

func (f *fetcher) fetchImage(ctx context.Context, rawURL string) (*fetchResult, error) {
	return fetch(ctx, f.client, rawURL, maxImageBytes, false)
}
