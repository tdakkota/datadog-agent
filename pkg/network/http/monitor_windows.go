// +build windows,npm

package http

import (
	"github.com/DataDog/datadog-agent/pkg/network/config"
)

// Monitor is responsible for:
// TODO
type Monitor struct {
	di *httpDriverInterface
}

// NewMonitor returns a new Monitor instance
func NewMonitor(c *config.Config) (*Monitor, error) {
	di, err := newDriverInterface()
	if err != nil {
		return nil, err
	}

	return &Monitor{
		di: di,
	}, nil
}

// Start consuming HTTP events
func (m *Monitor) Start() error {
	return nil
}

// GetHTTPStats returns a map of HTTP stats stored in the following format:
// [source, dest tuple, request path] -> RequestStats object
func (m *Monitor) GetHTTPStats() map[Key]RequestStats {
	return map[Key]RequestStats{}
}

// Stop HTTP monitoring
func (m *Monitor) Stop() {

}
