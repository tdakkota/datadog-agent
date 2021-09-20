// +build linux_bpf

package tracer

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/network"
	"github.com/DataDog/datadog-agent/pkg/network/config"
	"github.com/DataDog/datadog-agent/pkg/network/netlink"
	netlinktestutil "github.com/DataDog/datadog-agent/pkg/network/netlink/testutil"
	nettestutil "github.com/DataDog/datadog-agent/pkg/network/testutil"
	"github.com/DataDog/datadog-agent/pkg/process/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netns"
	"inet.af/netaddr"
)

const (
	natPort    = 5432
	nonNatPort = 9876
)

func TestConntrackers(t *testing.T) {
	conntrackers := []struct {
		name   string
		create func(*config.Config) (netlink.Conntracker, error)
	}{
		{"netlink", setupNetlinkConntracker},
		{"eBPF", setupEBPFConntracker},
	}
	for _, conntracker := range conntrackers {
		t.Run(conntracker.name, func(t *testing.T) {
			t.Run("IPv4", func(t *testing.T) {
				cfg := config.New()
				ct, err := conntracker.create(cfg)
				require.NoError(t, err)
				defer ct.Close()

				defer netlinktestutil.TeardownDNAT(t)
				netlinktestutil.SetupDNAT(t)

				testConntracker(t, netaddr.MustParseIP("1.1.1.1"), netaddr.MustParseIP("2.2.2.2"), ct)
			})
			t.Run("IPv6", func(t *testing.T) {
				cfg := config.New()
				ct, err := conntracker.create(cfg)
				require.NoError(t, err)
				defer ct.Close()

				defer netlinktestutil.TeardownDNAT6(t)
				netlinktestutil.SetupDNAT6(t)

				testConntracker(t, netaddr.MustParseIP("fd00::1"), netaddr.MustParseIP("fd00::2"), ct)
			})
			t.Run("cross namespace", func(t *testing.T) {
				cfg := config.New()
				cfg.EnableConntrackAllNamespaces = true
				ct, err := conntracker.create(cfg)
				require.NoError(t, err)
				defer ct.Close()

				testConntrackerCrossNamespace(t, ct)
			})
		})
	}
}

func setupEBPFConntracker(cfg *config.Config) (netlink.Conntracker, error) {
	if os.Getenv("BPF_DEBUG") != "" {
		cfg.BPFDebug = true
	}
	cfg.EnableRuntimeCompiler = true
	cfg.AllowPrecompiledFallback = false
	return NewEBPFConntracker(cfg)
}

func setupNetlinkConntracker(cfg *config.Config) (netlink.Conntracker, error) {
	cfg.ConntrackMaxStateSize = 100
	cfg.ConntrackRateLimit = 500
	ct, err := netlink.NewConntracker(cfg)
	time.Sleep(100 * time.Millisecond)
	return ct, err
}

func testConntracker(t *testing.T, serverIP, clientIP netaddr.IP, ct netlink.Conntracker) {
	srv1 := nettestutil.StartServerTCP(t, serverIP, natPort)
	defer srv1.Close()
	srv2 := nettestutil.StartServerTCP(t, serverIP, nonNatPort)
	defer srv2.Close()
	srv3 := nettestutil.StartServerUDP(t, serverIP, natPort)
	defer srv3.Close()

	localAddr := nettestutil.PingTCP(t, clientIP, natPort).LocalAddr().(*net.TCPAddr)
	time.Sleep(1 * time.Second)

	curNs, err := util.GetCurrentIno()
	require.NoError(t, err)

	family := network.AFINET
	if len(localAddr.IP) == net.IPv6len {
		family = network.AFINET6
	}

	lip, _ := netaddr.FromStdIP(localAddr.IP)

	trans := ct.GetTranslationForConn(
		network.ConnectionStats{
			Source: lip,
			SPort:  uint16(localAddr.Port),
			Dest:   clientIP,
			DPort:  uint16(natPort),
			Type:   network.TCP,
			Family: family,
			NetNS:  curNs,
		},
	)
	require.NotNil(t, trans)
	assert.Equal(t, serverIP, trans.ReplSrcIP)

	localAddrUDP := nettestutil.PingUDP(t, clientIP, natPort).LocalAddr().(*net.UDPAddr)
	time.Sleep(time.Second)

	family = network.AFINET
	if len(localAddrUDP.IP) == net.IPv6len {
		family = network.AFINET6
	}

	lip, _ = netaddr.FromStdIP(localAddrUDP.IP)
	trans = ct.GetTranslationForConn(
		network.ConnectionStats{
			Source: lip,
			SPort:  uint16(localAddrUDP.Port),
			Dest:   clientIP,
			DPort:  uint16(natPort),
			Type:   network.UDP,
			Family: family,
			NetNS:  curNs,
		},
	)
	require.NotNil(t, trans)
	assert.Equal(t, serverIP, trans.ReplSrcIP)

	// now dial TCP directly
	localAddr = nettestutil.PingTCP(t, serverIP, nonNatPort).LocalAddr().(*net.TCPAddr)
	time.Sleep(time.Second)

	lip, _ = netaddr.FromStdIP(localAddr.IP)
	trans = ct.GetTranslationForConn(
		network.ConnectionStats{
			Source: lip,
			SPort:  uint16(localAddr.Port),
			Dest:   serverIP,
			DPort:  uint16(nonNatPort),
			Type:   network.TCP,
			NetNS:  curNs,
		},
	)
	assert.Nil(t, trans)
}

func testConntrackerCrossNamespace(t *testing.T, ct netlink.Conntracker) {
	defer netlinktestutil.TeardownCrossNsDNAT(t)
	netlinktestutil.SetupCrossNsDNAT(t)

	closer := nettestutil.StartServerTCPNs(t, netaddr.MustParseIP("2.2.2.4"), 8080, "test")
	laddr := nettestutil.PingTCP(t, netaddr.MustParseIP("2.2.2.4"), 80).LocalAddr().(*net.TCPAddr)
	defer closer.Close()

	testNs, err := netns.GetFromName("test")
	require.NoError(t, err)
	defer testNs.Close()
	testIno, err := util.GetInoForNs(testNs)
	require.NoError(t, err)

	var trans *network.IPTranslation
	require.Eventually(t, func() bool {
		lip, _ := netaddr.FromStdIP(laddr.IP)
		trans = ct.GetTranslationForConn(
			network.ConnectionStats{
				Source: lip,
				SPort:  uint16(laddr.Port),
				Dest:   netaddr.MustParseIP("2.2.2.4"),
				DPort:  uint16(80),
				Type:   network.TCP,
				NetNS:  testIno,
			},
		)

		return trans != nil
	}, 5*time.Second, 1*time.Second, "timed out waiting for conntrack entry")

	assert.Equal(t, uint16(8080), trans.ReplSrcPort)
}
