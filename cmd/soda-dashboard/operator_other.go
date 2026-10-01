//go:build !linux

package main

import (
	"errors"
	"net"
)

func gateOperatorPeers(net.Listener, uint32) (net.Listener, error) {
	return nil, errors.New("operator endpoint requires Linux peer credentials")
}
