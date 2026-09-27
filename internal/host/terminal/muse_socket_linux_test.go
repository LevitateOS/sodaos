//go:build linux

package terminal

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	"golang.org/x/sys/unix"
)

func TestMuseSocketKernelPeerAndDescriptors(t *testing.T) {
	directory, err := os.MkdirTemp(os.TempDir(), "ms-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(directory) }()
	path := filepath.Join(directory, "launch.sock")
	listener, err := net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	client, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	server, err := listener.AcceptUnix()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	assertMuseKernelPeer(t, server)
	file, err := os.CreateTemp(t.TempDir(), "stdio")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if _, err = file.WriteString("shell bytes"); err != nil {
		t.Fatal(err)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	assertMuseDescriptors(t, client, server, file)
}

func assertMuseKernelPeer(t *testing.T, server *net.UnixConn) {
	t.Helper()
	peer, err := musePeer(server)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = unix.Close(peer.PIDFD) }()
	if peer.PID != os.Getpid() || peer.UID != uint32(os.Geteuid()) || peer.GID != uint32(os.Getegid()) || !musePeerAlive(peer) {
		t.Fatalf("wrong kernel peer: %#v", peer)
	}
}

func assertMuseDescriptors(t *testing.T, client, server *net.UnixConn, file *os.File) {
	t.Helper()
	body, err := json.Marshal(identity.LaunchRequest{CWD: "/workspace", Args: []string{"prompt with spaces"}})
	if err != nil {
		t.Fatal(err)
	}
	rights := unix.UnixRights(int(file.Fd()), int(file.Fd()), int(file.Fd()))
	if _, _, err = client.WriteMsgUnix(body, rights, nil); err != nil {
		t.Fatal(err)
	}
	request, files, err := museRequest(server)
	defer museCloseFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	if request.CWD != "/workspace" || request.Args[0] != "prompt with spaces" {
		t.Fatal("shell preferences changed")
	}
	payload, err := io.ReadAll(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != "shell bytes" {
		t.Fatal("stdin descriptor did not transfer")
	}
}

func TestMuseCommandExitSignal(t *testing.T) {
	command := exec.Command("/bin/sh", "-c", "kill -TERM $$")
	result := museCommandExit(command.Run())
	if result.Code != 143 || result.Error != "" {
		t.Fatalf("signal exit changed: %#v", result)
	}
}
