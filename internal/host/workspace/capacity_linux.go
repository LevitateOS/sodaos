//go:build linux

package workspace

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// CheckCapacity reserves room for the proxy and human use before allocation.
// Cgroup and tmpfs limits remain the enforcement boundary after admission.
func (w *Runtime) CheckCapacity() error {
	if float64(w.Config.CPUs)+0.25+0.5 > float64(runtime.NumCPU()) {
		return errors.New("insufficient CPU capacity")
	}
	available, err := availableMemory()
	if err != nil {
		return err
	}
	if available < w.Config.MemoryBytes+(128<<20)+(512<<20) {
		return errors.New("insufficient memory capacity")
	}
	var storage unix.Statfs_t
	if err = unix.Statfs(w.Config.Root, &storage); err != nil {
		return err
	}
	if storage.Bavail*uint64(storage.Bsize) < 256<<20 {
		return errors.New("insufficient writable storage capacity")
	}
	return nil
}

func availableMemory() (int64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "MemAvailable:" && fields[2] == "kB" {
			value, err := strconv.ParseInt(fields[1], 10, 64)
			return value * 1024, err
		}
	}
	return 0, errors.New("native memory capacity unavailable")
}
