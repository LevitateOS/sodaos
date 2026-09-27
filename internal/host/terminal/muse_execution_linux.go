//go:build linux

package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"golang.org/x/sys/unix"
)

type museExecution struct {
	caller     museCaller
	request    identity.LaunchRequest
	lease      identity.Lease
	binding    identity.Binding
	path, unit string
}

// Start creates a credential-free native boundary before delivery.
func (m *MuseRuntime) Start(ctx context.Context, peer MusePeer, in identity.LaunchRequest, stdio [3]*os.File) (MuseInvocation, error) {
	var out MuseInvocation
	execution, err := m.prepareExecution(ctx, peer, in)
	if err != nil {
		return out, err
	}
	out.Command = museCommand(ctx, execution.caller, execution.request, execution.unit, execution.path)
	out.Prepare = func(ctx context.Context) error { return m.deliverExecution(ctx, execution) }
	out.Finish = func(ctx context.Context) error {
		return m.stopExecution(ctx, execution.binding, execution.caller.Actor, execution.lease.ID, true)
	}
	out.Control = func(ctx context.Context, control identity.LaunchControl) error {
		return m.controlExecution(ctx, execution, stdio[0], out.Command, control)
	}
	return out, nil
}

func (m *MuseRuntime) launchReady() bool {
	return m.Acquire != nil && m.Attach != nil && m.End != nil && m.Select != nil
}

func (m *MuseRuntime) prepareExecution(ctx context.Context, peer MusePeer, in identity.LaunchRequest) (*museExecution, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !m.launchReady() || in.Validate() != nil {
		return nil, identity.ErrDenied
	}
	safe, err := identity.MuseArguments(in.Args)
	if err != nil {
		return nil, err
	}
	in.Args = safe
	caller, err := m.resolve(ctx, peer, identity.Muse)
	if err != nil {
		return nil, err
	}
	if err = m.verifyGuestBinary(ctx, caller.Container, caller.Child); err != nil {
		return nil, err
	}
	execution, err := m.reserveExecution(ctx, caller, in)
	if err != nil {
		return nil, err
	}
	if !musePeerAlive(peer) {
		clean, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = m.End(clean, caller.Actor, execution.lease.ID)
		return nil, identity.ErrDenied
	}
	if err = m.stage(ctx, caller, execution.path, in.ConfigHome); err != nil {
		clean, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = m.stopExecution(clean, execution.binding, caller.Actor, execution.lease.ID, true)
		return nil, err
	}
	return execution, nil
}

func (m *MuseRuntime) reserveExecution(ctx context.Context, caller museCaller, in identity.LaunchRequest) (*museExecution, error) {
	connection, err := m.Select(ctx, caller.Actor, caller.Project, in.ConnectionID)
	if err != nil {
		return nil, err
	}
	id, err := museID()
	if err != nil {
		return nil, err
	}
	lease, err := m.Acquire(ctx, identity.AcquireRequest{ExecutionID: id, ActorID: caller.Actor, ConnectionID: connection, ProjectID: caller.Project, Kind: identity.Terminal, ProviderID: identity.Muse, Deadline: time.Now().Add(12 * time.Hour)})
	if err != nil {
		return nil, err
	}
	path := "/run/soda-muse/" + id
	if caller.Child != "" {
		path = "/run/soda-muse/nested/" + caller.Registration + "/" + id
	}
	binding := identity.Binding{Kind: identity.Terminal, ID: id, Project: caller.Container, ChildID: caller.Child, UID: caller.UID, GID: caller.GID, Login: caller.Login, Generation: lease.Generation, Scope: "muse-project", CredentialRoot: path}
	return &museExecution{caller: caller, request: in, lease: lease, binding: binding, path: path, unit: "soda-muse-" + id + ".service"}, nil
}

