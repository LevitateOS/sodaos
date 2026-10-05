// ST15 composed development-factory demonstration (development-only).
// One real issue and its dependant complete the product loop through the
// actual factory chain: accepted input with blockers, automatic dispatch
// of real provider runs, native PR, separate review, complete CI,
// conditional merge, confirmed outcome and dependent pickup, watched
// through real Spaces and driven through real controls. Nothing here
// qualifies shipping bytes.
package control_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	hostexec "github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/identity"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
)

const st15Image = "localhost/soda-m2-st01:20261001"

type st15Config struct {
	FountainURL       string `json:"fountain_url"`
	BrowserURL        string `json:"browser_url"`
	Socket            string `json:"socket"`
	Owner             string `json:"owner"`
	Repo              string `json:"repo"`
	Repository        int64  `json:"repository"`
	ActorID           int64  `json:"actor_id"`
	TokenID           int64  `json:"token_id"`
	BaseBranch        string `json:"base_branch"`
	TokenFile         string `json:"token_file"`
	CreatorID         int64  `json:"creator_id"`
	CreatorTokenFile  string `json:"creator_token_file"`
	ReviewerID        int64  `json:"reviewer_id"`
	ReviewerTokenID   int64  `json:"reviewer_token_id"`
	ReviewerTokenFile string `json:"reviewer_token_file"`
	MaintainerPass    string `json:"maintainer_pass_file"`
	FountainDB        string `json:"fountain_db"`
	SodaDB            string `json:"soda_db"`
	GrantKeyFile      string `json:"grant_key_file"`
	HostSocket        string `json:"host_socket"`
	BrokerSocket      string `json:"broker_socket"`
}

func loadST15(t *testing.T) st15Config {
	t.Helper()
	raw := os.Getenv("SODA_ST15_NATIVE")
	if raw == "" {
		t.Skip("ST15 NOT RUN: SODA_ST15_NATIVE is not configured")
	}
	var cfg st15Config
	nativeMust(t, json.Unmarshal([]byte(raw), &cfg))
	if cfg.FountainURL == "" || cfg.BrowserURL == "" || cfg.Socket == "" || cfg.TokenFile == "" || cfg.Repository <= 0 ||
		cfg.ActorID <= 0 || cfg.TokenID <= 0 || cfg.CreatorTokenFile == "" || cfg.ReviewerTokenFile == "" ||
		cfg.MaintainerPass == "" || cfg.FountainDB == "" || cfg.SodaDB == "" || cfg.GrantKeyFile == "" ||
		cfg.HostSocket == "" || cfg.BrokerSocket == "" {
		t.Fatal("ST15 fixture requires the full native/daemon/browser configuration")
	}
	return cfg
}

func st15ReceiptDir(t *testing.T) string {
	t.Helper()
	root := os.Getenv("ST15_RECEIPT_DIR")
	if root == "" || !filepath.IsAbs(root) {
		t.Fatal("ST15_RECEIPT_DIR must be an absolute retained directory")
	}
	nativeMust(t, os.MkdirAll(root, 0o700))
	return root
}

func st15Receipt(t *testing.T, label string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	nativeMust(t, err)
	nativeMust(t, os.WriteFile(filepath.Join(st15ReceiptDir(t), label+".json"), append(raw, '\n'), 0o600))
}

// st15Fixture is the composed journey state: one native repository, one
// project container, one daemon/broker pair and one coordinator over the
// fixture dashboard store (the live browser reads the same store).
type st15Fixture struct {
	t   *testing.T
	ctx context.Context
	cfg st15Config

	scratch  string
	db       *store.Store
	client   *identityclient.Client
	coord    *control.Coordinator
	rest     *forgejo.Client
	observer *forgejo.ServiceObserver
	bg       *forgejo.ServiceBackground
	intake   api.IntakeHandler

	projectID   string
	container   string
	containerID string
	image       string
	versions    string
	harnessSHA  string
	coderPrep   string
	reviewPrep  string
	sourceHead  string
	synthetic   bool

	issueP, issueA, issueB, issueC int64
	issuePIndex, issueAIndex       int64
	issueBIndex, issueCIndex       int64
	acceptP, acceptA, acceptB      string
	assignA, assignB               string
	runA                           string
	runC                           string
	answerA                        int64
	pubA                           string
	prNumber                       int64
	head1, head2                   string
	review1, review2               factory.ReviewOutcome
	verdict1, verdict2             string
	ciFailRev, ciPassRev           int64

	// controlBrowser carries the background dependant-stop verdict:
	// the browser boots during review-2 and polls for B's run, so
	// the stop lands within seconds of record.
	controlBrowser chan error

	checks []st15Check
}

