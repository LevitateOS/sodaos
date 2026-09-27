//go:build linux

package terminal

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"golang.org/x/sys/unix"
)

// StartFactory runs under the existing broker operator, whose native resolver
// attests the caller against its current factory lease and immutable OCI labels.
type museFactoryExecution struct {
	caller     MuseFactoryCaller
	request    identity.LaunchRequest
	lease      identity.Lease
	binding    identity.Binding
	path, unit string
}

func (m *MuseRuntime) StartFactory(ctx context.Context, peer MusePeer, in identity.LaunchRequest, stdio [3]*os.File) (MuseInvocation, error) {
	var out MuseInvocation
	e, err := m.prepareFactory(ctx, peer, in)
	if err != nil {
		return out, err
	}
	out.Command = m.factoryCommand(ctx, e)
	out.Prepare = func(ctx context.Context) error { return m.deliverFactory(ctx, e) }
	out.Finish = func(ctx context.Context) error { return m.finishFactory(ctx, e) }
	out.Control = func(ctx context.Context, control identity.LaunchControl) error {
		return m.controlFactory(ctx, e, stdio[0], out.Command, control)
	}
	return out, nil
}

func (m *MuseRuntime) prepareFactory(ctx context.Context, peer MusePeer, in identity.LaunchRequest) (*museFactoryExecution, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !m.factoryReady(peer, in) {
		return nil, identity.ErrDenied
	}
	safe, err := identity.MuseArguments(in.Args)
	if err != nil {
		return nil, err
	}
	in.Args = safe
	caller, err := m.FactoryResolve(ctx, peer)
	if err != nil {
		return nil, err
	}
	if !m.validFactoryCaller(caller, peer) {
		return nil, identity.ErrDenied
	}
	if err = m.factoryMount(ctx, caller); err != nil {
		return nil, err
	}
	if err = m.verifyGuestBinary(ctx, caller.Container, ""); err != nil {
		return nil, err
	}
	return m.reserveFactory(ctx, caller, in)
}

func (m *MuseRuntime) validFactoryCaller(c MuseFactoryCaller, peer MusePeer) bool {
	if !containerID.MatchString(c.Container) || c.Actor <= 0 || !projectID.MatchString(c.Project) || c.UID <= 0 || c.HostPID <= 0 {
		return false
	}
	if peer.UID != uint32(os.Getuid()) || !filepath.IsAbs(m.FactoryRoot) {
		return false
	}
	return terminalID.MatchString(filepath.Base(c.CredentialRoot)) && filepath.Clean(c.CredentialRoot) == filepath.Join(m.FactoryRoot, filepath.Base(c.CredentialRoot))
}

func (m *MuseRuntime) reserveFactory(ctx context.Context, caller MuseFactoryCaller, in identity.LaunchRequest) (*museFactoryExecution, error) {
	selected, err := m.Select(ctx, caller.Actor, caller.Project, in.ConnectionID)
	if err != nil {
		return nil, err
	}
	id, err := museID()
	if err != nil {
		return nil, err
	}
	lease, err := m.Acquire(ctx, identity.AcquireRequest{ProviderID: identity.Muse, Kind: identity.Factory, ExecutionID: id, ActorID: caller.Actor, ProjectID: caller.Project, ConnectionID: selected, Deadline: museFactoryDeadline(caller.Deadline)})
	if err != nil {
		return nil, err
	}
	path := filepath.Join(caller.CredentialRoot, id)
	e := &museFactoryExecution{caller: caller, request: in, lease: lease, path: path, unit: "soda-muse-" + id + ".service", binding: identity.Binding{Kind: identity.Factory, ID: id, Project: caller.Container, Login: caller.Login, Generation: lease.Generation, Scope: "muse-factory", CredentialRoot: path}}
	if err = m.stageFactory(ctx, e); err != nil {
		clean, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = m.finishFactory(clean, e)
		return nil, err
	}
	return e, nil
}

func (m *MuseRuntime) stageFactory(ctx context.Context, e *museFactoryExecution) error {
	if err := os.Mkdir(e.path, 0o711); err != nil {
		return err
	}
	config := filepath.Join(e.path, "config", "muse")
	if err := os.MkdirAll(config, 0o755); err != nil {
		return err
	}
	if err := os.Symlink("../../auth.json", filepath.Join(config, "auth.json")); err != nil {
		return err
	}
	return m.copyFactoryConfig(ctx, e)
}

func (m *MuseRuntime) factoryCommand(ctx context.Context, e *museFactoryExecution) *exec.Cmd {
	args := []string{"--user", "--quiet", "--wait", "--collect", "--service-type=exec", "--unit=" + e.unit, "--property=KillMode=control-group", "--property=RuntimeMaxSec=43200", "--property=TimeoutStopSec=10", "--setenv=HOME=" + e.request.Home, "--setenv=TERM=" + e.request.Term}
	if e.request.TTY {
		args = append(args, "--pty")
	} else {
		args = append(args, "--pipe")
	}
	args = append(args, "--")
	args = append(args, museFactoryEnter(e.caller)...)
	args = append(args, "/usr/local/bin/muse", "--soda-exec", "/run/soda-muse/credentials/"+e.lease.ExecutionID, e.request.CWD)
	args = append(args, e.request.Args...)
	command := exec.CommandContext(ctx, "/usr/bin/systemd-run", args...)
	command.Env = append(museHostEnvironment(), "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/"+strconv.Itoa(os.Getuid())+"/bus")
	return command
}

func museFactoryEnter(c MuseFactoryCaller) []string {
	return []string{"/usr/bin/nsenter", "--target=" + strconv.Itoa(c.HostPID), "--user", "--mount", "--pid", "--uts", "--ipc", "--net", "--root", "--setuid=" + strconv.Itoa(c.UID), "--setgid=" + strconv.Itoa(c.GID), "--"}
}

