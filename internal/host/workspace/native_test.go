package workspace

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// This opt-in development check uses only resources recorded in its own run.
func TestNativeLifecycle(t *testing.T) {
	path := os.Getenv("SODA_FACTORY_NATIVE_CONFIG")
	if path == "" {
		t.Skip("isolated native configuration required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config Config
	if err = json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	runtime, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	r := testRun()
	r.ID = factory.NewID()
	r.Resources = nil
	config.Bind(&r)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	source := t.TempDir()
	git := func(args ...string) []byte {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = source
		out, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture git: %v", err)
		}
		return out
	}
	git("init", "--initial-branch=main")
	git("config", "user.name", "soda-tester")
	git("config", "user.email", "soda-tester@localhost")
	if err = os.WriteFile(filepath.Join(source, "input.txt"), []byte("isolated fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", "input.txt")
	git("commit", "-m", "fixture")
	sha := git("rev-parse", "HEAD")
	r.InputSHA = string(sha[:40])
	bundlePath := filepath.Join(source, "source.bundle")
	git("bundle", "create", bundlePath, "main")
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if err = runtime.Prepare(r, map[string]string{"role": "implementation"}, bundle); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if err := runtime.Cleanup(cleanup, &r); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	ledger := filepath.Join(config.Root, r.ID, "run.json")
	persist := func(run factory.Run) error {
		data, err := json.Marshal(run)
		if err != nil {
			return err
		}
		return os.WriteFile(ledger, data, 0o600)
	}
	if err = persist(r); err != nil {
		t.Fatal(err)
	}
	if err = runtime.Create(ctx, &r, persist); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err = runtime.Initialize(ctx, r); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	id := runtime.resource(r, "workspace").ID
	out, err := runtime.Exec.Run(ctx, nil, "podman", "exec", id, "git", "-C", "/workspace/repo", "rev-parse", "HEAD")
	if err != nil || string(out) != r.InputSHA+"\n" {
		t.Fatal("checkout differs from admitted commit")
	}
	bundle, err = runtime.ExportCandidate(ctx, r)
	if err != nil || len(bundle) == 0 {
		t.Fatalf("export frozen candidate: %v", err)
	}
	// Reconstruct the runtime and record to exercise restart cleanup, not names guessed from the engine.
	var recovered factory.Run
	data, err = os.ReadFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &recovered); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	if err = restarted.Cleanup(ctx, &recovered); err != nil {
		t.Fatalf("restart cleanup: %v", err)
	}
	if !recovered.CleanupComplete {
		t.Fatal("cleanup was not recorded")
	}
}
