//+build linux

package ebpf

import (
	"fmt"
	"net"
	"strconv"
	"unsafe"

	"inet.af/netaddr"
)

// IPv4 returns an IPv4 version of Inet6Addr as a netaddr.IP
func (a Inet6Addr) IPv4() netaddr.IP {
	return netaddr.IPv4(a.U[12], a.U[13], a.U[14], a.U[15])
}

// IPv6 returns an IPv6 version of Inet6Addr as a netaddr.IP
func (a Inet6Addr) IPv6() netaddr.IP {
	// use raw here to preserve 4over6 traffic as IPv6
	return netaddr.IPv6Raw(a.U)
}

// Family returns whether a tuple is IPv4 or IPv6
func (t ConnTuple) Family() ConnFamily {
	if t.Metadata&uint32(IPv6) != 0 {
		return IPv6
	}
	return IPv4
}

// Type returns whether a tuple is TCP or UDP
func (t ConnTuple) Type() ConnType {
	if t.Metadata&uint32(TCP) != 0 {
		return TCP
	}
	return UDP
}

// SourceAddress returns the source address
func (t ConnTuple) SourceAddress() netaddr.IP {
	if t.Family() == IPv6 {
		return t.Saddr.IPv6()
	}
	return t.Saddr.IPv4()
}

// SourceEndpoint returns the source address and source port joined
func (t ConnTuple) SourceEndpoint() string {
	return net.JoinHostPort(t.SourceAddress().String(), strconv.Itoa(int(t.Sport)))
}

// DestAddress returns the destination address
func (t ConnTuple) DestAddress() netaddr.IP {
	if t.Family() == IPv6 {
		return t.Daddr.IPv6()
	}
	return t.Daddr.IPv4()
}

// DestEndpoint returns the destination address and source port joined
func (t ConnTuple) DestEndpoint() string {
	return net.JoinHostPort(t.DestAddress().String(), strconv.Itoa(int(t.Dport)))
}

func (t ConnTuple) String() string {
	return fmt.Sprintf(
		"[%s%s] [PID: %d] [%s ⇄ %s] (ns: %d)",
		t.Type(),
		t.Family(),
		t.Pid,
		t.SourceEndpoint(),
		t.DestEndpoint(),
		t.Netns,
	)
}

// ConnectionDirection returns the direction of the connection (incoming vs outgoing).
func (cs ConnStats) ConnectionDirection() ConnDirection {
	return ConnDirection(cs.Direction)
}

// IsAssured returns whether the connection has seen traffic in both directions.
func (cs ConnStats) IsAssured() bool {
	return cs.Flags&uint32(Assured) != 0
}

// ToBatch converts a byte slice to a Batch pointer.
func ToBatch(data []byte) *Batch {
	return (*Batch)(unsafe.Pointer(&data[0]))
}

// FromIP updates the Inet6Addr from the netaddr.IP provided.
func (a *Inet6Addr) FromIP(ip netaddr.IP) {
	if ip.Is4() {
		var z [12]byte
		copy(a.U[:12], z[:])
		b := ip.As4()
		copy(a.U[12:], b[:])
	} else {
		a.U = ip.As16()
	}
}
