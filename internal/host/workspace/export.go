package workspace

import (
	"context"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// ExportCandidate captures bounded immutable bundle bytes, then terminates the
// whole worker. The publisher verifies the bundle's reported commit and ancestry.
// No filesystem tree, credential home or transcript is exported.
func (w *Runtime) ExportCandidate(ctx context.Context, r factory.Run) ([]byte, error) {
	if err := r.Authority(r.AttemptID, time.Now()); err != nil {
		return nil, err
	}
	if r.Role != factory.Implementation && r.Role != factory.Repair {
		return nil, errors.New("review cannot export implementation")
	}
	id := w.resource(r, "workspace").ID
	if !factory.ValidDigest(id) {
		return nil, errors.New("workspace is not recorded")
	}
	if _, err := w.Exec.Run(ctx, nil, "podman", "exec", "--workdir", "/workspace/repo", id, "git", "-c", "core.hooksPath=/dev/null", "bundle", "create", "/workspace/candidate.bundle", "HEAD"); err != nil {
		return nil, err
	}
	bundle, err := w.Exec.Run(ctx, nil, "podman", "exec", id, "cat", "/workspace/candidate.bundle")
	if err != nil {
		return nil, err
	}
	if len(bundle) == 0 {
		return nil, errors.New("candidate export is empty")
	}
	if _, err := w.Exec.Run(ctx, nil, "podman", "stop", "--time", "3", id); err != nil {
		return nil, err
	}
	return bundle, nil
}
