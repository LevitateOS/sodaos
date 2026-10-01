//go:build linux

package main

import (
	"net"

	"golang.org/x/sys/unix"
)

type operatorPeerListener struct {
	net.Listener
	uid uint32
}

func gateOperatorPeers(listener net.Listener, uid uint32) (net.Listener, error) {
	return &operatorPeerListener{Listener: listener, uid: uid}, nil
}

func (l *operatorPeerListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		if operatorPeerOK(conn, l.uid) {
			return conn, nil
		}
		_ = conn.Close()
	}
}

func operatorPeerOK(conn net.Conn, uid uint32) bool {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return false
	}
	file, err := unixConn.File()
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	credentials, err := unix.GetsockoptUcred(int(file.Fd()), unix.SOL_SOCKET, unix.SO_PEERCRED)
	return err == nil && credentials.Uid == uid
}
