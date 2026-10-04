package client

// Compatibility between the Go identity client and the Rust broker
// binary: builds soda-identity, seeds one subscription connection and
// drives every client call against it. Skips without the A10 fixture,
// a writable tmpfs provider root (/dev/shm) or a cargo toolchain.
import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

const compatCredential = `{"schema_version":1,"providers":{"meta":{"access_token":"synthetic","api_key":"synthetic","api_base_url":"https://api.meta.ai/v1","mechanism":"oauth","obtained_via":"device_code"}}}`

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
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

func buildBroker(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo unavailable")
	}
	root := repoRoot(t)
	build := exec.Command("cargo", "build", "-p", "soda-identity", "--bin", "soda-identity")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build soda-identity: %v\n%s", err, out)
	}
	return filepath.Join(root, "target", "debug", "soda-identity")
}

// tmpfsRoot rejects non-tmpfs parents: provider enrollment roots must be
// private tmpfs, and the broker validates that before serving.
func tmpfsRoot(t *testing.T) string {
	t.Helper()
	parent := filepath.Join("/dev/shm", fmt.Sprintf("soda-compat-%d", os.Getpid()))
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Skipf("tmpfs provider root unavailable: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	return parent
}

func fakeCodex(t *testing.T, root string) (binary, sum string) {
	t.Helper()
	binary = filepath.Join(root, "codex-fixture")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'codex-cli 0.153.4'; exit 0; fi\necho unexpected >&2\nexit 1\n"
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(script))
	return binary, hex.EncodeToString(digest[:])
}

// stubHost answers broker callbacks: validate/stop succeed, finish
// returns the maintained credential.
func stubHost(t *testing.T, socket, credential string) {
	t.Helper()
	_ = os.Remove(socket)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	mux := http.NewServeMux()
	mux.HandleFunc("/identity/validate", func(w http.ResponseWriter, r *http.Request) {
		var wire struct {
			Lease identity.Lease `json:"lease"`
		}
		if json.NewDecoder(r.Body).Decode(&wire) != nil || wire.Lease.Binding == nil {
			http.Error(w, "denied", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct{}{})
	})
	mux.HandleFunc("/identity/stop", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct{}{})
	})
	mux.HandleFunc("/identity/finish", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"lease":      map[string]any{},
			"credential": base64.StdEncoding.EncodeToString([]byte(credential)),
		})
	})
	server := &http.Server{Handler: mux}
	t.Cleanup(func() { _ = server.Close() })
	go func() { _ = server.Serve(listener) }()
}