func (m *MuseRuntime) deliverExecution(ctx context.Context, e *museExecution) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := m.awaitUnit(ctx, e.caller.Container, e.unit); err != nil {
		return err
	}
	body, err := m.guest(ctx, e.caller.Container, nil, "/usr/bin/systemctl", "show", "--property=InvocationID", "--value", e.unit)
	if err != nil {
		return err
	}
	e.binding.InvocationID = strings.TrimSpace(string(body))
	if !terminalID.MatchString(e.binding.InvocationID) {
		return identity.ErrDenied
	}
	delivery, err := m.Attach(ctx, e.lease.ID, e.binding)
	if err != nil {
		return err
	}
	defer clear(delivery.Credential)
	if !identity.CredentialValid(delivery.Credential) {
		return identity.ErrDenied
	}
	if _, err = m.guest(ctx, e.caller.Container, delivery.Credential, "/usr/bin/dd", "of="+e.path+"/auth.json", "status=none"); err != nil {
		return err
	}
	_, err = m.guest(ctx, e.caller.Container, nil, "/usr/bin/touch", e.path+"/ready")
	return err
}

func (m *MuseRuntime) controlExecution(ctx context.Context, e *museExecution, file *os.File, command *exec.Cmd, control identity.LaunchControl) error {
	if control.Signal != 0 {
		if control.Cols != 0 || control.Rows != 0 || !museSignal(control.Signal) {
			return identity.ErrDenied
		}
		_, err := m.guest(ctx, e.caller.Container, nil, "/usr/bin/systemctl", "kill", "--kill-whom=all", "--signal="+strconv.Itoa(control.Signal), e.unit)
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

func museSignal(sig int) bool {
	switch unix.Signal(sig) {
	case unix.SIGINT, unix.SIGTERM, unix.SIGHUP, unix.SIGQUIT, unix.SIGTSTP, unix.SIGCONT, unix.SIGUSR1, unix.SIGUSR2:
		return true
	}
	return false
}

func museCommand(ctx context.Context, c museCaller, in identity.LaunchRequest, unit, path string) *exec.Cmd {
	home := c.Home
	if in.Home != "" {
		home = in.Home
	}
	configPath := path
	if c.Child != "" {
		configPath = "/run/soda-muse/credentials/" + strings.TrimPrefix(path, "/run/soda-muse/nested/"+c.Registration+"/")
	}
	args := []string{"--remote=false", "exec", "--interactive"}
	if in.TTY {
		args = append(args, "--tty")
	}
	args = append(args, c.Container, "/usr/bin/systemd-run", "--quiet", "--wait", "--collect", "--service-type=exec", "--unit="+unit, "--property=KillMode=control-group", "--property=TimeoutStopSec=10", "--property=RuntimeMaxSec=43200", "--property=UMask=0077", "--setenv=HOME="+home, "--setenv=USER="+c.Login, "--setenv=LOGNAME="+c.Login, "--setenv=TERM="+in.Term, "--setenv=XDG_CONFIG_HOME="+path+"/config", "--setenv=XDG_STATE_HOME="+path+"/state", "--setenv=XDG_CACHE_HOME="+path+"/cache", "--setenv=TBH_CREDENTIAL_BACKEND=file", "--setenv=PATH=/usr/local/bin:/usr/bin:/bin")
	if c.Child == "" {
		args = append(args, "--uid="+c.Login, "--gid="+strconv.Itoa(c.GID), "--working-directory="+in.CWD, "--property=ReadOnlyPaths="+path+"/auth.json", "--property=BindReadOnlyPaths="+path+"/auth.json:"+path+"/config/muse/auth.json")
	}
	if in.TTY {
		args = append(args, "--pty")
	} else {
		args = append(args, "--pipe")
	}
	if c.Child != "" {
		args = append(args, "--", "/usr/bin/nsenter", "--target="+strconv.Itoa(c.NestedPID), "--mount", "--pid", "--uts", "--ipc", "--net", "--root", "--setuid="+strconv.Itoa(c.UID), "--setgid="+strconv.Itoa(c.GID), "--", "/usr/local/bin/muse", "--soda-exec", configPath, in.CWD)
	} else {
		args = append(args, "--", "/usr/local/bin/muse", "--soda-exec", configPath, in.CWD)
	}
	args = append(args, in.Args...)
	command := exec.CommandContext(ctx, "/usr/bin/podman", args...)
	// Preserve the operator-selected Podman context, never caller environment.
	command.Env = museHostEnvironment()
	return command
}

func (m *MuseRuntime) stage(ctx context.Context, c museCaller, path string, configHome string) error {
	_, err := m.guest(ctx, c.Container, nil, "/usr/bin/install", "--directory", "--mode=0700", path)
	if err != nil {
		return err
	}
	if c.Child == "" {
		_, err = m.guest(ctx, c.Container, nil, "/usr/bin/mount", "--types=tmpfs", "--options=mode=0700,size=4M", "tmpfs", path)
		if err != nil {
			return err
		}
	}
	if err = m.stageFiles(ctx, c, path); err != nil {
		return err
	}
	return m.stageConfig(ctx, c, path, configHome)
}

func (m *MuseRuntime) stageFiles(ctx context.Context, c museCaller, path string) error {
	var err error
	for _, dir := range []string{path + "/config/muse"} {
		if _, err = m.guest(ctx, c.Container, nil, "/usr/bin/install", "--directory", "--mode=0700", "--owner="+strconv.Itoa(c.UID), "--group="+strconv.Itoa(c.GID), dir); err != nil {
			return err
		}
	}
	if _, err = m.guest(ctx, c.Container, nil, "/usr/bin/chmod", "0711", path); err != nil {
		return err
	}
	if _, err = m.guest(ctx, c.Container, []byte("{}"), "/usr/bin/dd", "of="+path+"/auth.json", "status=none"); err != nil {
		return err
	}
	if _, err = m.guest(ctx, c.Container, nil, "/usr/bin/chown", strconv.Itoa(c.UID)+":"+strconv.Itoa(c.GID), path+"/auth.json"); err != nil {
		return err
	}
	if _, err = m.guest(ctx, c.Container, nil, "/usr/bin/chmod", "0600", path+"/auth.json"); err != nil {
		return err
	}
	return nil
}

func (m *MuseRuntime) stageConfig(ctx context.Context, c museCaller, path, configHome string) error {
	if configHome == "" {
		configHome = c.Home + "/.config"
	}
	if err := m.populateConfig(ctx, c, path, configHome); err != nil {
		return err
	}
	if _, err := m.guest(ctx, c.Container, nil, "/usr/bin/chmod", "0711", path+"/config"); err != nil {
		return err
	}
	return m.authMountTarget(ctx, c, path)
}

func (m *MuseRuntime) populateConfig(ctx context.Context, c museCaller, path, source string) error {
	if c.Child != "" {
		return m.copyNestedConfig(ctx, c, path, source)
	}
	_, err := m.podman(ctx, nil, "exec", "--user="+strconv.Itoa(c.UID)+":"+strconv.Itoa(c.GID), c.Container, "/usr/local/bin/muse", "--soda-copy-config", source+"/muse", path+"/config/muse")
	return err
}

func (m *MuseRuntime) authMountTarget(ctx context.Context, c museCaller, path string) error {
	if c.Child != "" {
		_, err := m.guest(ctx, c.Container, nil, "/usr/bin/ln", "--symbolic", "../../auth.json", path+"/config/muse/auth.json")
		return err
	}
	_, err := m.guest(ctx, c.Container, nil, "/usr/bin/touch", path+"/config/muse/auth.json")
	return err
}

func (m *MuseRuntime) stopExecution(ctx context.Context, binding identity.Binding, actor int64, leaseID string, returnCustody bool) error {
	container := binding.Project
	unit := "soda-muse-" + binding.ID + ".service"
	// systemctl stop waits for KillMode=control-group. Verify its cgroup is gone or
	// unpopulated before returning custody. Do not infer cleanup from CLI exit.
	_, stopErr := m.guest(ctx, container, nil, "/usr/bin/systemctl", "stop", unit)
	body, err := m.guest(ctx, container, nil, "/usr/bin/systemctl", "show", "--property=ActiveState", "--value", unit)
	if err != nil || strings.TrimSpace(string(body)) != "inactive" {
		return identity.ErrUncertain
	}
	if stopErr != nil {
		if _, err = m.guest(ctx, container, nil, "/usr/bin/test", "!", "-d", "/sys/fs/cgroup/system.slice/"+unit); err != nil {
			return identity.ErrUncertain
		}
	}
	if err = m.retireExecutionFiles(ctx, binding); err != nil {
		return err
	}
	if returnCustody {
		return m.End(ctx, actor, leaseID)
	}
	return nil
}

// ValidateMuseBinding checks the exact container incarnation and execution unit.
func (m *MuseRuntime) ValidateMuseBinding(ctx context.Context, b identity.Binding) error {
	if b.Validate() != nil || !terminalID.MatchString(b.ID) || !containerID.MatchString(b.Project) || !loginName.MatchString(b.Login) {
		return identity.ErrDenied
	}
	body, err := m.guest(ctx, b.Project, nil, "/usr/bin/systemctl", "show", "--property=ActiveState", "--value", "soda-muse-"+b.ID+".service")
	if err != nil || strings.TrimSpace(string(body)) != "active" {
		return identity.ErrStale
	}
	return nil
}

func (m *MuseRuntime) awaitUnit(ctx context.Context, container, unit string) error {
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		body, err := m.guest(ctx, container, nil, "/usr/bin/systemctl", "show", "--property=ActiveState", "--value", unit)
		if err == nil && strings.TrimSpace(string(body)) == "active" {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return identity.ErrDenied
		case <-tick.C:
		}
	}
}

func museResize(file *os.File, control identity.LaunchControl) error {
	if control.Signal != 0 || control.Cols == 0 || control.Rows == 0 {
		return identity.ErrDenied
	}
	return unix.IoctlSetWinsize(int(file.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Col: control.Cols, Row: control.Rows})
}

// Muse executes broker-selected operations against recorded native boundaries.
func (m *MuseRuntime) Muse(ctx context.Context, action string, delivery identity.Delivery) (identity.Delivery, error) {
	if !museDeliveryValid(delivery) {
		return delivery, identity.ErrDenied
	}
	b := delivery.Lease.Binding
	var err error
	switch b.Scope {
	case "muse-factory":
		err = m.factoryOperation(ctx, action, delivery)
	case "muse-project":
		err = m.projectOperation(ctx, action, delivery)
	default:
		err = identity.ErrDenied
	}
	return delivery, err
}

func museDeliveryValid(delivery identity.Delivery) bool {
	b := delivery.Lease.Binding
	if b == nil {
		return false
	}
	return delivery.Lease.ProviderID == identity.Muse && terminalID.MatchString(delivery.Lease.ExecutionID) && b.ID == delivery.Lease.ExecutionID && terminalID.MatchString(b.InvocationID)
}

func (m *MuseRuntime) factoryOperation(ctx context.Context, action string, delivery identity.Delivery) error {
	b := delivery.Lease.Binding
	unit := "soda-muse-" + b.ID + ".service"
	if !m.validFactoryCredentialRoot(b.CredentialRoot, b.ID) {
		return identity.ErrDenied
	}
	if action == "validate" {
		id, err := m.factoryInvocation(ctx, unit, false)
		if err != nil || id != b.InvocationID {
			return identity.ErrStale
		}
		return nil
	}
	if action != "stop" {
		return identity.ErrDenied
	}
	if err := m.stopFactoryUnit(ctx, unit, b.InvocationID); err != nil {
		return err
	}
	if err := m.cleanupExecutionState(ctx, *b); err != nil {
		return err
	}
	_, err := m.podman(ctx, nil, "unshare", "/usr/bin/rm", "--recursive", "--force", "--", b.CredentialRoot)
	return err
}

func (m *MuseRuntime) validFactoryCredentialRoot(path, id string) bool {
	if !filepath.IsAbs(m.FactoryRoot) || filepath.Clean(path) != path {
		return false
	}
	relative, err := filepath.Rel(m.FactoryRoot, path)
	if err != nil {
		return false
	}
	parts := strings.Split(relative, string(filepath.Separator))
	return len(parts) == 2 && terminalID.MatchString(parts[0]) && parts[1] == id
}

func (m *MuseRuntime) projectOperation(ctx context.Context, action string, delivery identity.Delivery) error {
	b := delivery.Lease.Binding
	unit := "soda-muse-" + b.ID + ".service"
	if !museProjectCredentialRoot(*b) {
		return identity.ErrDenied
	}
	body, err := m.guest(ctx, b.Project, nil, "/usr/bin/systemctl", "show", "--property=InvocationID", "--value", unit)
	if err == nil && strings.TrimSpace(string(body)) != "" && strings.TrimSpace(string(body)) != b.InvocationID {
		return identity.ErrStale
	}
	if action == "validate" {
		return m.ValidateMuseBinding(ctx, *b)
	}
	if action != "stop" {
		return identity.ErrDenied
	}
	return m.stopExecution(ctx, *b, delivery.Lease.ActorID, delivery.Lease.ID, false)
}

func museProjectCredentialRoot(b identity.Binding) bool {
	if !containerID.MatchString(b.Project) || filepath.Clean(b.CredentialRoot) != b.CredentialRoot || !strings.HasPrefix(b.CredentialRoot, "/run/soda-muse/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(b.CredentialRoot, "/run/soda-muse/"), "/")
	if len(parts) == 1 {
		return parts[0] == b.ID
	}
	return len(parts) == 3 && parts[0] == "nested" && terminalID.MatchString(parts[1]) && parts[2] == b.ID
}

func (m *MuseRuntime) copyNestedConfig(ctx context.Context, c museCaller, path, source string) error {
	body, err := m.guest(ctx, c.Container, nil, "/usr/bin/nsenter", "--target="+strconv.Itoa(c.NestedPID), "--mount", "--pid", "--uts", "--ipc", "--net", "--root", "--setuid="+strconv.Itoa(c.UID), "--setgid="+strconv.Itoa(c.GID), "--", "/usr/local/bin/muse", "--soda-read-config", source+"/muse")
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
		if _, err = m.guest(ctx, c.Container, value, "/usr/bin/dd", "of="+path+"/config/muse/"+name, "status=none"); err != nil {
			return err
		}
		clear(value)
		if _, err = m.guest(ctx, c.Container, nil, "/usr/bin/chown", strconv.Itoa(c.UID)+":"+strconv.Itoa(c.GID), path+"/config/muse/"+name); err != nil {
			return err
		}
		if _, err = m.guest(ctx, c.Container, nil, "/usr/bin/chmod", "0600", path+"/config/muse/"+name); err != nil {
			return err
		}
	}
	return nil
}

func museHostEnvironment() []string {
	var environment []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if name == "META_API_KEY" {
			continue
		}
		environment = append(environment, entry)
	}
	return environment
}

func (m *MuseRuntime) retireMount(ctx context.Context, container, path string) error {
	if _, err := m.guest(ctx, container, nil, "/usr/bin/umount", path); err == nil {
		return nil
	}
	if _, err := m.guest(ctx, container, nil, "/usr/bin/test", "!", "-e", path); err == nil {
		return nil
	}
	_, err := m.guest(ctx, container, nil, "/usr/bin/mountpoint", "--quiet", path)
	var exit interface{ ExitCode() int }
	if errors.As(err, &exit) && exit.ExitCode() == 32 {
		return nil
	}
	return identity.ErrUncertain
}
