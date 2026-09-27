//go:build linux

package terminal

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"golang.org/x/sys/unix"
)

type factoryGitExecution struct {
	caller     FactoryGitCaller
	lease      identity.Lease
	binding    identity.Binding
	capability string
	config     string
	listener   net.Listener
	server     *http.Server
	cancel     context.CancelFunc
	workerFD   int
	started    bool
}

func (f *FactoryGitRuntime) StartFactory(ctx context.Context, peer MusePeer, in identity.GitLaunchRequest, listenerFile *os.File) (GitInvocation, error) {
	var out GitInvocation
	caller, workerFD, err := f.factoryPeer(ctx, peer, in)
	if err != nil {
		return out, err
	}
	listener, err := gitListener(listenerFile, peer)
	if err != nil {
		_ = unix.Close(workerFD)
		return out, err
	}
	id, capability, lease, err := f.reserveFactory(ctx, caller, in)
	if err != nil {
		_ = listener.Close()
		_ = unix.Close(workerFD)
		return out, err
	}
	relayCtx, cancel := context.WithCancel(ctx)
	e := &factoryGitExecution{caller: caller, lease: lease, capability: capability, config: "/tmp/soda-git-" + id + ".config", listener: listener, cancel: cancel, workerFD: workerFD}
	e.binding = identity.Binding{Kind: identity.Factory, Scope: "git-factory", ID: id, Project: caller.Project, ContainerID: caller.Container, Generation: lease.Generation, UID: 1000, GID: 1000}
	e.server = &http.Server{Handler: gitRelay(capability, f.Proxy(lease.ID)), ReadHeaderTimeout: 10 * time.Second}
	if err := f.stageFactoryConfig(ctx, e); err != nil {
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		_ = f.removeFactoryConfig(cleanup, e)
		_ = f.End(cleanup, e.caller.Actor, e.lease.ID)
		cancel()
		_ = listener.Close()
		_ = unix.Close(workerFD)
		return out, err
	}
	out.Command = factoryGitCommand(ctx, e, in)
	out.Prepare = func(ctx context.Context) error { return f.prepareFactory(ctx, relayCtx, peer, e) }
	out.Finish = func(ctx context.Context) error {
		e.started = out.Command.Process != nil
		return f.finishFactory(ctx, e)
	}
	out.Control = func(ctx context.Context, control identity.LaunchControl) error {
		return f.controlFactory(ctx, e, control)
	}
	return out, nil
}

func (f *FactoryGitRuntime) factoryPeer(ctx context.Context, peer MusePeer, in identity.GitLaunchRequest) (FactoryGitCaller, int, error) {
	var caller FactoryGitCaller
	if !f.ready(in) || f.Resolve == nil || !musePeerAlive(peer) {
		return caller, -1, identity.ErrDenied
	}
	caller, err := f.Resolve(ctx, peer)
	if err != nil || !validFactoryGitCaller(caller) {
		return caller, -1, identity.ErrDenied
	}
	fd, err := unix.PidfdOpen(caller.HostPID, 0)
	if err != nil {
		return caller, -1, identity.ErrDenied
	}
	if !factoryGitPIDAlive(fd) {
		_ = unix.Close(fd)
		return caller, -1, identity.ErrDenied
	}
	return caller, fd, nil
}

func validFactoryGitCaller(c FactoryGitCaller) bool {
	return containerID.MatchString(c.Container) && projectID.MatchString(c.Project) && factoryGitExecutionID(c.ExecutionID) && c.Actor > 0 && c.RepositoryID > 0 && c.HostPID > 0 && c.Deadline.After(time.Now())
}

