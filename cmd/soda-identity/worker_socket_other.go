//go:build !linux

package main

import "net"

func listenWorkerSocket(path string) (*net.UnixListener, error) {
	return net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
}
