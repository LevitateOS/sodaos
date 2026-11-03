package serve

import (
	"context"
	"crypto/sha256"
	"database/sql"
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
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	factoryclient "github.com/levitateos/factory/pkg/factory/client"
	factoryruntime "github.com/levitateos/factory/pkg/factory/runtime"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
)

func TestShutdownServersClosesConnectionsAndReturnsTimeout(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = w.Write([]byte("finished"))
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		response, requestErr := http.Post("http://"+listener.Addr().String(), "text/plain", strings.NewReader("hold"))
		if requestErr == nil {
			_ = response.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("request did not reach held handler")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err = shutdownServers(ctx, server)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown timeout was not returned: %v", err)
	}
	close(release)
	select {
	case <-requestDone:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not finish after handler release")
	}
	if err := <-serveDone; err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatal(err)
	}
}

// Exercise the installed process boundary against a real PostgreSQL database:
// neither product's shutdown may stop or silently replace the other product.
func TestFactoryAndForgeIndependentProcesses(t *testing.T) {
	t.Run("appliance_products", checkFactoryAndForgeIndependentProcesses)
	t.Run("native_bootstrap", checkNativeFactoryBootstrap)
	t.Run("native_process", checkNativeFactoryProcess)
}

func checkFactoryAndForgeIndependentProcesses(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	db, dsn, cleanup, err := store.OpenEphemeral(t.Context(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	if err := db.UpsertUser(t.Context(), store.User{ID: 42, Login: "retained-owner"}); err != nil {
		t.Fatal(err)
	}
	retained := store.Project{ID: "retained-native-composition", Name: "Retained", RepositoryID: 7, OwnerID: 42, Repository: "team/retained"}
	if err := db.CreateProject(t.Context(), retained); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(home, "factory-split-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	if err := os.Mkdir(filepath.Join(dir, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(dir, "bin"), "./cmd/soda-dashboard", "./cmd/soda-factory-server")
	build.Dir = "../../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build products: %v\n%s", err, out)
	}
	dsnFile := filepath.Join(dir, "database.dsn")
	if err := os.WriteFile(dsnFile, []byte(dsn), 0o600); err != nil {
		t.Fatal(err)
	}
	address := func() string {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := listener.Addr().String()
		_ = listener.Close()
		return addr
	}
	factoryAddr, forgeAddr := address(), address()
	base := config.Config{ForgejoURL: "https://forgejo.test", ForgejoInternalURL: "http://127.0.0.1:3000", DatabaseDSNFile: dsnFile, HostSocket: filepath.Join(dir, "host.sock"), IdentitySocket: filepath.Join(dir, "broker.sock"), OperatorID: 1}
	writeConfig := func(name, listen string) string {
		c := base
		c.Listen, c.FactoryPublicationRoot = listen, filepath.Join(dir, name, "publication")
		data, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, name+".json")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	factoryConfig, forgeConfig := writeConfig("factory", factoryAddr), writeConfig("forge", forgeAddr)
	factorySocket, controlSocket := filepath.Join(dir, "factory.sock"), filepath.Join(dir, "control.sock")
	forgeSocket := filepath.Join(dir, "forge.sock")
	uid := strconv.Itoa(os.Geteuid())
	start := func(binary, conf, addr string, args ...string) func() {
		cmd := exec.Command(filepath.Join(dir, "bin", binary), append([]string{"--config=" + conf}, args...)...)
		logFile, err := os.Create(filepath.Join(dir, binary+".log"))
		if err != nil {
			t.Fatal(err)
		}
		cmd.Stdout, cmd.Stderr = logFile, logFile
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		stopped := false
		stop := func() {
			if stopped {
				return
			}
			stopped = true
			_ = cmd.Process.Signal(syscall.SIGTERM)
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("%s shutdown: %v", binary, err)
				}
			case <-time.After(25 * time.Second):
				_ = cmd.Process.Kill()
				<-done
				t.Errorf("%s did not drain", binary)
			}
			_ = logFile.Close()
		}
		t.Cleanup(stop)
		client := &http.Client{Timeout: time.Second}
		for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(25 * time.Millisecond) {
			response, err := client.Get("http://" + addr + "/healthz")
			if err == nil {
				_ = response.Body.Close()
				if response.StatusCode == http.StatusOK {
					return stop
				}
			}
		}
		logs, _ := os.ReadFile(logFile.Name())
		t.Fatalf("%s did not become ready: %s", binary, logs)
		return stop
	}
	startFactory := func() func() {
		return start("soda-factory-server", factoryConfig, factoryAddr, "--extension-socket="+factorySocket, "--control-socket="+controlSocket, "--operator-socket="+filepath.Join(dir, "factory-op.sock"), "--operator-uid="+uid)
	}
	startForge := func() func() {
		return start("soda-dashboard", forgeConfig, forgeAddr, "--extension-socket="+forgeSocket, "--factory-control-socket="+controlSocket, "--operator-socket="+filepath.Join(dir, "forge-op.sock"), "--operator-uid="+uid)
	}
	stopFactory, stopForge := startFactory(), startForge()
	observed, err := store.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer observed.Close()
	bound, err := observed.CollaborationHostByAPIBase(t.Context(), collaboration.Fountain, base.ForgejoURL)
	if err != nil {
		t.Fatalf("actual Factory process did not bind configured Fountain: %v", err)
	}
	if got, err := observed.NativeProjectForCollaborationRepository(t.Context(), 7); err != nil || got != retained {
		t.Fatalf("actual Factory process changed retained native Project: %+v %v", got, err)
	}
	request := func(socket, method, path, body string, want int) {
		transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}}
		defer transport.CloseIdleConnections()
		client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
		res, err := client.Do(mustProcessRequest(t, method, "http://product"+path, body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.StatusCode != want {
			t.Fatalf("%s %s: %d, want %d: %s", filepath.Base(socket), path, res.StatusCode, want, data)
		}
	}
	request(factorySocket, "GET", "/api/environments", "", 404)
	request(factorySocket, "GET", "/api/factory/departments", "", 403)
	request(forgeSocket, "GET", "/api/factory/departments", "", 404)
	request(filepath.Join(dir, "factory-op.sock"), "POST", "/operator/factory", `{"type":"status"}`, 200)
	request(filepath.Join(dir, "forge-op.sock"), "POST", "/operator/factory", `{"type":"status"}`, 404)
	request(controlSocket, "POST", "/projects", `{"action":"authority","repository":7}`, 200)
	remote := factoryclient.New(controlSocket)
	if _, err := remote.EffectiveAuthority(t.Context(), 7); err != nil {
		t.Fatal("Forge could not read remote Factory authority:", err)
	}
	if _, _, err := remote.StopProject(t.Context(), "human:1", "p0123456789abcdef01234567"); !errors.Is(err, factory.ErrNotFound) {
		t.Fatal("remote Project error lost its canonical identity:", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "forge", "factory-ipc", "mcp.sock")); !os.IsNotExist(err) {
		t.Fatal("Forge created a worker MCP socket")
	}
	stopForge()
	request(filepath.Join(dir, "factory-op.sock"), "POST", "/operator/factory", `{"type":"status"}`, 200)
	stopForge = startForge()
	stopFactory()
	request(forgeSocket, "GET", "/api/factory/departments", "", 404)
	request(forgeSocket, "GET", "/api/environments", "", 403)
	stopFactory = startFactory()
	if rebound, err := observed.CollaborationHostByAPIBase(t.Context(), collaboration.Fountain, base.ForgejoURL); err != nil || rebound != bound {
		t.Fatalf("Factory process restart forked provider identity: %+v %v", rebound, err)
	}
	request(factorySocket, "GET", "/api/factory/departments", "", 403)
	stopForge()
	stopFactory()
}

