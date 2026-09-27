//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

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
	cwd, err := os.Getwd()
	if err != nil {
		return 1, errors.New("git working directory unavailable")
	}
	request, err := remoteRequest(os.Args[1:], cwd, os.Getenv("SODA_GIT_CONNECTION"))
	if err != nil {
		return 1, errors.New("invalid Soda Git remote")
	}
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return 1, errors.New("git listener unavailable")
	}
	defer func() { _ = listener.Close() }()
	file, err := listener.File()
	if err != nil {
		return 1, errors.New("git listener unavailable")
	}
	defer func() { _ = file.Close() }()
	conn, err := net.DialUnix("unixpacket", nil, &net.UnixAddr{Name: identity.GitLaunchSocket, Net: "unixpacket"})
	if err != nil {
		return 1, errors.New("git launch service unavailable")
	}
	defer func() { _ = conn.Close() }()
	return launch(conn, request, file)
}

func launch(conn *net.UnixConn, request identity.GitLaunchRequest, listener *os.File) (int, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return 1, err
	}
	rights := unix.UnixRights(int(os.Stdin.Fd()), int(os.Stdout.Fd()), int(os.Stderr.Fd()), int(listener.Fd()))
	if _, _, err = conn.WriteMsgUnix(body, rights, nil); err != nil {
		return 1, errors.New("git launch failed")
	}
	stop := make(chan struct{})
	defer close(stop)
	go controls(conn, stop)
	var result identity.LaunchExit
	if err := json.NewDecoder(conn).Decode(&result); err != nil {
		return 1, errors.New("git launch service ended")
	}
	if result.Code < 0 || result.Code > 255 {
		return 1, errors.New("invalid Git exit status")
	}
	if result.Error != "" {
		return result.Code, errors.New(result.Error)
	}
	return result.Code, nil
}

func controls(conn *net.UnixConn, stop <-chan struct{}) {
	signals := make(chan os.Signal, 8)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Stop(signals)
	encoder := json.NewEncoder(conn)
	for {
		select {
		case <-stop:
			return
		case received := <-signals:
			if encoder.Encode(identity.LaunchControl{Signal: int(received.(syscall.Signal))}) != nil {
				return
			}
		}
	}
}