func (m *MuseRuntime) deliverFactory(ctx context.Context, e *museFactoryExecution) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	invocation, err := m.factoryInvocation(ctx, e.unit, true)
	if err != nil {
		return err
	}
	e.binding.InvocationID = invocation
	delivery, err := m.Attach(ctx, e.lease.ID, e.binding)
	if err != nil {
		return err
	}
	defer clear(delivery.Credential)
	if !identity.CredentialValid(delivery.Credential) {
		return identity.ErrDenied
	}
	if err = os.WriteFile(filepath.Join(e.path, "auth.json"), delivery.Credential, 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.path, "ready"), nil, 0o444)
}

func (m *MuseRuntime) finishFactory(ctx context.Context, e *museFactoryExecution) error {
	if err := m.stopFactoryUnit(ctx, e.unit, e.binding.InvocationID); err != nil {
		return err
	}
	if err := os.RemoveAll(e.path); err != nil {
		return err
	}
	return m.End(ctx, e.caller.Actor, e.lease.ID)
}

func (m *MuseRuntime) controlFactory(ctx context.Context, e *museFactoryExecution, file *os.File, command *exec.Cmd, control identity.LaunchControl) error {
	if control.Signal != 0 {
		if !museSignal(control.Signal) || control.Cols != 0 || control.Rows != 0 {
			return identity.ErrDenied
		}
		_, err := m.userSystemctl(ctx, "kill", "--kill-whom=all", "--signal="+strconv.Itoa(control.Signal), e.unit)
		return err
	}
	if !e.request.TTY {
		return identity.ErrDenied
	}
	if err := museResize(file, control); err != nil {
		return err
	}
	if command.Process != nil {
		return command.Process.Signal(unix.SIGWINCH)
	}
	return nil
}

func (m *MuseRuntime) copyFactoryConfig(ctx context.Context, e *museFactoryExecution) error {
	source := e.request.ConfigHome
	if source == "" {
		return nil
	}
	command := append(museFactoryEnter(e.caller), "/usr/local/bin/muse", "--soda-read-config", source+"/muse")
	body, err := m.Exec.Run(ctx, nil, command[0], command[1:]...)
	if err != nil {
		return err
	}
	defer clear(body)
	var view map[string][]byte
	if len(body) > 3<<20 || json.Unmarshal(body, &view) != nil {
		return identity.ErrDenied
	}
	for _, name := range []string{"settings.json", "trust.json"} {
		value, ok := view[name]
		if !ok {
			continue
		}
		if err = os.WriteFile(filepath.Join(e.path, "config", "muse", name), value, 0o600); err != nil {
			return err
		}
		clear(value)
	}
	return nil
}

func (m *MuseRuntime) factoryMount(ctx context.Context, c MuseFactoryCaller) error {
	body, err := m.podman(ctx, nil, "inspect", "--format", `{{.ID}} {{.State.Pid}} {{.State.Running}}`, c.Container)
	if err != nil || strings.TrimSpace(string(body)) != c.Container+" "+strconv.Itoa(c.HostPID)+" true" {
		return identity.ErrStale
	}
	body, err = m.podman(ctx, nil, "inspect", "--format", `{{json .Mounts}}`, c.Container)
	return museReadonlyMount(body, err, c.CredentialRoot)
}

func (m *MuseRuntime) factoryInvocation(ctx context.Context, unit string, wait bool) (string, error) {
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		body, err := m.userSystemctl(ctx, "show", "--property=ActiveState", "--property=InvocationID", unit)
		if err == nil {
			if id := museActiveInvocation(body); id != "" {
				return id, nil
			}
		}
		if !wait {
			return "", identity.ErrStale
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline.C:
			return "", identity.ErrStale
		case <-tick.C:
		}
	}
}

func (m *MuseRuntime) stopFactoryUnit(ctx context.Context, unit, invocation string) error {
	if invocation != "" {
		body, err := m.userSystemctl(ctx, "show", "--property=InvocationID", "--value", unit)
		if err == nil && strings.TrimSpace(string(body)) != "" && strings.TrimSpace(string(body)) != invocation {
			return identity.ErrStale
		}
	}
	_, _ = m.userSystemctl(ctx, "stop", unit)
	body, err := m.userSystemctl(ctx, "show", "--property=ActiveState", "--value", unit)
	if err != nil || strings.TrimSpace(string(body)) != "inactive" {
		return identity.ErrUncertain
	}
	return nil
}

func (m *MuseRuntime) userSystemctl(ctx context.Context, args ...string) ([]byte, error) {
	all := []string{"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/" + strconv.Itoa(os.Getuid()) + "/bus", "/usr/bin/systemctl", "--user"}
	all = append(all, args...)
	return m.Exec.Run(ctx, nil, "/usr/bin/env", all...)
}

func (m *MuseRuntime) factoryReady(peer MusePeer, in identity.LaunchRequest) bool {
	return m.launchReady() && m.FactoryResolve != nil && in.Validate() == nil && musePeerAlive(peer)
}

func museFactoryDeadline(original time.Time) time.Time {
	limit := time.Now().Add(12 * time.Hour)
	if !original.IsZero() && original.Before(limit) {
		return original
	}
	return limit
}

func museActiveInvocation(body []byte) string {
	var active, id string
	for _, field := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		if strings.HasPrefix(field, "ActiveState=") {
			active = strings.TrimPrefix(field, "ActiveState=")
		}
		if strings.HasPrefix(field, "InvocationID=") {
			id = strings.TrimPrefix(field, "InvocationID=")
		}
	}
	if active == "active" && terminalID.MatchString(id) {
		return id
	}
	return ""
}
