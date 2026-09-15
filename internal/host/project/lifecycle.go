package hostproject

import (
	"context"
	"errors"
	"strings"

	"github.com/levitateos/sodaos/internal/platform"
	"github.com/levitateos/sodaos/internal/project"
)

func validLifecycleAction(action string) bool {
	return action == "inspect" || action == "start" || action == "stop"
}

func parseUnitShowProperties(b []byte) (map[string]string, error) {
	if len(b) > 4096 {
		return nil, errors.New("native unit unavailable")
	}
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, errors.New("invalid native unit observation")
		}
		if _, exists := fields[key]; exists {
			return nil, errors.New("ambiguous native unit")
		}
		switch key {
		case "LoadState", "FragmentPath", "DropInPaths", "UnitFileState":
			fields[key] = value
		default:
			return nil, errors.New("unexpected unit property")
		}
	}
	return fields, nil
}

func validateUnitProperties(fields map[string]string) (bool, error) {
	dropIns := fields["DropInPaths"]
	if len(fields) != 4 || fields["LoadState"] != "loaded" || fields["FragmentPath"] != platform.ProjectUnit {
		return false, errors.New("native unit is not the selected project unit")
	}
	if dropIns != "" && dropIns != "/usr/lib/systemd/system/service.d/10-timeout-abort.conf" {
		return false, errors.New("native unit is not the selected project unit")
	}
	if fields["UnitFileState"] != "enabled" && fields["UnitFileState"] != "disabled" {
		return false, errors.New("native unit is not the selected project unit")
	}
	return fields["UnitFileState"] == "enabled", nil
}

func (r *Runtime) readProjectUnit(ctx context.Context, unit string) (bool, error) {
	b, err := r.Exec.Run(ctx, nil, "/usr/bin/systemctl", "show", unit, "--property=LoadState,FragmentPath,DropInPaths,UnitFileState")
	if err != nil {
		return false, errors.New("native unit unavailable")
	}
	fields, err := parseUnitShowProperties(b)
	if err != nil {
		return false, err
	}
	return validateUnitProperties(fields)
}

func (r *Runtime) applyLifecycleAction(ctx context.Context, action, unit string) error {
	if action == "inspect" {
		return nil
	}
	verb := "enable"
	if action == "stop" {
		verb = "disable"
	}
	// Stop also disables next-boot start. Start restores it. No persistent Soda
	// desired-state copy or direct Podman stop competing with systemd Restart.
	if _, err := r.Exec.Run(ctx, nil, "/usr/bin/systemctl", verb, "--now", unit); err != nil {
		return errors.New("native lifecycle outcome unconfirmed")
	}
	return nil
}

func verifyLifecycleOutcome(action string, result project.LifecycleState) error {
	if action == "start" && (!result.Environment.Running || !result.BootEnabled) {
		return errors.New("native lifecycle outcome unconfirmed")
	}
	if action == "stop" && (result.Environment.Running || result.BootEnabled) {
		return errors.New("native lifecycle outcome unconfirmed")
	}
	return nil
}

func (r *Runtime) Lifecycle(ctx context.Context, in project.Lifecycle) (project.LifecycleState, error) {
	var result project.LifecycleState
	if !validLifecycleAction(in.Action) {
		return result, errors.New("invalid lifecycle operation")
	}
	cid, err := r.ProjectContainer(ctx, in.Project, false)
	if err != nil {
		return result, err
	}
	unit := "soda-project@" + in.Project + ".service"
	if _, err = r.readProjectUnit(ctx, unit); err != nil {
		return result, err
	}
	if err = r.applyLifecycleAction(ctx, in.Action, unit); err != nil {
		return result, err
	}
	after, err := r.ProjectContainer(ctx, in.Project, false)
	if err != nil || after != cid {
		return result, errors.New("project identity changed during operation")
	}
	result.BootEnabled, err = r.readProjectUnit(ctx, unit)
	if err != nil {
		return result, err
	}
	result.Environment, _, err = r.Inspect(ctx, in.Project)
	if err != nil {
		return result, err
	}
	return result, verifyLifecycleOutcome(in.Action, result)
}
