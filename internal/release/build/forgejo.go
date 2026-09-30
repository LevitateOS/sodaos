package build

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// docker.io/library/golang:1.26.7-alpine multi-platform index.
const ForgejoCompilerImage = "docker.io/library/golang@sha256:28d89ee9cc0ff9fec75c82ca201e6bf7fdf9a679d4b7b24dfa04f2bb766bb468"

type ForgejoToolchain struct {
	CompilerImage string
	APKPackages   []string
}

func (t ForgejoToolchain) Validate() error {
	if t.CompilerImage != ForgejoCompilerImage || !validForgejoAPKList(t.APKPackages) {
		return errors.New("invalid Forgejo compiler provenance")
	}
	if !hasForgejoNativeBuildTools(t.APKPackages) {
		return errors.New("incomplete Forgejo APK provenance")
	}
	return nil
}

func validForgejoAPKList(packages []string) bool {
	if len(packages) == 0 || len(packages) > 256 || !slices.IsSorted(packages) {
		return false
	}
	for i, name := range packages {
		if name == "" || strings.ContainsAny(name, " \t\n\r\\\x00") || i > 0 && packages[i-1] == name {
			return false
		}
	}
	return true
}

func hasForgejoNativeBuildTools(packages []string) bool {
	hasBuildBase, hasGCC, hasMuslDev := false, false, false
	for _, name := range packages {
		hasBuildBase = hasBuildBase || name == "build-base-0.5-r4"
		hasGCC = hasGCC || strings.HasPrefix(name, "gcc-")
		hasMuslDev = hasMuslDev || strings.HasPrefix(name, "musl-dev-")
	}
	return hasBuildBase && hasGCC && hasMuslDev
}

func (p Production) recordForgejoToolchain(root string) error {
	data, err := os.ReadFile(filepath.Join(root, "apk-packages.txt"))
	if err != nil {
		return err
	}
	record := ForgejoToolchain{CompilerImage: ForgejoCompilerImage, APKPackages: strings.Split(strings.TrimSpace(string(data)), "\n")}
	if err = record.Validate(); err != nil {
		return err
	}
	data, err = json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return WriteNew(filepath.Join(p.Out, "forgejo-toolchain.json"), append(data, '\n'), 0o600)
}

// BuildForgejoBinary compiles the exact archived fork under musl. The normal
// Soda compiler remains CGO-free, while Forgejo's SQLite build needs Alpine's
// native C toolchain and the four generated bindata packages.
func (p Production) BuildForgejoBinary() (string, error) {
	if err := p.validateForgejoBuild(); err != nil {
		return "", err
	}
	if err := p.step("Build patched Forgejo binary"); err != nil {
		return "", err
	}
	root := filepath.Join(p.Native, "forgejo-build")
	if err := os.Mkdir(root, 0o700); err != nil {
		return "", err
	}
	platform, err := OCIArchitecture(p.Arch)
	if err != nil {
		return "", err
	}
	if err = p.Execute(p.ForgejoSource, "podman", forgejoBuildArgs(p, root, platform)...); err != nil {
		return "", err
	}
	return p.inspectForgejoBuild(root)
}

func (p Production) validateForgejoBuild() error {
	if !p.Vendor || !filepath.IsAbs(p.ForgejoSource) || p.Execute == nil {
		return errors.New("explicit archived Forgejo source required")
	}
	return nil
}

func forgejoBuildArgs(p Production, root, platform string) []string {
	const script = `set -eu
apk add --no-cache build-base=0.5-r4
mkdir -p /work/out/gocache /work/out/gopath /work/out/tmp
LC_ALL=C apk info -v | LC_ALL=C sort > /work/out/apk-packages.txt
for package in options public templates migration; do
  (cd modules/$package && go generate -tags bindata .)
done
go build -buildvcs=false -tags 'bindata sqlite sqlite_unlock_notify' -trimpath -o /work/out/forgejo-bin .`
	return []string{"--remote=false", "run", "--rm", "--pull=always", "--platform=linux/" + platform, "--volume=" + p.ForgejoSource + ":/work/source:Z", "--volume=" + root + ":/work/out:Z", "--workdir=/work/source", "--env=GOCACHE=/work/out/gocache", "--env=GOPATH=/work/out/gopath", "--env=TMPDIR=/work/out/tmp", ForgejoCompilerImage, "sh", "-ec", script}
}

func (p Production) inspectForgejoBuild(root string) (string, error) {
	if err := p.recordForgejoToolchain(root); err != nil {
		return "", err
	}
	binary := filepath.Join(root, "forgejo-bin")
	if err := os.Chmod(binary, 0o755); err != nil {
		return "", err
	}
	if err := inspectELF(binary, p.Arch); err != nil {
		return "", err
	}
	return binary, nil
}

func (p Production) StageForkBinary(context string) error {
	binary, err := p.BuildForgejoBinary()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(context, p.Out+string(os.PathSeparator)) {
		return errors.New("forgejo image context must belong to this build")
	}
	return os.Link(binary, filepath.Join(context, "forgejo-bin"))
}
