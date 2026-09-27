//go:build linux

// soda-muse delegates a normal shell invocation over the launch-only socket.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"golang.org/x/sys/unix"
)

func main() {
	code, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(code)
}

func run() (int, error) {
	if handled, code, err := nativeAction(os.Args[1:]); handled {
		return code, err
	}
	request, err := shellRequest()
	if err != nil {
		return 1, err
	}
	conn, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{Name: identity.MuseLaunchSocket, Net: "unixpacket"})
	if err != nil {
		return 1, errors.New("muse launch service unavailable")
	}
	defer func() { _ = conn.Close() }()
	return launchShell(conn, request)
}

func nativeAction(args []string) (bool, int, error) {
	if len(args) == 0 {
		return false, 0, nil
	}
	switch args[0] {
	case "--soda-copy-config":
		if len(args) != 3 {
			return true, 1, errors.New("config source and view required")
		}
		return true, 0, copyConfig(args[1], args[2])
	case "--soda-exec":
		if len(args) < 3 {
			return true, 1, errors.New("execution root and cwd required")
		}
		code, err := execute(args[1], args[2], args[3:])
		return true, code, err
	default:
		return metadataAction(args)
	}
}

func metadataAction(args []string) (bool, int, error) {
	switch args[0] {
	case "--soda-check", "--soda-read-config", "--soda-account":
	default:
		return false, 0, nil
	}
	if len(args) != 2 {
		return true, 1, errors.New("one metadata input required")
	}
	switch args[0] {
	case "--soda-check":
		return true, 0, checkRuntime(args[1])
	case "--soda-read-config":
		return true, 0, readConfig(args[1])
	default:
		return true, 0, accountFor(args[1])
	}
}

func shellRequest() (identity.LaunchRequest, error) {
	var request identity.LaunchRequest
	cwd, err := os.Getwd()
	if err != nil {
		return request, err
	}
	request = identity.LaunchRequest{CWD: cwd, Args: os.Args[1:], Home: os.Getenv("HOME"), ConnectionID: os.Getenv("SODA_MUSE_CONNECTION"), ConfigHome: os.Getenv("XDG_CONFIG_HOME"), Term: os.Getenv("TERM")}
	if request.ConfigHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return request, err
		}
		request.ConfigHome = filepath.Join(home, ".config")
	}
	if size, err := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ); err == nil {
		request.TTY = true
		request.Cols = size.Col
		request.Rows = size.Row
	}
	return request, request.Validate()
}

func launchShell(conn *net.UnixConn, request identity.LaunchRequest) (int, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return 1, err
	}
	rights := unix.UnixRights(int(os.Stdin.Fd()), int(os.Stdout.Fd()), int(os.Stderr.Fd()))
	if _, _, err = conn.WriteMsgUnix(body, rights, nil); err != nil {
		return 1, errors.New("muse launch failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go controls(ctx, conn, request.TTY)
	var result identity.LaunchExit
	if err = json.NewDecoder(conn).Decode(&result); err != nil {
		return 1, errors.New("muse launch service ended")
	}
	if result.Error != "" {
		return result.Code, errors.New(result.Error)
	}
	return result.Code, nil
}

func controls(ctx context.Context, conn *net.UnixConn, tty bool) {
	signals := make(chan os.Signal, 16)
	signal.Notify(signals, syscall.SIGWINCH, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTSTP, syscall.SIGCONT, syscall.SIGUSR1, syscall.SIGUSR2)
	defer signal.Stop(signals)
	encoder := json.NewEncoder(conn)
	for {
		select {
		case <-ctx.Done():
			return
		case received := <-signals:
			sig := received.(syscall.Signal)
			control := identity.LaunchControl{Signal: int(sig)}
			if sig == syscall.SIGWINCH {
				if !tty {
					continue
				}
				size, err := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ)
				if err != nil {
					continue
				}
				control = identity.LaunchControl{Cols: size.Col, Rows: size.Row}
			}
			if encoder.Encode(control) != nil {
				return
			}
		}
	}
}

// execute is fixed product code started by the native supervisor.
func execute(root, cwd string, args []string) (int, error) {
	if err := os.Chdir(cwd); err != nil {
		return 1, err
	}
	root = filepath.Clean(root)
	if !strings.HasPrefix(root, "/run/soda-muse/") || strings.Contains(root, "..") {
		return 1, errors.New("invalid Muse execution root")
	}
	if err := awaitAdmission(root); err != nil {
		return 1, err
	}
	state, err := executionState(root)
	if err != nil {
		return 1, err
	}
	env := museEnvironment(root, state)
	err = unix.Exec("/usr/local/libexec/soda/muse", append([]string{"muse"}, args...), env)
	return 1, fmt.Errorf("muse execution failed: %w", err)
}

func awaitAdmission(root string) error {
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		if info, err := os.Stat(root + "/ready"); err == nil && info.Mode().IsRegular() {
			return nil
		}
		select {
		case <-timeout.C:
			return errors.New("muse admission did not complete")
		case <-tick.C:
		}
	}
}

func museEnvironment(root, state string) []string {
	env := []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8", "TBH_CREDENTIAL_BACKEND=file", "XDG_CONFIG_HOME=" + root + "/config", "XDG_STATE_HOME=" + state + "/state", "XDG_CACHE_HOME=" + state + "/cache", "XDG_DATA_HOME=" + state + "/data", "TMPDIR=" + state + "/tmp"}
	for _, name := range []string{"HOME", "USER", "LOGNAME", "TERM", "COLORTERM"} {
		if value := os.Getenv(name); value != "" {
			env = append(env, name+"="+value)
		}
	}
	return env
}
