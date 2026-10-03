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
	// SyscallConn reads the fd without disturbing the connection: File()
	// would flip it to blocking mode, after which the HTTP server's
	// background read becomes uninterruptible and Close deadlocks.
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return false
	}
	var peer uint32
	var syscallErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, err := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if err != nil {
			syscallErr = err
			return
		}
		peer = credentials.Uid
	}); err != nil {
		return false
	}
	return syscallErr == nil && peer == uid
}
