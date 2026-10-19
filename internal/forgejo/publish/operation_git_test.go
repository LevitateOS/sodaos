package publish

import (
	"context"
	"errors"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

func TestPushBranchMovesExactlyItsTarget(t *testing.T) {
	c, r := candidateFixture(t, "README.md")
	remote := filepath.Join(c.Root, "remote.git")
	command := exec.Command("git", "init", "--bare", remote)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bare: %v %s", err, out)
	}
	c.Remote = servePublicationGit(t, c.Root) + "/remote.git"
	validated, err := c.PrepareValidated(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer validated.Close()
	target := "refs/heads/soda/factory/testpush"
	ops := &fakeBackgroundOps{}
	if err := validated.PushBranch(context.Background(), c, ops, "soda-test-publish-1", target); err != nil {
		t.Fatalf("push: %v", err)
	}
	if len(ops.pushRebinds) != 1 || ops.pushRebinds[0] {
		t.Fatalf("push env: %+v", ops.pushRebinds)
	}
	out, err := exec.Command("git", "--git-dir="+remote, "for-each-ref", "--format=%(refname) %(objectname)").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != target+" "+r.Commit {
		t.Fatalf("remote refs: %s", out)
	}
	if err := validated.PushBranch(context.Background(), c, ops, "bad id", target); err == nil {
		t.Fatal("malformed operation pushed")
	} else {
		var refusal *Refusal
		if !errors.As(err, &refusal) {
			t.Fatalf("operation: %v", err)
		}
	}
	if err := validated.PushBranch(context.Background(), c, ops, "soda-test-publish-1", "main"); err == nil {
		t.Fatal("malformed ref pushed")
	}
}

func TestObserveTipParsesExactAdvertisement(t *testing.T) {
	c, _ := candidateFixture(t, "README.md")
	remote := filepath.Join(c.Root, "remote.git")
	command := exec.Command("git", "init", "--bare", remote)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bare: %v %s", err, out)
	}
	seed := filepath.Join(c.Root, "seed")
	if err := os.Mkdir(seed, 0o700); err != nil {
		t.Fatal(err)
	}
	run := func(dir string, args ...string) string {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = dir
		out, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("seed Git: %v %s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run(seed, "init", "--initial-branch=main")
	run(seed, "config", "user.name", "soda-tester")
	run(seed, "config", "user.email", "soda-tester@localhost")
	if err := os.WriteFile(filepath.Join(seed, "file"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(seed, "add", ".")
	run(seed, "commit", "-m", "seed")
	tip := run(seed, "rev-parse", "HEAD")
	run(seed, "remote", "add", "origin", remote)
	run(seed, "push", "origin", "main")
	git, cleanup, err := c.sourceRepository(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	got, err := observeTip(context.Background(), git, remote, "refs/heads/main", false)
	if err != nil || got != tip {
		t.Fatalf("tip: %q %v", got, err)
	}
	got, err = observeTip(context.Background(), git, remote, "refs/heads/absent", true)
	if err != nil || got != "" {
		t.Fatalf("absent: %q %v", got, err)
	}
	if _, err := observeTip(context.Background(), git, remote, "refs/heads/absent", false); err == nil {
		t.Fatal("missing comparison accepted")
	} else {
		var refusal *Refusal
		if !errors.As(err, &refusal) {
			t.Fatalf("comparison: %v", err)
		}
	}
}

func TestObserveForPublishRefusesBeforeAnyCall(t *testing.T) {
	c, _ := candidateFixture(t, "README.md")
	ops := &fakeBackgroundOps{revision: extensions.NativeRevisionObservation{Revision: 9, Idle: true}}
	if _, err := c.ObserveForPublish(context.Background(), ops, "main", "refs/heads/main"); err == nil {
		t.Fatal("malformed refs accepted")
	}
	if _, err := c.ObserveForPublish(context.Background(), ops, "refs/heads/main", "refs/heads/main"); err == nil {
		t.Fatal("identical refs accepted")
	}
	if _, err := c.ObserveForPublish(context.Background(), nil, "refs/heads/a", "refs/heads/main"); err == nil {
		t.Fatal("missing transport accepted")
	}
	ops = &fakeBackgroundOps{revision: extensions.NativeRevisionObservation{Revision: 9, Idle: false}}
	if _, err := c.ObserveForPublish(context.Background(), ops, "refs/heads/a", "refs/heads/main"); err == nil {
		t.Fatal("busy revision accepted")
	} else {
		var wait *Wait
		if !errors.As(err, &wait) {
			t.Fatalf("busy: %v", err)
		}
	}
}

// Serve real smart HTTP for the publisher's Git transport test. The fixture
// only proves the exact refspec; native operation enforcement needs Fountain.
func servePublicationGit(t *testing.T, root string) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(&cgi.Handler{
		Path: git, Args: []string{"http-backend"}, Dir: root,
		Env: []string{"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1", "REMOTE_USER=soda-tester"},
	})
	t.Cleanup(server.Close)
	return server.URL
}
