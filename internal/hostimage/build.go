package hostimage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/appliancerelease"
	"github.com/levitateos/sodaos/internal/nativebuild"
)

// Request describes one fresh P1–P6 build, never installation or publication.
type Request struct{ Source, Out, Arch, RepositoryPrefix, Revision string }
type Result struct{ Revision, Architecture, Candidate, Scope string }

// Build is the single candidate execution owner. Qualification and protected
// delivery consume its unchanged bytes; this result is not a qualified release.
func Build(ctx context.Context, r Request, progress *nativebuild.BuildProgress) (Result, error) {
	var log io.Writer = io.Discard
	capture := func(dir, name string, args ...string) (string, error) {
		return runBuildCommand(ctx, log, nil, dir, name, args...)
	}
	execute := func(dir, name string, args ...string) error {
		_, err := runBuildCommand(ctx, log, os.Stdout, dir, name, args...)
		return err
	}
	return build(r, progress, execute, capture, func(path string) (func() error, error) {
		f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return nil, e
		}
		log = f
		return f.Close, nil
	})
}

func build(r Request, progress *nativebuild.BuildProgress, execute nativebuild.BuildExec, capture nativebuild.BuildCapture, openLog func(string) (func() error, error)) (result Result, err error) {
	next := progress.Phase
	if err = next("P1 / Admit and freeze inputs"); err != nil {
		return
	}
	if err = nativebuild.RequireNative(r.Arch); err != nil {
		return
	}
	if runtime.Version() != "go1.26.7" {
		return result, errors.New("pinned Go 1.26.7 required")
	}
	if !appliancerelease.ValidRepositoryPrefix(r.RepositoryPrefix) {
		return result, errors.New("explicit intended repository prefix required")
	}
	source, e := capture(r.Source, "git", "rev-parse", "--show-toplevel")
	if e != nil {
		return result, e
	}
	if source != r.Source || !filepath.IsAbs(source) {
		return result, errors.New("canonical checkout root required")
	}
	info, e := os.Lstat(filepath.Join(source, ".git"))
	if e != nil || !info.IsDir() {
		return result, errors.New("canonical checkout required; no worktree")
	}
	status, e := capture(source, "git", "status", "--porcelain", "--untracked-files=normal")
	if e != nil {
		return result, e
	}
	if status != "" {
		return result, errors.New("clean committed source required")
	}
	revision, e := capture(source, "git", "rev-parse", "HEAD")
	if e != nil || !nativebuild.Revision(revision) {
		return result, errors.New("exact committed revision required")
	}
	goVersion, e := capture(source, "go", "env", "GOVERSION")
	if e != nil {
		return result, e
	}
	if goVersion != runtime.Version() {
		return result, errors.New("pinned Go compiler unavailable")
	}
	if r.Revision != "" && r.Revision != revision {
		return result, errors.New("controller/source revision mismatch")
	}
	if !filepath.IsAbs(r.Out) || !strings.HasPrefix(filepath.Clean(r.Out), filepath.Join(source, ".artifacts/releases")+string(os.PathSeparator)) {
		return result, errors.New("fresh output must be below .artifacts/releases; parent must exist")
	}
	if err = nativebuild.FreshDirectory(r.Out); err != nil {
		return
	}
	for _, dir := range []string{"inputs", "work", "artifacts", "evidence", "release", "logs"} {
		if err = os.Mkdir(filepath.Join(r.Out, dir), 0700); err != nil {
			return
		}
	}
	if err = progress.CreateLog(filepath.Join(r.Out, "logs/timing.log")); err != nil {
		return
	}
	closeLog, e := openLog(filepath.Join(r.Out, "logs/build.log"))
	if e != nil {
		return result, e
	}
	defer func() { err = errors.Join(err, closeLog()) }()
	snapshot := filepath.Join(r.Out, "work/source")
	if err = os.Mkdir(snapshot, 0700); err != nil {
		return
	}
	archive := filepath.Join(r.Out, "inputs/source.tar")
	if err = execute(source, "git", "archive", "--format=tar", "--output", archive, revision); err != nil {
		return
	}
	if err = execute(snapshot, "tar", "--extract", "--file", archive, "--no-same-owner"); err != nil {
		return
	}
	contextDir := filepath.Join(r.Out, "work/host-context")
	base, e := Prepare(snapshot, contextDir, r.Arch, revision)
	if e != nil {
		return result, e
	}
	artifacts := filepath.Join(r.Out, "artifacts")
	p := nativebuild.Production{Source: snapshot, Native: filepath.Join(snapshot, ".artifacts/native", r.Arch), Out: artifacts, Arch: r.Arch, Revision: revision, Vendor: true, Execute: execute, Capture: capture, Next: progress.Next}
	// Refuse an unavailable native package transaction before production work.
	packageHash, e := LockHostPackages(snapshot, contextDir, r.Arch, base)
	if e != nil {
		return result, e
	}
	if err = p.ResolveInputs(); err != nil {
		return
	}
	platform, _ := nativebuild.OCIArchitecture(r.Arch)
	pinned := base.Images[r.Arch]
	if err = execute(snapshot, "podman", "--remote=false", "pull", "--platform=linux/"+platform, pinned); err != nil {
		return
	}
	// The upstream image metadata remains upstream-owned; change only the origin.
	metadata, e := capture(snapshot, "podman", "--remote=false", "run", "--cidfile", filepath.Join(r.Out, "evidence/base-config.cid"), "--network=none", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--entrypoint=/usr/bin/cat", pinned, "/usr/share/coreos-assembler/image.json")
	if e != nil {
		return result, e
	}
	var imageConfig map[string]json.RawMessage
	if err = json.Unmarshal([]byte(metadata), &imageConfig); err != nil {
		return
	}
	if len(imageConfig) == 0 {
		return result, errors.New("missing upstream image configuration")
	}
	imageConfig["container-imgref"], _ = json.Marshal("ostree-image-signed:docker://" + r.RepositoryPrefix + "-host:candidate")
	imageConfig["bootc-install-to-fs"] = json.RawMessage("false")
	imageData, e := json.MarshalIndent(imageConfig, "", "  ")
	if e != nil {
		return result, e
	}
	if err = ownedWrite(filepath.Join(contextDir, "rootfs/usr/share/coreos-assembler/image.json"), append(imageData, '\n'), 0644); err != nil {
		return
	}
	if err = next("P2 / Verify and install frozen dependencies"); err != nil {
		return
	}
	if err = p.Dependencies(); err != nil {
		return
	}
	if err = next("P3 / Compile shipping programs and prepared tools"); err != nil {
		return
	}
	names, e := nativebuild.SodaCommands(snapshot)
	if e != nil {
		return result, e
	}
	for _, name := range names {
		if err = p.Compile(name, "./cmd/"+name, filepath.Join(contextDir, "rootfs/usr/libexec/soda", name)); err != nil {
			return
		}
	}
	tools := filepath.Join(artifacts, "tools")
	if err = os.Mkdir(tools, 0755); err != nil {
		return
	}
	for _, tool := range []struct{ name, pkg string }{{"soda-installer", "./appliance/installer"}, {"soda-artifacts", "./tools/soda-artifacts"}, {"soda-acceptance", "./tools/soda-acceptance"}} {
		if err = p.Compile(tool.name, tool.pkg, filepath.Join(tools, tool.name)); err != nil {
			return
		}
	}
	toolFiles := map[string]nativebuild.File{}
	entries, e := os.ReadDir(tools)
	if e != nil {
		return result, e
	}
	for _, entry := range entries {
		hash, e := nativebuild.HashFile(filepath.Join(tools, entry.Name()))
		if e != nil {
			return result, e
		}
		toolFiles[entry.Name()] = nativebuild.File{SHA256: hash, Mode: 0755}
	}
	toolData, e := json.MarshalIndent(struct {
		Revision, Architecture string
		Files                  map[string]nativebuild.File
	}{revision, r.Arch, toolFiles}, "", "  ")
	if e != nil {
		return result, e
	}
	if err = nativebuild.WriteNew(filepath.Join(artifacts, "tools.json"), append(toolData, '\n'), 0600); err != nil {
		return
	}
	// This concrete sequence prepares assets/tests, builds five images once and
	// assembles their fixed references/archives directly into the host context.
	if _, err = completeCandidate(snapshot, contextDir, artifacts, r.Arch, revision, r.RepositoryPrefix, base, packageHash, p, progress.Phase); err != nil {
		return
	}
	if err = next("P5 / Build FCOS host candidate"); err != nil {
		return
	}
	if err = Inventory(contextDir); err != nil {
		return
	}
	if err = buildHost(contextDir, artifacts, r.Arch, revision, r.RepositoryPrefix, base, p, progress.Phase); err != nil {
		return
	}
	result = Result{revision, r.Arch, filepath.Join(artifacts, "candidate.json"), "P1-P6 verified unsigned candidate; not a qualified release"}
	data, _ := json.MarshalIndent(result, "", "  ")
	err = nativebuild.WriteNew(filepath.Join(r.Out, "evidence/build.json"), append(data, '\n'), 0600)
	if err == nil {
		err = errors.Join(progress.End(nil), progress.EndPhase(nil))
	}
	return
}

// Only the build's tool/cache environment is inherited. In particular, provider,
// installed-test, signing and private-token variables cannot activate extra work.
func buildEnvironment() []string {
	var env []string
	for _, key := range []string{"HOME", "PATH", "TMPDIR", "XDG_RUNTIME_DIR", "XDG_CACHE_HOME", "GOCACHE", "GOMODCACHE", "PLAYWRIGHT_BROWSERS_PATH", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"} {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	// A -trimpath controller can have no embedded GOROOT. Use Go's pinned
	// upstream toolchain selection, not a relative bin/go or the ambient version.
	return append(env, "GOTOOLCHAIN="+runtime.Version(), "GOWORK=off", "GOFLAGS=-mod=readonly", "CGO_ENABLED=0")
}
func runBuildCommand(ctx context.Context, log, output io.Writer, dir, name string, args ...string) (string, error) {
	if name == "go" && runtime.GOROOT() != "" {
		name = filepath.Join(runtime.GOROOT(), "bin/go")
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = buildEnvironment()
	var data bytes.Buffer
	if output == nil {
		cmd.Stdout = &data
	} else {
		cmd.Stdout = io.MultiWriter(log, output)
	}
	cmd.Stderr = log
	// Never copy raw argv, environment or stdin into the progress/evidence stream.
	if _, err := fmt.Fprintf(log, "\nCOMMAND %s\n", filepath.Base(name)); err != nil {
		return "", err
	}
	process, err := acceptance.StartCommand(ctx, cmd)
	if err != nil {
		return "", err
	}
	err = process.Wait(ctx)
	if ctx.Err() != nil {
		err = errors.Join(context.Cause(ctx), process.Stop(), err)
	}
	return strings.TrimSpace(data.String()), err
}

func preparedChecks(p nativebuild.Production) error {
	// Existing suites consume these aliases, never rebuild shipping assets.
	for link, target := range map[string]string{
		filepath.Join(p.Source, ".artifacts/forgejo-js"):              filepath.Join(p.Native, "forgejo-js"),
		filepath.Join(p.Source, ".artifacts/browser-terminal/vendor"): filepath.Join(p.Native, "terminal-assets"),
	} {
		if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
			return err
		}
		if err := os.Symlink(target, link); err != nil {
			return err
		}
	}
	for _, check := range []struct {
		label, name string
		args        []string
	}{
		{"Go source tests", "go", []string{"test", "./..."}},
		{"TypeScript and Lit checks", "bun", []string{"run", "typecheck"}},
		{"Prepared frontend tests", "bun", []string{"run", "test:frontend:prepared"}},
		{"Prepared Forgejo tests", "bun", []string{"run", "test:forgejo:prepared"}},
		{"Prepared layout tests", "bun", []string{"run", "test:layout:prepared"}},
		{"Build/source fixtures", "python3", []string{"-m", "unittest", "discover", "-s", "tests/build"}},
	} {
		if err := p.Next("P3 / " + check.label); err != nil {
			return err
		}
		if err := p.Execute(p.Source, check.name, check.args...); err != nil {
			return err
		}
	}
	return nil
}
