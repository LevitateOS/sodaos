package host

import (
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

const projectAdmissionDirectory = "/var/lib/soda/project-maintenance"

// ProjectAdmissionError reports a bounded cross-process maintenance refusal.
type ProjectAdmissionError struct{ code string }

func (e *ProjectAdmissionError) Error() string { return e.code }

// Code is either project_maintenance or project_admission_unavailable.
func (e *ProjectAdmissionError) Code() string { return e.code }

// AcquireProjectCreation holds the same shared directory lock as native managed
// admission before a caller reserves any durable Project identity. It reads no
// maintenance content and never creates a directory, lock file, or marker.
func AcquireProjectCreation() (func(), error) {
	return acquireProjectCreation(projectAdmissionDirectory, 0)
}

func acquireProjectCreation(directory string, owner uint32) (func(), error) {
	unavailable := &ProjectAdmissionError{code: "project_admission_unavailable"}
	maintenance := &ProjectAdmissionError{code: "project_maintenance"}
	fd, err := unix.Open(directory, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, unavailable
	}
	file := os.NewFile(uintptr(fd), directory)
	admitted := false
	defer func() {
		if !admitted {
			_ = file.Close()
		}
	}()
	var info unix.Stat_t
	if unix.Fstat(fd, &info) != nil || info.Uid != owner || info.Mode&unix.S_IFMT != unix.S_IFDIR || info.Mode&0o7777 != 0o755 {
		return nil, unavailable
	}
	if err := unix.Flock(fd, unix.LOCK_SH|unix.LOCK_NB); err != nil {
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, maintenance
		}
		return nil, unavailable
	}
	// Any retained operation closes creation, including a Project-specific
	// interrupted repair. Names suffice: dashboard cannot read private journals.
	names, err := file.Readdirnames(129)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, unavailable
	}
	for _, name := range names {
		if strings.HasSuffix(name, ".json") {
			return nil, maintenance
		}
	}
	if len(names) == 129 {
		return nil, unavailable
	}
	admitted = true
	return func() { _ = file.Close() }, nil
}
