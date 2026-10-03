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

// ForgejoBunVersion is the only JS toolchain permitted for the archived
// fork's frontend. The container fetches the musl build from the upstream
// release and verifies ForgejoBunSHA256 before use.
const ForgejoBunVersion = "1.4.2"

// ForgejoBunSHA256 pins bun-linux-x64-musl.zip for ForgejoBunVersion.
const ForgejoBunSHA256 = "4835eca59d6da70f4674f5642f6e459dcadab773695b2ed9922d131057989742"

// Fountain derives from Forgejo 15.0 LTS. The compatibility token follows
// the fork Makefile's GITEA_COMPATIBILITY contract.
const (
	ForgejoUpstreamBase = "15.0.9"
	ForgejoCompatToken  = "gitea-1.22.0"
)

// forgejoVersionStamp maps the exact archived revision to the
// Makefile-shaped version identity: <base>-soda.<short>+<compat>. The
// Makefile appends +gitea-1.22.0 to every fork version; the describe count
// is unknowable without git, so the soda marker carries the short revision
// instead of faking a describe count.
func forgejoVersionStamp(revision string) (string, error) {
	if !Revision(revision) {
		return "", errors.New("exact Forgejo source revision required")
	}
	return ForgejoUpstreamBase + "-soda." + revision[:12] + "+" + ForgejoCompatToken, nil
}

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
	// Production callers always set ForgejoRevision; the archive gate
	// rejects anything else before this build runs. An empty stamp keeps
	// upstream defaults for direct unit-test callers only.
	stamp, err := forgejoVersionStamp(p.ForgejoRevision)
	if err != nil {
		stamp = ""
	}
	script := `set -eu
apk add --no-cache build-base=0.5-r4
wget -O /tmp/bun.zip "https://github.com/oven-sh/bun/releases/download/bun-v` + ForgejoBunVersion + `/bun-linux-x64-musl.zip"
echo "` + ForgejoBunSHA256 + `  /tmp/bun.zip" | sha256sum -c -
rm -rf /tmp/bun-extract && mkdir -p /tmp/bun-extract
unzip -q -o -d /tmp/bun-extract /tmp/bun.zip
install -m 755 /tmp/bun-extract/bun-linux-x64-musl/bun /usr/local/bin/bun
test "$(bun --version)" = "` + ForgejoBunVersion + `"
mkdir -p /work/out/gocache /work/out/gopath /work/out/tmp
LC_ALL=C apk info -v | LC_ALL=C sort > /work/out/apk-packages.txt
# The git archive never contains built frontend outputs; generate them with
# the pinned Bun toolchain before bindata embeds public/. The frozen install
# resolves from the archived package lock; Bun is the only JS toolchain.
bun install --frozen-lockfile --no-progress
BROWSERSLIST_IGNORE_OLD_DATA=true bun ./node_modules/.bin/webpack
test -s public/assets/js/index.js
test -s public/assets/css/index.css
for package in options public templates migration; do
  (cd modules/$package && go generate -tags bindata .)
done
LDFLAGS=""
if [ -n "${FORGEJO_VERSION:-}" ]; then
  LDFLAGS="-X main.Version=${FORGEJO_VERSION} -X main.ForgejoVersion=${FORGEJO_VERSION} -X main.ReleaseVersion=${FORGEJO_VERSION} -X \"main.Tags=bindata sqlite sqlite_unlock_notify\""
fi
go build -buildvcs=false -tags 'bindata sqlite sqlite_unlock_notify' -ldflags "${LDFLAGS}" -trimpath -o /work/out/forgejo-bin .`
	return []string{"--remote=false", "run", "--rm", "--pull=always", "--platform=linux/" + platform, "--volume=" + p.ForgejoSource + ":/work/source:Z", "--volume=" + root + ":/work/out:Z", "--workdir=/work/source", "--env=GOCACHE=/work/out/gocache", "--env=GOPATH=/work/out/gopath", "--env=TMPDIR=/work/out/tmp", "--env=FORGEJO_VERSION=" + stamp, ForgejoCompilerImage, "sh", "-ec", script}
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
