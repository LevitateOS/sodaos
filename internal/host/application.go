package host

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/levitateos/sodaos/internal/application"
)

type applicationResponse struct{ application.Observation }

func (r *applicationResponse) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	keys := []string{"id", "revision", "container", "image", "running", "ready", "origin"}
	if _, ok := fields["bundle_file_sha256"]; ok {
		keys = append(keys, "bundle_file_sha256")
	}
	if len(fields) != len(keys) {
		return fmt.Errorf("invalid native application observation shape")
	}
	for _, key := range keys {
		if len(fields[key]) == 0 || string(fields[key]) == "null" {
			return fmt.Errorf("missing native application observation field")
		}
	}
	return json.Unmarshal(data, &r.Observation)
}

// Application invokes the admitted native operation and verifies its confirmed
// references and generation. Any transport or observation failure is uncertain.
func (c *Client) Application(ctx context.Context, in application.Request) (application.Observation, error) {
	var out application.Observation
	if err := in.Validate(); err != nil {
		return out, err
	}
	var response applicationResponse
	if err := c.callLimit(ctx, "/application", in, &response, 4096); err != nil {
		return out, err
	}
	out = response.Observation
	if err := out.Validate(in.Application); err != nil {
		return out, err
	}
	if !applicationGenerationConfirmed(in, out) || !applicationActionConfirmed(in.Action, out) {
		return application.Observation{}, fmt.Errorf("native application result did not confirm the requested operation")
	}
	return out, nil
}

func applicationGenerationConfirmed(in application.Request, out application.Observation) bool {
	if out.Container == "" {
		return in.Action == "inspect" && in.ExpectedContainer == "" && !out.Running && !out.Ready
	}
	if in.Action == "install" {
		return true
	}
	if in.Action == "replace" || in.Action == "restore" {
		return out.Container != in.ExpectedContainer
	}
	return in.ExpectedContainer == "" || out.Container == in.ExpectedContainer
}

func applicationActionConfirmed(action string, out application.Observation) bool {
	switch action {
	case "install", "stop", "replace", "restore", "restore_abort":
		return !out.Running && !out.Ready
	case "start":
		return out.Running
	default:
		return true
	}
}
