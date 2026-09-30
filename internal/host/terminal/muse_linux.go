//go:build linux

package terminal

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/sys/unix"
)

type (
	museInspection struct {
		ID         string `json:"id"`
		PID        int    `json:"pid"`
		Project    string `json:"project"`
		Running    bool   `json:"running"`
		Privileged bool   `json:"privileged"`
		Userns     string `json:"userns"`
	}
)

const museInspect = `{"id":{{json .ID}},"pid":{{json .State.Pid}},"project":{{json (index .Config.Labels "org.soda.project")}},"running":{{json .State.Running}},"privileged":{{json .HostConfig.Privileged}},"userns":{{json .HostConfig.UsernsMode}}}`

func (m *MuseRuntime) podman(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	return m.Exec.Run(ctx, in, "/usr/bin/podman", append([]string{"--remote=false"}, args...)...)
}

func (m *MuseRuntime) guest(ctx context.Context, c string, in []byte, args ...string) ([]byte, error) {
	return m.podman(ctx, in, append([]string{"exec", "--interactive", c}, args...)...)
}

func musePeerAlive(peer MusePeer) bool {
	fds := []unix.PollFd{{Fd: int32(peer.PIDFD), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, 0)
	return err == nil && n == 0
}

func (m *MuseRuntime) inspect(ctx context.Context, c string) (museInspection, error) {
	var out museInspection
	body, err := m.podman(ctx, nil, "inspect", "--format", museInspect, c)
	if err != nil || len(body) > 4096 || strictjson.Decode(bytes.NewReader(body), &out) != nil || out.ID != c || !out.Running || out.PID <= 0 || !projectID.MatchString(out.Project) || out.Privileged {
		return out, identity.ErrDenied
	}
	return out, nil
}

func museProjectCgroup(value string) (string, error) {
	// Match the complete OCI scope component, never a substring or name guess.
	for _, part := range strings.Split(strings.TrimSpace(value), "/") {
		if strings.HasPrefix(part, "libpod-") && strings.HasSuffix(part, ".scope") {
			id := strings.TrimSuffix(strings.TrimPrefix(part, "libpod-"), ".scope")
			if containerID.MatchString(id) {
				return id, nil
			}
		}
	}
	return "", identity.ErrDenied
}

func museMappedUID(data string, uid uint32) (int, error) {
	for _, line := range strings.Split(data, "\n") {
		p := strings.Fields(line)
		if len(p) != 3 {
			continue
		}
		inside, e1 := strconv.ParseUint(p[0], 10, 32)
		outside, e2 := strconv.ParseUint(p[1], 10, 32)
		count, e3 := strconv.ParseUint(p[2], 10, 32)
		if e1 == nil && e2 == nil && e3 == nil && outside > 0 && uint64(uid) >= outside && uint64(uid) < outside+count {
			return int(inside + uint64(uid) - outside), nil
		}
	}
	return 0, identity.ErrDenied
}

func (m *MuseRuntime) resolveProject(ctx context.Context, peer MusePeer) (museCaller, error) {
	c, err := museKernelCaller(peer)
	if err != nil {
		return c, err
	}
	inspected, err := m.inspect(ctx, c.Container)
	if err != nil {
		return c, err
	}
	c.Project = inspected.Project
	c.ProjectPID = inspected.PID
	verified, err := (&Service{Exec: m.Exec}).projectContainer(ctx, c.Project, true)
	if err != nil || verified != c.Container {
		return c, identity.ErrDenied
	}
	projectNS, err := os.Readlink("/proc/" + strconv.Itoa(c.ProjectPID) + "/ns/pid")
	if err != nil {
		return c, identity.ErrDenied
	}
	if c.Namespace != projectNS {
		return m.registeredCaller(ctx, c, peer)
	}
	if !musePeerAlive(peer) {
		return c, identity.ErrDenied
	}
	return c, nil
}

func museKernelCaller(peer MusePeer) (museCaller, error) {
	var c museCaller
	if !musePeerAlive(peer) {
		return c, identity.ErrDenied
	}
	proc := "/proc/" + strconv.Itoa(peer.PID)
	cg, err := os.ReadFile(proc + "/cgroup")
	if err != nil {
		return c, identity.ErrDenied
	}
	c.Container, err = museProjectCgroup(string(cg))
	if err != nil {
		return c, err
	}
	mappings, err := os.ReadFile(proc + "/uid_map")
	if err != nil {
		return c, identity.ErrDenied
	}
	c.UID, err = museMappedUID(string(mappings), peer.UID)
	if err != nil {
		return c, err
	}
	c.Namespace, err = os.Readlink(proc + "/ns/pid")
	return c, err
}

func (m *MuseRuntime) registeredCaller(ctx context.Context, c museCaller, peer MusePeer) (museCaller, error) {
	m.mu.Lock()
	registered, ok := m.nested[c.Namespace]
	m.mu.Unlock()
	if !ok || registered.Parent != c.Container || registered.Project != c.Project {
		return c, identity.ErrDenied
	}
	if err := m.validateNested(ctx, registered); err != nil {
		return c, err
	}
	c.Actor = registered.Actor
	c.Child = registered.Child
	c.Registration = registered.Registration
	c.NestedPID = registered.PID
	c.MuseAllowed = registered.Muse
	if !musePeerAlive(peer) {
		return c, identity.ErrDenied
	}
	return c, nil
}

func (m *MuseRuntime) resolve(ctx context.Context, peer MusePeer) (museCaller, error) {
	c, err := m.resolveProject(ctx, peer)
	if err != nil {
		return c, identity.ErrDenied
	}
	if c.Child != "" {
		if !c.MuseAllowed {
			return c, identity.ErrDenied
		}
		return m.nestedCaller(ctx, c, peer)
	}
	if c.UID == 0 {
		return c, identity.ErrDenied
	}
	c, err = m.projectAccount(ctx, c)
	if err != nil {
		return c, err
	}
	c.Actor, err = m.projectActor(ctx, c)
	if err != nil {
		return c, err
	}
	if !m.authorizedCaller(ctx, c, peer) {
		return c, identity.ErrDenied
	}
	return c, nil
}

func (m *MuseRuntime) projectAccount(ctx context.Context, c museCaller) (museCaller, error) {
	body, err := m.guest(ctx, c.Container, nil, "/usr/bin/getent", "passwd", strconv.Itoa(c.UID))
	if err != nil || len(body) > 4096 {
		return c, identity.ErrDenied
	}
	account := strings.Split(strings.TrimSpace(string(body)), ":")
	if !musePasswdValid(account, c.UID) {
		return c, identity.ErrDenied
	}
	c.Login = account[0]
	c.Home = account[5]
	c.GID, err = strconv.Atoi(account[3])
	if err != nil {
		return c, identity.ErrDenied
	}
	return c, nil
}

func musePasswdValid(account []string, uid int) bool {
	if len(account) != 7 {
		return false
	}
	return loginName.MatchString(account[0]) && account[0] != "root" && account[2] == strconv.Itoa(uid) && filepath.IsAbs(account[5])
}

func (m *MuseRuntime) projectActor(ctx context.Context, c museCaller) (int64, error) {
	marker := "/var/lib/soda/accounts/" + c.Login
	body, err := m.guest(ctx, c.Container, nil, "/usr/bin/stat", "--format=%u:%g:%a:%F", "/var", "/var/lib", "/var/lib/soda", "/var/lib/soda/accounts", marker)
	if err != nil || !museAccountModes(string(body)) {
		return 0, identity.ErrDenied
	}
	body, err = m.guest(ctx, c.Container, nil, "/usr/bin/cat", marker)
	if err != nil || len(body) > 64 {
		return 0, identity.ErrDenied
	}
	actor, err := strconv.ParseInt(strings.TrimSpace(string(body)), 10, 64)
	if err != nil || actor <= 0 {
		return 0, identity.ErrDenied
	}
	return actor, nil
}

func (m *MuseRuntime) authorizedCaller(ctx context.Context, c museCaller, peer MusePeer) bool {
	return m.Authorize != nil && m.Authorize(ctx, c.Actor, c.Project) == nil && musePeerAlive(peer)
}

func museAccountModes(body string) bool {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 5 {
		return false
	}
	for i, line := range lines {
		if !museAccountNode(line, i == 4) {
			return false
		}
	}
	return true
}

func museAccountNode(line string, marker bool) bool {
	parts := strings.SplitN(line, ":", 4)
	if len(parts) != 4 || parts[0] != "0" || parts[1] != "0" {
		return false
	}
	mode, err := strconv.ParseUint(parts[2], 8, 32)
	if err != nil || mode&0o022 != 0 {
		return false
	}
	if marker {
		return parts[3] == "regular file" && mode == 0o600
	}
	return parts[3] == "directory"
}

// RegisterNested binds only a current project-root registration. Records are
// deliberately ephemeral: a host restart requires fresh root authorization.
func (m *MuseRuntime) RegisterNested(ctx context.Context, peer MusePeer, in identity.NestedRegistration) error {
	if !museRegistrationValid(in) {
		return identity.ErrDenied
	}
	c, err := m.resolveProject(ctx, peer)
	if err != nil {
		return err
	}
	if !m.registrationAuthority(ctx, c, in.ActorID) {
		return identity.ErrDenied
	}
	record, err := m.registeredChild(ctx, c, in)
	if err != nil {
		return err
	}
	if err = m.validateNested(ctx, record); err != nil {
		return err
	}
	if !musePeerAlive(peer) {
		return identity.ErrDenied
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nested == nil {
		m.nested = map[string]museNested{}
	}
	m.nested[record.Namespace] = record
	return nil
}

func museRegistrationValid(in identity.NestedRegistration) bool {
	return containerID.MatchString(in.ChildID) && in.ActorID > 0 && terminalID.MatchString(in.RegistrationID) && in.Muse
}

func (m *MuseRuntime) registrationAuthority(ctx context.Context, c museCaller, actor int64) bool {
	if c.UID != 0 || m.NestedAuthorize == nil || m.NestedAuthorize(ctx, actor, c.Project) != nil {
		return false
	}
	ns, err := os.Readlink("/proc/" + strconv.Itoa(c.ProjectPID) + "/ns/pid")
	if err != nil || c.Namespace != ns {
		return false
	}
	_, err = m.actorAccount(ctx, c.Container, actor)
	return err == nil
}

func (m *MuseRuntime) registeredChild(ctx context.Context, c museCaller, in identity.NestedRegistration) (museNested, error) {
	var record museNested
	body, err := m.guest(ctx, c.Container, nil, "/usr/bin/podman", "--remote=false", "inspect", "--format", `{"id":{{json .ID}},"pid":{{json .State.Pid}},"running":{{json .State.Running}}}`, in.ChildID)
	pid, err := museChildPID(body, err, in.ChildID)
	if err != nil {
		return record, err
	}
	body, err = m.guest(ctx, c.Container, nil, "/usr/bin/readlink", "/proc/"+strconv.Itoa(pid)+"/ns/pid")
	if err != nil {
		return record, identity.ErrDenied
	}
	ns := strings.TrimSpace(string(body))
	if !strings.HasPrefix(ns, "pid:[") || ns == c.Namespace {
		return record, identity.ErrDenied
	}
	return museNested{Parent: c.Container, Project: c.Project, Child: in.ChildID, Namespace: ns, Registration: in.RegistrationID, Actor: in.ActorID, PID: pid, Muse: in.Muse}, nil
}

func museChildPID(body []byte, err error, id string) (int, error) {
	if err != nil || len(body) > 4096 {
		return 0, identity.ErrDenied
	}
	var v struct {
		ID      string `json:"id"`
		PID     int    `json:"pid"`
		Running bool   `json:"running"`
	}
	if strictjson.Decode(bytes.NewReader(body), &v) != nil || v.ID != id || !v.Running || v.PID <= 0 {
		return 0, identity.ErrDenied
	}
	return v.PID, nil
}

func (m *MuseRuntime) validateNested(ctx context.Context, r museNested) error {
	body, err := m.guest(ctx, r.Parent, nil, "/usr/bin/podman", "--remote=false", "inspect", "--format", `{{.ID}} {{.State.Pid}} {{.State.Running}}`, r.Child)
	if err != nil || strings.TrimSpace(string(body)) != r.Child+" "+strconv.Itoa(r.PID)+" true" {
		return identity.ErrDenied
	}
	if err = m.nestedNamespace(ctx, r); err != nil {
		return err
	}
	body, err = m.guest(ctx, r.Parent, nil, "/usr/bin/podman", "--remote=false", "inspect", "--format", `{{json .Mounts}}`, r.Child)
	if r.Muse && museReadonlyMount(body, err, "/run/soda-muse/nested/"+r.Registration, "/run/soda-muse/credentials") != nil {
		return identity.ErrDenied
	}
	return nil
}

func (m *MuseRuntime) nestedNamespace(ctx context.Context, r museNested) error {
	body, err := m.guest(ctx, r.Parent, nil, "/usr/bin/readlink", "/proc/"+strconv.Itoa(r.PID)+"/ns/pid")
	if err != nil || strings.TrimSpace(string(body)) != r.Namespace {
		return identity.ErrDenied
	}
	body, err = m.guest(ctx, r.Parent, nil, "/usr/bin/readlink", "/proc/"+strconv.Itoa(r.PID)+"/ns/user", "/proc/1/ns/user")
	if err != nil {
		return identity.ErrDenied
	}
	names := strings.Fields(string(body))
	if len(names) != 2 || names[0] != names[1] {
		return identity.ErrDenied
	}
	return nil
}

func museID() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	return hex.EncodeToString(b[:]), err
}

func museReadonlyMount(body []byte, err error, source, destination string) error {
	var mounts []struct {
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	}
	if err != nil || len(body) > 32768 || json.Unmarshal(body, &mounts) != nil {
		return identity.ErrDenied
	}
	for _, mount := range mounts {
		if mount.Source == source && mount.Destination == destination && !mount.RW {
			return nil
		}
	}
	return identity.ErrDenied
}

type museActorAccount struct {
	Username string
	Uid      string
	Gid      string
	HomeDir  string
}

func (m *MuseRuntime) actorAccount(ctx context.Context, container string, actor int64) (museActorAccount, error) {
	var account museActorAccount
	body, err := m.guest(ctx, container, nil, "/usr/local/bin/muse", "--soda-account", strconv.FormatInt(actor, 10))
	if err != nil || len(body) > 4096 {
		return account, identity.ErrDenied
	}
	if json.Unmarshal(body, &account) != nil || !loginName.MatchString(account.Username) || account.Username == "root" {
		return account, identity.ErrDenied
	}
	return account, nil
}

func (m *MuseRuntime) nestedCaller(ctx context.Context, c museCaller, peer MusePeer) (museCaller, error) {
	account, err := m.actorAccount(ctx, c.Container, c.Actor)
	if err != nil {
		return c, err
	}
	c.Login = account.Username
	c.Home = account.HomeDir
	mappings, err := os.ReadFile("/proc/" + strconv.Itoa(peer.PID) + "/gid_map")
	if err != nil {
		return c, identity.ErrDenied
	}
	c.GID, err = museMappedUID(string(mappings), peer.GID)
	if err != nil {
		return c, err
	}
	if !m.authorizedCaller(ctx, c, peer) {
		return c, identity.ErrDenied
	}
	return c, nil
}
