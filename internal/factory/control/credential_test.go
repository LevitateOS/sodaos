package control

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host/workspace"
	"github.com/levitateos/sodaos/internal/store"
)

func TestDelegatedCredentialSnapshotUsesInjectedBytes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "auth.json")
	original := []byte(`{"tokens":{"access_token":"synthetic-injected-credential"}}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(filepath.Join(root, "execution.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	a := testAttempt(t, s)
	copy := a
	r, err := copy.BeginRun(factory.Implementation, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Image = "sha256:" + strings.Repeat("c", 64)
	r.Harness, r.Model = "test", "test"
	r.CredentialClaimed = true
	if err = s.StartFactoryRun(t.Context(), &a, r); err != nil {
		t.Fatal(err)
	}
	runtime := &workspace.Runtime{Exec: executorFunc(func(_ context.Context, data []byte, _ string, _ ...string) ([]byte, error) {
		if string(data) != string(original) {
			t.Fatal("injected different credential state")
		}
		return nil, os.WriteFile(path, []byte(`{"tokens":{"access_token":"synthetic-later-enrollment"}}`), 0o600)
	})}
	c := Controller{Store: s, Config: Config{Workspace: workspace.Config{CredentialHome: root}}, Workspace: runtime}
	secrets, err := c.delegateCredential(t.Context(), &r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resultBytes(factory.Result{Summary: "synthetic-injected-credential"}, secrets); err == nil {
		t.Fatal("later enrollment displaced injected credential snapshot")
	}
}

func TestSaveCredentialPreservesEnrollment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	original := []byte(`{"tokens":{"refresh_token":"synthetic-original"}}`)
	renewed := []byte(`{"tokens":{"refresh_token":"synthetic-renewed"}}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveCredential(path, credentialSHA(original), renewed); err != nil {
		t.Fatal(err)
	}
	if err := saveCredential(path, credentialSHA(original), original); err == nil {
		t.Fatal("overwrote changed enrollment")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(renewed) {
		t.Fatal("lost maintained credential state")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatal("credential permissions widened")
	}
	if err := saveCredential(path, credentialSHA(renewed), []byte("invalid")); err == nil {
		t.Fatal("accepted invalid state")
	}
}

func TestCredentialStreamRequiresReenrollmentAfterLostReturn(t *testing.T) {
	root := t.TempDir()
	state := []byte(`{"tokens":{"refresh_token":"synthetic-first"}}`)
	if err := os.WriteFile(filepath.Join(root, "auth.json"), state, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(filepath.Join(root, "execution.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	a := testAttempt(t, s)
	copy := a
	r, err := copy.BeginRun(factory.Implementation, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Image = "sha256:" + strings.Repeat("c", 64)
	r.Harness = "test"
	r.Model = "test"
	if err := s.StartFactoryRun(t.Context(), &a, r); err != nil {
		t.Fatal(err)
	}
	r.CredentialClaimed = true
	r.CredentialDelegated = true
	r.CredentialSeedSHA = credentialSHA(state)
	if err := s.SaveFactoryRun(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	c := Controller{Store: s, Config: Config{Workspace: workspace.Config{CredentialHome: root}}}
	if err := c.checkCredentialStream(t.Context()); err == nil {
		t.Fatal("reused running credential stream")
	}
	r.Outcome = factory.NeedsHuman
	r.CleanupComplete = true
	if err := s.SaveFactoryRun(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	if err := c.checkCredentialStream(t.Context()); err == nil {
		t.Fatal("cleanup allowed stale credential reuse")
	}
	if err := os.WriteFile(filepath.Join(root, "auth.json"), []byte(`{"tokens":{"refresh_token":"synthetic-new-enrollment"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := c.checkCredentialStream(t.Context()); err != nil {
		t.Fatal(err)
	}
}
