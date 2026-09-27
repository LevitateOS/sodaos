//go:build linux

package main

import (
	"crypto/rand"
	"encoding/json"
	"net"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	"golang.org/x/sys/unix"
)

func TestLaunchTransfersListenerAndPreservesExit(t *testing.T) {
	address := &net.UnixAddr{Name: "@soda-git-test-" + rand.Text(), Net: "unixpacket"}
	server, err := net.ListenUnix("unixpacket", address)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	file, err := listener.File()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	done := make(chan error, 1)
	go func() {
		conn, err := server.AcceptUnix()
		if err != nil {
			done <- err
			return
		}
		defer func() { _ = conn.Close() }()
		done <- receiveLaunch(conn)
	}()
	conn, err := net.DialUnix("unixpacket", nil, address)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	code, err := launch(conn, identity.GitLaunchRequest{CWD: "/home/soda-tester/work", Remote: "origin", Owner: "soda-tester", Repository: "repo"}, file)
	if err != nil || code != 17 {
		t.Fatal("helper exit changed", code, err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func receiveLaunch(conn *net.UnixConn) error {
	body, control := make([]byte, 4096), make([]byte, unix.CmsgSpace(16))
	n, oob, flags, _, err := conn.ReadMsgUnix(body, control)
	if err != nil {
		return err
	}
	if flags&(unix.MSG_TRUNC|unix.MSG_CTRUNC) != 0 {
		return identity.ErrDenied
	}
	messages, err := unix.ParseSocketControlMessage(control[:oob])
	if err != nil || len(messages) != 1 {
		return identity.ErrDenied
	}
	fds, err := unix.ParseUnixRights(&messages[0])
	if err != nil {
		return err
	}
	defer func() {
		for _, fd := range fds {
			_ = unix.Close(fd)
		}
	}()
	if len(fds) != 4 {
		return identity.ErrDenied
	}
	accepting, err := unix.GetsockoptInt(fds[3], unix.SOL_SOCKET, unix.SO_ACCEPTCONN)
	if err != nil || accepting != 1 {
		return identity.ErrDenied
	}
	var request identity.GitLaunchRequest
	if json.Unmarshal(body[:n], &request) != nil || request.Validate() != nil {
		return identity.ErrDenied
	}
	return json.NewEncoder(conn).Encode(identity.LaunchExit{Code: 17})
}
