package publish

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func candidateFixture(t *testing.T, changed string) (Config, Request) {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = source
		out, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git: %v", err)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "--initial-branch=main")
	git("config", "user.name", "soda-tester")
	git("config", "user.email", "soda-tester@localhost")
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-m", "base")
	base := git("rev-parse", "HEAD")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(source, changed)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, changed), []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-m", "candidate")
	sha := git("rev-parse", "HEAD")
	path := filepath.Join(root, "candidate.bundle")
	git("bundle", "create", path, "HEAD")
	bundle, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	token := filepath.Join(root, "token")
	if err := os.WriteFile(token, []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	r := factory.Run{ID: factory.NewID(), ProjectID: "p123456789012345678901234", Role: "soda-coder", InputSHA: base, Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("b", 64), Harness: "test", Model: "test"}
	return Config{Root: root, Remote: "http://127.0.0.1:1/target.git", Username: "soda-tester", TokenFile: token}, Request{Run: r, BaseSHA: base, Commit: sha, Bundle: bundle}
}

func TestPublisherValidatesCandidate(t *testing.T) {
	c, r := candidateFixture(t, "README.md")
	if err := c.ValidateCandidate(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	c, r = candidateFixture(t, ".forgejo/workflows/verify.yaml")
	if err := c.ValidateCandidate(context.Background(), r); err == nil {
		t.Fatal("protected path accepted")
	}
}

func TestPublisherRejectsMismatchedBundle(t *testing.T) {
	c, r := candidateFixture(t, "README.md")
	r.Commit = strings.Repeat("c", 40)
	if err := c.ValidateCandidate(context.Background(), r); err == nil {
		t.Fatal("mismatched bundle accepted")
	}
}
