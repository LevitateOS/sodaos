package workspace

import (
	"context"
	"strings"
	"testing"
)

func TestFactoryGitRemoteRequiresExactRepository(t *testing.T) {
	remote := "https://forge.example.test/soda-tester/repo.git"
	if got, err := gitRemote(remote); err != nil || got != "soda://soda-tester/repo.git" {
		t.Fatal("exact admitted remote rejected", err)
	}
	for _, invalid := range []string{
		"http://forge.example.test/soda-tester/repo.git",
		"https://token@forge.example.test/soda-tester/repo.git",
		"https://forge.example.test/soda-tester/repo.git?other=1",
		"https://forge.example.test/soda-tester/other/repo.git",
		"https://forge.example.test/soda-tester/repo",
		"https://forge.example.test/soda-tester/repo.git#fragment",
	} {
		if _, err := gitRemote(invalid); err == nil {
			t.Fatalf("invalid remote admitted: %s", invalid)
		}
	}
}

func TestFactoryGitConfigurationStopsOnInvalidRemote(t *testing.T) {
	w := Runtime{Config: Config{GitSocket: "/run/soda-git-interface/launch.sock"}, GitRemote: "https://forge.example.test/other/repo.git?override=1", Exec: executorFunc(func(context.Context, []byte, string, ...string) ([]byte, error) {
		t.Fatal("invalid Git remote reached native execution")
		return nil, nil
	})}
	if err := w.configureGit(t.Context(), "container"); err == nil {
		t.Fatal("invalid Git remote accepted")
	}
}

func TestFactoryGitConfigurationKeepsBundleCheckoutAndPinsOrigin(t *testing.T) {
	remote := "https://forge.example.test/soda-tester/repo.git"
	var calls []string
	w := Runtime{Config: Config{GitSocket: "/run/soda-git-interface/launch.sock"}, GitRemote: remote, Exec: executorFunc(func(_ context.Context, _ []byte, executable string, args ...string) ([]byte, error) {
		calls = append(calls, executable+" "+strings.Join(args, " "))
		return nil, nil
	})}
	if err := w.configureGit(t.Context(), "container"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || !strings.Contains(calls[0], "url.soda://soda-tester/repo.git.insteadOf "+remote) || !strings.Contains(calls[1], "remote set-url origin "+remote) {
		t.Fatalf("factory remote was not pinned: %v", calls)
	}
}
