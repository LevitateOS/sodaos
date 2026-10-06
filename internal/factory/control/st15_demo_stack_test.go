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
	"time"

	"github.com/levitateos/sodaos/internal/config"
	hostexec "github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/identity"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

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
