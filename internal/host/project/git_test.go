package project

import (
	"context"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

type gitCommands struct {
	executable string
	args       []string
}

func (c *gitCommands) Run(_ context.Context, _ []byte, executable string, args ...string) ([]byte, error) {
	c.executable, c.args = executable, append([]string(nil), args...)
	return nil, nil
}

func TestGitOriginRewrite(t *testing.T) {
	for _, raw := range []string{"https://forge.example.test", "https://forge.example.test/", "https://forge.example.test:8443"} {
		c := &gitCommands{}
		r := Runtime{Exec: c, Config: Config{GitSocket: "/run/soda-git-interface/launch.sock", ForgejoURL: raw}}
		if err := r.configureGit(context.Background(), "p123456789012345678901234"); err != nil {
			t.Fatal(err)
		}
		origin, _ := gitOrigin(raw)
		want := []string{"exec", "soda-p123456789012345678901234", "/usr/bin/git", "config", "--system", "url.soda://.insteadOf", origin}
		if c.executable != "/usr/bin/podman" || !reflect.DeepEqual(c.args, want) {
			t.Fatalf("unexpected command: %s %q", c.executable, c.args)
		}
		if origin[len(origin)-1] != '/' {
			t.Fatal("origin rewrite must end with a slash")
		}
	}
}

func TestGitRewriteRejectsBroadOrCredentialOrigins(t *testing.T) {
	for _, raw := range []string{"", "https://", "http://forge.example.test", "https://user:secret@forge.example.test", "https://forge.example.test/owner", "https://forge.example.test?", "https://forge.example.test?q=1", "https://forge.example.test#fragment", "https://forge.example.test\n"} {
		c := &gitCommands{}
		r := Runtime{Exec: c, Config: Config{GitSocket: "/run/soda-git-interface/launch.sock", ForgejoURL: raw}}
		if err := r.configureGit(context.Background(), "p123456789012345678901234"); err == nil || c.executable != "" {
			t.Fatalf("unsafe rewrite admitted: %q", raw)
		}
	}
	c := &gitCommands{}
	r := Runtime{Exec: c}
	if err := r.configureGit(context.Background(), "p123456789012345678901234"); err != nil || c.executable != "" {
		t.Fatal("disabled Git modified the project")
	}
}

func TestGitRewriteSelectsOnlyNativeOrigin(t *testing.T) {
	for remote, want := range map[string]string{
		"https://forge.example.test/owner/repo.git":           "soda://owner/repo.git",
		"https://forge.example.test.evil.test/owner/repo.git": "https://forge.example.test.evil.test/owner/repo.git",
		"https://other.example.test/owner/repo.git":           "https://other.example.test/owner/repo.git",
		"https://forge.example.test:8443/owner/repo.git":      "https://forge.example.test:8443/owner/repo.git",
	} {
		command := exec.Command("git", "-c", "url.soda://.insteadOf=https://forge.example.test/", "ls-remote", "--get-url", remote)
		body, err := command.Output()
		if err != nil || strings.TrimSpace(string(body)) != want {
			t.Fatalf("rewrite %q: %q, %v", remote, body, err)
		}
	}
}
