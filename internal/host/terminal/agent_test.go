package terminal

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAgentExecUsesFixedProgram(t *testing.T) {
	container := strings.Repeat("c", 64)
	cmd := AgentExec(container, []byte(`{"action":"stage"}`), "broker")
	if cmd.Path != "/usr/bin/podman" {
		t.Fatal("agent launcher moved", cmd.Path)
	}
	want := []string{"/usr/bin/podman", "--remote=false", "exec", "--interactive", container, "/usr/libexec/soda/project-terminal", "broker"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Fatal("agent argv changed", cmd.Args)
	}
	if cmd.Stdin == nil {
		t.Fatal("agent stdin not wired")
	}
	if stream := AgentExec(container, nil, "attach"); stream.Stdin != nil {
		t.Fatal("streaming launcher cannot take StdinPipe")
	}
}

func agentFixture(t *testing.T, mode os.FileMode) (string, []byte) {
	t.Helper()
	content := []byte("soda-test-agent-binary")
	path := filepath.Join(t.TempDir(), "project-terminal")
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SODA_PROJECT_TERMINAL", path)
	return path, content
}

func TestAgentProgramHash(t *testing.T) {
	uid := uint32(os.Geteuid())
	t.Run("good file", func(t *testing.T) {
		_, content := agentFixture(t, 0o755)
		hash, err := agentProgramHash(uid)
		if err != nil {
			t.Fatal(err)
		}
		if want := fmt.Sprintf("%x", sha256.Sum256(content)); hash != want {
			t.Fatal("agent digest differs", hash)
		}
	})
	t.Run("bad mode", func(t *testing.T) {
		for _, mode := range []os.FileMode{0o666, 0o777, 0o775} {
			agentFixture(t, mode)
			if _, err := agentProgramHash(uid); err == nil {
				t.Fatal("writable agent accepted", mode)
			}
		}
	})
	t.Run("missing", func(t *testing.T) {
		t.Setenv("SODA_PROJECT_TERMINAL", filepath.Join(t.TempDir(), "absent"))
		if _, err := agentProgramHash(uid); err == nil {
			t.Fatal("missing agent accepted")
		}
	})
	t.Run("too big", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "project-terminal")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o755)
		if err != nil {
			t.Fatal(err)
		}
		if err = file.Truncate(1 + 32<<20); err != nil {
			t.Fatal(err)
		}
		if err = file.Close(); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SODA_PROJECT_TERMINAL", path)
		if _, err := agentProgramHash(uid); err == nil {
			t.Fatal("oversized agent accepted")
		}
	})
	t.Run("not regular", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("SODA_PROJECT_TERMINAL", dir)
		if _, err := agentProgramHash(uid); err == nil {
			t.Fatal("directory agent accepted")
		}
		link := filepath.Join(t.TempDir(), "link")
		if err := os.Symlink(dir, link); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SODA_PROJECT_TERMINAL", link)
		if _, err := agentProgramHash(uid); err == nil {
			t.Fatal("symlink agent accepted")
		}
	})
	t.Run("wrong owner", func(t *testing.T) {
		agentFixture(t, 0o755)
		if _, err := agentProgramHash(uid ^ 0x1bad); err == nil {
			t.Fatal("foreign-owned agent accepted")
		}
	})
	t.Run("empty", func(t *testing.T) {
		agentFixture(t, 0o755)
		if err := os.Truncate(os.Getenv("SODA_PROJECT_TERMINAL"), 0); err != nil {
			t.Fatal(err)
		}
		if _, err := agentProgramHash(uid); err == nil {
			t.Fatal("empty agent accepted")
		}
	})
}

func TestAgentProgramHashRequiresRootOwnership(t *testing.T) {
	path, content := agentFixture(t, 0o755)
	hash, err := AgentProgramHash()
	if os.Geteuid() == 0 {
		if err != nil {
			t.Fatal(err)
		}
		if want := fmt.Sprintf("%x", sha256.Sum256(content)); hash != want {
			t.Fatal("agent digest differs", hash)
		}
		return
	}
	if err == nil {
		t.Fatal("non-root agent accepted", path)
	}
}
