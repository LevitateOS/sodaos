package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/installlayout"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/projectos"
	"github.com/levitateos/sodaos/internal/runners"
	"github.com/levitateos/sodaos/internal/strictjson"
	"github.com/levitateos/sodaos/internal/tailnet"
	"golang.org/x/crypto/ssh"
)

var (
	projectID   = regexp.MustCompile(`^p[0-9a-f]{24}$`)
	loginName   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)
	imageID     = regexp.MustCompile(`^(?:sha256:)?[0-9a-f]{64}$`)
	networkName = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)
)

type Config struct {
	TailnetManagement bool   `json:"tailnet_management,omitempty"`
	TailnetImage      string `json:"tailnet_image,omitempty"`
	Image             string `json:"image"`
	Network           string `json:"network"`
	Subnet            string `json:"subnet"`
	Bridge            string `json:"bridge"`
}

func LoadConfig(path string) (Config, error) {
	return loadConfig(path, installlayout.Release)
}

func applyReleaseImages(c *Config, releasePath string) error {
	if releasePath == "" {
		return nil
	}
	p, err := appliancerelease.Load(releasePath)
	if err != nil || nativebuild.RequireNative(p.Architecture) != nil {
		return errors.New("immutable appliance image defaults unavailable")
	}
	// Never silently replace an operator's saved image choice. New installs
	// omit these fields; a conflicting old install needs explicit migration.
	project, companion := p.Images["project-os"].Config, p.Images["tailnet"].Config
	if (c.Image != "" && c.Image != project) || (c.TailnetImage != "" && c.TailnetImage != companion) {
		return errors.New("saved image selection conflicts with appliance release; explicit migration required")
	}
	c.Image = project
	if c.TailnetManagement {
		c.TailnetImage = companion
	}
	return nil
}

func validTailnetImage(c Config) bool {
	if c.TailnetImage == "" {
		return true
	}
	return c.TailnetManagement && strings.HasPrefix(c.TailnetImage, "sha256:") && imageID.MatchString(c.TailnetImage)
}

func validNetworkNames(c Config) bool {
	return c.Image != "" && !strings.HasPrefix(c.Image, "-") && networkName.MatchString(c.Network) && networkName.MatchString(c.Bridge)
}

func validateRuntimeConfig(c Config) error {
	if _, err := netip.ParsePrefix(c.Subnet); err != nil {
		return err
	}
	if !validTailnetImage(c) {
		return errors.New("invalid immutable Tailnet companion configuration")
	}
	if !validNetworkNames(c) {
		return errors.New("invalid native runtime configuration")
	}
	return nil
}

