package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var projectID = regexp.MustCompile(`^p[0-9a-f]{24}$`)
var loginName = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)
var networkName = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)

type Config struct {
	Image   string `json:"image"`
	Network string `json:"network"`
	Subnet  string `json:"subnet"`
	Bridge  string `json:"bridge"`
}

func LoadConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err = d.Decode(&c); err != nil {
		return c, err
	}
	if _, err = netip.ParsePrefix(c.Subnet); err != nil {
		return c, err
	}
	if c.Image == "" || strings.HasPrefix(c.Image, "-") || !networkName.MatchString(c.Network) || !networkName.MatchString(c.Bridge) {
		return c, errors.New("invalid native runtime configuration")
	}
	return c, nil
}

type Executor interface {
	Run(context.Context, []byte, string, ...string) ([]byte, error)
}
type Native struct{}

func (Native) Run(ctx context.Context, in []byte, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = bytes.NewReader(in)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s failed: %w: %s", command, err, stderr.String())
	}
	return out, nil
}

type Daemon struct {
	Config         Config
	Exec           Executor
	mu             sync.Mutex
	terminalMu     sync.Mutex
	terminals      map[*http.Request]context.CancelFunc
	terminalClosed bool
	terminalWG     sync.WaitGroup
}

func (d *Daemon) podman(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	return d.Exec.Run(ctx, in, "/usr/bin/podman", args...)
}
func (d *Daemon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/terminal" {
		d.terminalHandler(w, r)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "POST required", 405)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	decode := func(v any) error {
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(v); err != nil {
			return err
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			return errors.New("one object required")
		}
		return nil
	}
	// Serialize the small native mutations; this is not a persistent workflow engine.
	d.mu.Lock()
	defer d.mu.Unlock()
	var out any
	var err error
	switch r.URL.Path {
	case "/create":
		var in Create
		if err = decode(&in); err == nil {
			if !projectID.MatchString(in.ID) || in.Owner <= 0 {
				err = errors.New("invalid project identity")
			} else {
				out, err = d.create(ctx, in)
			}
		}
	case "/inspect":
		var in Create
		if err = decode(&in); err == nil {
			out, _, err = d.inspect(ctx, in.ID)
		}
	case "/connection":
		var in Create
		if err = decode(&in); err == nil {
			out, err = d.connection(ctx, in.ID)
		}
	case "/account":
		var in Account
		if err = decode(&in); err == nil {
			err = d.account(ctx, in)
		}
		out = map[string]bool{"ok": err == nil}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		slog.Error("project native operation failed", "operation", r.URL.Path, "error", err)
		http.Error(w, "native operation failed; inspect operator journal", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
func (d *Daemon) create(ctx context.Context, in Create) (Environment, error) {
	if _, err := d.podman(ctx, nil, "network", "exists", d.Config.Network); err != nil {
		if _, err = d.podman(ctx, nil, "network", "create", "--driver", "bridge", "--subnet", d.Config.Subnet, "--interface-name", d.Config.Bridge, d.Config.Network); err != nil {
			return Environment{}, err
		}
	}
	name := "soda-" + in.ID
	// No --replace or --rm: the writable userspace is a lasting environment.
	// NET_ADMIN lets nested netavark configure project-owned networking;
	// SYS_PTRACE lets the engine enter different-UID workload process namespaces.
	// Both are confined to the project's user namespace, not the appliance.
	args := []string{"create", "--name", name, "--label", "org.soda.project=" + in.ID, "--label", "org.soda.owner=" + strconv.FormatInt(in.Owner, 10), "--network", d.Config.Network, "--userns=auto:size=262144", "--systemd=always", "--cgroupns=private", "--cap-add=SYS_ADMIN,MKNOD,NET_ADMIN,SYS_PTRACE", "--device=/dev/fuse", "--security-opt=label=disable", d.Config.Image}
	if _, err := d.podman(ctx, nil, args...); err != nil {
		return Environment{}, err
	}
	if _, err := d.Exec.Run(ctx, nil, "/usr/bin/systemctl", "enable", "--now", "soda-project@"+in.ID+".service"); err != nil {
		return Environment{}, err
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		if _, err := d.podman(ctx, nil, "exec", name, "/usr/bin/test", "-f", "/run/soda-project-ready"); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return Environment{}, ctx.Err()
		case <-tick.C:
		}
	}
	env, _, err := d.inspect(ctx, in.ID)
	if err != nil {
		return env, err
	}
	if !env.Running || env.IP == "" {
		return env, errors.New("project did not report a running endpoint")
	}
	return env, nil
}
func (d *Daemon) inspect(ctx context.Context, id string) (Environment, int64, error) {
	env := Environment{ID: id}
	if !projectID.MatchString(id) {
		return env, 0, errors.New("invalid project id")
	}
	out, err := d.podman(ctx, nil, "inspect", "soda-"+id)
	if err != nil {
		return env, 0, err
	}
	var items []struct {
		Config          struct{ Labels map[string]string }
		State           struct{ Running bool }
		NetworkSettings struct {
			Networks map[string]struct{ IPAddress string }
		}
	}
	if err = json.Unmarshal(out, &items); err != nil || len(items) != 1 {
		return env, 0, errors.New("invalid native inspection")
	}
	item := items[0]
	if item.Config.Labels["org.soda.project"] != id {
		return env, 0, errors.New("container is not owned by this project")
	}
	owner, err := strconv.ParseInt(item.Config.Labels["org.soda.owner"], 10, 64)
	if err != nil || owner <= 0 {
		return env, 0, errors.New("invalid native project owner")
	}
	env.Running = item.State.Running
	env.IP = item.NetworkSettings.Networks[d.Config.Network].IPAddress
	if env.IP != "" {
		ip, err := netip.ParseAddr(env.IP)
		if err != nil {
			return env, 0, err
		}
		prefix, _ := netip.ParsePrefix(d.Config.Subnet)
		if !prefix.Contains(ip) {
			return env, 0, errors.New("project IP outside configured network")
		}
	}
	return env, owner, nil
}

// Only the fixed public Ed25519 host key is readable. No caller path or full
// container inspection crosses this boundary, and stopped containers stay stopped.
func (d *Daemon) connection(ctx context.Context, id string) (Connection, error) {
	env, _, err := d.inspect(ctx, id)
	result := Connection{Environment: env}
	if err != nil {
		return result, err
	}
	if !env.Running {
		return result, nil
	}
	data, err := d.podman(ctx, nil, "exec", "soda-"+id, "/usr/bin/head", "-c", "16385", "/etc/ssh/ssh_host_ed25519_key.pub")
	if err != nil || len(data) > 16384 {
		return result, errors.New("public host key unavailable")
	}
	key, _, options, rest, err := ssh.ParseAuthorizedKey(data)
	if err != nil || key.Type() != ssh.KeyAlgoED25519 || len(options) != 0 || len(bytes.TrimSpace(rest)) != 0 {
		return result, errors.New("invalid public host key")
	}
	result.HostKey = string(ssh.MarshalAuthorizedKey(key))
	result.Fingerprint = ssh.FingerprintSHA256(key)
	return result, nil
}

func (d *Daemon) account(ctx context.Context, in Account) error {
	if !loginName.MatchString(in.Login) || in.Identity <= 0 || len(in.Keys) == 0 || len(in.Keys) > 32 {
		return errors.New("invalid project account")
	}
	env, owner, err := d.inspect(ctx, in.Project)
	if err != nil {
		return err
	}
	if !env.Running {
		return errors.New("project is stopped")
	}
	keys := []string{}
	for _, value := range in.Keys {
		key, _, options, rest, err := ssh.ParseAuthorizedKey([]byte(value))
		if err != nil || len(options) != 0 || len(bytes.TrimSpace(rest)) != 0 {
			return errors.New("invalid public key")
		}
		keys = append(keys, string(ssh.MarshalAuthorizedKey(key)))
	}
	body, _ := json.Marshal(map[string]any{"login": in.Login, "identity": in.Identity, "admin": in.Identity == owner, "keys": keys})
	_, err = d.podman(ctx, body, "exec", "--interactive", "soda-"+in.Project, "/usr/libexec/soda/project-account")
	return err
}
