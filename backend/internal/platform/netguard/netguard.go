// Package netguard keeps outbound requests that the server makes on a user's
// behalf away from the server's own network.
//
// Two features send requests to addresses a user supplied: link previews
// (internal/linkpreview) and browser push (internal/push — the subscription
// endpoint comes from the browser). Both must never reach loopback, the
// private ranges, link-local — 169.254.169.254 is the cloud metadata service
// — or carrier-grade NAT. The check runs in the dialer, after DNS resolution,
// on the concrete address about to be connected: a hostname that resolves to a
// private address, including one that changes its answer between a check and
// the connection (DNS rebinding), is still refused.
package netguard

import (
	"errors"
	"net"
	"syscall"
)

// ErrBlocked: the address is inside a range outbound requests may not reach.
var ErrBlocked = errors.New("netguard: address not allowed")

// BlockedIP reports whether ip is an address outbound requests must not
// reach. A nil (unparseable) address is blocked.
func BlockedIP(ip net.IP) bool {
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

// Control is a net.Dialer Control hook enforcing BlockedIP on the resolved
// address of every connection.
func Control(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return ErrBlocked
	}
	if BlockedIP(net.ParseIP(host)) {
		return ErrBlocked
	}
	return nil
}
