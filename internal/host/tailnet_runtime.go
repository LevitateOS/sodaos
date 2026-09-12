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

func processRunIdentity(pid int, read func(string) ([]byte, error), link func(string) (string, error)) (processIdentity, error) {
	var out processIdentity
	if pid <= 1 {
		return out, tailnet.ErrConflict
	}
	base := "/proc/" + strconv.Itoa(pid)
	data, e := read(base + "/stat")
	if e != nil || len(data) > 8192 {
		return out, tailnet.ErrUnavailable
	}
	// comm may contain spaces/parentheses. The final ')' terminates field 2;
	// starttime is field 22, offset 19 after it. Require the expected PID too.
	end := strings.LastIndexByte(string(data), ')')
	first := strings.IndexByte(string(data), ' ')
	if end < 0 || first < 0 || string(data[:first]) != strconv.Itoa(pid) {
		return out, tailnet.ErrUnavailable
	}
	fields := strings.Fields(string(data[end+1:]))
	if len(fields) < 20 || fields[0] == "Z" || fields[0] == "X" {
		return out, tailnet.ErrUnavailable
	}
	n, e := strconv.ParseUint(fields[19], 10, 64)
	if e != nil || n == 0 || strconv.FormatUint(n, 10) != fields[19] {
		return out, tailnet.ErrUnavailable
	}
	out.Start = fields[19]
	mapping := func(name string) (uint32, error) {
		b, e := read(base + "/" + name)
		if e != nil || len(b) > 4096 {
			return 0, tailnet.ErrUnavailable
		}
		f := strings.Fields(string(b))
		if len(f) != 3 || !terminalIDMap([]string{strings.Join(f, ":")}) {
			return 0, tailnet.ErrUnsupported
		}
		n, _ := strconv.ParseUint(f[1], 10, 32)
		return uint32(n), nil
	}
	if out.UID, e = mapping("uid_map"); e != nil {
		return out, e
	}
	if out.GID, e = mapping("gid_map"); e != nil {
		return out, e
	}
	namespace := func(name string) (string, error) {
		v, e := link(base + "/ns/" + name)
		prefix := name + ":["
		if e != nil || !strings.HasPrefix(v, prefix) || !strings.HasSuffix(v, "]") {
			return "", tailnet.ErrUnavailable
		}
		s := strings.TrimSuffix(strings.TrimPrefix(v, prefix), "]")
		n, e := strconv.ParseUint(s, 10, 64)
		if e != nil || n == 0 || strconv.FormatUint(n, 10) != s {
			return "", tailnet.ErrUnavailable
		}
		host, e := link("/proc/1/ns/" + name)
		if e != nil || host == v {
			return "", tailnet.ErrUnsupported
		}
		return v, nil
	}
	if out.UserNS, e = namespace("user"); e != nil {
		return out, e
	}
	if out.NetNS, e = namespace("net"); e != nil {
		return out, e
	}
	boot, e := read("/proc/sys/kernel/random/boot_id")
	s := strings.TrimSpace(string(boot))
	if e != nil || len(s) != 36 || strings.Count(s, "-") != 4 {
		return out, tailnet.ErrUnavailable
	}
	compact := strings.ReplaceAll(s, "-", "")
	if _, e = hex.DecodeString(compact); e != nil || len(compact) != 32 {
		return out, tailnet.ErrUnavailable
	}
	out.Boot = s
	return out, nil
}

