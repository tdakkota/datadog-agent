package http

import (
	"fmt"
	"net"

	"github.com/DataDog/datadog-agent/pkg/network/driver"
	"golang.org/x/sys/windows"
)

type httpDriverInterface struct {
	driverHTTPHandle *driver.Handle
}

func newDriverInterface() (*httpDriverInterface, error) {
	d := &httpDriverInterface{}
	err := d.setupHTTPHandle()
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (di *httpDriverInterface) setupHTTPHandle() error {
	dh, err := driver.NewHandle(0, driver.HTTPHandle) // 0 or windows.FILE_FLAG_OVERLAPPED ?
	if err != nil {
		return err
	}
	di.driverHTTPHandle = dh

	filters, err := createHTTPFilters()
	if err != nil {
		return err
	}

	if err := di.driverHTTPHandle.SetHTTPFilters(filters); err != nil {
		return err
	}

	return nil
}

func (di *httpDriverInterface) Close() error {
	if err := di.driverHTTPHandle.Close(); err != nil {
		return fmt.Errorf("error closing HTTP file handle: %w", err)
	}
	return nil
}

func createHTTPFilters() ([]driver.FilterDefinition, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var filters []driver.FilterDefinition
	for _, iface := range ifaces {
		// IPv4
		filters = append(filters, driver.FilterDefinition{
			FilterVersion:  driver.Signature,
			Size:           driver.FilterDefinitionSize,
			Direction:      driver.DirectionOutbound,
			FilterLayer:    driver.LayerTransport,
			InterfaceIndex: uint64(iface.Index),
			Af:             windows.AF_INET,
			Protocol:       windows.IPPROTO_TCP,
		}, driver.FilterDefinition{
			FilterVersion:  driver.Signature,
			Size:           driver.FilterDefinitionSize,
			Direction:      driver.DirectionInbound,
			FilterLayer:    driver.LayerTransport,
			InterfaceIndex: uint64(iface.Index),
			Af:             windows.AF_INET,
			Protocol:       windows.IPPROTO_TCP,
		})

		// IPv6
		filters = append(filters, driver.FilterDefinition{
			FilterVersion:  driver.Signature,
			Size:           driver.FilterDefinitionSize,
			Direction:      driver.DirectionOutbound,
			FilterLayer:    driver.LayerTransport,
			InterfaceIndex: uint64(iface.Index),
			Af:             windows.AF_INET6,
			Protocol:       windows.IPPROTO_TCP,
		}, driver.FilterDefinition{
			FilterVersion:  driver.Signature,
			Size:           driver.FilterDefinitionSize,
			Direction:      driver.DirectionInbound,
			FilterLayer:    driver.LayerTransport,
			InterfaceIndex: uint64(iface.Index),
			Af:             windows.AF_INET6,
			Protocol:       windows.IPPROTO_TCP,
		})
	}

	return filters, nil
}