// The native Rust lifecycle writes these real private files before each Go
// process. No appliance config, browser session or mocked database is involved.
func checkNativeFactoryBootstrap(t *testing.T) {
	path := os.Getenv("SODA_NATIVE_FACTORY_CONFIG")
	if path == "" {
		t.Skip("owned native PostgreSQL/bootstrap driver required")
	}
	c, err := factoryruntime.LoadNative(path)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(path)
	if !collaboration.ValidID(string(c.Profile)) || c.Directory != root || c.Socket != filepath.Join(root, "backend.sock") || len(c.IPCKey()) != 32 {
		t.Fatal("native bootstrap lost its owned profile binding")
	}
	key := c.IPCKey()
	originalByte := key[0]
	key[0] ^= 255
	if c.IPCKey()[0] != originalByte {
		t.Fatal("native bootstrap exposed its mutable admission key")
	}
	uri, err := os.ReadFile(os.Getenv("SODA_NATIVE_PG_DSN_FILE"))
	if err != nil || string(uri) != c.DatabaseDSN() {
		t.Fatal("Go bootstrap differs from the owned native PostgreSQL connection")
	}
	encoded, err := json.Marshal(c)
	if err != nil || strings.Contains(string(encoded), c.DatabaseDSN()) || strings.Contains(string(encoded), hex.EncodeToString(c.IPCKey())) || strings.Contains(fmt.Sprint(c), c.DatabaseDSN()) {
		t.Fatal("native bootstrap exposes private database or IPC inputs")
	}
	db, err := store.Open(c.DatabaseDSN())
	if err != nil {
		t.Fatal("native bootstrap could not open the real Factory store")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		t.Fatal("native bootstrap did not reach owned PostgreSQL")
	}
	raw, err := sql.Open("pgx", c.DatabaseDSN())
	if err != nil {
		t.Fatal("native bootstrap inspection could not open")
	}
	defer raw.Close()
	assertNoWork := func() {
		t.Helper()
		var count int
		if err := raw.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM factory_commands)+(SELECT count(*) FROM factory_runs)+(SELECT count(*) FROM projects)+(SELECT count(*) FROM collaboration_profiles)`).Scan(&count); err != nil || count != 0 {
			t.Fatal("native input loading created work, a Project or profile enrollment", count)
		}
	}
	assertNoWork()
	manifest, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	keyPath, dsnPath := filepath.Join(root, "backend.key"), filepath.Join(root, "backend.dsn")
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	phase := os.Getenv("SODA_NATIVE_PG_PHASE")
	if phase == "write" {
		refuse := func(change, restore func()) {
			t.Helper()
			change()
			defer restore()
			if _, err := factoryruntime.LoadNative(path); err == nil || strings.Contains(err.Error(), "never-expose-this-config-secret") || strings.Contains(err.Error(), c.DatabaseDSN()) || strings.Contains(err.Error(), string(keyBytes)) {
				t.Fatal("unsafe native bootstrap was admitted or exposed private input")
			}
		}
		write := func(file string, data []byte) {
			t.Helper()
			if err := os.WriteFile(file, data, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		for _, body := range [][]byte{
			append(append([]byte{}, manifest...), manifest...),
			[]byte(`{"profile_id":"` + string(c.Profile) + `","profile_id":"` + string(c.Profile) + `"}`),
			[]byte(`{"profile_id":"` + string(c.Profile) + `","forgejo_url":"never-expose-this-config-secret"}`),
			[]byte(`{"profile_id":"not-a-profile"}`),
			[]byte(`{"profile_id":"` + factory.NewID() + `"}`),
			[]byte(strings.Repeat("x", 65537)),
		} {
			refuse(func() { write(path, body) }, func() { write(path, manifest) })
		}
		for _, target := range []string{path, keyPath, dsnPath, root} {
			restoreMode := os.FileMode(0o600)
			if target == root {
				restoreMode = 0o700
			}
			refuse(func() {
				if err := os.Chmod(target, restoreMode|0o040); err != nil {
					t.Fatal(err)
				}
			}, func() {
				if err := os.Chmod(target, restoreMode); err != nil {
					t.Fatal(err)
				}
			})
		}
		refuse(func() {
			if err := os.Rename(keyPath, keyPath+".fixture"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(keyPath+".fixture", keyPath); err != nil {
				t.Fatal(err)
			}
		}, func() {
			_ = os.Remove(keyPath)
			if err := os.Rename(keyPath+".fixture", keyPath); err != nil {
				t.Fatal(err)
			}
		})
		refuse(func() { write(keyPath, []byte("invalid")) }, func() { write(keyPath, keyBytes) })
		refuse(func() {
			write(dsnPath, []byte("postgresql://factory:never-expose-this-config-secret@127.0.0.1/postgres"))
		}, func() { write(dsnPath, uri) })
		alias := filepath.Join(root, "alias")
		if err := os.Symlink(root, alias); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(alias) })
		if _, err := factoryruntime.LoadNative(filepath.Join(alias, "backend.json")); err == nil {
			t.Fatal("native bootstrap followed a symlink profile")
		}
		if _, err := factoryruntime.LoadNative("backend.json"); err == nil {
			t.Fatal("native bootstrap selected an implicit relative profile")
		}
	}
	type proof struct {
		Profile collaboration.ProfileID `json:"profile"`
		KeyHash string                  `json:"key_hash"`
	}
	hash := sha256.Sum256(c.IPCKey())
	current := proof{Profile: c.Profile, KeyHash: hex.EncodeToString(hash[:])}
	proofPath := filepath.Join(root, "bootstrap-proof.json")
	if phase != "write" {
		previous, err := os.ReadFile(proofPath)
		var saved proof
		if err != nil || json.Unmarshal(previous, &saved) != nil || saved.Profile != c.Profile || saved.KeyHash == current.KeyHash {
			t.Fatal("native restart replaced profile identity or reused the prior IPC key")
		}
	}
	data, _ := json.Marshal(current)
	if err := os.WriteFile(proofPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := factoryruntime.LoadNative(path); err != nil {
		t.Fatal("refusal checks damaged valid retained native inputs")
	}
	assertNoWork()
}

// Build and run the real native executable against Rust-owned PostgreSQL. The
// single captured bootstrap is reused here to isolate network/process recovery;
// managed per-start key rotation belongs to the Rust supervisor's start ordering.
func checkNativeFactoryProcess(t *testing.T) {
	configPath := os.Getenv("SODA_NATIVE_FACTORY_CONFIG")
	if configPath == "" {
		t.Skip("owned native PostgreSQL/bootstrap driver required")
	}
	c, err := factoryruntime.LoadNative(configPath)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(c.Directory, "soda-factory-server-proof")
	build := exec.CommandContext(t.Context(), "go", "build", "-mod=readonly", "-tags=factory_native", "-o", binary, "./cmd/soda-factory-server")
	build.Dir = "../../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build native Factory: %v\n%s", err, out)
	}
	db, err := store.Open(c.DatabaseDSN())
	if err != nil {
		t.Fatal("could not open native process proof store")
	}
	defer db.Close()
	now := time.Now().UTC()
	retained := factory.Run{ID: factory.NewID(), ProjectID: "p0123456789abcdef01234567", Role: "coder", InputSHA: strings.Repeat("a", 40),
		Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: "muse", Model: "declared-retained-peer"}
	if err := db.RecordFactoryRun(t.Context(), retained); err != nil {
		t.Fatal(err)
	}
	raw, err := sql.Open("pgx", c.DatabaseDSN())
	if err != nil {
		t.Fatal("could not inspect native process proof store")
	}
	defer raw.Close()
	// Declared pending journal fixtures, not launched workers. The production
	// owner/claim methods establish real PostgreSQL liveness for startup recovery.
	var producers []*store.ContributionOwner
	defer func() {
		for _, owner := range producers {
			_ = owner.Close()
		}
	}()
	newProducer := func() *store.ContributionOwner {
		t.Helper()
		owner, err := db.OwnContributions(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		producers = append(producers, owner)
		return owner
	}
	pending := func(owner *store.ContributionOwner) factory.ContributionReview {
		t.Helper()
		id, _, err := db.SaveFeedPost(t.Context(), factory.PostRequest{Repository: 7, Body: "Declared source retained through native startup", IdempotencyKey: factory.NewID()}, "42", "Declared owner", now.Unix())
		if err != nil {
			t.Fatal(err)
		}
		review := factory.ContributionReview{Source: id, SourceKind: factory.PostSourcePost, Repository: 7,
			Body: "Declared source retained through native startup", SourceActor: "42", State: "pending",
			Roles: []factory.ContributionRole{{Assignment: factory.NewID(), Profile: factory.MCPProfileInspector, Capability: factory.NewID(), Scope: "Declared pending assessment"}}}
		if claimed, err := owner.ClaimContributionReview(t.Context(), review); err != nil || !claimed {
			t.Fatal("could not seed owned pending assessment", claimed, err)
		}
		return review
	}
	liveOwner, releasedOwner, malformedOwner := newProducer(), newProducer(), newProducer()
	liveReview := pending(liveOwner)
	var released []factory.ContributionReview
	for range 33 { // More than one bounded recovery page.
		released = append(released, pending(releasedOwner))
	}
	unknownReview := pending(releasedOwner)
	if _, err := raw.ExecContext(t.Context(), `UPDATE factory_feed_posts SET data=data-'contribution_owner' WHERE id=$1`, factory.ContributionReviewInvitationID(unknownReview)); err != nil {
		t.Fatal(err)
	}
	malformedReview := pending(malformedOwner)
	if err := releasedOwner.Close(); err != nil {
		t.Fatal(err)
	}
	assertRecovery := func(malformedRecovered bool) {
		t.Helper()
		check := func(review factory.ContributionReview, recovered bool) {
			t.Helper()
			if recovered {
				review.State = "unavailable"
			}
			detail, err := db.FeedPost(t.Context(), factory.ContributionReviewInvitationID(review))
			want, _ := json.Marshal(review)
			got, _ := json.Marshal(detail.Contribution)
			if err != nil || string(want) != string(got) {
				t.Fatal("native startup changed assessment scope or guessed live/unknown ownership", err)
			}
		}
		for _, review := range released {
			check(review, true)
		}
		check(liveReview, false)
		check(unknownReview, false)
		check(malformedReview, malformedRecovered)
	}
	snapshot := func() string {
		t.Helper()
		var data string
		if err := raw.QueryRowContext(t.Context(), `SELECT jsonb_build_array(
		 (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM factory_runs r),
		 (SELECT jsonb_agg(to_jsonb(c) ORDER BY id) FROM factory_commands c),
		 (SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM factory_feed_posts p WHERE NOT data ? 'contribution'),
		 (SELECT count(*) FROM factory_assignments),(SELECT count(*) FROM factory_priority_judgments),
		 (SELECT count(*) FROM factory_usage),(SELECT count(*) FROM factory_post_intakes),
		 (SELECT count(*) FROM projects),(SELECT count(*) FROM collaboration_profiles),
		 (SELECT count(*) FROM collaboration_runtimes))::text`).Scan(&data); err != nil {
			t.Fatal(err)
		}
		return data
	}
	before := snapshot()
	token := hex.EncodeToString(c.IPCKey())
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", c.Socket)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	request := func(method, path string, headers http.Header) (int, []byte, http.Header, error) {
		req, err := http.NewRequestWithContext(t.Context(), method, "http://factory"+path, nil)
		if err != nil {
			return 0, nil, nil, err
		}
		req.Header = headers
		res, err := client.Do(req)
		if err != nil {
			return 0, nil, nil, err
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		return res.StatusCode, body, res.Header, err
	}
	admitted := http.Header{"Authorization": {"Bearer " + token}}
	type child struct {
		cmd     *exec.Cmd
		done    chan error
		log     *os.File
		stopped bool
	}
	var children []*child
	launch := func() *child {
		t.Helper()
		log, err := os.OpenFile(filepath.Join(c.Directory, fmt.Sprintf("backend-process-%d.log", len(children))), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(binary, "--config="+configPath)
		cmd.Stdout, cmd.Stderr = log, log
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		p := &child{cmd: cmd, done: make(chan error, 1), log: log}
		children = append(children, p)
		go func() { p.done <- cmd.Wait() }()
		t.Cleanup(func() {
			if !p.stopped {
				_ = cmd.Process.Kill()
				<-p.done
			}
			_ = log.Close()
		})
		return p
	}
	waitExit := func(p *child, success bool) {
		t.Helper()
		select {
		case err := <-p.done:
			p.stopped = true
			if (err == nil) != success {
				t.Fatal("native process exit did not match its admitted lifetime", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("native process did not exit within its bounded lifetime")
		}
		if data, err := os.ReadFile(p.log.Name()); err != nil || strings.Contains(string(data), c.DatabaseDSN()) || strings.Contains(string(data), token) {
			t.Fatal("native process log exposed private inputs or could not be read")
		}
	}
	waitReady := func(p *child) string {
		t.Helper()
		for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(25 * time.Millisecond) {
			code, body, headers, err := request(http.MethodGet, "/healthz", admitted)
			if err == nil && code == 200 {
				var ready map[string]string
				if json.Unmarshal(body, &ready) != nil || ready["status"] != "ready" || ready["profile"] != string(c.Profile) || !collaboration.ValidID(ready["instance"]) || headers.Get("Cache-Control") != "no-store" {
					t.Fatal("native readiness lost profile/instance binding", string(body))
				}
				return ready["instance"]
			}
			select {
			case err := <-p.done:
				p.stopped = true
				t.Fatal("native process exited before readiness", err)
			default:
			}
		}
		t.Fatal("native process did not become ready")
		return ""
	}
	assertRequest := func(method, path string, headers http.Header, expected int) {
		t.Helper()
		code, body, responseHeaders, err := request(method, path, headers)
		if err != nil || code != expected || responseHeaders.Get("Cache-Control") != "no-store" || strings.Contains(string(body), token) || strings.Contains(string(body), c.DatabaseDSN()) {
			t.Fatal("native IPC admission", method, path, code, err)
		}
	}
	first := launch()
	instance := waitReady(first)
	assertRecovery(false)
	if info, err := os.Lstat(c.Socket); err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm() != 0o600 {
		t.Fatal("native backend socket is not private")
	}
	for _, header := range []http.Header{
		{}, {"Authorization": {"Bearer wrong"}}, {"Authorization": {token}},
		{"Authorization": {"Bearer " + token, "Bearer " + token}},
	} {
		assertRequest(http.MethodGet, "/healthz", header, 401)
	}
	for _, name := range []string{"Origin", "Sec-Fetch-Site"} {
		header := admitted.Clone()
		header.Set(name, "https://unadmitted.test")
		assertRequest(http.MethodGet, "/healthz", header, 403)
	}
	assertRequest(http.MethodHead, "/healthz", admitted, 405)
	assertRequest(http.MethodGet, "/healthz?other=1", admitted, 400)
	// IPC owns a process channel, not a provider/account login. Product routes
	// are mounted but cannot disclose facts or create work without a connector.
	assertRequest(http.MethodGet, "/api/factory/identity", admitted, 503)
	assertRequest(http.MethodGet, "/api/factory/repositories", admitted, 503)
	assertRequest(http.MethodGet, "/api/factory/departments", admitted, 503)
	second := launch()
	waitExit(second, false)
	if current := waitReady(first); current != instance {
		t.Fatal("second native process replaced the live instance")
	}
	_ = first.cmd.Process.Signal(syscall.SIGTERM)
	waitExit(first, true)
	transport.CloseIdleConnections()
	if _, err := os.Lstat(c.Socket); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("clean native shutdown retained its socket", err)
	}
	third := launch()
	if next := waitReady(third); next == instance {
		t.Fatal("new native process reused the previous process identity")
	}
	_ = third.cmd.Process.Kill()
	waitExit(third, false)
	transport.CloseIdleConnections()
	if _, err := os.Lstat(c.Socket); err != nil {
		t.Fatal("abrupt-stop proof did not retain its stale socket")
	}
	fourth := launch()
	waitReady(fourth)
	_ = fourth.cmd.Process.Signal(syscall.SIGTERM)
	waitExit(fourth, true)
	transport.CloseIdleConnections()
	marker := []byte("foreign endpoint: preserve exactly")
	if err := os.WriteFile(c.Socket, marker, 0o600); err != nil {
		t.Fatal(err)
	}
	waitExit(launch(), false)
	if data, err := os.ReadFile(c.Socket); err != nil || string(data) != string(marker) {
		t.Fatal("native startup replaced a foreign socket-path resource")
	}
	if err := os.Remove(c.Socket); err != nil {
		t.Fatal(err)
	}
	foreign, err := net.ListenUnix("unix", &net.UnixAddr{Name: c.Socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	waitExit(launch(), false)
	probe, err := net.DialTimeout("unix", c.Socket, time.Second)
	if err != nil {
		t.Fatal("native startup removed an unowned live listener")
	}
	_ = probe.Close()
	_ = foreign.Close()
	// Shutdown owns the inode it bound, not anything later placed at its name.
	replaced := launch()
	waitReady(replaced)
	transport.CloseIdleConnections()
	ownedSocket := c.Socket + ".owned-fixture"
	if err := os.Rename(c.Socket, ownedSocket); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.Socket, marker, 0o600); err != nil {
		t.Fatal(err)
	}
	_ = replaced.cmd.Process.Signal(syscall.SIGTERM)
	waitExit(replaced, true)
	if data, err := os.ReadFile(c.Socket); err != nil || string(data) != string(marker) {
		t.Fatal("native shutdown removed or changed a replacement resource")
	}
	if err := os.Remove(c.Socket); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(ownedSocket); err != nil {
		t.Fatal(err)
	}
	// A signal must cancel actual SQL schema admission, before readiness, while
	// the PostgreSQL blocker remains held. It cannot wait for a future unlock.
	held, err := raw.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Rollback()
	if _, err := held.ExecContext(t.Context(), "LOCK TABLE schema_version IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	waiting := launch()
	blocked := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(25 * time.Millisecond) {
		var count int
		if err := raw.QueryRowContext(t.Context(), "SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND pid<>pg_backend_pid()").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count > 0 {
			blocked = true
			break
		}
	}
	if !blocked {
		t.Fatal("native startup did not reach the actual held SQL schema lock")
	}
	_ = waiting.cmd.Process.Signal(syscall.SIGTERM)
	waitExit(waiting, true)
	if err := held.Rollback(); err != nil {
		t.Fatal(err)
	}
	// Recovery is part of startup admission too: readiness cannot precede a
	// held assessment lock, and a signal must cancel that wait before unlock.
	recoveryLock, err := raw.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer recoveryLock.Rollback()
	if _, err := recoveryLock.ExecContext(t.Context(), `SELECT id FROM factory_feed_posts WHERE id=$1 FOR UPDATE`, factory.ContributionReviewInvitationID(liveReview)); err != nil {
		t.Fatal(err)
	}
	recovering := launch()
	blocked = false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(25 * time.Millisecond) {
		var count int
		if err := raw.QueryRowContext(t.Context(), `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database()
 AND wait_event_type='Lock' AND query LIKE '%contribution_owner%' AND pid<>pg_backend_pid()`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count > 0 {
			blocked = true
			break
		}
	}
	if !blocked {
		t.Fatal("native startup did not reach the held recovery row")
	}
	if code, _, _, err := request(http.MethodGet, "/healthz", admitted); err == nil && code == 200 {
		t.Fatal("native readiness escaped unfinished retained recovery")
	}
	_ = recovering.cmd.Process.Signal(syscall.SIGTERM)
	waitExit(recovering, true)
	if err := recoveryLock.Rollback(); err != nil {
		t.Fatal(err)
	}
	transport.CloseIdleConnections()
	assertRecovery(false)
	// Unknown/malformed released state refuses startup without exposing retained
	// source contents. Restoring that same receipt lets the next start recover it.
	if err := malformedOwner.Close(); err != nil {
		t.Fatal(err)
	}
	const privateSource = "declared-private-invalid-recovery-source"
	assessmentID := factory.ContributionReviewInvitationID(malformedReview)
	if _, err := raw.ExecContext(t.Context(), `UPDATE factory_feed_posts SET data=jsonb_set(data,'{contribution,source}',to_jsonb($2::text)) WHERE id=$1`, assessmentID, privateSource); err != nil {
		t.Fatal(err)
	}
	malformed := launch()
	waitExit(malformed, false)
	if log, err := os.ReadFile(malformed.log.Name()); err != nil || strings.Contains(string(log), privateSource) {
		t.Fatal("native recovery failure disclosed retained source details", err)
	}
	if _, err := raw.ExecContext(t.Context(), `UPDATE factory_feed_posts SET data=jsonb_set(data,'{contribution,source}',to_jsonb($2::text)) WHERE id=$1`, assessmentID, malformedReview.Source); err != nil {
		t.Fatal(err)
	}
	recovered := launch()
	waitReady(recovered)
	assertRecovery(true)
	_ = recovered.cmd.Process.Signal(syscall.SIGTERM)
	waitExit(recovered, true)
	transport.CloseIdleConnections()
	if snapshot() != before {
		t.Fatal("native process startup, readiness or restart changed retained work/ownership")
	}
}

func mustProcessRequest(t *testing.T, method, url, body string) *http.Request {
	t.Helper()
	r, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(fmt.Errorf("product request: %w", err))
	}
	r.Header.Set("Content-Type", "application/json")
	return r
}
