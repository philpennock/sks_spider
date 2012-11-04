package sks_spider

import "net"

func init() {
	prepDisallowedIPs()
}

var disallowedIPs []*net.IPNet

// RFC 5735 / BCP 153  Special Use IPv4 Addresses (DSUA)
// RFC 5736 IANA IPv4  Special Purpose Address Registry
// RFC 5737            IPv4 Address Blocks Reserved for Documentation
// RFC 5156            Special-Use IPv6 Addresses
func prepDisallowedIPs() {
	list := make([]*net.IPNet, 0, 50)
	for _, spec := range []string{
		"0.0.0.0/8",          // DSUA this
		"10.0.0.0/8",         // DSUA RFC1918
		"127.0.0.0/8",        // DSUA loopback
		"169.254.0.0/16",     // DSUA link-local
		"172.16.0.0/12",      // DSUA RFC1918
		"192.0.2.0/24",       // TEST-NET-1
		"192.88.99.0/24",     // DSUA 6to4 anycast relay; should not be sending SKS traffic to this underlying IP
		"192.168.0.0/16",     // DSUA RFC1918
		"198.18.0.0/15",      // DSUA Benchmarking
		"198.51.100.0/24",    // TEST-NET-2
		"203.0.113.0/24",     // TEST-NET-3
		"224.0.0.0/4",        // DSUA Class D Multicast
		"240.0.0.0/4",        // DSUA Class E
		"255.255.255.255/32", // DSUA Limited Broadcast

		"192.0.0.0/29", // http://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xml

		"2001:db8::/32",       // Documentation
		"2001:10::/28",        // ORCHID
		"2002:c058:6301::/48", // 6to4 anycast relay, IPv6-side
		"fc00::/7",            // RFC 4193 unique local unicast addresses
		"fe00::/8",            // various non-global scoped addresses
		"ff00::/8",            // Multicast
		"0100::/64",           // Blackhole / Discard prefix; RFC 6666
		// ignore (permit): 6bone, 6to4, teredo
		// For fe00::/8: the 16 feXE::/16 blocks are nominally global; we skip them too for sanity
	} {
		_, block, _ := net.ParseCIDR(spec)
		list = append(list, block)
	}
}

func IPDisallowed(ipstr string) bool {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return true
	}
	for _, block := range disallowedIPs {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