func factoryGitExecutionID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, ch := range id {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func (f *FactoryGitRuntime) reserveFactory(ctx context.Context, c FactoryGitCaller, in identity.GitLaunchRequest) (string, string, identity.Lease, error) {
	connection, err := f.Select(ctx, c.Actor, c.Project, in.ConnectionID)
	if err != nil {
		return "", "", identity.Lease{}, err
	}
	id, err := museID()
	if err != nil {
		return "", "", identity.Lease{}, err
	}
	capability, err := gitCapability()
	if err != nil {
		return "", "", identity.Lease{}, err
	}
	deadline := c.Deadline
	if limit := time.Now().Add(12 * time.Hour); deadline.After(limit) {
		deadline = limit
	}
	lease, err := f.Acquire(ctx, identity.GitAcquireRequest{Owner: in.Owner, Repository: in.Repository, ExpectedRepositoryID: c.RepositoryID, Acquire: identity.AcquireRequest{ActorID: c.Actor, ProjectID: c.Project, ConnectionID: connection, ExecutionID: id, ProviderID: identity.Forgejo, Kind: identity.Factory, Deadline: deadline}})
	return id, capability, lease, err
}

func (f *FactoryGitRuntime) stageFactoryConfig(ctx context.Context, e *factoryGitExecution) error {
	config := "[http]\n\textraHeader =\n\textraHeader = X-Soda-Git-Invocation: " + e.capability + "\n"
	runtime := &MuseRuntime{Exec: f.Exec}
	_, err := runtime.guest(ctx, e.caller.Container, []byte(config), "/usr/bin/sh", "-c", "umask 077; cat > "+e.config)
	return err
}

func factoryGitCommand(ctx context.Context, e *factoryGitExecution, in identity.GitLaunchRequest) *exec.Cmd {
	local := "http://" + e.listener.Addr().String() + "/" + in.Owner + "/" + in.Repository + ".git"
	args := []string{"--user", "--quiet", "--wait", "--collect", "--service-type=exec", "--pipe", "--unit=" + factoryGitUnitName(e.binding.ID), "--property=KillMode=control-group", "--property=TimeoutStopSec=10", "--property=RuntimeMaxSec=" + museRuntimeMax(e.lease.Deadline), "--working-directory=/", "--setenv=HOME=/workspace/home", "--setenv=PATH=/usr/bin:/bin", "--setenv=GIT_CONFIG_NOSYSTEM=1", "--setenv=GIT_CONFIG_GLOBAL=/dev/null", "--setenv=GIT_TERMINAL_PROMPT=0", "--setenv=GIT_CONFIG_COUNT=4", "--setenv=GIT_CONFIG_KEY_0=http.proxy", "--setenv=GIT_CONFIG_VALUE_0=", "--setenv=GIT_CONFIG_KEY_1=http.followRedirects", "--setenv=GIT_CONFIG_VALUE_1=false", "--setenv=GIT_CONFIG_KEY_2=credential.helper", "--setenv=GIT_CONFIG_VALUE_2=", "--setenv=GIT_CONFIG_KEY_3=include.path", "--setenv=GIT_CONFIG_VALUE_3=" + e.config, "--setenv=HTTP_PROXY=", "--setenv=HTTPS_PROXY=", "--setenv=ALL_PROXY=", "--setenv=GIT_TRACE=", "--setenv=GIT_TRACE_CURL=", "--setenv=GIT_CURL_VERBOSE=", "--", "/usr/bin/nsenter", "--target=" + strconv.Itoa(e.caller.HostPID), "--user", "--mount", "--net", "--root", "--setuid=1000", "--setgid=1000", "--wdns=" + in.CWD, "--", "/usr/libexec/git-core/git-remote-http", in.Remote, local}
	command := exec.CommandContext(ctx, "/usr/bin/systemd-run", args...)
	command.Env = append(museHostEnvironment(), "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/"+strconv.Itoa(os.Getuid())+"/bus")
	return command
}

func (f *FactoryGitRuntime) prepareFactory(ctx, relayCtx context.Context, peer MusePeer, e *factoryGitExecution) error {
	unit, err := f.awaitFactoryUnit(ctx, peer, e)
	if err != nil {
		return err
	}
	e.binding.InvocationID = unit.invocation
	// ChildID is the host PID of the user-unit MainPID, not an OCI ID.
	e.binding.ChildID = strconv.Itoa(unit.mainPID)
	if err := f.attestFactoryGitPeer(ctx, peer, e); err != nil {
		return err
	}
	session, err := f.Register(ctx, e.lease.ID, e.binding)
	if err != nil || !gitSessionMatches(session, e.lease, e.binding) {
		return identity.ErrDenied
	}
	gitServe(relayCtx, e.listener, e.server)
	return nil
}

func (f *FactoryGitRuntime) awaitFactoryUnit(ctx context.Context, peer MusePeer, e *factoryGitExecution) (factoryGitUnit, error) {
	deadline := time.NewTimer(30 * time.Second)
	defer deadline.Stop()
	for {
		unit, err := f.unit(ctx, e.binding.ID)
		if err == nil && factoryGitUnitReady(unit, e.binding.ID) {
			return unit, nil
		}
		if !musePeerAlive(peer) || !factoryGitPIDAlive(e.workerFD) {
			return factoryGitUnit{}, identity.ErrStale
		}
		select {
		case <-ctx.Done():
			return factoryGitUnit{}, ctx.Err()
		case <-deadline.C:
			return factoryGitUnit{}, identity.ErrStale
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func factoryGitUnitReady(unit factoryGitUnit, id string) bool {
	return unit.state == "active" && terminalID.MatchString(unit.invocation) && unit.mainPID > 0 && strings.HasSuffix(unit.group, "/"+factoryGitUnitName(id))
}

func (f *FactoryGitRuntime) attestFactoryGitPeer(ctx context.Context, peer MusePeer, e *factoryGitExecution) error {
	current, err := f.Resolve(ctx, peer)
	if err != nil || current != e.caller || !factoryGitPIDAlive(e.workerFD) || factoryGitNamespaces(unitPID(e.binding), e.caller.HostPID) != nil {
		return identity.ErrStale
	}
	return nil
}

func unitPID(b identity.Binding) int { pid, _ := strconv.Atoi(b.ChildID); return pid }

func (f *FactoryGitRuntime) controlFactory(ctx context.Context, e *factoryGitExecution, control identity.LaunchControl) error {
	if control.Signal == 0 || control.Cols != 0 || control.Rows != 0 || !museSignal(control.Signal) {
		return identity.ErrDenied
	}
	if err := f.Validate(ctx, identity.Lease{Binding: &e.binding}, e.caller.HostPID); err != nil {
		return err
	}
	runtime := &MuseRuntime{Exec: f.Exec}
	_, err := runtime.userSystemctl(ctx, "kill", "--kill-whom=all", "--signal="+strconv.Itoa(control.Signal), factoryGitUnitName(e.binding.ID))
	return err
}

func (f *FactoryGitRuntime) finishFactory(ctx context.Context, e *factoryGitExecution) error {
	e.cancel()
	_ = e.server.Close()
	_ = e.listener.Close()
	defer unix.Close(e.workerFD)
	if e.started && e.binding.InvocationID == "" {
		if err := f.recoverFactoryUnit(ctx, e); err != nil {
			return err
		}
	}
	if e.binding.InvocationID != "" {
		if err := f.Stop(ctx, e.binding); err != nil {
			return err
		}
	}
	cleanupErr := f.removeFactoryConfig(ctx, e)
	endErr := f.End(ctx, e.caller.Actor, e.lease.ID)
	return errors.Join(cleanupErr, endErr)
}

func (f *FactoryGitRuntime) removeFactoryConfig(ctx context.Context, e *factoryGitExecution) error {
	runtime := &MuseRuntime{Exec: f.Exec}
	_, err := runtime.guest(ctx, e.caller.Container, nil, "/usr/bin/rm", "-f", e.config)
	return err
}

func (f *FactoryGitRuntime) recoverFactoryUnit(ctx context.Context, e *factoryGitExecution) error {
	unit, err := f.unit(ctx, e.binding.ID)
	if err != nil {
		return identity.ErrUncertain
	}
	if unit.state == "inactive" && unit.group == "" {
		return nil
	}
	if !factoryGitUnitReady(unit, e.binding.ID) || !factoryGitPIDAlive(e.workerFD) {
		return identity.ErrUncertain
	}
	if err := factoryGitNamespaces(unit.mainPID, e.caller.HostPID); err != nil {
		return identity.ErrUncertain
	}
	e.binding.InvocationID = unit.invocation
	e.binding.ChildID = strconv.Itoa(unit.mainPID)
	return nil
}
