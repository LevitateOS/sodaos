// ST15 composed development-factory demonstration (development-only).
// One real issue and its dependant complete the product loop through the
// actual factory chain: accepted input with blockers, automatic dispatch
// of real provider runs, native PR, separate review, complete CI,
// conditional merge, confirmed outcome and dependent pickup, watched
// through real Spaces and driven through real controls. Nothing here
// qualifies shipping bytes.
package control_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	hostexec "github.com/levitateos/sodaos/internal/host"
	projectexec "github.com/levitateos/sodaos/internal/host/project"
	terminalexec "github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/identity"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	identitycontrol "github.com/levitateos/sodaos/internal/identity/control"
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
	brokerDB *store.Store
	broker   *identitycontrol.Controller
	client   *identityclient.Client
	term     *terminalexec.Service
	rt       *projectexec.Runtime
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

func st15Podman(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/podman", args...)
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

type st15HostEndpoint struct {
	daemon *hostexec.Daemon
	mu     sync.Mutex
	calls  map[string]int64
}

func (h *st15HostEndpoint) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mu.Lock()
		h.calls[r.URL.Path]++
		h.mu.Unlock()
		h.daemon.ServeHTTP(w, r)
	})
}

type st15HostRuntime struct {
	client *http.Client
}

func st15NewHostRuntime(socket string) *st15HostRuntime {
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, "unix", socket)
	}}
	return &st15HostRuntime{client: &http.Client{Transport: transport, Timeout: 90 * time.Second}}
}

func (h *st15HostRuntime) call(ctx context.Context, op string, l identity.Lease) ([]byte, error) {
	body, err := json.Marshal(identity.DeliveryWire{Lease: l})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://host/identity/"+op, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	response, err := h.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("identity operation unconfirmed (HTTP %d)", response.StatusCode)
	}
	var out identity.DeliveryWire
	if err = json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out.Credential, nil
}

func (h *st15HostRuntime) Validate(ctx context.Context, l identity.Lease) error {
	_, err := h.call(ctx, "validate", l)
	return err
}

func (h *st15HostRuntime) Stop(ctx context.Context, l identity.Lease) error {
	_, err := h.call(ctx, "stop", l)
	return err
}

func (h *st15HostRuntime) Finish(ctx context.Context, l identity.Lease) ([]byte, error) {
	return h.call(ctx, "finish", l)
}

// st15StubProvider satisfies broker construction; ST15 never enrolls (the
// connection is seeded directly from the configured credential file).
type st15StubProvider struct{}

func (st15StubProvider) Start(context.Context, int64) (identity.EnrollmentSession, error) {
	return nil, errors.New("enrollment unavailable in the ST15 fixture")
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
	stateDir := filepath.Join(scratch, "runs")
	if err := os.Mkdir(stateDir, 0o700); err != nil {
		return err
	}
	helper, err := os.ReadFile("/home/vince/Projects/sodaos/project-os/rootfs/usr/libexec/soda/project-factory-roles")
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
	if _, err := st15Podman(fx.ctx, nil, "cp", "/home/vince/Projects/sodaos/project-os/rootfs/usr/libexec/soda/project-factory-roles", name+":/usr/libexec/soda/project-factory-roles"); err != nil {
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
	codex, err := os.ReadFile("/home/vince/.local/bin/codex")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(harnessDir, "codex"), codex, 0o755); err != nil {
		return err
	}
	host, err := os.ReadFile("/home/vince/.codex/packages/standalone/current/bin/codex-code-mode-host")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(harnessDir, "codex-code-mode-host"), host, 0o755); err != nil {
		return err
	}
	versionOut, err := exec.CommandContext(fx.ctx, "/home/vince/.local/bin/codex", "--version").CombinedOutput()
	if err != nil {
		return err
	}
	fx.versions = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(versionOut)), "codex-cli "))
	fx.harnessSHA = st15SHA256(codex)
	fx.term = &terminalexec.Service{Exec: hostexec.Native{}, CodexHarness: filepath.Join(scratch, "harness"), CodexHarnessSHA256: fx.harnessSHA, CodexHarnessVersion: fx.versions}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	brokerDB, err := store.OpenEncrypted(filepath.Join(scratch, "broker.db"), key)
	if err != nil {
		return err
	}
	fx.brokerDB = brokerDB
	t.Cleanup(func() { _ = brokerDB.Close() })
	credentialPath := os.Getenv("SODA_ST15_PROVIDER_CREDENTIAL")
	var auth []byte
	if credentialPath == "" {
		fx.synthetic = true
		auth = []byte(`{"tokens":{"id_token":"ST15-SYNTHETIC-NONCREDENTIAL","access_token":"ST15-SYNTHETIC-NONCREDENTIAL","refresh_token":"ST15-SYNTHETIC-NONCREDENTIAL","account_id":"st15-synthetic"}}`)
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
	conn := identity.Connection{ID: "st15-codex", ProviderID: identity.Codex, OwnerID: fx.cfg.CreatorID, Label: "st15-fixture", State: identity.Ready, Generation: 1}
	if err = brokerDB.IdentitySaveConnection(fx.ctx, conn, auth); err != nil {
		return err
	}
	for _, sock := range []string{fx.cfg.BrokerSocket, fx.cfg.HostSocket} {
		_ = os.Remove(sock)
	}
	endpoint := &st15HostEndpoint{daemon: &hostexec.Daemon{
		Config:   hostexec.Config{Image: fx.image},
		Terminal: fx.term,
		Project:  &projectexec.Runtime{Exec: hostexec.Native{}},
	}, calls: map[string]int64{}}
	hostListener, err := net.Listen("unix", fx.cfg.HostSocket)
	if err != nil {
		return err
	}
	hostSrv := &http.Server{Handler: endpoint.handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = hostSrv.Serve(hostListener) }()
	t.Cleanup(func() { _ = hostSrv.Close() })
	broker, err := identitycontrol.New(fx.brokerDB, map[string]identity.Provider{identity.Codex: st15StubProvider{}}, st15NewHostRuntime(fx.cfg.HostSocket))
	if err != nil {
		return err
	}
	fx.broker = broker
	brokerListener, err := net.Listen("unix", fx.cfg.BrokerSocket)
	if err != nil {
		return err
	}
	brokerSrv := &http.Server{Handler: broker.Handler(true), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = brokerSrv.Serve(brokerListener) }()
	t.Cleanup(func() { _ = brokerSrv.Close() })
	fx.client = identityclient.New(fx.cfg.BrokerSocket)
	if _, err = fx.client.GetExecution(fx.ctx, identity.Factory, strings.Repeat("f", 32)); !errors.Is(err, identity.ErrNotFound) {
		return fmt.Errorf("broker transport check failed: %v", err)
	}
	fx.rt = &projectexec.Runtime{Exec: hostexec.Native{}}
	f, err := projectexec.OpenFactory(stateDir, fx.term, fx.client)
	if err != nil {
		return err
	}
	endpoint.daemon.Factory = f
	endpoint.daemon.Identity = fx.client
	pin, err := hostexec.NewClient(fx.cfg.HostSocket).FactoryHarness(fx.ctx)
	if err != nil {
		return fmt.Errorf("harness pin route failed: %w", err)
	}
	t.Logf("ST15 host stack: container %.12s image %.16s harness codex-%s project-os %s",
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