func (d *Daemon) projectRun(ctx context.Context, id string) (projectRun, error) {
	var out projectRun
	cid, e := d.projectContainer(ctx, id, true)
	if e != nil {
		return out, tailnet.ErrUnavailable
	}
	observe := func() ([]byte, error) {
		return d.podman(ctx, nil, "--remote=false", "inspect", "--format", tailnetRunInspect, cid)
	}
	data, e := observe()
	if e != nil || len(data) > 4096 {
		return out, tailnet.ErrUnavailable
	}
	var raw struct {
		ID       string `json:"id"`
		Running  bool   `json:"running"`
		PID      int    `json:"pid"`
		Started  string `json:"started"`
		Resolver string `json:"resolver"`
	}
	if strictjson.Decode(strings.NewReader(string(data)), &raw) != nil || raw.ID != cid || !raw.Running || raw.PID <= 1 {
		return out, tailnet.ErrConflict
	}
	started, e := time.Parse(time.RFC3339Nano, raw.Started)
	if e != nil || started.IsZero() {
		return out, tailnet.ErrUnavailable
	}
	// Only the selected native rootful storage resolver is considered. Custom
	// resolver sources/storage layouts are unsupported, not a path-repair task.
	resolver := "/var/lib/containers/storage/overlay-containers/" + cid + "/userdata/resolv.conf"
	if raw.Resolver != resolver {
		return out, tailnet.ErrUnsupported
	}
	identity, e := processRunIdentity(raw.PID, os.ReadFile, os.Readlink)
	if e != nil {
		return out, e
	}
	again, e := observe()
	if e != nil || string(again) != string(data) {
		return out, tailnet.ErrConflict
	}
	second, e := processRunIdentity(raw.PID, os.ReadFile, os.Readlink)
	if e != nil || second != identity {
		return out, tailnet.ErrConflict
	}
	current, e := d.projectContainer(ctx, id, true)
	if e != nil || current != cid {
		return out, tailnet.ErrConflict
	}
	b, _ := json.Marshal(struct {
		CID, Started string
		PID          int
		Identity     processIdentity
	}{cid, raw.Started, raw.PID, identity})
	sum := sha256.Sum256(b)
	out = projectRun{Target: tailnet.RunTarget{Project: id, Container: cid, Run: hex.EncodeToString(sum[:])}, PID: raw.PID, Started: raw.Started, UserNS: identity.UserNS, NetNS: identity.NetNS, UID: identity.UID, GID: identity.GID, Resolver: resolver}
	return out, nil
}

// The immutable image ID is host configuration, never a browser or project field.
// This recipe uses upstream Podman namespace joining rather than a setns launcher.
// Creating a recipe does not claim installed TUN, resolver or namespace proof.
func companionCreateArgs(run projectRun, image string) ([]string, error) {
	if !projectID.MatchString(run.Target.Project) || !containerID.MatchString(run.Target.Container) || !containerID.MatchString(run.Target.Run) || !strings.HasPrefix(image, "sha256:") || !imageID.MatchString(image) || run.UID == 0 || run.GID == 0 {
		return nil, tailnet.ErrInvalid
	}
	base := filepath.Join("/run/soda-tailnet", run.Target.Project, run.Target.Run)
	args := []string{"--remote=false", "create", "--name", "soda-tailnet-" + run.Target.Project + "-" + run.Target.Run,
		"--label", "org.soda.tailnet.project=" + run.Target.Project, "--label", "org.soda.tailnet.run=" + run.Target.Run,
		"--label", "org.soda.tailnet.parent=" + run.Target.Container,
		"--userns=container:" + run.Target.Container, "--network=container:" + run.Target.Container,
		"--pid=private", "--ipc=private", "--uts=private", "--cgroupns=private", "--user=0:0",
		"--cap-drop=ALL", "--cap-add=NET_ADMIN", "--device=/dev/net/tun", "--security-opt=label=disable",
		"--no-hosts", "--log-driver=none", "--pull=never",
		"--volume", base + "/state:/var/lib/tailscale:rw",
		"--volume", base + "/control:/run/tailscale:rw",
		"--volume", base + "/input:/run/soda-enrollment:ro",
		"--entrypoint=/usr/local/bin/tailscaled", image,
		"--state=/var/lib/tailscale/tailscaled.state", "--socket=/run/tailscale/tailscaled.sock", "--tun=tailscale0", "--no-logs-no-support"}
	return args, nil
}

func (d *Daemon) recheckProjectRun(ctx context.Context, before projectRun) error {
	after, e := d.projectRun(ctx, before.Target.Project)
	if e != nil || after != before {
		return tailnet.ErrConflict
	}
	return nil
}
