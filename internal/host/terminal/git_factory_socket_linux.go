//go:build linux

package terminal

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"golang.org/x/sys/unix"
)

func (f *FactoryGitRuntime) ServeFactory(ctx context.Context, listener *net.UnixListener) error {
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
		go func() { defer workers.Done(); f.serveFactoryGit(ctx, connection) }()
	}
}

func (f *FactoryGitRuntime) serveFactoryGit(parent context.Context, connection *net.UnixConn) {
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
	invocation, err := f.StartFactory(ctx, peer, request, files[3])
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
