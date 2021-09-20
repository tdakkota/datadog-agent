package testutil

import (
	"fmt"
	"net"
	"testing"

	"inet.af/netaddr"

	"github.com/stretchr/testify/require"
)

// PingTCP connects to the provided IP address over TCP/TCPv6, sends the string "ping",
// reads from the connection, and returns the open connection for further use/inspection.
func PingTCP(t *testing.T, ip netaddr.IP, port uint16) net.Conn {
	addr := fmt.Sprintf("%s:%d", ip, port)
	network := "tcp"
	if ip.Is6() {
		network = "tcp6"
		addr = fmt.Sprintf("[%s]:%d", ip, port)
	}

	conn, err := net.Dial(network, addr)
	require.NoError(t, err)

	_, err = conn.Write([]byte("ping"))
	require.NoError(t, err)
	bs := make([]byte, 10)
	_, err = conn.Read(bs)
	require.NoError(t, err)

	return conn
}

// PingUDP connects to the provided IP address over UDP/UDPv6, sends the string "ping",
// and returns the open connection for further use/inspection.
func PingUDP(t *testing.T, ip netaddr.IP, port uint16) net.Conn {
	network := "udp"
	if ip.Is6() {
		network = "udp6"
	}

	conn, err := net.DialUDP(network, nil, netaddr.IPPortFrom(ip, port).UDPAddr())
	require.NoError(t, err)

	_, err = conn.Write([]byte("ping"))
	require.NoError(t, err)

	return conn
}
