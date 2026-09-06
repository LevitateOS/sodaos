package acceptance

import (
	"errors"
	"golang.org/x/sys/unix"
)

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
