package build

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStageForkBinaryUsesExplicitFrozenSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "frozen-fork")
	out := filepath.Join(root, "output")
	context := filepath.Join(out, "forgejo-context")
	for _, dir := range []string{source, context, filepath.Join(root, "native")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	var command []string
	p := Production{ForgejoSource: source, Native: filepath.Join(root, "native"), Out: out, Arch: "x86_64", Vendor: true}
	p.Next = func(string) error { return nil }
	p.Execute = func(dir, name string, args ...string) error {
		if dir != source || name != "podman" {
			t.Fatalf("builder used an unexpected source or command: %s %s", dir, name)
		}
		command = args
		elf := make([]byte, 64)
		copy(elf, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1})
		binary.LittleEndian.PutUint16(elf[16:], 2)
		binary.LittleEndian.PutUint16(elf[18:], 62)
		binary.LittleEndian.PutUint32(elf[20:], 1)
		binary.LittleEndian.PutUint16(elf[52:], 64)
		if err := os.WriteFile(filepath.Join(p.Native, "forgejo-build/apk-packages.txt"), []byte("build-base-0.5-r4\ngcc-14.2.0-r6\nmusl-dev-1.2.5-r10\n"), 0o600); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(p.Native, "forgejo-build/forgejo-bin"), elf, 0o700)
	}
	if err := p.StageForkBinary(context); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(command, " ")
	for _, required := range []string{"--pull=always", "--platform=linux/amd64", "--volume=" + source + ":/work/source:Z", ForgejoCompilerImage, "apk add --no-cache build-base=0.5-r4", "apk info -v", "go generate -tags bindata", "go build -buildvcs=false -tags 'bindata sqlite sqlite_unlock_notify'"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing frozen fork build input %q in %s", required, joined)
		}
	}
	for _, forbidden := range []string{"golang:1.26.7-alpine", "apk add --no-cache build-base git"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("compiler command retained mutable input %q in %s", forbidden, joined)
		}
	}
	if _, err := os.Stat(filepath.Join(context, "forgejo-bin")); err != nil {
		t.Fatal(err)
	}
	var toolchain ForgejoToolchain
	if err := ReadJSON(filepath.Join(out, "forgejo-toolchain.json"), &toolchain); err != nil {
		t.Fatal(err)
	}
	if err := toolchain.Validate(); err != nil {
		t.Fatal(err)
	}
}