type st15Check struct {
	Name   string `json:"name"`
	Result string `json:"result"`
}

func (fx *st15Fixture) check(name string, err error) {
	fx.t.Helper()
	if err != nil {
		fx.checks = append(fx.checks, st15Check{Name: name, Result: "fail: " + err.Error()})
		fx.t.Fatalf("ST15 %s: %v", name, err)
	}
	fx.checks = append(fx.checks, st15Check{Name: name, Result: "pass"})
	fx.t.Logf("ST15 pass: %s", name)
}

// st15Podman drives the project container through root podman: the
// production daemon runs as root and only sees root container storage,
// so the fixture manages the container in that same storage.
func st15Podman(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "sudo", append([]string{"-n", "/usr/bin/podman"}, args...)...)
	env := []string{}
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "TMPDIR=") {
			env = append(env, entry)
		}
	}
	cmd.Env = env
	if stdin != nil {
		cmd.Stdin = strings.NewReader(string(stdin))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("podman %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func st15Pexec(ctx context.Context, name, user string, stdin []byte, args ...string) ([]byte, error) {
	full := []string{"exec"}
	if user != "" {
		full = append(full, "--user", user)
	}
	if stdin != nil {
		full = append(full, "-i")
	}
	full = append(full, name)
	full = append(full, args...)
	return st15Podman(ctx, stdin, full...)
}

func st15Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=soda-tester", "GIT_AUTHOR_EMAIL=soda-tester@example.test",
		"GIT_COMMITTER_NAME=soda-tester", "GIT_COMMITTER_EMAIL=soda-tester@example.test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func st15SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func st15RandHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// st15RepoRoot locates the checkout so the fixture builds the broker from
// the tree under test, including worktrees.
func st15RepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	nativeMust(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}

// st15BuildBroker compiles the Rust identity broker; ST15 never
// enrolls, so the pinned provider below is construction-only (the
// connection is seeded directly from the configured credential file).
func st15BuildBroker(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo unavailable")
	}
	root := st15RepoRoot(t)
	build := exec.Command("cargo", "build", "-p", "soda-identity", "--bin", "soda-identity")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build soda-identity: %v\n%s", err, out)
	}
	return filepath.Join(root, "target", "debug", "soda-identity")
}

// st15BuildHost compiles the production `soda-host` daemon the fixture
// spawns over the pre-bound host socket.
func st15BuildHost(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo unavailable")
	}
	root := st15RepoRoot(t)
	build := exec.Command("cargo", "build", "-p", "soda-host", "--bin", "soda-host")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build soda-host: %v\n%s", err, out)
	}
	return filepath.Join(root, "target", "debug", "soda-host")
}

// st15BuildFactoryRoles compiles the Rust factory-roles helper the
// fixture stages into the project container.
func st15BuildFactoryRoles(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo unavailable")
	}
	root := st15RepoRoot(t)
	build := exec.Command("cargo", "build", "-p", "soda-project-factory-roles", "--bin", "project-factory-roles")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build project-factory-roles: %v\n%s", err, out)
	}
	return filepath.Join(root, "target", "debug", "project-factory-roles")
}

