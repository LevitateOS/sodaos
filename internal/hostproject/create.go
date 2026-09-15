package hostproject

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/projectos"
	"github.com/levitateos/sodaos/internal/strictjson"
	"golang.org/x/crypto/ssh"
)

func (c Create) Validate() error {
	if !projectID.MatchString(c.ID) || c.Owner <= 0 || c.Profile == nil || c.Profile.Validate() != nil {
		return errors.New("invalid creation identity")
	}
	return nil
}

// Create provisions and starts a project environment. Callers that serialize
// mutations must acquire their admission gate before invoking this method.
func (r *Runtime) Create(ctx context.Context, in Create) (Environment, error) {
	if err := in.Validate(); err != nil {
		return Environment{}, err
	}
	if err := r.createContainer(ctx, in); err != nil {
		return Environment{}, err
	}
	return r.startCreated(ctx, in)
}

func (r *Runtime) createContainer(ctx context.Context, in Create) error {
	profile, err := r.ResolveProfile(ctx)
	if err != nil {
		return err
	}
	if profile != *in.Profile {
		return errors.New("installed profile changed; reservation retained")
	}
	encoded, _ := json.Marshal(profile)
	if _, err := r.podman(ctx, nil, "network", "exists", r.Config.Network); err != nil {
		if _, err = r.podman(ctx, nil, "network", "create", "--driver", "bridge", "--subnet", r.Config.Subnet, "--interface-name", r.Config.Bridge, r.Config.Network); err != nil {
			return err
		}
	}
	name := "soda-" + in.ID
	// No --replace or --rm: the writable userspace is a lasting environment.
	// NET_ADMIN lets nested netavark configure project-owned networking;
	// SYS_PTRACE lets the engine enter different-UID workload process namespaces.
	// Both are confined to the project's user namespace, not the appliance.
	args := []string{"create", "--name", name, "--label", "org.soda.project=" + in.ID, "--label", "org.soda.owner=" + strconv.FormatInt(in.Owner, 10), "--network", r.Config.Network, "--userns=auto:size=262144", "--systemd=always", "--cgroupns=private", "--cap-add=SYS_ADMIN,MKNOD,NET_ADMIN,SYS_PTRACE", "--device=/dev/fuse", "--security-opt=label=disable", "--label", "org.soda.profile=" + profile.ID, "--label", "org.soda.creation-profile=" + string(encoded), "--pull=never", profile.Image}
	_, err = r.podman(ctx, nil, args...)
	return err
}

type inspectItem struct {
	Image           string
	Config          struct{ Labels map[string]string }
	State           struct{ Running bool }
	NetworkSettings struct {
		Networks map[string]struct{ IPAddress string }
	}
}

func applyCreationProfile(env *Environment, item inspectItem) error {
	raw := item.Config.Labels["org.soda.creation-profile"]
	if raw == "" {
		return nil
	}
	p, decodeErr := projectos.Decode(raw)
	image := item.Image
	if !strings.HasPrefix(image, "sha256:") {
		image = "sha256:" + image
	}
	if decodeErr != nil || p.ID != item.Config.Labels["org.soda.profile"] || p.Image != image {
		return errors.New("native creation profile mismatch")
	}
	env.Profile = p
	return nil
}

func admitProjectIP(env *Environment, network, subnet string, networks map[string]struct{ IPAddress string }) error {
	env.IP = networks[network].IPAddress
	if env.IP == "" {
		return nil
	}
	ip, err := netip.ParseAddr(env.IP)
	if err != nil {
		return err
	}
	prefix, _ := netip.ParsePrefix(subnet)
	if !prefix.Contains(ip) {
		return errors.New("project IP outside configured network")
	}
	return nil
}

func inspectOwner(item inspectItem, id string) (int64, error) {
	if item.Config.Labels["org.soda.project"] != id {
		return 0, errors.New("container is not owned by this project")
	}
	owner, err := strconv.ParseInt(item.Config.Labels["org.soda.owner"], 10, 64)
	if err != nil || owner <= 0 {
		return 0, errors.New("invalid native project owner")
	}
	return owner, nil
}