func loadConfig(path, releasePath string) (Config, error) {
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
	if err = applyReleaseImages(&c, releasePath); err != nil {
		return c, err
	}
	if err = validateRuntimeConfig(c); err != nil {
		return c, err
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
	Tailnet *tailnet.Management
	Runners *runners.Operations
	Config  Config
	Exec    Executor
	// tailnetEnabledCheck stubs the Off pre-check in tests. Production leaves
	// it nil so StartTailnet uses the real policy owner with native identity.
	tailnetEnabledCheck func(ctx context.Context, project, cid string) (bool, error)
	admissionOnce       sync.Once
	admission           chan struct{}
	terminalMu          sync.Mutex
	terminals           map[*http.Request]context.CancelFunc
	terminalClosed      bool
	terminalWG          sync.WaitGroup
}

func (d *Daemon) podman(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	return d.Exec.Run(ctx, in, "/usr/bin/podman", args...)
}

// Keep mutations globally serialized, but let cancelled waiters leave.
// Lazy initialization keeps Daemon struct literals usable without a constructor.
// This gate is independent of terminal streams and runner file locks.
func (d *Daemon) acquireAdmission(ctx context.Context) error {
	d.admissionOnce.Do(func() { d.admission = make(chan struct{}, 1) })
	select {
	case d.admission <- struct{}{}:
		// Done and the gate can both be ready; select does not prioritize Done.
		if err := ctx.Err(); err != nil {
			<-d.admission
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var errNotFound = errors.New("not found")

func (d *Daemon) routeSubsystem(w http.ResponseWriter, r *http.Request) bool {
	if strings.HasPrefix(r.URL.Path, "/tailnet/") {
		d.tailnetHandler(w, r)
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/runners/") {
		d.runnerHandler(w, r)
		return true
	}
	if r.URL.Path == "/terminal" {
		d.terminalHandler(w, r)
		return true
	}
	return false
}

func hasNativeCleanPath(r *http.Request) bool {
	switch r.URL.Path {
	case "/lifecycle", "/access-keys", "/profile", "/create", "/os":
		return r.URL.RawQuery == "" && !r.URL.ForceQuery && r.URL.RawPath == ""
	default:
		return true
	}
}

func validateNativeOperationRequest(r *http.Request) int {
	if !hasNativeCleanPath(r) {
		return http.StatusBadRequest
	}
	if r.Method != "POST" {
		return http.StatusMethodNotAllowed
	}
	return 0
}

func nativeRequestErrorMessage(code int) string {
	if code == http.StatusBadRequest {
		return "invalid native operation path"
	}
	return "POST required"
}

func isAdmittedMutationPath(path string) bool {
	return path == "/lifecycle" || path == "/access-keys" || path == "/account"
}

func makeBodyDecoder(body io.Reader, ctx context.Context) func(any) error {
	return func(v any) error {
		if err := strictjson.Decode(body, v); err != nil {
			return err
		}
		// A body read can outlive admission. Refuse cancellation before dispatch;
		// already-running native commands retain their existing context handling.
		return ctx.Err()
	}
}

func (d *Daemon) dispatchProfile(ctx context.Context, decode func(any) error) (any, error) {
	var in struct{}
	if err := decode(&in); err != nil {
		return nil, err
	}
	return d.resolveProfile(ctx)
}

func (d *Daemon) dispatchCreate(ctx context.Context, decode func(any) error) (any, error) {
	var in Create
	if err := decode(&in); err != nil {
		return nil, err
	}
	if !projectID.MatchString(in.ID) || in.Owner <= 0 {
		return nil, errors.New("invalid project identity")
	}
	return d.create(ctx, in)
}

func (d *Daemon) dispatchCreateTargeted(ctx context.Context, path string, decode func(any) error) (any, error) {
	var in Create
	if err := decode(&in); err != nil {
		return nil, err
	}
	switch path {
	case "/inspect":
		out, _, err := d.inspect(ctx, in.ID)
		return out, err
	case "/os":
		return d.observeOS(ctx, in.ID)
	case "/connection":
		return d.connection(ctx, in.ID)
	default:
		return nil, errNotFound
	}
}

func (d *Daemon) dispatchMutation(ctx context.Context, path string, decode func(any) error) (any, error) {
	switch path {
	case "/lifecycle":
		var in Lifecycle
		if err := decode(&in); err != nil {
			return nil, err
		}
		return d.lifecycle(ctx, in)
	case "/access-keys":
		var in AccessKeys
		if err := decode(&in); err != nil {
			return nil, err
		}
		return d.accessKeys(ctx, in)
	case "/account":
		var in Account
		if err := decode(&in); err != nil {
			return nil, err
		}
		err := d.account(ctx, in)
		return map[string]bool{"ok": err == nil}, nil
	default:
		return nil, errNotFound
	}
}

func (d *Daemon) dispatchOperation(ctx context.Context, path string, decode func(any) error) (any, error) {
	switch path {
	case "/profile":
		return d.dispatchProfile(ctx, decode)
	case "/create":
		return d.dispatchCreate(ctx, decode)
	case "/inspect", "/os", "/connection":
		return d.dispatchCreateTargeted(ctx, path, decode)
	case "/lifecycle", "/access-keys", "/account":
		return d.dispatchMutation(ctx, path, decode)
	default:
		return nil, errNotFound
	}
}

func (d *Daemon) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d.routeSubsystem(w, r) {
		return
	}
	if code := validateNativeOperationRequest(r); code != 0 {
		http.Error(w, nativeRequestErrorMessage(code), code)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	decode := makeBodyDecoder(r.Body, ctx)
	if ctx.Err() != nil {
		http.Error(w, "native operation cancelled before admission", http.StatusRequestTimeout)
		return
	}
	// Read-only observations may overlap mutations and report transitional state.
	// Create owns the same writer gate inside its native provisioning phase.
	if isAdmittedMutationPath(r.URL.Path) {
		if err := d.acquireAdmission(ctx); err != nil {
			http.Error(w, "native operation cancelled before admission", http.StatusRequestTimeout)
			return
		}
		defer func() { <-d.admission }()
	}
	out, err := d.dispatchOperation(ctx, r.URL.Path, decode)
	if err != nil {
		if errors.Is(err, errNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.Error("project native operation failed", "operation", r.URL.Path, "error", err)
		http.Error(w, "native operation failed; inspect operator journal", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (d *Daemon) create(ctx context.Context, in Create) (Environment, error) {
	if !projectID.MatchString(in.ID) || in.Owner <= 0 || in.Profile == nil || in.Profile.Validate() != nil {
		return Environment{}, errors.New("invalid creation identity")
	}
	if err := d.acquireAdmission(ctx); err != nil {
		return Environment{}, err
	}
	defer func() { <-d.admission }()
	if err := d.createContainer(ctx, in); err != nil {
		return Environment{}, err
	}
	return d.startCreated(ctx, in)
}

func (d *Daemon) createContainer(ctx context.Context, in Create) error {
	profile, err := d.resolveProfile(ctx)
	if err != nil {
		return err
	}
	if profile != *in.Profile {
		return errors.New("installed profile changed; reservation retained")
	}
	encoded, _ := json.Marshal(profile)
	if _, err := d.podman(ctx, nil, "network", "exists", d.Config.Network); err != nil {
		if _, err = d.podman(ctx, nil, "network", "create", "--driver", "bridge", "--subnet", d.Config.Subnet, "--interface-name", d.Config.Bridge, d.Config.Network); err != nil {
			return err
		}
	}
	name := "soda-" + in.ID
	// No --replace or --rm: the writable userspace is a lasting environment.
	// NET_ADMIN lets nested netavark configure project-owned networking;
	// SYS_PTRACE lets the engine enter different-UID workload process namespaces.
	// Both are confined to the project's user namespace, not the appliance.
	args := []string{"create", "--name", name, "--label", "org.soda.project=" + in.ID, "--label", "org.soda.owner=" + strconv.FormatInt(in.Owner, 10), "--network", d.Config.Network, "--userns=auto:size=262144", "--systemd=always", "--cgroupns=private", "--cap-add=SYS_ADMIN,MKNOD,NET_ADMIN,SYS_PTRACE", "--device=/dev/fuse", "--security-opt=label=disable", "--label", "org.soda.profile=" + profile.ID, "--label", "org.soda.creation-profile=" + string(encoded), "--pull=never", profile.Image}
	_, err = d.podman(ctx, nil, args...)
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

func (d *Daemon) inspect(ctx context.Context, id string) (Environment, int64, error) {
	env := Environment{ID: id}
	if !projectID.MatchString(id) {
		return env, 0, errors.New("invalid project id")
	}
	out, err := d.podman(ctx, nil, "inspect", "soda-"+id)
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
	if err = admitProjectIP(&env, d.Config.Network, d.Config.Subnet, item.NetworkSettings.Networks); err != nil {
		return env, 0, err
	}
	return env, owner, nil
}

func waitProjectReady(ctx context.Context, d *Daemon, name string) error {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		if _, err := d.podman(ctx, nil, "exec", name, "/usr/bin/test", "-f", "/run/soda-project-ready"); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

func (d *Daemon) startCreated(ctx context.Context, in Create) (Environment, error) {
	name := "soda-" + in.ID
	if _, err := d.Exec.Run(ctx, nil, "/usr/bin/systemctl", "enable", "--now", "soda-project@"+in.ID+".service"); err != nil {
		return Environment{}, err
	}
	if err := waitProjectReady(ctx, d, name); err != nil {
		return Environment{}, err
	}
	env, _, err := d.inspect(ctx, in.ID)
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

func (d *Daemon) account(ctx context.Context, in Account) error {
	if !loginName.MatchString(in.Login) || in.Login == "root" || in.Identity <= 0 || len(in.Keys) > 32 {
		return errors.New("invalid project account")
	}
	env, owner, err := d.inspect(ctx, in.Project)
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
	out, err := d.podman(ctx, body, "exec", "--interactive", "soda-"+in.Project, "/usr/libexec/soda/project-account")
	if err != nil {
		return err
	}
	return confirmAccount(out, in)
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
