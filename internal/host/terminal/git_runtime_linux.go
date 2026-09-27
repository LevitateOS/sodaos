//go:build linux

package terminal

import (
	"context"
	"errors"
	"maps"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

func (g *GitRuntime) Start(ctx context.Context, peer MusePeer, in identity.GitLaunchRequest, listenerFile *os.File) (GitInvocation, error) {
	var out GitInvocation
	if !g.ready(in) {
		return out, identity.ErrDenied
	}
	caller, resolver, err := g.resolveCaller(ctx, peer)
	if err != nil {
		return out, identity.ErrDenied
	}
	listener, err := gitListener(listenerFile, peer)
	if err != nil {
		return out, err
	}
	owned := true
	defer func() {
		if owned {
			_ = listener.Close()
		}
	}()
	id, capability, lease, err := g.reserveHelper(ctx, caller, in)
	if err != nil {
		return out, err
	}
	binding := identity.Binding{Kind: identity.Terminal, ID: id, Project: caller.Project, ContainerID: caller.Container, ChildID: caller.Child, UID: caller.UID, GID: caller.GID, Login: caller.Login, Generation: lease.Generation, Scope: "git"}
	unit := "soda-git-" + id + ".service"
	local := "http://" + listener.Addr().String() + "/" + in.Owner + "/" + in.Repository + ".git"
	path := "/run/soda-git/" + id
	config := path + "/gitconfig"
	if caller.Child != "" {
		path = "/run/soda-muse/nested/" + caller.Registration + "/git-" + id
		config = "/run/soda-git/credentials/git-" + id + "/gitconfig"
	}
	binding.CredentialRoot = path
	if err = g.stageConfig(ctx, caller, path, capability); err != nil {
		clean, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		cleanupErr := g.removeConfig(clean, caller.Container, path)
		endErr := g.End(clean, caller.Actor, lease.ID)
		return out, errors.Join(err, cleanupErr, endErr)
	}
	out.Command = gitCommand(ctx, caller, in, unit, local, config)
	server := &http.Server{Handler: gitRelay(capability, g.Proxy(lease.ID)), ReadHeaderTimeout: 10 * time.Second}
	relayCtx, cancel := context.WithCancel(ctx)
	out.Prepare = g.prepareHelper(resolver, caller, unit, &binding, lease, relayCtx, listener, server)
	out.Control = g.controlHelper(resolver, caller, unit, &binding)
	out.Finish = g.finishHelper(cancel, server, listener, &binding, caller, lease)

	owned = false
	return out, nil
}

func (g *GitRuntime) resolveCaller(ctx context.Context, peer MusePeer) (museCaller, *MuseRuntime, error) {
	resolver := g.resolver()
	caller, err := resolver.resolve(ctx, peer, identity.Forgejo)
	if err == nil && caller.Child != "" {
		caller, err = g.nestedGitCaller(ctx, resolver, caller)
	}
	return caller, resolver, err
}

func (g *GitRuntime) resolver() *MuseRuntime {
	resolver := &MuseRuntime{Exec: g.Exec, Authorize: g.Authorize}
	if g.Nested != nil {
		g.Nested.mu.Lock()
		resolver.nested = maps.Clone(g.Nested.nested)
		g.Nested.mu.Unlock()
	}
	return resolver
}

func (g *GitRuntime) nestedGitCaller(ctx context.Context, resolver *MuseRuntime, c museCaller) (museCaller, error) {
	body, err := resolver.guest(ctx, c.Container, nil, "/usr/bin/podman", "--remote=false", "exec", c.Child, "/usr/bin/getent", "passwd", strconv.Itoa(c.UID))
	if err != nil || len(body) > 4096 {
		return c, identity.ErrDenied
	}
	account := strings.Split(strings.TrimSpace(string(body)), ":")
	if len(account) != 7 || account[2] != strconv.Itoa(c.UID) || !strings.HasPrefix(account[5], "/") || strings.ContainsAny(account[5], "\x00\r\n") {
		return c, identity.ErrDenied
	}
	c.Home = account[5]
	return c, nil
}

func (g *GitRuntime) reserveHelper(ctx context.Context, caller museCaller, in identity.GitLaunchRequest) (string, string, identity.Lease, error) {
	connection, err := g.Select(ctx, caller.Actor, caller.Project, in.ConnectionID)
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
	lease, err := g.Acquire(ctx, identity.GitAcquireRequest{Owner: in.Owner, Repository: in.Repository, Acquire: identity.AcquireRequest{ActorID: caller.Actor, ProjectID: caller.Project, ConnectionID: connection, ExecutionID: id, ProviderID: identity.Forgejo, Kind: identity.Terminal, Deadline: time.Now().Add(12 * time.Hour)}})
	if err != nil {
		return "", "", identity.Lease{}, err
	}
	return id, capability, lease, nil
}

func (g *GitRuntime) ready(in identity.GitLaunchRequest) bool {
	return in.Validate() == nil && g.Authorize != nil && g.Select != nil && g.Acquire != nil && g.Register != nil && g.End != nil && g.Proxy != nil
}

func (g *GitRuntime) prepareHelper(resolver *MuseRuntime, caller museCaller, unit string, binding *identity.Binding, lease identity.Lease, relayCtx context.Context, listener net.Listener, server *http.Server) func(context.Context) error {
	return func(ctx context.Context) error {
		if err := resolver.awaitUnit(ctx, caller.Container, unit); err != nil {
			return err
		}
		body, err := resolver.guest(ctx, caller.Container, nil, "/usr/bin/systemctl", "show", "--property=InvocationID", "--value", unit)
		binding.InvocationID = strings.TrimSpace(string(body))
		if err != nil || !terminalID.MatchString(binding.InvocationID) {
			return identity.ErrDenied
		}
		session, err := g.Register(ctx, lease.ID, *binding)
		if err != nil {
			return err
		}
		if !gitSessionMatches(session, lease, *binding) {
			return identity.ErrDenied
		}
		if err := g.configureGitIdentity(ctx, caller, session); err != nil {
			return err
		}
		gitServe(relayCtx, listener, server)
		return nil
	}
}

func gitSessionMatches(session identity.GitSession, lease identity.Lease, binding identity.Binding) bool {
	if session.Lease.Binding == nil || *session.Lease.Binding != binding || !session.Lease.Deadline.Equal(lease.Deadline) {
		return false
	}
	registered := session.Lease
	registered.Binding = nil
	registered.Deadline = lease.Deadline
	lease.Binding = nil
	return registered == lease
}

func (g *GitRuntime) controlHelper(resolver *MuseRuntime, caller museCaller, unit string, binding *identity.Binding) func(context.Context, identity.LaunchControl) error {
	return func(ctx context.Context, control identity.LaunchControl) error {
		if control.Signal == 0 || control.Cols != 0 || control.Rows != 0 || !museSignal(control.Signal) {
			return identity.ErrDenied
		}
		if err := g.Validate(ctx, *binding); err != nil {
			return err
		}
		_, err := resolver.guest(ctx, caller.Container, nil, "/usr/bin/systemctl", "kill", "--kill-whom=all", "--signal="+strconv.Itoa(control.Signal), unit)
		return err
	}
}

func (g *GitRuntime) finishHelper(cancel context.CancelFunc, server *http.Server, listener net.Listener, binding *identity.Binding, caller museCaller, lease identity.Lease) func(context.Context) error {
	return func(ctx context.Context) error {
		cancel()
		_ = server.Close()
		_ = listener.Close()
		if err := g.Stop(ctx, *binding); err != nil {
			return err
		}
		return g.End(ctx, caller.Actor, lease.ID)
	}
}

func gitCommand(ctx context.Context, c museCaller, in identity.GitLaunchRequest, unit, remote, config string) *exec.Cmd {
	args := []string{"--remote=false", "exec", "--interactive", c.Container, "/usr/bin/systemd-run", "--quiet", "--wait", "--collect", "--service-type=exec", "--pipe", "--unit=" + unit, "--property=KillMode=control-group", "--property=TimeoutStopSec=10", "--property=RuntimeMaxSec=43200"}
	if c.Child == "" {
		args = append(args, "--uid="+c.Login, "--gid="+strconv.Itoa(c.GID), "--working-directory="+in.CWD)
	} else {
		args = append(args, "--uid=root", "--gid=0")
	}
	args = append(args, "--setenv=HOME="+c.Home, "--setenv=GIT_CONFIG_NOSYSTEM=1", "--setenv=GIT_CONFIG_GLOBAL=/dev/null", "--setenv=GIT_TERMINAL_PROMPT=0", "--setenv=PATH=/usr/bin:/bin", "--setenv=GIT_CONFIG_COUNT=4", "--setenv=GIT_CONFIG_KEY_0=http.proxy", "--setenv=GIT_CONFIG_VALUE_0=", "--setenv=GIT_CONFIG_KEY_1=http.followRedirects", "--setenv=GIT_CONFIG_VALUE_1=false", "--setenv=GIT_CONFIG_KEY_2=credential.helper", "--setenv=GIT_CONFIG_VALUE_2=", "--setenv=GIT_CONFIG_KEY_3=include.path", "--setenv=GIT_CONFIG_VALUE_3="+config, "--setenv=HTTP_PROXY=", "--setenv=HTTPS_PROXY=", "--setenv=ALL_PROXY=", "--setenv=GIT_TRACE=", "--setenv=GIT_TRACE_CURL=", "--setenv=GIT_CURL_VERBOSE=", "--")
	if c.Child != "" {
		args = append(args, "/usr/bin/nsenter", "--target="+strconv.Itoa(c.NestedPID), "--mount", "--pid", "--uts", "--ipc", "--net", "--root", "--setuid="+strconv.Itoa(c.UID), "--setgid="+strconv.Itoa(c.GID), "--wdns="+in.CWD, "--")
	}
	args = append(args, "/usr/libexec/git-core/git-remote-http", in.Remote, remote)
	command := exec.CommandContext(ctx, "/usr/bin/podman", args...)
	command.Env = museHostEnvironment()
	return command
}

func gitBinding(b identity.Binding) bool {
	return b.Validate() == nil && b.Kind == identity.Terminal && b.Scope == "git" && terminalID.MatchString(b.ID) && terminalID.MatchString(b.InvocationID) && containerID.MatchString(b.ContainerID) && loginName.MatchString(b.Login) && gitConfigBinding(b)
}

func gitConfigBinding(b identity.Binding) bool {
	if b.ChildID == "" {
		return b.CredentialRoot == "/run/soda-git/"+b.ID
	}
	return containerID.MatchString(b.ChildID) && gitNestedConfigPath(b.CredentialRoot) && strings.HasSuffix(b.CredentialRoot, "/git-"+b.ID)
}

func (g *GitRuntime) Validate(ctx context.Context, b identity.Binding) error {
	if !gitBinding(b) {
		return identity.ErrDenied
	}
	runtime := &MuseRuntime{Exec: g.Exec}
	if err := g.nativeAccount(ctx, b); err != nil {
		return err
	}
	body, err := runtime.guest(ctx, b.ContainerID, nil, "/usr/bin/systemctl", "show", "--property=ActiveState", "--property=InvocationID", "--property=User", "--property=Group", "--property=ControlGroup", "soda-git-"+b.ID+".service")
	if err != nil || !gitUnitMatches(body, b) {
		return identity.ErrStale
	}
	return nil
}

func (g *GitRuntime) nativeAccount(ctx context.Context, b identity.Binding) error {
	if err := g.nativeStopTarget(ctx, b); err != nil {
		return err
	}
	if b.ChildID == "" {
		return nil
	}
	record, ok := g.nestedRecord(b)
	if !ok || g.Nested.validateNested(ctx, record) != nil {
		return identity.ErrStale
	}
	account, err := g.Nested.actorAccount(ctx, b.ContainerID, record.Actor)
	if err != nil || account.Username != b.Login {
		return identity.ErrStale
	}
	return nil
}

func (g *GitRuntime) nativeStopTarget(ctx context.Context, b identity.Binding) error {
	runtime := &MuseRuntime{Exec: g.Exec}
	inspection, err := runtime.inspect(ctx, b.ContainerID)
	if err != nil || inspection.Project != b.Project {
		return identity.ErrStale
	}
	if b.ChildID != "" {
		return nil
	}
	account, err := runtime.projectAccount(ctx, museCaller{Container: b.ContainerID, UID: b.UID})
	if err != nil || account.Login != b.Login || account.GID != b.GID {
		return identity.ErrStale
	}
	return nil
}

func (g *GitRuntime) nestedRecord(b identity.Binding) (museNested, bool) {
	if g.Nested == nil || !gitConfigBinding(b) {
		return museNested{}, false
	}
	parts := strings.Split(b.CredentialRoot, "/")
	g.Nested.mu.Lock()
	defer g.Nested.mu.Unlock()
	for _, record := range g.Nested.nested {
		if record.Git && record.Registration == parts[4] && record.Child == b.ChildID && record.Parent == b.ContainerID && record.Project == b.Project {
			return record, true
		}
	}
	return museNested{}, false
}

func gitUnitMatches(body []byte, b identity.Binding) bool {
	properties := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			properties[key] = value
		}
	}
	user, group := b.Login, strconv.Itoa(b.GID)
	if b.ChildID != "" {
		user, group = "root", "0"
	}
	return properties["ActiveState"] == "active" && properties["InvocationID"] == b.InvocationID && properties["User"] == user && properties["Group"] == group && properties["ControlGroup"] == "/system.slice/soda-git-"+b.ID+".service"
}