func (r *Runtime) Inspect(ctx context.Context, id string) (Environment, int64, error) {
	env := Environment{ID: id}
	if !projectID.MatchString(id) {
		return env, 0, errors.New("invalid project id")
	}
	out, err := r.podman(ctx, nil, "inspect", "soda-"+id)
	if err != nil {
		return env, 0, err
	}
	var items []inspectItem
	if err = json.Unmarshal(out, &items); err != nil || len(items) != 1 {
		return env, 0, errors.New("invalid native inspection")
	}
	item := items[0]
	if imageID.MatchString(item.Image) {
		env.Image = "sha256:" + strings.TrimPrefix(item.Image, "sha256:")
	}
	owner, err := inspectOwner(item, id)
	if err != nil {
		return env, 0, err
	}
	if err = applyCreationProfile(&env, item); err != nil {
		return env, 0, err
	}
	env.Running = item.State.Running
	if err = admitProjectIP(&env, r.Config.Network, r.Config.Subnet, item.NetworkSettings.Networks); err != nil {
		return env, 0, err
	}
	return env, owner, nil
}

func (r *Runtime) waitProjectReady(ctx context.Context, name string) error {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		if _, err := r.podman(ctx, nil, "exec", name, "/usr/bin/test", "-f", "/run/soda-project-ready"); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

func (r *Runtime) startCreated(ctx context.Context, in Create) (Environment, error) {
	name := "soda-" + in.ID
	if _, err := r.Exec.Run(ctx, nil, "/usr/bin/systemctl", "enable", "--now", "soda-project@"+in.ID+".service"); err != nil {
		return Environment{}, err
	}
	if err := r.waitProjectReady(ctx, name); err != nil {
		return Environment{}, err
	}
	env, _, err := r.Inspect(ctx, in.ID)
	if err != nil {
		return env, err
	}
	if !env.Running || env.IP == "" || env.Profile == nil || *env.Profile != *in.Profile {
		return env, errors.New("project did not report the expected profile and running endpoint")
	}
	return env, nil
}

func canonicalizeAccountKeys(values []string) ([]string, error) {
	keys := []string{}
	for _, value := range values {
		key, _, options, rest, err := ssh.ParseAuthorizedKey([]byte(value))
		if err != nil || len(options) != 0 || len(bytes.TrimSpace(rest)) != 0 {
			return nil, errors.New("invalid public key")
		}
		keys = append(keys, string(ssh.MarshalAuthorizedKey(key)))
	}
	return keys, nil
}

func confirmAccount(out []byte, in Account) error {
	var result struct {
		Login    string `json:"login"`
		Identity int64  `json:"identity"`
	}
	if len(out) > 4096 || strictjson.Decode(bytes.NewReader(out), &result) != nil || result.Login != in.Login || result.Identity != in.Identity {
		return errors.New("native account identity was not confirmed")
	}
	return nil
}

func (r *Runtime) Account(ctx context.Context, in Account) error {
	if !loginName.MatchString(in.Login) || in.Login == "root" || in.Identity <= 0 || len(in.Keys) > 32 {
		return errors.New("invalid project account")
	}
	env, owner, err := r.Inspect(ctx, in.Project)
	if err != nil {
		return err
	}
	if !env.Running {
		return errors.New("project is stopped")
	}
	keys, err := canonicalizeAccountKeys(in.Keys)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{"login": in.Login, "identity": in.Identity, "admin": in.Identity == owner, "keys": keys})
	out, err := r.podman(ctx, body, "exec", "--interactive", "soda-"+in.Project, "/usr/libexec/soda/project-account")
	if err != nil {
		return err
	}
	return confirmAccount(out, in)
}

// Connection returns only the fixed public Ed25519 host key. No caller path or
// full container inspection crosses this boundary, and stopped containers stay stopped.
func (r *Runtime) Connection(ctx context.Context, id string) (Connection, error) {
	env, _, err := r.Inspect(ctx, id)
	result := Connection{Environment: env}
	if err != nil {
		return result, err
	}
	if !env.Running {
		return result, nil
	}
	data, err := r.podman(ctx, nil, "exec", "soda-"+id, "/usr/bin/head", "-c", "16385", "/etc/ssh/ssh_host_ed25519_key.pub")
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
