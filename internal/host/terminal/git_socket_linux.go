//go:build linux

package terminal

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/sys/unix"
)

// Serve admits native Git helper sessions on a dedicated Unix seqpacket socket.
func (g *GitRuntime) Serve(ctx context.Context, listener *net.UnixListener) error {
	var workers sync.WaitGroup
	defer workers.Wait()
	go func() { <-ctx.Done(); _ = listener.Close() }()
	for {
		connection, err := listener.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		workers.Add(1)
		go func() { defer workers.Done(); g.serveGit(ctx, connection) }()
	}
}

func (g *GitRuntime) serveGit(parent context.Context, connection *net.UnixConn) {
	defer connection.Close()
	result := identity.LaunchExit{Code: 1, Error: "Git launch denied"}
	defer func() { _ = json.NewEncoder(connection).Encode(result) }()
	peer, err := musePeer(connection)
	if err != nil {
		return
	}
	defer unix.Close(peer.PIDFD)
	_ = connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	request, files, err := gitRequest(connection)
	defer func() {
		for _, file := range files {
			if file != nil {
				_ = file.Close()
			}
		}
	}()
	if err != nil {
		return
	}
	_ = connection.SetReadDeadline(time.Time{})
	ctx, cancel := context.WithTimeout(parent, 12*time.Hour)
	defer cancel()
	invocation, err := g.Start(ctx, peer, request, files[3])
	if err != nil {
		return
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		if invocation.Finish(cleanup) != nil {
			result = identity.LaunchExit{Code: 1, Error: "Git cleanup unconfirmed"}
		}
	}()
	command := invocation.Command
	command.Stdin = files[0]
	command.Stdout = files[1]
	command.Stderr = files[2]
	if command.Start() != nil {
		return
	}
	if invocation.Prepare(ctx) != nil {
		cancel()
		_ = command.Wait()
		return
	}
	go museControls(ctx, cancel, connection, invocation.Control)
	result = museCommandExit(command.Wait())
}

func gitRequest(connection *net.UnixConn) (identity.GitLaunchRequest, [4]*os.File, error) {
	var request identity.GitLaunchRequest
	var files [4]*os.File
	body := make([]byte, 8192)
	control := make([]byte, unix.CmsgSpace(16))
	n, oob, flags, _, err := connection.ReadMsgUnix(body, control)
	if err != nil {
		return request, files, err
	}
	descriptors, err := museRights(control[:oob])
	if err != nil || len(descriptors) != 4 || flags&(unix.MSG_TRUNC|unix.MSG_CTRUNC) != 0 {
		for _, fd := range descriptors {
			_ = unix.Close(fd)
		}
		return request, files, identity.ErrDenied
	}
	for i, fd := range descriptors {
		unix.CloseOnExec(fd)
		files[i] = os.NewFile(uintptr(fd), "git-launch")
	}
	if err = strictjson.Decode(bytes.NewReader(body[:n]), &request); err != nil {
		return request, files, err
	}
	return request, files, request.Validate()
}
