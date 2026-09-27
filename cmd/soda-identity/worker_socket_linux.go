//go:build linux

package main

import (
	"errors"
	"net"
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

const (
	workerSocketContext   = "system_u:system_r:container_t:s0"
	socketCreateAttribute = "/proc/thread-self/attr/sockcreate"
)

type workerSocketResult struct {
	listener *net.UnixListener
	err      error
}

// listenWorkerSocket gives only the public launch socket a container_t SID.
// The service keeps its own process label and private sockets unchanged.
func listenWorkerSocket(path string) (*net.UnixListener, error) {
	result := make(chan workerSocketResult, 1)
	go func() {
		runtime.LockOSThread()
		listener, safe, err := createWorkerSocket(path)
		if safe {
			runtime.UnlockOSThread()
		}
		// An unsafe result exits while locked and destroys this OS thread.
		result <- workerSocketResult{listener: listener, err: err}
	}()
	out := <-result
	return out.listener, out.err
}

func createWorkerSocket(path string) (*net.UnixListener, bool, error) {
	before, err := os.ReadFile(socketCreateAttribute)
	if err != nil || len(before) != 0 {
		return nil, true, errors.New("worker socket creation context unavailable")
	}
	fd, err := unix.Open(socketCreateAttribute, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, true, err
	}
	defer unix.Close(fd)
	if _, err := unix.Write(fd, []byte(workerSocketContext+"\x00")); err != nil {
		return nil, false, err
	}
	listener, listenErr := net.ListenUnix("unixpacket", &net.UnixAddr{Name: path, Net: "unixpacket"})
	if err := restoreWorkerSocketContext(fd); err != nil {
		if listener != nil {
			_ = listener.Close()
		}
		return nil, false, err
	}
	return listener, true, listenErr
}

func restoreWorkerSocketContext(fd int) error {
	_, err := unix.Write(fd, nil)
	after, readErr := os.ReadFile(socketCreateAttribute)
	if err != nil || readErr != nil || len(after) != 0 {
		return errors.New("worker socket creation context was not restored")
	}
	return nil
}
