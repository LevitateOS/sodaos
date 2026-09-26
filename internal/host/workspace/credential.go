package workspace

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

// SeedCredential delivers only authentication bytes over stdin. The protected
// host credential directory is never mounted into the workspace.
func (w *Runtime) SeedCredential(ctx context.Context, r factory.Run, state []byte) error {
	if len(state) == 0 || len(state) > 256<<10 {
		return errors.New("invalid credential state size")
	}
	id := w.resource(r, "workspace").ID
	_, err := w.Exec.Run(ctx, state, "podman", "exec", "--interactive", id, "sh", "-c", "umask 077; cat > /run/codex/auth.json")
	return err
}

// CaptureCredential reads a frozen tmpfs through the verified container PID's
// namespace. A SIGKILL terminates all processes before the caller saves state.
func (w *Runtime) CaptureCredential(ctx context.Context, r factory.Run) ([]byte, error) {
	resource := w.resource(r, "workspace")
	id, exists, err := w.observeResource(ctx, r, resource)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("credential workspace ended before state return")
	}
	if _, err = w.Exec.Run(ctx, nil, "podman", "pause", id); err != nil {
		return nil, err
	}
	state, err := w.readCredential(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err = w.Exec.Run(ctx, nil, "podman", "kill", "--signal=KILL", id); err != nil {
		return nil, err
	}
	return state, nil
}

func (w *Runtime) readCredential(ctx context.Context, id string) ([]byte, error) {
	out, err := w.Exec.Run(ctx, nil, "podman", "inspect", "--format", "{{.State.Pid}}", id)
	if err != nil {
		return nil, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || pid <= 0 {
		return nil, errors.New("invalid credential workspace PID")
	}
	path := fmt.Sprintf("/proc/%d/root/run/codex/auth.json", pid)
	script := `test -f "$1" && ! test -L "$1" && head -c 262145 -- "$1"`
	state, err := w.Exec.Run(ctx, nil, "podman", "unshare", "sh", "-c", script, "capture-credential", path)
	if err != nil {
		return nil, err
	}
	if len(state) == 0 || len(state) > 256<<10 {
		return nil, errors.New("credential return exceeds state limit")
	}
	return state, nil
}
