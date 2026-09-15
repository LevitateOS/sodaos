package tailnet

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
	domain "github.com/levitateos/sodaos/internal/tailnet"
)

// This observation deliberately excludes Config.Env and other credential-bearing
// full-inspection fields. Container names/labels alone are not a run identity.
const tailnetRunInspect = `{"id":{{json .ID}},"running":{{json .State.Running}},"pid":{{json .State.Pid}},"started":{{json .State.StartedAt}},"resolver":{{json .ResolvConfPath}}}`

type projectRun struct {
	Target   domain.RunTarget
	PID      int
	Started  string
	UserNS   string
	NetNS    string
	UID      uint32
	GID      uint32
	Resolver string
}
type processIdentity struct {
	Start, UserNS, NetNS, Boot string
	UID, GID                   uint32
}

func parseStatFields(pid int, data []byte) ([]string, error) {
	if len(data) > 8192 {
		return nil, domain.ErrUnavailable
	}
	end := strings.LastIndexByte(string(data), ')')
	first := strings.IndexByte(string(data), ' ')
	if end < 0 || first < 0 || string(data[:first]) != strconv.Itoa(pid) {
		return nil, domain.ErrUnavailable
	}
	fields := strings.Fields(string(data[end+1:]))
	if len(fields) < 20 || fields[0] == "Z" || fields[0] == "X" {
		return nil, domain.ErrUnavailable
	}
	return fields, nil
}

func parseProcessStartTime(pid int, read func(string) ([]byte, error)) (string, error) {
	data, err := read("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return "", domain.ErrUnavailable
	}
	fields, err := parseStatFields(pid, data)
	if err != nil {
		return "", err
	}
	n, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil || n == 0 || strconv.FormatUint(n, 10) != fields[19] {
		return "", domain.ErrUnavailable
	}
	return fields[19], nil
}

func readProcessIDMap(base, name string, read func(string) ([]byte, error)) (uint32, error) {
	b, err := read(base + "/" + name)
	if err != nil || len(b) > 4096 {
		return 0, domain.ErrUnavailable
	}
	f := strings.Fields(string(b))
	if len(f) != 3 || !idMap([]string{strings.Join(f, ":")}) {
		return 0, domain.ErrUnsupported
	}
	n, _ := strconv.ParseUint(f[1], 10, 32)
	return uint32(n), nil
}

func parseNamespaceLink(name, v string) (string, error) {
	prefix := name + ":["
	if !strings.HasPrefix(v, prefix) || !strings.HasSuffix(v, "]") {
		return "", domain.ErrUnavailable
	}
	s := strings.TrimSuffix(strings.TrimPrefix(v, prefix), "]")
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n == 0 || strconv.FormatUint(n, 10) != s {
		return "", domain.ErrUnavailable
	}
	return v, nil
}

func readProcessNamespace(base, name string, link func(string) (string, error)) (string, error) {
	v, err := link(base + "/ns/" + name)
	if err != nil {
		return "", domain.ErrUnavailable
	}
	if _, err = parseNamespaceLink(name, v); err != nil {
		return "", err
	}
	host, err := link("/proc/1/ns/" + name)
	if err != nil || host == v {
		return "", domain.ErrUnsupported
	}
	return v, nil
}

func readSystemBootID(read func(string) ([]byte, error)) (string, error) {
	boot, err := read("/proc/sys/kernel/random/boot_id")
	s := strings.TrimSpace(string(boot))
	if err != nil || len(s) != 36 || strings.Count(s, "-") != 4 {
		return "", domain.ErrUnavailable
	}
	compact := strings.ReplaceAll(s, "-", "")
	if _, err = hex.DecodeString(compact); err != nil || len(compact) != 32 {
		return "", domain.ErrUnavailable
	}
	return s, nil
}