func (g *GitRuntime) Stop(ctx context.Context, b identity.Binding) error {
	// Before registration, a failed launch still owns its random unit ID. Once
	// registered, a replacement invocation must never be stopped.
	if !gitStopBinding(b) {
		return identity.ErrDenied
	}
	if err := g.nativeStopTarget(ctx, b); err != nil {
		return err
	}
	runtime := &MuseRuntime{Exec: g.Exec}
	unit := "soda-git-" + b.ID + ".service"
	body, err := runtime.guest(ctx, b.ContainerID, nil, "/usr/bin/systemctl", "show", "--property=InvocationID", "--value", unit)
	if err != nil {
		return identity.ErrUncertain
	}
	current := strings.TrimSpace(string(body))
	if b.InvocationID != "" && current != "" && current != b.InvocationID {
		return identity.ErrStale
	}
	if err := gitStopUnit(ctx, runtime, b.ContainerID, unit); err != nil {
		return err
	}
	if err := gitEmptyUnit(ctx, runtime, b.ContainerID, unit); err != nil {
		return err
	}
	return g.removeConfig(ctx, b.ContainerID, b.CredentialRoot)
}

func gitStopUnit(ctx context.Context, runtime *MuseRuntime, container, unit string) error {
	_, _ = runtime.guest(ctx, container, nil, "/usr/bin/systemctl", "stop", unit)
	body, err := runtime.guest(ctx, container, nil, "/usr/bin/systemctl", "show", "--property=ActiveState", "--value", unit)
	if err != nil || strings.TrimSpace(string(body)) != "inactive" {
		return identity.ErrUncertain
	}
	return nil
}

func gitEmptyUnit(ctx context.Context, runtime *MuseRuntime, container, unit string) error {
	body, err := runtime.guest(ctx, container, nil, "/usr/bin/cat", "/sys/fs/cgroup/system.slice/"+unit+"/cgroup.events")
	if err == nil {
		if !strings.Contains(string(body), "populated 0\n") {
			return identity.ErrUncertain
		}
	} else {
		if _, err = runtime.guest(ctx, container, nil, "/usr/bin/test", "!", "-d", "/sys/fs/cgroup/system.slice/"+unit); err != nil {
			return errors.Join(identity.ErrUncertain, err)
		}
	}
	return nil
}

func gitStopBinding(b identity.Binding) bool {
	return b.Scope == "git" && terminalID.MatchString(b.ID) && containerID.MatchString(b.ContainerID) && gitConfigBinding(b)
}
