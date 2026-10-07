package netguard

import (
	"net"
	"testing"
)

// The ranges outbound requests must never reach. Audit A-43 found the list
// stopping short of NAT64 and 6to4 — IPv6 prefixes that carry an IPv4 address
// and route to it — and of 198.18/15, 240/4 and 192.0.0/24.
func TestBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", // loopback, RFC 1918
		"169.254.169.254",       // cloud metadata
		"100.64.0.1", "0.1.2.3", // CGNAT, "this network"
		"192.0.0.8", "198.18.0.1", "240.0.0.1", // protocol assignments, benchmarking, class E
		"::1", "fe80::1", "fc00::1", // v6 loopback, link-local, ULA
		"64:ff9b::a9fe:a9fe", // NAT64 of 169.254.169.254
		"2002:a9fe:a9fe::1",  // 6to4 of 169.254.169.254
		"",                   // unparseable
	}
	for _, s := range blocked {
		if !BlockedIP(net.ParseIP(s)) {
			t.Errorf("%q must be blocked", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "142.250.74.46", "2a00:1450:4001:82a::200e"} {
		if BlockedIP(net.ParseIP(s)) {
			t.Errorf("%q is a public address and must be reachable", s)
		}
	}
}
