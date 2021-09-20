package util

import (
	"net"
	"runtime"
	"testing"

	"inet.af/netaddr"
)

func BenchmarkNetIPFromIP(b *testing.B) {
	var (
		buf  = make([]byte, 16)
		addr = netaddr.MustParseIP("8.8.8.8")
		ip   net.IP
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip = NetIPFromIP(addr, buf)
	}
	runtime.KeepAlive(ip)
}
