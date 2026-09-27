//go:build linux

package terminal

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
)

// State names are derived only from the recorded execution, never caller paths.
func (m *MuseRuntime) cleanupExecutionState(ctx context.Context, b identity.Binding) error {
	container, err := stateContainer(b)
	if err != nil || container == "" {
		return err
	}
	if _, err := m.invokeState(ctx, b, "container", "exists", container); err != nil {
		if removedStateContainer(err) {
			return nil
		}
		return identity.ErrUncertain
	}
	state := "/tmp/soda-muse-state-" + b.ID
	owner := "--user=" + strconv.Itoa(b.UID) + ":" + strconv.Itoa(b.GID)
	if _, err := m.invokeState(ctx, b, "exec", owner, container, "/usr/bin/rm", "--recursive", "--force", "--", state); err != nil {
		return identity.ErrUncertain
	}
	if _, err := m.invokeState(ctx, b, "exec", owner, container, "/usr/bin/test", "!", "-e", state); err != nil {
		return identity.ErrUncertain
	}
	return nil
}

func (m *MuseRuntime) invokeState(ctx context.Context, b identity.Binding, args ...string) ([]byte, error) {
	if b.Scope == "muse-project" && b.ChildID != "" {
		return m.guest(ctx, b.Project, nil, append([]string{"/usr/bin/podman", "--remote=false"}, args...)...)
	}
	return m.podman(ctx, nil, args...)
}

func removedStateContainer(err error) bool {
	var status *exec.ExitError
	return errors.As(err, &status) && status.ExitCode() == 1
}

func (m *MuseRuntime) retireExecutionFiles(ctx context.Context, binding identity.Binding) error {
	container, path := binding.Project, binding.CredentialRoot
	var err error
	if err = m.cleanupExecutionState(ctx, binding); err != nil {
		return err
	}
	if !strings.Contains(path, "/nested/") {
		if err = m.retireMount(ctx, container, path); err != nil {
			return err
		}
	}
	if _, err = m.guest(ctx, container, nil, "/usr/bin/rm", "--recursive", "--force", "--", path); err != nil {
		return identity.ErrUncertain
	}
	return nil
}

func stateContainer(b identity.Binding) (string, error) {
	if !terminalID.MatchString(b.ID) || b.UID < 0 || b.GID < 0 {
		return "", identity.ErrDenied
	}
	container := b.Project
	if b.Scope == "muse-project" && b.ChildID != "" {
		container = b.ChildID
	}
	if !containerID.MatchString(container) {
		return "", identity.ErrDenied
	}
	return container, nil
}
