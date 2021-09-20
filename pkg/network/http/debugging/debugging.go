package debugging

import (
	"github.com/DataDog/datadog-agent/pkg/network/http"
	"github.com/DataDog/sketches-go/ddsketch"
	"inet.af/netaddr"
)

// RequestSummary represents a (debug-friendly) aggregated view of requests
// matching a (client, server, path, method) tuple
type RequestSummary struct {
	Client   Address
	Server   Address
	DNS      string
	Path     string
	Method   string
	ByStatus map[int]Stats
}

// Address represents represents a IP:Port
type Address struct {
	IP   string
	Port uint16
}

// Stats consolidates request count and latency information for a certain status code
type Stats struct {
	Count              int
	FirstLatencySample float64
	LatencyP50         float64
}

// HTTP returns a debug-friendly representation of map[http.Key]http.RequestStats
func HTTP(stats map[http.Key]http.RequestStats, dns map[netaddr.IP][]string) []RequestSummary {
	all := make([]RequestSummary, 0, len(stats))
	for k, v := range stats {
		debug := RequestSummary{
			Client: Address{
				IP:   k.SrcIP.String(),
				Port: k.SrcPort,
			},
			Server: Address{
				IP:   k.DstIP.String(),
				Port: k.DstPort,
			},
			DNS:      getDNS(dns, k.DstIP),
			Path:     k.Path,
			Method:   k.Method.String(),
			ByStatus: make(map[int]Stats),
		}

		for i, stat := range v {
			if stat.Count == 0 {
				continue
			}

			status := (i + 1) * 100
			debug.ByStatus[status] = Stats{
				Count:              stat.Count,
				FirstLatencySample: stat.FirstLatencySample,
				LatencyP50:         getSketchQuantile(stat.Latencies, 0.5),
			}
		}

		all = append(all, debug)
	}

	return all
}

func getDNS(dns map[netaddr.IP][]string, addr netaddr.IP) string {
	if names := dns[addr]; len(names) > 0 {
		return names[0]
	}

	return ""
}

func getSketchQuantile(sketch *ddsketch.DDSketch, percentile float64) float64 {
	if sketch == nil {
		return 0.0
	}

	val, _ := sketch.GetValueAtQuantile(percentile)
	return val
}
