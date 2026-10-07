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

// blockedNets are ranges the standard library's predicates do not cover
// (audit A-43): "this network", carrier-grade NAT, IETF protocol assignments,
// benchmarking, the reserved class E, and the IPv6 transition prefixes that
// carry an IPv4 address inside — NAT64 and 6to4 — which would otherwise route
// to a private v4 address through a public-looking v6 one.
var blockedNets = func() []*net.IPNet {
	var out []*net.IPNet
	for _, cidr := range []string{
		"0.0.0.0/8",
		"100.64.0.0/10",
		"192.0.0.0/24",
		"198.18.0.0/15",
		"240.0.0.0/4",
		"64:ff9b::/96",
		"64:ff9b:1::/48",
		"2002::/16",
	} {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("netguard: bad CIDR " + cidr)
		}
		out = append(out, n)
	}
	return out
}()

// BlockedIP reports whether ip is an address outbound requests must not
// reach. A nil (unparseable) address is blocked.
func BlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast() {
		return true
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
