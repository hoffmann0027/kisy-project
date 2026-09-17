// Package turnstile verifies Cloudflare Turnstile tokens: the proof, obtained
// by the sign-up and password-recovery screens, that a person rather than a
// script filled the form in.
//
// The check is server-side and mandatory where it is wired: a client that
// skips the widget gets a 403, not a registered account. See
// https://developers.cloudflare.com/turnstile/get-started/server-side-validation/.
package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultEndpoint is Cloudflare's validation API.
const DefaultEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

var (
	// ErrMissing: the request carried no token at all.
	ErrMissing = errors.New("turnstile: token missing")
	// ErrRejected: Cloudflare looked at the token and said no — forged,
	// expired, already used, or issued for another site.
	ErrRejected = errors.New("turnstile: token rejected")
	// ErrUnavailable: the check could not be made (Cloudflare unreachable, or
	// the deployment is not configured). Fails closed: nothing is let through.
	ErrUnavailable = errors.New("turnstile: verification unavailable")
)

// Verifier decides whether a token is valid for a request from remoteIP.
type Verifier interface {
	Verify(ctx context.Context, token, remoteIP string) error
}

// Client calls siteverify.
type Client struct {
	secret   string
	endpoint string
	http     *http.Client
	// hostnames, when set, are the only hostnames a token may have been
	// issued on. The widget's own allowlist in the Cloudflare dashboard is the
	// first line; this is the second, so a token solved on some other page
	// that happens to share the site key is still refused.
	hostnames map[string]bool
}

// NewClient returns a verifier for secret. A zero timeout means five seconds:
// a sign-up waiting longer than that on a third party is a hung request.
func NewClient(secret, endpoint string, timeout time.Duration, allowedHostnames []string) *Client {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	hosts := map[string]bool{}
	for _, h := range allowedHostnames {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			hosts[h] = true
		}
	}
	return &Client{secret: secret, endpoint: endpoint, http: &http.Client{Timeout: timeout}, hostnames: hosts}
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
	Hostname   string   `json:"hostname"`
}

// maxTokenLength: Cloudflare documents tokens of up to 2048 characters.
const maxTokenLength = 2048

func (c *Client) Verify(ctx context.Context, token, remoteIP string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrMissing
	}
	if len(token) > maxTokenLength {
		return ErrRejected
	}

	form := url.Values{"secret": {c.secret}, "response": {token}}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: siteverify status %d", ErrUnavailable, resp.StatusCode)
	}

	var out siteverifyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&out); err != nil {
		return fmt.Errorf("%w: decode: %v", ErrUnavailable, err)
	}
	if !out.Success {
		// A secret Cloudflare does not recognise is our misconfiguration, not
		// the user's failed challenge — report it as unavailable so it is
		// logged and surfaces as a 503, not as "you look like a bot".
		for _, code := range out.ErrorCodes {
			if code == "invalid-input-secret" || code == "missing-input-secret" {
				return fmt.Errorf("%w: %s", ErrUnavailable, code)
			}
		}
		return fmt.Errorf("%w: %s", ErrRejected, strings.Join(out.ErrorCodes, ","))
	}
	if len(c.hostnames) > 0 && !c.hostnames[strings.ToLower(out.Hostname)] {
		return fmt.Errorf("%w: issued on hostname %q", ErrRejected, out.Hostname)
	}
	return nil
}

// Disabled accepts every request. Only ever wired outside production (see
// internal/config): local development and tests have no Cloudflare widget.
type Disabled struct{}

func (Disabled) Verify(context.Context, string, string) error { return nil }

// Unconfigured refuses every request. Production without Turnstile keys gets
// this, so an unfinished setup closes registration instead of opening it to
// scripts — while the rest of the app (login, chats) keeps working.
type Unconfigured struct{}

func (Unconfigured) Verify(context.Context, string, string) error {
	return fmt.Errorf("%w: TURNSTILE_SECRET is not configured", ErrUnavailable)
}

// Select picks the verifier for a deployment and the site key the sign-up
// screen should render. Disabled (development only — config never passes
// enabled=false in production) checks nothing and shows no widget; enabled
// without both keys refuses every sign-up rather than silently opening it.
func Select(enabled bool, siteKey, secret string, allowedHostnames []string) (Verifier, string) {
	switch {
	case !enabled:
		return Disabled{}, ""
	case siteKey == "" || secret == "":
		return Unconfigured{}, ""
	default:
		return NewClient(secret, "", 0, allowedHostnames), siteKey
	}
}
