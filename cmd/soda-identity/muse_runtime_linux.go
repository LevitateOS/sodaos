//go:build linux

package main

import (
	"errors"
	"net"
	"os"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

func workerRuntimeAvailable(root string) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return errors.New("muse worker root must be a private directory")
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(root, &fs); err != nil {
		return err
	}
	if int64(fs.Type) != 0x01021994 {
		return errors.New("muse worker credentials require tmpfs")
	}
	conn, err := net.DialTimeout("unix", "/run/user/"+strconv.Itoa(os.Getuid())+"/bus", 2*time.Second)
	if err != nil {
		return errors.New("muse workers require the operator user systemd bus")
	}
	return conn.Close()
}
