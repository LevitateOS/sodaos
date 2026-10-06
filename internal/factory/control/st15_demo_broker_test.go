package control_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// st15RepoRoot locates the checkout so the fixture builds the broker from
// the tree under test, including worktrees.
func st15RepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	nativeMust(t, err)
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

// st15BuildBroker compiles the Rust identity broker; ST15 never
// enrolls, so the pinned provider below is construction-only (the
// connection is seeded directly from the configured credential file).
func st15BuildBroker(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo unavailable")
	}
	root := st15RepoRoot(t)
	build := exec.Command("cargo", "build", "-p", "soda-identity", "--bin", "soda-identity")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build soda-identity: %v\n%s", err, out)
	}
	return filepath.Join(root, "target", "debug", "soda-identity")
}

// st15BuildHost compiles the production `soda-host` daemon the fixture
// spawns over the pre-bound host socket.
func st15BuildHost(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo unavailable")
	}
	root := st15RepoRoot(t)
	build := exec.Command("cargo", "build", "-p", "soda-host", "--bin", "soda-host")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build soda-host: %v\n%s", err, out)
	}
	return filepath.Join(root, "target", "debug", "soda-host")
}

// st15BuildFactoryRoles compiles the Rust factory-roles helper the
// fixture stages into the project container.
func st15BuildFactoryRoles(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo unavailable")
	}
	root := st15RepoRoot(t)
	build := exec.Command("cargo", "build", "-p", "soda-project-terminal", "--bin", "project-factory-roles")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build project-factory-roles: %v\n%s", err, out)
	}
	return filepath.Join(root, "target", "debug", "project-factory-roles")
}

// st15TmpfsRoot confines the construction-only provider root to private
// tmpfs, which the broker validates before serving.
func st15TmpfsRoot(t *testing.T) string {
	t.Helper()
	parent := filepath.Join("/dev/shm", fmt.Sprintf("soda-st15-%d", os.Getpid()))
	if err := os.MkdirAll(parent, 0o700); err != nil {
		t.Skipf("tmpfs provider root unavailable: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	return parent
}

func st15FakeCodex(t *testing.T, dir string) (binary, sum string) {
	t.Helper()
	binary = filepath.Join(dir, "codex-fixture")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'codex-cli 0.153.4'; exit 0; fi\necho unexpected >&2\nexit 1\n"
	nativeMust(t, os.WriteFile(binary, []byte(script), 0o700))
	return binary, st15SHA256([]byte(script))
}

// st15FakeMuse registers the broker's muse provider the same way: the
// fake answers the pinned version line the adapter requires at
// construction. The journey never enrolls through it (the credential is
// seeded from file); registration only satisfies the acquire-time
// provider gate for the st15-muse connection.
func st15FakeMuse(t *testing.T, dir string) (binary, sum string) {
	t.Helper()
	binary = filepath.Join(dir, "muse-fixture")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'Muse Code 1.4.0 (1.4.0-R4161.1)'; exit 0; fi\necho unexpected >&2\nexit 1\n"
	nativeMust(t, os.WriteFile(binary, []byte(script), 0o700))
	return binary, st15SHA256([]byte(script))
}

func st15WaitSocket(t *testing.T, path string) {
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
