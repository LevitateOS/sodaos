//go:build linux

package terminal

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host/publish"
	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

// Only the container boundary is replaced. The production export program
// and all Git commands run unchanged against disposable local files.
type exportExec struct {
	root        string
	outputBytes int
	err         error
}

func (f *exportExec) Run(ctx context.Context, in []byte, name string, args ...string) ([]byte, error) {
	if name != "/usr/bin/podman" || len(args) < 2 || args[0] != "--remote=false" {
		return nil, errors.New("unexpected export command")
	}
	if args[1] == "inspect" {
		return []byte(takeoverInspect(takeoverProject, true)), nil
	}
	if len(args) < 6 || args[1] != "exec" || args[2] != "--user" || args[3] != domain.RoleCoder || args[4] != takeoverContainer {
		return nil, errors.New("export did not use the recorded role and container")
	}
	if f.err != nil {
		return nil, f.err
	}
	local := append([]string(nil), args[5:]...)
	for i, arg := range local {
		local[i] = strings.ReplaceAll(arg, "/home/"+domain.RoleCoder, f.root)
	}
	command := exec.CommandContext(ctx, local[0], local[1:]...)
	command.Stdin = strings.NewReader(string(in))
	out, err := command.Output()
	f.outputBytes = len(out)
	return out, err
}

func (f *exportExec) RunReader(ctx context.Context, in io.Reader, name string, args ...string) ([]byte, error) {
	return f.Run(ctx, nil, name, args...)
}

type exportFixture struct {
	root, source, base, candidate string
	exec                          *exportExec
}

func (f exportFixture) git(t *testing.T, args ...string) string {
	t.Helper()
	command := exec.Command("/usr/bin/git", args...)
	command.Dir = f.source
	command.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + f.root, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null"}
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture Git: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func (f exportFixture) commit(t *testing.T, name string, data []byte) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.source, name), data, 0o600); err != nil {
		t.Fatal(err)
	}
	f.git(t, "add", name)
	f.git(t, "commit", "-m", "fixture")
	return f.git(t, "rev-parse", "HEAD")
}

func newExportFixture(t *testing.T) exportFixture {
	t.Helper()
	f := exportFixture{root: t.TempDir()}
	f.source = filepath.Join(f.root, "checkouts", takeoverPrep)
	f.exec = &exportExec{root: f.root}
	if err := os.MkdirAll(f.source, 0o700); err != nil {
		t.Fatal(err)
	}
	f.git(t, "init", "--initial-branch=main")
	f.git(t, "config", "user.name", "soda-tester")
	f.git(t, "config", "user.email", "soda-tester@localhost")
	f.base = f.commit(t, "README.md", []byte("base"))
	f.candidate = f.commit(t, "README.md", []byte("candidate"))
	return f
}

func (f exportFixture) export(ctx context.Context, candidate string) ([]byte, error) {
	s := &Service{Exec: f.exec}
	return s.FactoryExportBundle(ctx, takeoverProject, takeoverContainer, domain.RoleCoder, takeoverPrep, candidate)
}

func (f exportFixture) assertCleaned(t *testing.T) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(f.root, "checkouts", ".soda-export-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("export workspace remains: %v %v", matches, err)
	}
}

func TestFactoryExportBundleReadsExactCandidate(t *testing.T) {
	f := newExportFixture(t)
	later := f.commit(t, "README.md", []byte("later work must not be exported"))
	// The exporter must never parse source or inherited Git configuration.
	badConfig := filepath.Join(f.source, ".git", "config")
	if err := os.WriteFile(badConfig, []byte("[invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", badConfig)
	t.Setenv("GIT_CONFIG_SYSTEM", badConfig)
	bundle, err := f.export(t.Context(), f.candidate)
	if err != nil {
		t.Fatal(err)
	}
	f.assertCleaned(t)
	path := filepath.Join(f.root, "candidate.bundle")
	if err := os.WriteFile(path, bundle, 0o600); err != nil {
		t.Fatal(err)
	}
	clean := f
	clean.source = f.root
	if got := clean.git(t, "bundle", "list-heads", path); got != f.candidate+" HEAD" {
		t.Fatalf("export refs: %q", got)
	}
	ref, err := os.ReadFile(filepath.Join(f.source, ".git", "refs", "heads", "main"))
	if err != nil || strings.TrimSpace(string(ref)) != later {
		t.Fatalf("source ref changed: %v", err)
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	token := filepath.Join(root, "token")
	if err := os.WriteFile(token, []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := publish.Config{Root: root, Remote: "http://127.0.0.1:1/target.git", Username: "soda-tester", TokenFile: token}
	now := time.Now()
	run := factory.Run{ID: factory.NewID(), ProjectID: takeoverProject, Role: "coder", InputSHA: f.base,
		Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("a", 64), Harness: "test", Model: "test"}
	if err := cfg.ValidateCandidate(t.Context(), publish.Request{Run: run, BaseSHA: f.base, Commit: f.candidate, Bundle: bundle}); err != nil {
		t.Fatalf("real export rejected by retained publisher: %v", err)
	}
}

func TestFactoryExportBundleRefusesWrongCandidate(t *testing.T) {
	f := newExportFixture(t)
	for _, candidate := range []string{strings.Repeat("f", 40), f.git(t, "rev-parse", "HEAD:README.md")} {
		if _, err := f.export(t.Context(), candidate); !errors.Is(err, domain.ErrFactoryExportCandidate) {
			t.Fatalf("non-commit candidate: %v", err)
		}
		f.assertCleaned(t)
	}
}

func TestFactoryExportBundleBoundsNativeOutput(t *testing.T) {
	f := newExportFixture(t)
	large := make([]byte, domain.MaxFactoryExportBundle+65536)
	if _, err := rand.Read(large); err != nil {
		t.Fatal(err)
	}
	candidate := f.commit(t, "large.bin", large)
	if _, err := f.export(t.Context(), candidate); !errors.Is(err, domain.ErrFactoryExportBounds) {
		t.Fatalf("oversized candidate: %v", err)
	}
	if f.exec.outputBytes != domain.MaxFactoryExportBundle+1 {
		t.Fatalf("native output bound: %d", f.exec.outputBytes)
	}
	f.assertCleaned(t)
}

func TestFactoryExportBundleRefusesStaleIncarnation(t *testing.T) {
	candidate := strings.Repeat("a", 40)
	s := &Service{Exec: &exportExec{root: t.TempDir()}}
	if _, err := s.FactoryExportBundle(context.Background(), takeoverProject, strings.Repeat("d", 64), domain.RoleCoder, takeoverPrep, candidate); !errors.Is(err, identity.ErrStale) {
		t.Fatalf("replaced container: %v", err)
	}
	if _, err := s.FactoryExportBundle(context.Background(), takeoverProject, takeoverContainer, domain.RoleCoder, "bad", candidate); !errors.Is(err, identity.ErrDenied) {
		t.Fatalf("bad preparation: %v", err)
	}
}

func TestFactoryExportBundleDistinguishesMissingAndTransport(t *testing.T) {
	f := newExportFixture(t)
	if err := os.RemoveAll(f.source); err != nil {
		t.Fatal(err)
	}
	if _, err := f.export(t.Context(), f.candidate); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("missing checkout: %v", err)
	}
	f.exec.err = errors.New("untrusted native diagnostic")
	if _, err := f.export(t.Context(), f.candidate); err == nil || err.Error() != "candidate export execution unconfirmed" {
		t.Fatalf("transport classified or diagnostic exposed: %v", err)
	}
}