func processRunIdentity(pid int, read func(string) ([]byte, error), link func(string) (string, error)) (processIdentity, error) {
	var out processIdentity
	if pid <= 1 {
		return out, domain.ErrConflict
	}
	var err error
	if out.Start, err = parseProcessStartTime(pid, read); err != nil {
		return out, err
	}
	base := "/proc/" + strconv.Itoa(pid)
	if out.UID, err = readProcessIDMap(base, "uid_map", read); err != nil {
		return out, err
	}
	if out.GID, err = readProcessIDMap(base, "gid_map", read); err != nil {
		return out, err
	}
	if out.UserNS, err = readProcessNamespace(base, "user", link); err != nil {
		return out, err
	}
	if out.NetNS, err = readProcessNamespace(base, "net", link); err != nil {
		return out, err
	}
	out.Boot, err = readSystemBootID(read)
	return out, err
}

type projectRunInspect struct {
	ID       string `json:"id"`
	Running  bool   `json:"running"`
	PID      int    `json:"pid"`
	Started  string `json:"started"`
	Resolver string `json:"resolver"`
}

func decodeProjectRunInspect(data []byte, cid string) (projectRunInspect, error) {
	var raw projectRunInspect
	if strictjson.Decode(strings.NewReader(string(data)), &raw) != nil || raw.ID != cid || !raw.Running || raw.PID <= 1 {
		return raw, domain.ErrConflict
	}
	return raw, nil
}

func (c *Companion) confirmProjectRunIdentity(ctx context.Context, cid string, data []byte, pid int, identity processIdentity) error {
	again, e := c.inspectProjectRun(ctx, cid)
	if e != nil || string(again) != string(data) {
		return domain.ErrConflict
	}
	second, e := processRunIdentity(pid, os.ReadFile, os.Readlink)
	if e != nil || second != identity {
		return domain.ErrConflict
	}
	return nil
}

func (c *Companion) inspectProjectRun(ctx context.Context, cid string) ([]byte, error) {
	return c.podman(ctx, nil, "--remote=false", "inspect", "--format", tailnetRunInspect, cid)
}

func assembleProjectRun(id, cid, resolver string, raw projectRunInspect, identity processIdentity) projectRun {
	b, _ := json.Marshal(struct {
		CID, Started string
		PID          int
		Identity     processIdentity
	}{cid, raw.Started, raw.PID, identity})
	sum := sha256.Sum256(b)
	return projectRun{Target: domain.RunTarget{Project: id, Container: cid, Run: hex.EncodeToString(sum[:])}, PID: raw.PID, Started: raw.Started, UserNS: identity.UserNS, NetNS: identity.NetNS, UID: identity.UID, GID: identity.GID, Resolver: resolver}
}

func admitProjectRunSnapshot(cid string, raw projectRunInspect) (string, processIdentity, error) {
	started, e := time.Parse(time.RFC3339Nano, raw.Started)
	if e != nil || started.IsZero() {
		return "", processIdentity{}, domain.ErrUnavailable
	}
	// Only the selected native rootful storage resolver is considered. Custom
	// resolver sources/storage layouts are unsupported, not a path-repair task.
	resolver := "/var/lib/containers/storage/overlay-containers/" + cid + "/userdata/resolv.conf"
	if raw.Resolver != resolver {
		return "", processIdentity{}, domain.ErrUnsupported
	}
	identity, e := processRunIdentity(raw.PID, os.ReadFile, os.Readlink)
	if e != nil {
		return "", processIdentity{}, e
	}
	return resolver, identity, nil
}

func (c *Companion) projectRun(ctx context.Context, id string) (projectRun, error) {
	var out projectRun
	cid, e := c.projectContainer(ctx, id, true)
	if e != nil {
		return out, domain.ErrUnavailable
	}
	data, e := c.inspectProjectRun(ctx, cid)
	if e != nil || len(data) > 4096 {
		return out, domain.ErrUnavailable
	}
	raw, e := decodeProjectRunInspect(data, cid)
	if e != nil {
		return out, e
	}
	resolver, identity, e := admitProjectRunSnapshot(cid, raw)
	if e != nil {
		return out, e
	}
	if e = c.confirmProjectRunIdentity(ctx, cid, data, raw.PID, identity); e != nil {
		return out, e
	}
	current, e := c.projectContainer(ctx, id, true)
	if e != nil || current != cid {
		return out, domain.ErrConflict
	}
	return assembleProjectRun(id, cid, resolver, raw, identity), nil
}

