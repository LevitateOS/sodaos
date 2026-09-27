//go:build linux

package terminal

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
)

type gitConfigExecutor struct {
	t         *testing.T
	secret    string
	protected bool
	wrote     bool
}

func (e *gitConfigExecutor) Run(_ context.Context, body []byte, _ string, args ...string) ([]byte, error) {
	command := strings.Join(args, " ")
	if strings.Contains(command, e.secret) {
		e.t.Fatal("capability exposed in argv")
	}
	if strings.Contains(command, "--mode=0600 /dev/null") {
		e.protected = true
	}
	if len(body) > 0 {
		if !e.protected || !strings.Contains(string(body), e.secret) || !strings.Contains(command, "/usr/bin/dd") {
			e.t.Fatal("unprotected capability input")
		}
		e.wrote = true
	}
	return nil, nil
}

func (e *gitConfigExecutor) RunReader(context.Context, io.Reader, string, ...string) ([]byte, error) {
	e.t.Fatal("unexpected reader call")
	return nil, nil
}

func TestGitCapabilityRestrictedConfigInput(t *testing.T) {
	executor := &gitConfigExecutor{t: t, secret: "synthetic-invocation-capability"}
	runtime := &GitRuntime{Exec: executor}
	if err := runtime.stageConfig(context.Background(), museCaller{Container: strings.Repeat("a", 64), UID: 1000, GID: 1000}, "/run/soda-git/"+strings.Repeat("b", 32), executor.secret); err != nil {
		t.Fatal(err)
	}
	if !executor.wrote {
		t.Fatal("missing protected config input")
	}
}

type gitIdentityExecutor struct {
	t    *testing.T
	home string
}

func (e *gitIdentityExecutor) Run(ctx context.Context, body []byte, name string, args ...string) ([]byte, error) {
	e.t.Helper()
	if len(body) != 0 || name != "/usr/bin/podman" || len(args) != 11 || args[0] != "--remote=false" || args[1] != "exec" || args[2] != "--user=1000:1000" || args[3] != "--env=HOME="+e.home || args[4] != strings.Repeat("a", 64) || args[5] != "/usr/bin/git" || args[6] != "config" || args[7] != "--global" || args[8] != "--replace-all" {
		e.t.Fatalf("unexpected native Git identity command: %q %q", name, args)
	}
	command := exec.CommandContext(ctx, "git", args[6:]...)
	command.Env = append(os.Environ(), "HOME="+e.home, "XDG_CONFIG_HOME="+filepath.Join(e.home, ".config"), "GIT_CONFIG_NOSYSTEM=1")
	return command.CombinedOutput()
}

func (e *gitIdentityExecutor) RunReader(context.Context, io.Reader, string, ...string) ([]byte, error) {
	e.t.Fatal("unexpected reader call")
	return nil, nil
}

func TestGitIdentityRefreshesGlobalDefaultsWithoutChangingOtherSettings(t *testing.T) {
	home := t.TempDir()
	env := append(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"), "GIT_CONFIG_NOSYSTEM=1")
	git := func(args ...string) string {
		t.Helper()
		command := exec.Command("git", args...)
		command.Env = env
		out, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %q: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("config", "--global", "user.name", "old name")
	git("config", "--global", "user.email", "old@example.test")
	git("config", "--global", "core.editor", "synthetic-editor")
	localRepo := filepath.Join(t.TempDir(), "local-repo")
	git("init", "--quiet", localRepo)
	git("-C", localRepo, "config", "user.name", "Local Tester")
	container := strings.Repeat("a", 64)
	runtime := &GitRuntime{Exec: &gitIdentityExecutor{t: t, home: home}}
	caller := museCaller{Container: container, UID: 1000, GID: 1000, Home: home}
	if err := runtime.configureGitIdentity(t.Context(), caller, identity.GitSession{Name: "Soda Tester", Email: "soda-tester@example.test"}); err != nil {
		t.Fatal(err)
	}
	if got := git("config", "--global", "user.name"); got != "Soda Tester" {
		t.Fatal(got)
	}
	if got := git("config", "--global", "user.email"); got != "soda-tester@example.test" {
		t.Fatal(got)
	}
	if got := git("config", "--global", "core.editor"); got != "synthetic-editor" {
		t.Fatal(got)
	}
	if got := git("-C", localRepo, "config", "user.name"); got != "Local Tester" {
		t.Fatal(got)
	}
	repo := filepath.Join(t.TempDir(), "repo")
	git("init", "--quiet", repo)
	git("-C", repo, "commit", "--quiet", "--allow-empty", "-m", "test")
	if got := git("-C", repo, "log", "-1", "--format=%an <%ae>"); got != "Soda Tester <soda-tester@example.test>" {
		t.Fatal(got)
	}
}

func TestGitIdentityRejectsInvalidValuesBeforeWriting(t *testing.T) {
	runtime := &GitRuntime{Exec: &gitConfigExecutor{t: t, secret: "unused"}}
	for _, session := range []identity.GitSession{{Name: "", Email: "valid@example.test"}, {Name: "bad\nname", Email: "valid@example.test"}, {Name: "Valid", Email: "bad\x00@example.test"}} {
		if err := runtime.configureGitIdentity(t.Context(), museCaller{}, session); err != identity.ErrDenied {
			t.Fatalf("invalid Git identity admitted: %v", err)
		}
	}
}
