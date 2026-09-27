//go:build linux

package main

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestWorkerSocketHasOnlyPublicContainerSID(t *testing.T) {
	if _, err := os.Stat("/sys/fs/selinux/enforce"); err != nil {
		t.Skip("SELinux is unavailable")
	}
	root, err := os.MkdirTemp(os.TempDir(), "ws-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	public, err := listenWorkerSocket(filepath.Join(root, "public.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer public.Close()
	private, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: filepath.Join(root, "private.sock"), Net: "unixpacket"})
	if err != nil {
		t.Fatal(err)
	}
	defer private.Close()
	if got := peerSocketContext(t, public.Addr().String()); !strings.Contains(got, ":container_t:") {
		t.Fatalf("public worker socket has wrong SID: %s", got)
	}
	if got := peerSocketContext(t, private.Addr().String()); strings.Contains(got, ":container_t:") {
		t.Fatalf("private service socket inherited worker SID: %s", got)
	}
}

func peerSocketContext(t *testing.T, path string) string {
	t.Helper()
	conn, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	raw, err := conn.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	var context string
	var socketErr error
	if err := raw.Control(func(fd uintptr) {
		context, socketErr = unix.GetsockoptString(int(fd), unix.SOL_SOCKET, unix.SO_PEERSEC)
	}); err != nil || socketErr != nil {
		t.Fatal(err, socketErr)
	}
	return context
}
