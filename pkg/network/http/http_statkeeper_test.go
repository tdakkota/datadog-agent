// +build linux_bpf

package http

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"inet.af/netaddr"
)

var (
	nativeEndian binary.ByteOrder
)

// In lack of binary.NativeEndian ...
func init() {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	if b == 0x04 {
		nativeEndian = binary.LittleEndian
	} else {
		nativeEndian = binary.BigEndian
	}
}

func TestProcessHTTPTransactions(t *testing.T) {
	sk := newHTTPStatkeeper(1000, newTelemetry())
	txs := make([]httpTX, 100)

	sourceIP := netaddr.MustParseIP("1.1.1.1")
	sourcePort := 1234
	destIP := netaddr.MustParseIP("2.2.2.2")
	destPort := 8080

	const numPaths = 10
	for i := 0; i < numPaths; i++ {
		path := "/testpath" + strconv.Itoa(i)

		for j := 0; j < 10; j++ {
			statusCode := (j%5 + 1) * 100
			latency := time.Duration(j%5) * time.Millisecond
			txs[i*10+j] = generateIPv4HTTPTransaction(sourceIP, destIP, sourcePort, destPort, path, statusCode, latency)
		}
	}

	sk.Process(txs)

	stats := sk.GetAndResetAllStats()
	assert.Equal(t, 0, len(sk.stats))
	assert.Equal(t, numPaths, len(stats))
	for key, stats := range stats {
		assert.Equal(t, "/testpath", key.Path[:9])
		for i := 0; i < 5; i++ {
			assert.Equal(t, 2, stats[i].Count)
			assert.Equal(t, 2.0, stats[i].Latencies.GetCount())

			p50, err := stats[i].Latencies.GetValueAtQuantile(0.5)
			assert.Nil(t, err)

			expectedLatency := float64(time.Duration(i) * time.Millisecond)
			acceptableError := expectedLatency * stats[i].Latencies.IndexMapping.RelativeAccuracy()
			assert.True(t, p50 >= expectedLatency-acceptableError)
			assert.True(t, p50 <= expectedLatency+acceptableError)
		}
	}
}

func generateIPv4HTTPTransaction(source netaddr.IP, dest netaddr.IP, sourcePort int, destPort int, path string, code int, latency time.Duration) httpTX {
	var tx httpTX

	reqFragment := fmt.Sprintf("GET %s HTTP/1.1\nHost: example.com\nUser-Agent: example-browser/1.0", path)
	latencyNS := _Ctype_ulonglong(uint64(latency))
	tx.request_started = 1
	tx.response_last_seen = tx.request_started + latencyNS
	tx.response_status_code = _Ctype_ushort(code)
	tx.request_fragment = requestFragment([]byte(reqFragment))
	if source.Is6() {
		tx.tup.saddr.in6_u = source.As16()
	} else {
		b := source.As4()
		copy(tx.tup.saddr.in6_u[12:], b[:])
	}
	tx.tup.sport = _Ctype_ushort(sourcePort)
	if dest.Is6() {
		tx.tup.daddr.in6_u = dest.As16()
	} else {
		b := dest.As4()
		copy(tx.tup.daddr.in6_u[12:], b[:])
	}

	tx.tup.dport = _Ctype_ushort(destPort)
	tx.tup.metadata = 1

	return tx
}

func BenchmarkProcessSameConn(b *testing.B) {
	sk := newHTTPStatkeeper(1000, newTelemetry())
	tx := generateIPv4HTTPTransaction(
		netaddr.MustParseIP("1.1.1.1"),
		netaddr.MustParseIP("2.2.2.2"),
		1234,
		8080,
		"foobar",
		404,
		30*time.Millisecond,
	)
	transactions := []httpTX{tx}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.Process(transactions)
	}
}
