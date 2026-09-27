//go:build linux

package workspace

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFactoryGitMountExposesOnlyPublicLaunchInterface(t *testing.T) {
	root := t.TempDir()
	interfaceDir := filepath.Join(root, "interface")
	toolsDir := filepath.Join(root, "tools")
	if err := os.Mkdir(interfaceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(toolsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(interfaceDir, "launch.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	if err := os.WriteFile(filepath.Join(toolsDir, "git-remote-soda"), []byte("public helper"), 0o700); err != nil {
		t.Fatal(err)
	}
	w := Runtime{Config: Config{GitSocket: socket, GitToolsDirectory: toolsDir}}
	args, err := w.gitArguments([]string{"create"})
	if err != nil || !strings.Contains(strings.Join(args, " "), interfaceDir+":/run/soda-git-interface:ro,z") {
		t.Fatal("public interface unavailable", err)
	}
	if err := os.WriteFile(filepath.Join(interfaceDir, "admin.sock"), []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := w.gitArguments(nil); err == nil {
		t.Fatal("private sibling entered worker")
	}
}
