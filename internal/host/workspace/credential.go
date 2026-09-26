package workspace

import (
	"context"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/filelock"
	"golang.org/x/sys/unix"
)

// AcquireCredential serializes the enrolled stream outside the agent mount.
// The controller closes the lease only after whole-workspace termination.
func (w *Runtime) AcquireCredential(ctx context.Context) (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(w.Config.Root, "credential-stream.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err = filelock.Acquire(ctx, file, unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}
