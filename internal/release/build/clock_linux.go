package build

import (
	"golang.org/x/sys/unix"
	"time"
)

func monotonic() time.Duration {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		panic(err)
	}
	return time.Duration(ts.Nano())
}
