package workspace

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// ExportCandidate captures bounded immutable bundle bytes. Credential return
// subsequently freezes and terminates the whole worker before publication. The publisher verifies the bundle's reported commit and ancestry.
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
	return bundle, nil
}

func (w *Runtime) Revision(ctx context.Context, r factory.Run) (string, error) {
	id := w.resource(r, "workspace").ID
	out, err := w.Exec.Run(ctx, nil, "podman", "exec", "--workdir", "/workspace/repo", id, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	sha := strings.TrimSpace(string(out))
	if !factory.ValidCommit(sha) {
		return "", errors.New("workspace HEAD is invalid")
	}
	return sha, nil
}
