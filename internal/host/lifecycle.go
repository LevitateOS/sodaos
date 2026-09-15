package host

import (
	"context"
	"errors"
)

type Lifecycle struct {
	Project string `json:"project"`
	Action  string `json:"action"`
}
type LifecycleState struct {
	Environment Environment `json:"environment"`
	BootEnabled bool        `json:"boot_enabled"`
}

func lifecycleActionConfirmed(action string, out LifecycleState) bool {
	switch action {
	case "start":
		return out.Environment.Running && out.BootEnabled
	case "stop":
		return !out.Environment.Running && !out.BootEnabled
	default:
		return true
	}
}

func lifecycleOutcomeConfirmed(in Lifecycle, out LifecycleState) bool {
	if out.Environment.ID != in.Project {
		return false
	}
	if out.Environment.IP != "" && !validAddress(out.Environment.IP) {
		return false
	}
	return lifecycleActionConfirmed(in.Action, out)
}

func (c *Client) Lifecycle(ctx context.Context, in Lifecycle) (LifecycleState, error) {
	var out LifecycleState
	err := c.call(ctx, "/lifecycle", in, &out)
	if err == nil && !lifecycleOutcomeConfirmed(in, out) {
		err = errors.New("native lifecycle outcome not confirmed")
	}
	return out, err
}
