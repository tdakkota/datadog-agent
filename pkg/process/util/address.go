package util

import (
	"net"
	"sync"

	"inet.af/netaddr"
)

// NetIPFromIP returns a net.IP from a netaddr.IP
func NetIPFromIP(ip netaddr.IP, buf []byte) net.IP {
	if ip.Is4() {
		b := ip.As4()
		if len(buf) < len(b) {
			buf = make([]byte, len(b))
		}
		n := copy(buf, b[:])
		return net.IP(buf[:n])
	}

	if ip.Is6() {
		b := ip.As16()
		if len(buf) < len(b) {
			buf = make([]byte, len(b))
		}
		n := copy(buf, b[:])
		return net.IP(buf[:n])
	}
	return nil
}

// IPBufferPool is meant to be used in conjunction with `NetIPFromIP`
var IPBufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, net.IPv6len)
	},
}
