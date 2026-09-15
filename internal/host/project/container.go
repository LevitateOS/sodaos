package project

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"

	domain "github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/strictjson"
)

const projectInspect = `{"id":{{json .ID}},"running":{{json .State.Running}},"project":{{json (index .Config.Labels "org.soda.project")}},"owner":{{json (index .Config.Labels "org.soda.owner")}},"privileged":{{json .HostConfig.Privileged}},"userns":{{json .HostConfig.UsernsMode}},"mappings":{{json .HostConfig.IDMappings}}}`

type projectInspection struct {
	ID         string `json:"id"`
	Running    bool   `json:"running"`
	Project    string `json:"project"`
	Owner      string `json:"owner"`
	Privileged bool   `json:"privileged"`
	Userns     string `json:"userns"`
	Mappings   struct {
		UIDMap []string `json:"UidMap"`
		GIDMap []string `json:"GidMap"`
	} `json:"mappings"`
}

func (r *Runtime) TerminalContainer(ctx context.Context, id string) (string, error) {
	return r.ProjectContainer(ctx, id, true)
}

// Native lifecycle may inspect stopped containers, never missing/replacement ones.
func projectIsolation(v projectInspection, id string) bool {
	if !domain.ValidContainerID(v.ID) || v.Project != id || v.Privileged || v.Userns != "private" {
		return false
	}
	return projectIDMap(v.Mappings.UIDMap) && projectIDMap(v.Mappings.GIDMap)
}

func projectTargetReady(v projectInspection, id string, requireRunning bool) bool {
	owner, err := strconv.ParseInt(v.Owner, 10, 64)
	if err != nil || owner <= 0 {
		return false
	}
	if requireRunning && !v.Running {
		return false
	}
	return projectIsolation(v, id)
}

func (r *Runtime) ProjectContainer(ctx context.Context, id string, requireRunning bool) (string, error) {
	if !domain.ValidID(id) {
		return "", errors.New("invalid project")
	}
	data, err := r.podman(ctx, nil, "--remote=false", "inspect", "--format", projectInspect, "soda-"+id)
	if err != nil || len(data) > 4096 {
		return "", errors.New("terminal inspection unavailable")
	}
	var v projectInspection
	if err = strictjson.Decode(bytes.NewReader(data), &v); err != nil {
		return "", errors.New("invalid terminal inspection")
	}
	if !projectTargetReady(v, id, requireRunning) {
		return "", errors.New("terminal target not ready or isolated")
	}
	return v.ID, nil
}

// Podman reports the resulting namespace as "private", not its create-time
// "auto" selector. Require the production candidate's single 262144-ID mapping
// with container root shifted away from host root; do not infer isolation from
// the mode string alone or introduce identity-remapping policy.
func projectIDMap(values []string) bool {
	if len(values) != 1 {
		return false
	}
	parts := strings.Split(values[0], ":")
	if len(parts) != 3 || parts[0] != "0" || parts[2] != "262144" {
		return false
	}
	base, err := strconv.ParseUint(parts[1], 10, 32)
	return err == nil && strconv.FormatUint(base, 10) == parts[1] && base > 0 && base+262144 <= 4294967295
}
