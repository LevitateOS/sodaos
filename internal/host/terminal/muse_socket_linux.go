//go:build linux

package terminal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/sys/unix"
)

func musePeer(conn *net.UnixConn) (MusePeer, error) {
	var p MusePeer
	raw, err := conn.SyscallConn()
	if err != nil {
		return p, err
	}
	var inner error
	err = raw.Control(func(fd uintptr) {
		credentials, e := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
		if e != nil {
			inner = e
			return
		}
		// SO_PEERPIDFD (Linux 6.5+) fails closed on an unsupported kernel.
		pidfd, e := unix.GetsockoptInt(int(fd), unix.SOL_SOCKET, 77)
		if e != nil {
			inner = e
			return
		}
		p = MusePeer{PID: int(credentials.Pid), UID: credentials.Uid, GID: credentials.Gid, PIDFD: pidfd}
	})
	if err != nil {
		return p, err
	}
	return p, inner
}

func museRequest(conn *net.UnixConn) (identity.LaunchRequest, [3]*os.File, error) {
	var request identity.LaunchRequest
	body := make([]byte, 65536)
	control := make([]byte, unix.CmsgSpace(3*4))
	n, oob, flags, _, err := conn.ReadMsgUnix(body, control)
	if err != nil {
		return request, [3]*os.File{}, err
	}
	files, err := museFiles(control[:oob], flags)
	if err != nil {
		return request, files, err
	}
	if err = strictjson.Decode(bytes.NewReader(body[:n]), &request); err != nil {
		return request, files, err
	}
	if !museDescriptorsValid(request, files) {
		return request, files, errors.New("invalid launch descriptors")
	}
	return request, files, request.Validate()
}

func museFiles(control []byte, flags int) ([3]*os.File, error) {
	var files [3]*os.File
	fds, err := museRights(control)
	if flags&(unix.MSG_TRUNC|unix.MSG_CTRUNC) != 0 || !museFDCount(len(fds)) || err != nil {
		for _, fd := range fds {
			_ = unix.Close(fd)
		}
		return files, errors.New("invalid launch descriptors")
	}
	for i, fd := range fds {
		unix.CloseOnExec(fd)
		files[i] = os.NewFile(uintptr(fd), "muse-stdio")
	}
	return files, nil
}
func museFDCount(n int) bool { return n == 0 || n == 3 }
func museRights(control []byte) ([]int, error) {
	messages, err := unix.ParseSocketControlMessage(control)
	if err != nil {
		return nil, err
	}
	var fds []int
	for _, message := range messages {
		rights, err := unix.ParseUnixRights(&message)
		if err != nil {
			return fds, err
		}
		fds = append(fds, rights...)
	}
	return fds, nil
}

func museDescriptorsValid(request identity.LaunchRequest, files [3]*os.File) bool {
	if request.Register != nil {
		return files[0] == nil
	}
	return files[0] != nil && files[1] != nil && files[2] != nil
}

func museCloseFiles(files [3]*os.File) {
	for _, f := range files {
		if f != nil {
			_ = f.Close()
		}
	}
}

// Serve runs a dedicated SOCK_SEQPACKET launch listener. Service cancellation
// cancels every live invocation; the native unit owns complete descendant cleanup.
func (s *MuseLaunch) Serve(ctx context.Context, listener *net.UnixListener) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	go func() { <-ctx.Done(); _ = listener.Close() }()
	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		wg.Add(1)
		go func() { defer wg.Done(); s.serve(ctx, conn) }()
	}
}

func (s *MuseLaunch) serve(parent context.Context, conn *net.UnixConn) {
	defer func() { _ = conn.Close() }()
	result := identity.LaunchExit{Code: 1, Error: "Muse launch denied"}
	defer func() { _ = json.NewEncoder(conn).Encode(result) }()
	peer, err := musePeer(conn)
	if err != nil {
		return
	}
	defer func() { _ = unix.Close(peer.PIDFD) }()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	request, files, err := museRequest(conn)
	defer museCloseFiles(files)
	if err != nil {
		return
	}
	if request.Register != nil {
		if s.Register != nil && s.Register(parent, peer, *request.Register) == nil {
			result = identity.LaunchExit{Code: 0}
		}
		return
	}
	_ = conn.SetReadDeadline(time.Time{})
	result = s.shell(parent, conn, peer, request, files)
}

func (s *MuseLaunch) shell(parent context.Context, conn *net.UnixConn, peer MusePeer, in identity.LaunchRequest, files [3]*os.File) (result identity.LaunchExit) {
	result = identity.LaunchExit{Code: 1, Error: "Muse launch denied"}
	if s.Start == nil {
		return result
	}
	ctx, cancel := context.WithTimeout(parent, 12*time.Hour)
	defer cancel()
	invocation, err := s.Start(ctx, peer, in, files)
	if err != nil {
		return result
	}
	if invocation.Command == nil || invocation.Finish == nil {
		return result
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		if invocation.Finish(cleanup) != nil {
			result = identity.LaunchExit{Code: 1, Error: "Muse cleanup unconfirmed"}
		}
	}()
	command := invocation.Command
	command.Stdin = files[0]
	command.Stdout = files[1]
	command.Stderr = files[2]
	if command.Start() != nil {
		return result
	}
	if invocation.Prepare != nil && invocation.Prepare(ctx) != nil {
		cancel()
		_ = command.Wait()
		return result
	}
	go museControls(ctx, cancel, conn, invocation.Control)
	return museCommandExit(command.Wait())
}

func museCommandExit(err error) identity.LaunchExit {
	result := identity.LaunchExit{Code: 0}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		result.Code = exit.ExitCode()
		if result.Code < 0 {
			result.Code = 128 + int(exit.Sys().(syscall.WaitStatus).Signal())
		}
		return result
	}
	if err != nil {
		result.Code = 1
	}
	return result
}

func museControls(ctx context.Context, cancel context.CancelFunc, conn *net.UnixConn, control func(context.Context, identity.LaunchControl) error) {
	defer cancel()
	decoder := json.NewDecoder(io.LimitReader(conn, 1<<20))
	decoder.DisallowUnknownFields()
	for {
		var in identity.LaunchControl
		if decoder.Decode(&in) != nil {
			return
		}
		if control == nil || control(ctx, in) != nil {
			return
		}
	}
}
