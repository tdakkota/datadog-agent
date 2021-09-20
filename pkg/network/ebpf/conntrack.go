//+build linux

package ebpf

import (
	"fmt"
	"net"
	"strconv"

	"inet.af/netaddr"
)

// IPv4 returns an IPv4 version of In6Addr as a netaddr.IP
func (a In6Addr) IPv4() netaddr.IP {
	return netaddr.IPv4(a.U[12], a.U[13], a.U[14], a.U[15])
}

// IPv6 returns an IPv6 version of In6Addr as a netaddr.IP
func (a In6Addr) IPv6() netaddr.IP {
	// use raw here to preserve 4over6 traffic as IPv6
	return netaddr.IPv6Raw(a.U)
}

// Family returns whether a tuple is IPv4 or IPv6
func (t ConntrackTuple) Family() ConnFamily {
	if t.Metadata&uint32(IPv6) != 0 {
		return IPv6
	}
	return IPv4
}

// Type returns whether a tuple is TCP or UDP
func (t ConntrackTuple) Type() ConnType {
	if t.Metadata&uint32(TCP) != 0 {
		return TCP
	}
	return UDP
}

// SourceAddress returns the source address
func (t ConntrackTuple) SourceAddress() netaddr.IP {
	if t.Family() == IPv6 {
		return t.Saddr.IPv6()
	}
	return t.Saddr.IPv4()
}

// SourceEndpoint returns the source address and source port joined
func (t ConntrackTuple) SourceEndpoint() string {
	return net.JoinHostPort(t.SourceAddress().String(), strconv.Itoa(int(t.Sport)))
}

// DestAddress returns the destination address
func (t ConntrackTuple) DestAddress() netaddr.IP {
	if t.Family() == IPv6 {
		return t.Daddr.IPv6()
	}
	return t.Daddr.IPv4()
}

// DestEndpoint returns the destination address and source port joined
func (t ConntrackTuple) DestEndpoint() string {
	return net.JoinHostPort(t.DestAddress().String(), strconv.Itoa(int(t.Dport)))
}

func (t ConntrackTuple) String() string {
	return fmt.Sprintf(
		"[%s%s] [%s ⇄ %s] (ns: %d)",
		t.Type(),
		t.Family(),
		t.SourceEndpoint(),
		t.DestEndpoint(),
		t.Netns,
	)
}

// FromIP updates the In6Addr from the netaddr.IP provided.
func (a *In6Addr) FromIP(ip netaddr.IP) {
	if ip.Is4() {
		var z [12]byte
		copy(a.U[:12], z[:])
		b := ip.As4()
		copy(a.U[12:], b[:])
	} else {
		a.U = ip.As16()
	}
}
