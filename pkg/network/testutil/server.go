// +build linux

package testutil

import (
	"fmt"
	"io"
	"net"
	"testing"

	"inet.af/netaddr"

	"github.com/DataDog/datadog-agent/pkg/process/util"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netns"
)

// StartServerTCPNs is identical to StartServerTCP, but it operates with the
// network namespace provided by name.
func StartServerTCPNs(t *testing.T, ip netaddr.IP, port int, ns string) io.Closer {
	h, err := netns.GetFromName(ns)
	require.NoError(t, err)

	var closer io.Closer
	_ = util.WithNS("/proc", h, func() error {
		closer = StartServerTCP(t, ip, port)
		return nil
	})

	return closer
}

// StartServerTCP starts a TCP server listening at provided IP address and port.
// It will respond to any connection with "hello" and then close the connection.
// It returns an io.Closer that should be Close'd when you are finished with it.
func StartServerTCP(t *testing.T, ip netaddr.IP, port int) io.Closer {
	_, closer, err := NewTCPServerOnAddress(netaddr.IPPortFrom(ip, uint16(port)), func(c net.Conn) {
		_, _ = c.Write([]byte("hello"))
		c.Close()
	})
	require.NoError(t, err)
	return closer
}

func NewTCPServer(onMessage func(c net.Conn)) (string, io.Closer, error) {
	return NewTCPServerOnAddress(netaddr.MustParseIPPort("127.0.0.1:0"), onMessage)
}

func NewTCPServerOnAddress(addr netaddr.IPPort, onMessage func(c net.Conn)) (string, io.Closer, error) {
	network := "tcp"
	if addr.IP().Is6() {
		network = "tcp6"
	}
	ln, err := net.Listen(network, addr.String())
	if err != nil {
		return "", nil, err
	}

	started := make(chan struct{})
	go func() {
		close(started)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go onMessage(conn)
		}
	}()

	<-started
	return ln.Addr().String(), ln, nil
}

func NewUDPServer(payloadSize int, onMessage func(b []byte, n int) []byte) (string, io.Closer, error) {
	return NewUDPServerOnAddress(netaddr.MustParseIPPort("127.0.0.1:0"), payloadSize, onMessage)
}

func NewUDPServerOnAddress(addr netaddr.IPPort, payloadSize int, onMessage func(b []byte, n int) []byte) (string, io.Closer, error) {
	network := "udp"
	if addr.IP().Is6() {
		network = "udp6"
	}
	ln, err := net.ListenUDP(network, addr.UDPAddr())
	if err != nil {
		return "", nil, err
	}

	started := make(chan struct{})
	go func() {
		close(started)
		buf := make([]byte, payloadSize)
		for {
			n, addr, err := ln.ReadFrom(buf)
			if err != nil {
				break
			}
			_, err = ln.WriteTo(onMessage(buf, n), addr)
			if err != nil {
				fmt.Println(err)
				break
			}
		}
		ln.Close()
	}()

	<-started
	return ln.LocalAddr().String(), ln, nil
}

// StartServerUDPNs is identical to StartServerUDP, but it operates with the
// network namespace provided by name.
func StartServerUDPNs(t *testing.T, ip netaddr.IP, port uint16, ns string) io.Closer {
	h, err := netns.GetFromName(ns)
	require.NoError(t, err)

	var closer io.Closer
	_ = util.WithNS("/proc", h, func() error {
		closer = StartServerUDP(t, ip, port)
		return nil
	})

	return closer
}

// StartServerUDP starts a UDP server listening at provided IP address and port.
// It does not respond in any fashion to sent datagrams.
// It returns an io.Closer that should be Close'd when you are finished with it.
func StartServerUDP(t *testing.T, ip netaddr.IP, port uint16) io.Closer {
	ch := make(chan struct{})
	network := "udp"
	if ip.Is6() {
		network = "udp6"
	}

	l, err := net.ListenUDP(network, netaddr.IPPortFrom(ip, port).UDPAddr())
	require.NoError(t, err)
	go func() {
		close(ch)

		for {
			bs := make([]byte, 10)
			_, err := l.Read(bs)
			if err != nil {
				return
			}
		}
	}()
	<-ch

	return l
}