func waitForSocket(t *testing.T, path string) {
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

func TestRustBrokerCompatibility(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	binary := buildBroker(t)
	shm := tmpfsRoot(t)
	ctx := context.Background()

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	db, dsn, cleanup, err := store.OpenEphemeral(ctx, admin, key)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)

	scratch := t.TempDir()
	writeSecret := func(name, content string) string {
		path := filepath.Join(scratch, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	dsnFile := writeSecret("dsn", dsn+"\n")
	keyFile := writeSecret("key", base64.StdEncoding.EncodeToString(key)+"\n")
	adminSock := filepath.Join(scratch, "admin.sock")
	runtimeSock := filepath.Join(scratch, "runtime.sock")
	hostSock := filepath.Join(scratch, "host.sock")
	stubHost(t, hostSock, compatCredential)

	codexBinary, codexSum := fakeCodex(t, scratch)
	codexRoot := filepath.Join(shm, "codex")
	if err := os.MkdirAll(codexRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	settings, _ := json.Marshal(map[string]any{
		"database_dsn_file": dsnFile,
		"key_file":          keyFile,
		"admin_socket":      adminSock,
		"runtime_socket":    runtimeSock,
		"host_socket":       hostSock,
		"codex": map[string]any{
			"binary": codexBinary, "version": "0.153.4", "sha256": codexSum, "root": codexRoot,
		},
		"muse": map[string]any{},
	})
	config := filepath.Join(scratch, "identity.json")
	if err := os.WriteFile(config, settings, 0o600); err != nil {
		t.Fatal(err)
	}

	broker := exec.Command(binary, "--config", config)
	var brokerLog bytes.Buffer
	broker.Stdout = &brokerLog
	broker.Stderr = &brokerLog
	if err := broker.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = broker.Process.Kill()
		_ = broker.Wait()
	})
	waitForSocket(t, adminSock)
	waitForSocket(t, runtimeSock)

	// Seed one subscription connection directly; enrollment needs native binaries.
	conn := identity.Connection{
		ProviderID: identity.Codex, ID: "conn-compat", OwnerID: 1,
		Label: "compat", Email: "soda-tester@example.invalid", Plan: "plus",
		Generation: 1, State: identity.Ready,
	}
	if err := db.IdentitySaveConnection(ctx, conn, []byte(compatCredential)); err != nil {
		t.Fatal(err)
	}

	adminClient := New(adminSock)
	runtimeClient := New(runtimeSock)

	connections, err := adminClient.Connections(ctx, 1)
	if err != nil || len(connections) != 1 || connections[0].ID != "conn-compat" {
		t.Fatalf("connections: %v %+v", err, connections)
	}
	available, err := adminClient.Available(ctx, 1, "project")
	if err != nil || len(available) != 1 {
		t.Fatalf("available: %v", err)
	}
	grant, err := adminClient.CreateGrant(ctx, 1, identity.GrantRequest{
		ConnectionID: "conn-compat", UserID: 2, ProjectID: "project",
		ConfirmSubscription: true, ConfirmCredentialExposure: true,
	})
	if err != nil || grant.Revision != 1 {
		t.Fatalf("create grant: %v", err)
	}

	deadline := time.Now().Add(time.Hour)
	lease, err := runtimeClient.Acquire(ctx, identity.AcquireRequest{
		ProviderID: identity.Codex, ExecutionID: "execution-compat", ActorID: 2,
		ConnectionID: "conn-compat", ProjectID: "project", Kind: identity.Factory,
		Deadline: deadline,
	})
	if err != nil || lease.ConnectionID != "conn-compat" || lease.GrantID != grant.ID {
		t.Fatalf("acquire: %v %+v", err, lease)
	}
	execution, err := runtimeClient.GetExecution(ctx, identity.Factory, "execution-compat")
	if err != nil || execution.State != "live" || execution.LeaseID != lease.ID {
		t.Fatalf("get execution: %v %+v", err, execution)
	}
	binding := identity.Binding{Kind: identity.Factory, ID: "execution-compat", Generation: 1}
	delivery, err := runtimeClient.Register(ctx, lease.ID, binding)
	if err != nil || string(delivery.Credential) != compatCredential {
		t.Fatalf("register: %v", err)
	}
	_ = delivery
	leases, err := adminClient.Leases(ctx, 1, "conn-compat")
	if err != nil || len(leases) != 1 || leases[0].Binding != nil {
		t.Fatalf("leases strip bindings: %v %+v", err, leases)
	}
	if err := runtimeClient.Return(ctx, lease.ID, binding, []byte(compatCredential)); err != nil {
		t.Fatalf("return: %v", err)
	}
	execution, err = runtimeClient.GetExecution(ctx, identity.Factory, "execution-compat")
	if err != nil || execution.State != "terminal" {
		t.Fatalf("terminal execution: %v %+v", err, execution)
	}

	// Error codes cross the wire unchanged.
	if _, err := runtimeClient.GetExecution(ctx, identity.Factory, "missing"); err != identity.ErrNotFound {
		t.Fatalf("missing execution: %v", err)
	}
	if err := runtimeClient.CloseExecution(ctx, identity.Factory, "execution-compat"); err != nil {
		t.Fatalf("close execution: %v", err)
	}
	if err := adminClient.RevokeGrant(ctx, 1, grant.ID); err != nil {
		t.Fatalf("revoke grant: %v", err)
	}
	if err := adminClient.Revoke(ctx, 1, "conn-compat"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := adminClient.Connections(ctx, 0); err != identity.ErrDenied {
		t.Fatalf("owner check: %v", err)
	}
	t.Logf("broker log: %s", brokerLog.String())
}