// The immutable image ID is host configuration, never a browser or project field.
// This recipe uses upstream Podman namespace joining rather than a setns launcher.
// Creating a recipe does not claim installed TUN, resolver or namespace proof.
func companionCreateArgs(run projectRun, image string) ([]string, error) {
	if !projectID.MatchString(run.Target.Project) || !containerID.MatchString(run.Target.Container) || !containerID.MatchString(run.Target.Run) || !strings.HasPrefix(image, "sha256:") || !imageID.MatchString(image) || run.UID == 0 || run.GID == 0 {
		return nil, domain.ErrInvalid
	}
	base := filepath.Join("/run/soda-tailnet", run.Target.Project, run.Target.Run)
	args := []string{
		"--remote=false", "create", "--name", "soda-tailnet-" + run.Target.Project + "-" + run.Target.Run,
		"--label", "org.soda.tailnet.project=" + run.Target.Project, "--label", "org.soda.tailnet.run=" + run.Target.Run,
		"--label", "org.soda.tailnet.parent=" + run.Target.Container,
		"--userns=container:" + run.Target.Container, "--network=container:" + run.Target.Container,
		"--pid=private", "--ipc=private", "--uts=private", "--cgroupns=private", "--user=0:0",
		"--cap-drop=ALL", "--cap-add=NET_ADMIN", "--device=/dev/net/tun", "--security-opt=label=disable",
		"--no-hosts", "--log-driver=none", "--pull=never",
		"--volume", base + "/control:/run/tailscale:rw",
		"--volume", base + "/input:/run/soda-enrollment:ro",
		"--entrypoint=/usr/local/bin/tailscaled", image,
		"--state=mem:", "--socket=/run/tailscale/tailscaled.sock", "--tun=tailscale0", "--no-logs-no-support",
	}
	return args, nil
}

func (c *Companion) recheckProjectRun(ctx context.Context, before projectRun) error {
	after, e := c.projectRun(ctx, before.Target.Project)
	if e != nil || after != before {
		return domain.ErrConflict
	}
	return nil
}

const projectInspectFormat = `{"id":{{json .ID}},"running":{{json .State.Running}},"project":{{json (index .Config.Labels "org.soda.project")}},"owner":{{json (index .Config.Labels "org.soda.owner")}},"privileged":{{json .HostConfig.Privileged}},"userns":{{json .HostConfig.UsernsMode}},"mappings":{{json .HostConfig.IDMappings}}}`

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

func idMap(values []string) bool {
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

func projectIsolation(v projectInspection, id string) bool {
	if !containerID.MatchString(v.ID) || v.Project != id || v.Privileged || v.Userns != "private" {
		return false
	}
	return idMap(v.Mappings.UIDMap) && idMap(v.Mappings.GIDMap)
}

func (c *Companion) inspectProject(ctx context.Context, id string) (projectInspection, error) {
	var v projectInspection
	if !projectID.MatchString(id) {
		return v, errors.New("invalid project")
	}
	data, err := c.podman(ctx, nil, "--remote=false", "inspect", "--format", projectInspectFormat, "soda-"+id)
	if err != nil || len(data) > 4096 {
		return v, errors.New("terminal inspection unavailable")
	}
	if err = strictjson.Decode(bytes.NewReader(data), &v); err != nil {
		return v, errors.New("invalid terminal inspection")
	}
	owner, err := strconv.ParseInt(v.Owner, 10, 64)
	if err != nil || owner <= 0 || !projectIsolation(v, id) {
		return v, errors.New("terminal target not ready or isolated")
	}
	return v, nil
}

func (c *Companion) projectContainer(ctx context.Context, id string, requireRunning bool) (string, error) {
	v, err := c.inspectProject(ctx, id)
	if err != nil {
		return "", err
	}
	if requireRunning && !v.Running {
		return "", errors.New("terminal target not ready or isolated")
	}
	return v.ID, nil
}

func (c *Companion) projectRunning(ctx context.Context, id string) (bool, error) {
	v, err := c.inspectProject(ctx, id)
	if err != nil {
		return false, err
	}
	return v.Running, nil
}
