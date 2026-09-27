//go:build linux

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
)

var containerID = regexp.MustCompile(`^[0-9a-f]{64}$`)

type observation struct {
	ID      string `json:"id"`
	Project string `json:"project"`
	Owner   string `json:"owner"`
	PID     int    `json:"pid"`
	Running bool   `json:"running"`
}

func podman(ctx context.Context, input io.Reader, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "podman", append([]string{"--remote=false"}, args...)...)
	command.Stdin = input
	out, err := command.Output()
	if err != nil {
		return nil, errors.New("project maintenance command failed")
	}
	return out, nil
}

func inspectProject(ctx context.Context, key, project string) (observation, error) {
	var o observation
	body, err := podman(ctx, nil, "inspect", "--format", `{"id":{{json .ID}},"project":{{json (index .Config.Labels "org.soda.project")}},"owner":{{json (index .Config.Labels "org.soda.owner")}},"pid":{{json .State.Pid}},"running":{{json .State.Running}}}`, key)
	if err != nil {
		return o, err
	}
	if len(body) > 8192 || strictjson.Decode(bytes.NewReader(body), &o) != nil {
		return o, errors.New("invalid project observation")
	}
	if err := o.validate(project); err != nil {
		return o, err
	}
	return o, nil
}

func (o observation) validate(project string) error {
	if !containerID.MatchString(o.ID) || o.Project != project {
		return errors.New("project container identity differs")
	}
	owner, err := strconv.ParseInt(o.Owner, 10, 64)
	if err != nil || owner <= 0 {
		return errors.New("project owner label is invalid")
	}
	return nil
}

func waitProject(ctx context.Context, project string) (observation, error) {
	bounded, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		o, err := inspectProject(bounded, "soda-"+project, project)
		if err != nil {
			return o, err
		}
		if o.Running && o.PID > 0 {
			return o, nil
		}
		select {
		case <-bounded.Done():
			return o, errors.New("project did not become running within maintenance deadline")
		case <-tick.C:
		}
	}
}

func confirmProject(ctx context.Context, o observation) error {
	current, err := inspectProject(ctx, o.ID, o.Project)
	if err != nil {
		return err
	}
	if current != o || !current.Running {
		return errors.New("project incarnation changed during maintenance")
	}
	return nil
}
