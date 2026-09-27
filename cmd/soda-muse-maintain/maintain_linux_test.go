//go:build linux

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity/muse"
)

func syntheticTools(t *testing.T) []tool {
	t.Helper()
	var sources []tool
	for _, name := range []string{"muse", "soda-muse-compose", "muse-native"} {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte("synthetic "+name), 0o755); err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		info, err := file.Stat()
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, tool{name: name, file: file, size: info.Size()})
	}
	t.Cleanup(func() { closeTools(sources) })
	return sources
}

func TestPublicToolReplacementPreservesOtherFilesAndRefusesSymlinks(t *testing.T) {
	root := t.TempDir()
	targets := []string{filepath.Join(root, "bin", "muse"), filepath.Join(root, "bin", "soda-muse-compose"), filepath.Join(root, "libexec", "soda", "muse")}
	for _, target := range targets {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte("obsolete public tool"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	retained := filepath.Join(root, "account-state")
	if err := os.WriteFile(retained, []byte("preserved account and project state"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(retained)
	if err != nil {
		t.Fatal(err)
	}
	var stream bytes.Buffer
	if err := archiveTools(&stream, syntheticTools(t)); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("/bin/sh", append([]string{"-ceu", installScript, "soda-muse-maintain"}, targets...)...)
	command.Stdin = bytes.NewReader(stream.Bytes())
	if body, err := command.CombinedOutput(); err != nil {
		t.Fatal("public replacement failed", err, string(body))
	}
	after, err := os.Stat(retained)
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("unrelated retained inode changed", err)
	}
	content, err := os.ReadFile(retained)
	if err != nil || string(content) != "preserved account and project state" {
		t.Fatal("retained state changed", err)
	}
	for i, target := range targets {
		body, err := os.ReadFile(target)
		if err != nil || !bytes.HasPrefix(body, []byte("synthetic ")) {
			t.Fatal("public tool not replaced", i, err)
		}
	}
	if err := os.Remove(targets[0]); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(retained, targets[0]); err != nil {
		t.Fatal(err)
	}
	denied := exec.Command("/bin/sh", append([]string{"-ceu", installScript, "soda-muse-maintain"}, targets...)...)
	denied.Stdin = bytes.NewReader(stream.Bytes())
	if err := denied.Run(); err == nil {
		t.Fatal("target symlink accepted")
	}
	content, err = os.ReadFile(retained)
	if err != nil || string(content) != "preserved account and project state" {
		t.Fatal("symlink target changed", err)
	}
}

func TestSourceAndNativeDigestAdmission(t *testing.T) {
	root := t.TempDir()
	native := filepath.Join(root, "muse-native")
	body := []byte("synthetic native bytes")
	if err := os.WriteFile(native, body, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "muse")
	if err := os.Symlink(native, link); err != nil {
		t.Fatal(err)
	}
	if _, err := openTool(link, "muse"); err == nil {
		t.Fatal("source symlink admitted")
	}
	file, err := os.Open(native)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if err := verifyNative(file, strings.Repeat("0", 64)); err == nil {
		t.Fatal("unverified native bytes admitted")
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if err := verifyNative(file, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		source, err := openTool(native, "muse-native")
		if err != nil {
			t.Fatal("root-owned native source refused", err)
		}
		_ = source.file.Close()
	} else {
		if _, err := openTool(native, "muse-native"); err == nil {
			t.Fatal("non-root source admitted")
		}
	}
	if _, err := loadTools(root, hex.EncodeToString(sum[:]), muse.Version+"-other"); err == nil {
		t.Fatal("unqualified version admitted")
	}
}

func TestProjectIdentityAndPublicInterfaceAdmission(t *testing.T) {
	valid := observation{ID: strings.Repeat("a", 64), Project: "p0123456789abcdef01234567", Owner: "42", PID: 1, Running: true}
	if err := valid.validate(valid.Project); err != nil {
		t.Fatal(err)
	}
	foreign := valid
	foreign.Project = "pabcdef0123456789abcdef01"
	if err := foreign.validate(valid.Project); err == nil {
		t.Fatal("foreign project label accepted")
	}
	invalid := valid
	invalid.Owner = "0"
	if err := invalid.validate(valid.Project); err == nil {
		t.Fatal("invalid owner accepted")
	}
	root, err := os.MkdirTemp(os.TempDir(), "mi-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	socket := filepath.Join(root, "launch.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	if err := os.Chmod(socket, 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		if _, err := publicSocketDirectory(socket); err != nil {
			t.Fatal("public socket refused", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "credential"), []byte("synthetic private data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := publicSocketDirectory(socket); err == nil {
		t.Fatal("directory containing private data admitted")
	}
	if _, err := parse([]string{"--project", "../foreign"}); err == nil {
		t.Fatal("invalid project admitted")
	}
}

func TestGitMaintenanceModeAndPreservation(t *testing.T) {
	o, err := parse([]string{"--git-only", "--bind-only", "--project", "p0123456789abcdef01234567"})
	if err != nil || !o.gitOnly || !o.bindOnly {
		t.Fatal("Git startup mode unavailable", err)
	}
	root := t.TempDir()
	binary := filepath.Join(root, "git-remote-soda")
	retained := filepath.Join(root, "account-state")
	if err := os.WriteFile(retained, []byte("preserved"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "public-source")
	if err := os.WriteFile(path, []byte("synthetic helper"), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var stream bytes.Buffer
	if err := archiveTools(&stream, []tool{{name: "git-remote-soda", file: f, size: int64(len("synthetic helper"))}}); err != nil {
		t.Fatal(err)
	}
	script := strings.ReplaceAll(installGitScript, "/usr/local/bin", root)
	command := exec.Command("/bin/sh", "-ceu", script)
	command.Stdin = bytes.NewReader(stream.Bytes())
	if body, err := command.CombinedOutput(); err != nil {
		t.Fatal(err, string(body))
	}
	if body, err := os.ReadFile(binary); err != nil || string(body) != "synthetic helper" {
		t.Fatal("Git helper missing", err)
	}
	if body, err := os.ReadFile(retained); err != nil || string(body) != "preserved" {
		t.Fatal("account state changed", err)
	}
	if err := os.Remove(binary); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(retained, binary); err != nil {
		t.Fatal(err)
	}
	denied := exec.Command("/bin/sh", "-ceu", script)
	denied.Stdin = bytes.NewReader(stream.Bytes())
	if err := denied.Run(); err == nil {
		t.Fatal("Git target symlink accepted")
	}
}
