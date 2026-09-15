package acceptance

import (
	"errors"

	"golang.org/x/sys/unix"
)

// A controller may be a subreaper; reap only adopted members of its killed group.
func reapOwnedChildren(pgid int) error {
	for {
		var status unix.WaitStatus
		_, err := unix.Wait4(-pgid, &status, 0, nil)
		if errors.Is(err, unix.ECHILD) {
			return nil
		}
		if err != nil && !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}

func ownedGroupsSupported() error { return nil }
func waitOwnedExit(pid int) error {
	var info unix.Siginfo
	for {
		err := unix.Waitid(unix.P_PID, pid, &info, unix.WEXITED|unix.WNOWAIT, nil)
		if !errors.Is(err, unix.EINTR) {
			return err
		}
	}
}
