package push

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kisy-backend/internal/platform/netguard"
)

// A browser push subscription names the URL the server will POST to, and the
// browser — that is, any client — supplies it. It used to be stored as given
// and called with a default http.Client: no timeout, ten redirects, any
// scheme, any host. That made the server a blind SSRF probe into its own
// network (an endpoint of http://169.254.169.254/... was accepted and stored)
// and let a slow host hold a goroutine per push forever (audit A-17).
//
// Real subscriptions only ever point at a handful of push services, so that
// is the rule: https, on the default port, at one of these hosts.

// ErrBadEndpoint: not the address of a known browser push service.
var ErrBadEndpoint = errors.New("push: not a browser push service endpoint")

// pushHosts are the push services of the browsers KISY runs in. An entry
// starting with "." matches that domain's subdomains.
var pushHosts = []string{
	"fcm.googleapis.com",                // Chrome, Edge (Chromium), Opera, Samsung Internet
	"updates.push.services.mozilla.com", // Firefox
	".push.services.mozilla.com",
	".notify.windows.com", // Edge on Windows (WNS)
	"web.push.apple.com",  // Safari
	".push.apple.com",
}

// sendTimeout bounds one delivery to one push service.
const sendTimeout = 10 * time.Second

// ValidateEndpoint accepts only the URL of a known push service.
func ValidateEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" {
		return ErrBadEndpoint
	}
	if port := u.Port(); port != "" && port != "443" {
		return ErrBadEndpoint
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || net.ParseIP(host) != nil {
		return ErrBadEndpoint
	}
	for _, allowed := range pushHosts {
		if strings.HasPrefix(allowed, ".") {
			if strings.HasSuffix(host, allowed) {
				return nil
			}
			continue
		}
		if host == allowed {
			return nil
		}
	}
	return ErrBadEndpoint
}

// newSendClient is the client every browser push goes through: bounded in
// time, never following a redirect (a push service answers, it does not send
// you elsewhere), and refusing private addresses at dial time even if a
// listed host should ever resolve to one.
func newSendClient() *http.Client {
	dialer := &net.Dialer{Timeout: sendTimeout, Control: netguard.Control}
	return &http.Client{
		Timeout: sendTimeout,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   sendTimeout,
			ResponseHeaderTimeout: sendTimeout,
			MaxIdleConnsPerHost:   4,
			IdleConnTimeout:       90 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
