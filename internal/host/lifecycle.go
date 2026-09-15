package host

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

func lifecycleActionConfirmed(action string, out project.LifecycleState) bool {
	switch action {
	case "start":
		return out.Environment.Running && out.BootEnabled
	case "stop":
		return !out.Environment.Running && !out.BootEnabled
	default:
		return true
	}
}

func lifecycleOutcomeConfirmed(in project.Lifecycle, out project.LifecycleState) bool {
	if out.Environment.ID != in.Project {
		return false
	}
	if out.Environment.IP != "" && !validAddress(out.Environment.IP) {
		return false
	}
	return lifecycleActionConfirmed(in.Action, out)
}

func (c *Client) Lifecycle(ctx context.Context, in project.Lifecycle) (project.LifecycleState, error) {
	var out project.LifecycleState
	err := c.call(ctx, "/lifecycle", in, &out)
	if err == nil && !lifecycleOutcomeConfirmed(in, out) {
		err = errors.New("native lifecycle outcome not confirmed")
	}
	return out, err
}
