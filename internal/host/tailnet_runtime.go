package host

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
	"github.com/levitateos/sodaos/internal/tailnet"
)

// This observation deliberately excludes Config.Env and other credential-bearing
// full-inspection fields. Container names/labels alone are not a run identity.
const tailnetRunInspect = `{"id":{{json .ID}},"running":{{json .State.Running}},"pid":{{json .State.Pid}},"started":{{json .State.StartedAt}},"resolver":{{json .ResolvConfPath}}}`

type projectRun struct {
	Target   tailnet.RunTarget
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
		return nil, tailnet.ErrUnavailable
	}
	end := strings.LastIndexByte(string(data), ')')
	first := strings.IndexByte(string(data), ' ')
	if end < 0 || first < 0 || string(data[:first]) != strconv.Itoa(pid) {
		return nil, tailnet.ErrUnavailable
	}
	fields := strings.Fields(string(data[end+1:]))
	if len(fields) < 20 || fields[0] == "Z" || fields[0] == "X" {
		return nil, tailnet.ErrUnavailable
	}
	return fields, nil
}

func parseProcessStartTime(pid int, read func(string) ([]byte, error)) (string, error) {
	data, err := read("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return "", tailnet.ErrUnavailable
	}
	fields, err := parseStatFields(pid, data)
	if err != nil {
		return "", err
	}
	n, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil || n == 0 || strconv.FormatUint(n, 10) != fields[19] {
		return "", tailnet.ErrUnavailable
	}
	return fields[19], nil
}

func readProcessIDMap(base, name string, read func(string) ([]byte, error)) (uint32, error) {
	b, err := read(base + "/" + name)
	if err != nil || len(b) > 4096 {
		return 0, tailnet.ErrUnavailable
	}
	f := strings.Fields(string(b))
	if len(f) != 3 || !terminalIDMap([]string{strings.Join(f, ":")}) {
		return 0, tailnet.ErrUnsupported
	}
	n, _ := strconv.ParseUint(f[1], 10, 32)
	return uint32(n), nil
}

func parseNamespaceLink(name, v string) (string, error) {
	prefix := name + ":["
	if !strings.HasPrefix(v, prefix) || !strings.HasSuffix(v, "]") {
		return "", tailnet.ErrUnavailable
	}
	s := strings.TrimSuffix(strings.TrimPrefix(v, prefix), "]")
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n == 0 || strconv.FormatUint(n, 10) != s {
		return "", tailnet.ErrUnavailable
	}
	return v, nil
}

func readProcessNamespace(base, name string, link func(string) (string, error)) (string, error) {
	v, err := link(base + "/ns/" + name)
	if err != nil {
		return "", tailnet.ErrUnavailable
	}
	if _, err = parseNamespaceLink(name, v); err != nil {
		return "", err
	}
	host, err := link("/proc/1/ns/" + name)
	if err != nil || host == v {
		return "", tailnet.ErrUnsupported
	}
	return v, nil
}

func readSystemBootID(read func(string) ([]byte, error)) (string, error) {
	boot, err := read("/proc/sys/kernel/random/boot_id")
	s := strings.TrimSpace(string(boot))
	if err != nil || len(s) != 36 || strings.Count(s, "-") != 4 {
		return "", tailnet.ErrUnavailable
	}
	compact := strings.ReplaceAll(s, "-", "")
	if _, err = hex.DecodeString(compact); err != nil || len(compact) != 32 {
		return "", tailnet.ErrUnavailable
	}
	return s, nil
}

func processRunIdentity(pid int, read func(string) ([]byte, error), link func(string) (string, error)) (processIdentity, error) {
	var out processIdentity
	if pid <= 1 {
		return out, tailnet.ErrConflict
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
		return raw, tailnet.ErrConflict
	}
	return raw, nil
}

func (d *Daemon) confirmProjectRunIdentity(ctx context.Context, cid string, data []byte, pid int, identity processIdentity) error {
	again, e := d.inspectProjectRun(ctx, cid)
	if e != nil || string(again) != string(data) {
		return tailnet.ErrConflict
	}
	second, e := processRunIdentity(pid, os.ReadFile, os.Readlink)
	if e != nil || second != identity {
		return tailnet.ErrConflict
	}
	return nil
}

func (d *Daemon) inspectProjectRun(ctx context.Context, cid string) ([]byte, error) {
	return d.podman(ctx, nil, "--remote=false", "inspect", "--format", tailnetRunInspect, cid)
}

func assembleProjectRun(id, cid, resolver string, raw projectRunInspect, identity processIdentity) projectRun {
	b, _ := json.Marshal(struct {
		CID, Started string
		PID          int
		Identity     processIdentity
	}{cid, raw.Started, raw.PID, identity})
	sum := sha256.Sum256(b)
	return projectRun{Target: tailnet.RunTarget{Project: id, Container: cid, Run: hex.EncodeToString(sum[:])}, PID: raw.PID, Started: raw.Started, UserNS: identity.UserNS, NetNS: identity.NetNS, UID: identity.UID, GID: identity.GID, Resolver: resolver}
}

func admitProjectRunSnapshot(cid string, raw projectRunInspect) (string, processIdentity, error) {
	started, e := time.Parse(time.RFC3339Nano, raw.Started)
	if e != nil || started.IsZero() {
		return "", processIdentity{}, tailnet.ErrUnavailable
	}
	// Only the selected native rootful storage resolver is considered. Custom
	// resolver sources/storage layouts are unsupported, not a path-repair task.
	resolver := "/var/lib/containers/storage/overlay-containers/" + cid + "/userdata/resolv.conf"
	if raw.Resolver != resolver {
		return "", processIdentity{}, tailnet.ErrUnsupported
	}
	identity, e := processRunIdentity(raw.PID, os.ReadFile, os.Readlink)
	if e != nil {
		return "", processIdentity{}, e
	}
	return resolver, identity, nil
}

func (d *Daemon) projectRun(ctx context.Context, id string) (projectRun, error) {
	var out projectRun
	cid, e := d.projectContainer(ctx, id, true)
	if e != nil {
		return out, tailnet.ErrUnavailable
	}
	data, e := d.inspectProjectRun(ctx, cid)
	if e != nil || len(data) > 4096 {
		return out, tailnet.ErrUnavailable
	}
	raw, e := decodeProjectRunInspect(data, cid)
	if e != nil {
		return out, e
	}
	resolver, identity, e := admitProjectRunSnapshot(cid, raw)
	if e != nil {
		return out, e
	}
	if e = d.confirmProjectRunIdentity(ctx, cid, data, raw.PID, identity); e != nil {
		return out, e
	}
	current, e := d.projectContainer(ctx, id, true)
	if e != nil || current != cid {
		return out, tailnet.ErrConflict
	}
	return assembleProjectRun(id, cid, resolver, raw, identity), nil
}

// The immutable image ID is host configuration, never a browser or project field.
// This recipe uses upstream Podman namespace joining rather than a setns launcher.
// Creating a recipe does not claim installed TUN, resolver or namespace proof.
func companionCreateArgs(run projectRun, image string) ([]string, error) {
	if !projectID.MatchString(run.Target.Project) || !containerID.MatchString(run.Target.Container) || !containerID.MatchString(run.Target.Run) || !strings.HasPrefix(image, "sha256:") || !imageID.MatchString(image) || run.UID == 0 || run.GID == 0 {
		return nil, tailnet.ErrInvalid
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

func (d *Daemon) recheckProjectRun(ctx context.Context, before projectRun) error {
	after, e := d.projectRun(ctx, before.Target.Project)
	if e != nil || after != before {
		return tailnet.ErrConflict
	}
	return nil
}