// st15TmpfsRoot confines the construction-only provider root to private
// tmpfs, which the broker validates before serving.
func st15TmpfsRoot(t *testing.T) string {
	t.Helper()
	parent := filepath.Join("/dev/shm", fmt.Sprintf("soda-st15-%d", os.Getpid()))
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Skipf("tmpfs provider root unavailable: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	return parent
}

func st15FakeCodex(t *testing.T, dir string) (binary, sum string) {
	t.Helper()
	binary = filepath.Join(dir, "codex-fixture")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'codex-cli 0.153.4'; exit 0; fi\necho unexpected >&2\nexit 1\n"
	nativeMust(t, os.WriteFile(binary, []byte(script), 0o700))
	return binary, st15SHA256([]byte(script))
}

// st15FakeMuse registers the broker's muse provider the same way: the
// fake answers the pinned version line the adapter requires at
// construction. The journey never enrolls through it (the credential is
// seeded from file); registration only satisfies the acquire-time
// provider gate for the st15-muse connection.
func st15FakeMuse(t *testing.T, dir string) (binary, sum string) {
	t.Helper()
	binary = filepath.Join(dir, "muse-fixture")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'Muse Code 1.4.0 (1.4.0-R4161.1)'; exit 0; fi\necho unexpected >&2\nexit 1\n"
	nativeMust(t, os.WriteFile(binary, []byte(script), 0o700))
	return binary, st15SHA256([]byte(script))
}

func st15WaitSocket(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Lstat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("broker socket %s never appeared", path)
}

// setupHostStack stands up the container, helper, harness, daemon and
// broker on the fixture sockets the dashboard already dials.
func (fx *st15Fixture) setupHostStack() error {
	t := fx.t
	scratch, err := os.MkdirTemp("/home/vince/tmp", "st15-*")
	if err != nil {
		return err
	}
	fx.scratch = scratch
	t.Cleanup(func() { _ = os.RemoveAll(scratch) })
	helperPath := st15BuildFactoryRoles(t)
	helper, err := os.ReadFile(helperPath)
	if err != nil {
		return err
	}
	helperSHA := st15SHA256(helper)
	digest, err := st15Podman(fx.ctx, nil, "inspect", "--format", "{{.Digest}}", st15Image)
	if err != nil {
		return err
	}
	fx.image = strings.TrimSpace(string(digest))
	sum := sha256.Sum256([]byte("soda-st15-project"))
	fx.projectID = "p" + hex.EncodeToString(sum[:])[:24]
	name := "soda-" + fx.projectID
	if _, err := st15Podman(fx.ctx, nil, "rm", "-f", name); err != nil {
		t.Logf("ST15 container pre-clean: %v", err)
	}
	if _, err := st15Podman(fx.ctx, nil, "create", "--name", name, "--init",
		"--label", "org.soda.project="+fx.projectID, "--label", fmt.Sprintf("org.soda.owner=%d", fx.cfg.CreatorID),
		"--uidmap", "0:1:65536", "--gidmap", "0:1:65536", st15Image, "sleep", "infinity"); err != nil {
		return err
	}
	t.Cleanup(func() { _, _ = st15Podman(context.Background(), nil, "rm", "-f", name) })
	var startErr error
	for _, wait := range []time.Duration{0, 10 * time.Second, 30 * time.Second, 60 * time.Second} {
		time.Sleep(wait)
		if _, startErr = st15Podman(fx.ctx, nil, "start", name); startErr == nil {
			break
		}
		t.Logf("ST15 container start attempt failed, retrying: %v", startErr)
	}
	if startErr != nil {
		return startErr
	}
	id, err := st15Podman(fx.ctx, nil, "inspect", "--format", "{{.ID}}", name)
	if err != nil {
		return err
	}
	fx.containerID = strings.TrimSpace(string(id))
	userns, err := st15Podman(fx.ctx, nil, "inspect", "--format", "{{.HostConfig.UsernsMode}}", name)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(userns)) != "private" {
		return fmt.Errorf("fixture userns is %q, need private", strings.TrimSpace(string(userns)))
	}
	fx.container = name
	osRelease, err := st15Pexec(fx.ctx, name, "", nil, "/bin/bash", "-c", ". /etc/os-release && echo $ID:$VERSION_ID")
	if err != nil {
		return err
	}
	if _, err := st15Podman(fx.ctx, nil, "cp", helperPath, name+":/usr/libexec/soda/project-factory-roles"); err != nil {
		return err
	}
	if _, err := st15Pexec(fx.ctx, name, "", nil, "/usr/bin/chmod", "0755", "/usr/libexec/soda/project-factory-roles"); err != nil {
		return err
	}
	staged, err := st15Pexec(fx.ctx, name, "", nil, "/usr/bin/sha256sum", "/usr/libexec/soda/project-factory-roles")
	if err != nil {
		return err
	}
	if fields := strings.Fields(strings.TrimSpace(string(staged))); len(fields) != 2 || fields[0] != helperSHA {
		return errors.New("staged helper digest differs")
	}
	if _, err := st15Pexec(fx.ctx, name, "", nil, "/usr/libexec/soda/project-init"); err != nil {
		return fmt.Errorf("project boot preparation: %w", err)
	}
	out, err := st15Pexec(fx.ctx, name, "", []byte(`{"op":"ensure"}`), "/usr/libexec/soda/project-factory-roles")
	if err != nil {
		return fmt.Errorf("role ensure: %w", err)
	}
	var ensured struct {
		Roles []string `json:"roles"`
	}
	if err = json.Unmarshal(out, &ensured); err != nil || len(ensured.Roles) != 2 {
		return fmt.Errorf("role ensure unconfirmed: %s", strings.TrimSpace(string(out)))
	}
	harnessDir := filepath.Join(scratch, "harness", "bin")
	if err := os.MkdirAll(harnessDir, 0o700); err != nil {
		return err
	}
	// The staged guest is the static muse binary itself, resolved the
	// way the launcher does (never the auto-updating shell wrapper).
	installed, err := os.ReadFile("/home/vince/.local/bin/.muse-version")
	if err != nil {
		return err
	}
	museBin := filepath.Join("/home/vince/.local/bin", "muse-bin-"+strings.TrimSpace(string(installed)))
	muse, err := os.ReadFile(museBin)
	if err != nil {
		return err
	}
	stagedBin := filepath.Join(harnessDir, "muse")
	if err = os.WriteFile(stagedBin, muse, 0o755); err != nil {
		return err
	}
	versionOut, err := exec.CommandContext(fx.ctx, stagedBin, "--version").CombinedOutput()
	if err != nil {
		return err
	}
	fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(versionOut)), "Muse Code ")))
	if len(fields) == 0 {
		return fmt.Errorf("muse version unparseable: %q", strings.TrimSpace(string(versionOut)))
	}
	fx.versions = fields[0]
	fx.harnessSHA = st15SHA256(muse)
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	brokerDB, brokerDSN := postgresFixture(t, key)
	credentialPath := os.Getenv("SODA_ST15_PROVIDER_CREDENTIAL")
	var auth []byte
	if credentialPath == "" {
		fx.synthetic = true
		auth = []byte(`{"schema_version":1,"providers":{},"st15_synthetic":true}`)
	} else {
		raw, err := os.ReadFile(credentialPath)
		if err != nil {
			return fmt.Errorf("provider credential unreadable: %w", err)
		}
		auth = raw
	}
	defer clear(auth)
	if !identity.CredentialValid(auth) {
		return errors.New("provider credential unusable")
	}
	conn := identity.Connection{ID: "st15-muse", ProviderID: identity.Muse, OwnerID: fx.cfg.CreatorID, Label: "st15-fixture", State: identity.Ready, Generation: 1}
	if err = brokerDB.SeedIdentityConnection(fx.ctx, conn, auth); err != nil {
		return err
	}
	runtimeSocket := fx.cfg.BrokerSocket + ".runtime"
	for _, sock := range []string{fx.cfg.BrokerSocket, runtimeSocket, fx.cfg.HostSocket} {
		_ = os.Remove(sock)
	}
	// The production `soda-host` daemon serves the fixture host socket:
	// the harness spawns the compiled binary as root (its production
	// identity) with --listen-path, so ST15 drives the shipped executor
	// instead of an in-test double. The empty release path disables the
	// appliance image overlay: the fixture writes the full pinned config
	// itself.
	if out, err := exec.CommandContext(fx.ctx, "sudo", "-n", "install", "-d", "-m", "700", "/var/lib/soda/host/factory").CombinedOutput(); err != nil {
		return fmt.Errorf("factory state dir: %w: %s", err, strings.TrimSpace(string(out)))
	}
	// Factory spawn runs systemd-run --user on the daemon's (root)
	// user bus; a bare host never started user@0, so start it
	// transiently for the fixture. Production must guarantee the same
	// bus (linger at install); without it every reserve refuses.
	if out, err := exec.CommandContext(fx.ctx, "sudo", "-n", "systemctl", "start", "user@0.service").CombinedOutput(); err != nil {
		return fmt.Errorf("root user manager: %w: %s", err, strings.TrimSpace(string(out)))
	}
	daemonCfg, err := json.Marshal(map[string]any{
		"muse_sha256": "", "muse_version": "", "muse_socket": "",
		"identity_socket": runtimeSocket,
		"codex_harness":   "", "codex_harness_sha256": "", "codex_harness_version": "",
		"muse_harness":         filepath.Join(scratch, "harness"),
		"muse_harness_sha256":  fx.harnessSHA,
		"muse_harness_version": fx.versions,
		"tailnet_management":   false, "tailnet_image": "",
		"image": fx.image, "network": "soda-projects",
		"subnet": "10.89.0.0/24", "bridge": "soda0",
	})
	if err != nil {
		return err
	}
	daemonCfgPath := filepath.Join(scratch, "soda-host.json")
	if err := os.WriteFile(daemonCfgPath, daemonCfg, 0o600); err != nil {
		return err
	}
	daemonLogPath := filepath.Join(scratch, "soda-host.log")
	daemonLog, err := os.Create(daemonLogPath)
	if err != nil {
		return err
	}
	daemonCmd := exec.Command("sudo", "-n", st15BuildHost(t), "--config", daemonCfgPath,
		"--release", "", "--listen-path", fx.cfg.HostSocket)
	daemonCmd.Stdout = daemonLog
	daemonCmd.Stderr = daemonLog
	if err := daemonCmd.Start(); err != nil {
		return err
	}
	t.Cleanup(func() {
		_ = daemonCmd.Process.Kill()
		_ = daemonCmd.Wait()
		// Killing sudo orphans the daemon: reap it by its unique
		// fixture config path (the test binary never matches).
		_ = exec.Command("sudo", "-n", "pkill", "-f", "soda-host --config "+daemonCfgPath).Run()
		_ = daemonLog.Close()
	})
	// The Rust broker serves the fixture sockets: the dashboard keeps
	// dialing the configured admin socket while the factory client uses
	// the runtime socket, matching the production topology.
	brokerBinary := st15BuildBroker(t)
	writeSecret := func(name, content string) string {
		path := filepath.Join(scratch, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	dsnFile := writeSecret("broker-dsn", brokerDSN+"\n")
	keyFile := writeSecret("broker-key", base64.StdEncoding.EncodeToString(key)+"\n")
	codexBinary, codexSum := st15FakeCodex(t, scratch)
	codexRoot := filepath.Join(st15TmpfsRoot(t), "codex")
	if err := os.MkdirAll(codexRoot, 0o700); err != nil {
		return err
	}
	museBinary, museSum := st15FakeMuse(t, scratch)
	museRoot := filepath.Join(st15TmpfsRoot(t), "muse")
	if err := os.MkdirAll(museRoot, 0o700); err != nil {
		return err
	}
	settings, err := json.Marshal(map[string]any{
		"database_dsn_file": dsnFile,
		"key_file":          keyFile,
		"admin_socket":      fx.cfg.BrokerSocket,
		"runtime_socket":    runtimeSocket,
		"host_socket":       fx.cfg.HostSocket,
		"codex": map[string]any{
			"binary": codexBinary, "version": "0.153.4", "sha256": codexSum, "root": codexRoot,
		},
		"muse": map[string]any{
			"binary": museBinary, "version": "1.4.0-R4161.1", "sha256": museSum, "root": museRoot,
		},
	})
	if err != nil {
		return err
	}
	configPath := filepath.Join(scratch, "identity.json")
	if err := os.WriteFile(configPath, settings, 0o600); err != nil {
		return err
	}
	brokerCmd := exec.Command(brokerBinary, "--config", configPath)
	brokerLogPath := filepath.Join(scratch, "soda-identity.log")
	brokerLog, err := os.Create(brokerLogPath)
	if err != nil {
		return err
	}
	brokerCmd.Stdout = brokerLog
	brokerCmd.Stderr = brokerLog
	if err := brokerCmd.Start(); err != nil {
		return err
	}
	t.Cleanup(func() {
		_ = brokerCmd.Process.Kill()
		_ = brokerCmd.Wait()
		_ = brokerLog.Close()
	})
	st15WaitSocket(t, fx.cfg.BrokerSocket)
	st15WaitSocket(t, runtimeSocket)
	fx.client = identityclient.New(runtimeSocket)
	if _, err = fx.client.GetExecution(fx.ctx, identity.Factory, strings.Repeat("f", 32)); !errors.Is(err, identity.ErrNotFound) {
		return fmt.Errorf("broker transport check failed: %v", err)
	}
	hostClient := hostexec.NewClient(fx.cfg.HostSocket)
	var pin project.FactoryHarnessPin
	readyDeadline := time.Now().Add(30 * time.Second)
	for {
		pin, err = hostClient.FactoryHarness(fx.ctx)
		if err == nil {
			break
		}
		if time.Now().After(readyDeadline) {
			raw, _ := os.ReadFile(daemonLogPath)
			return fmt.Errorf("daemon harness pin route failed: %w (log %s: %s)", err, daemonLogPath, strings.TrimSpace(string(raw)))
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Logf("ST15 host stack: container %.12s image %.16s harness muse-%s project-os %s",
		strings.TrimSpace(string(id)), fx.image, fx.versions, strings.TrimSpace(string(osRelease)))
	if pin.Version != fx.versions {
		return fmt.Errorf("harness pin %q differs from staged %q", pin.Version, fx.versions)
	}
	return nil
}

// openFactoryStore opens the fixture dashboard store the live browser reads.
func (fx *st15Fixture) openFactoryStore() error {
	key, err := config.GrantKey(fx.cfg.GrantKeyFile)
	if err != nil {
		return err
	}
	db, err := store.OpenEncrypted(fx.cfg.SodaDB, key)
	if err != nil {
		return err
	}
	fx.db = db
	fx.t.Cleanup(func() { _ = db.Close() })
	return nil
}
