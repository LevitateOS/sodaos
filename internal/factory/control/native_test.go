package control

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// Uses task-owned disposable state and no model calls. Cleanup must work even
// when Forgejo and provider authentication are unavailable after a restart.
func TestNativeWithdrawAndRecover(t *testing.T) {
	path := os.Getenv("SODA_FACTORY_CONTROLLER_CONFIG")
	if path == "" {
		t.Skip("private native controller fixture required")
	}
	config, policy, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	online, err := Open(t.Context(), config, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = online.Store.Close() }()
	base, err := online.Publisher.BranchRevision(t.Context(), online.Repository.DefaultBranch)
	if err != nil {
		t.Fatal(err)
	}
	source, err := online.Publisher.Source(t.Context(), base)
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"cancel", "recover"} {
		t.Run(operation, func(t *testing.T) { nativeWithdrawal(t, config, policy, base, source, operation) })
	}
}

func nativeWithdrawal(t *testing.T, config Config, policy, base string, source []byte, operation string) {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	config.Root = root
	config.Workspace.Root = filepath.Join(root, "workspaces")
	config.Workspace.CredentialHome = filepath.Join(root, "credentials")
	for _, path := range []string{config.Workspace.Root, config.Workspace.CredentialHome} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	local, err := OpenLocal(config, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = local.Store.Close() }()
	a, err := factory.New(factory.WorkItem{RepositoryID: config.RepositoryID, Issue: 1, HumanID: 1, Objective: "native cancellation fixture", BaseSHA: base, PolicySHA: policy}, factory.NewID(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	a, _, err = local.Store.AdmitFactory(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	copy := a
	r, err := copy.BeginRun(factory.Implementation, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	config.Workspace.Bind(&r)
	if err = local.Store.StartFactoryRun(t.Context(), &a, r); err != nil {
		t.Fatal(err)
	}
	if err = local.Workspace.Prepare(r, map[string]string{"role": "fixture"}, source); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		ctx, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if err := local.Workspace.Cleanup(ctx, &r); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(cleanup)
	if err = local.Workspace.Create(t.Context(), &r, func(run factory.Run) error { return local.Store.SaveFactoryRun(t.Context(), run) }); err != nil {
		t.Fatal(err)
	}
	if err = local.Workspace.Initialize(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	// A subprocess survives its launching shell until whole-container withdrawal.
	id := r.Resources[2].ID
	if _, err = local.Workspace.Exec.Run(t.Context(), nil, "podman", "exec", id, "sh", "-c", "sleep 300 >/dev/null 2>&1 &"); err != nil {
		t.Fatal(err)
	}
	if operation == "recover" {
		if err = local.Store.Close(); err != nil {
			t.Fatal(err)
		}
		local, err = OpenLocal(config, policy)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = local.Store.Close() }()
		err = local.Recover(t.Context())
	} else {
		err = local.Cancel(t.Context(), a.ID)
	}
	if err != nil {
		t.Fatal(err)
	}
	got, err := local.Store.FactoryAttempt(t.Context(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := factory.Cancelled
	if operation == "recover" {
		want = factory.NeedsHuman
	}
	if got.Outcome != want || !got.CleanupComplete || got.Authority(time.Now()) == nil {
		t.Fatal("withdrawal retained authority or resources")
	}
	if _, err = local.Workspace.Exec.Run(t.Context(), nil, "podman", "container", "exists", id); err == nil {
		t.Fatal("surviving subprocess container was retained")
	}
}
