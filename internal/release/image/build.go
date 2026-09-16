package image

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
	"time"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

// Request selects a boundary of the same producer, never installation or publication.
type Request struct {
	Source, Out, Arch, RepositoryPrefix, Revision, RootfsBaseURL, MediaAuthority, CoreOSInputs string
	Development                                                                                bool
	Target, MediaCompression                                                                   string
}
type Result struct {
	Revision, Architecture, Candidate, CandidateSHA256, HostManifest, PayloadSHA256, Scope string
	Media                                                                                  string `json:",omitempty"`
	Purpose, RequestedTarget, CompletedTarget                                              string
	Checks                                                                                 []string
	MediaCompression                                                                       string `json:",omitempty"`
}

// ValidateTarget rejects implicit partial production and irrelevant media inputs.
// MediaAuthority is admitted separately inside the isolated media worker.
func (r Request) validateDevelopmentTarget() error {
	if r.Development {
		if r.Target != "candidate" && r.Target != "media" {
			return errors.New("--development requires --target candidate or media")
		}
		return nil
	}
	if r.Target != "" {
		return errors.New("--target requires --development")
	}
	return nil
}

func (r Request) validateMediaInputs() error {
	if r.MediaCompression != "" && (!r.Development || r.Target != "media" || r.MediaCompression != "fast") {
		return errors.New("--media-compression accepts only fast with --development --target media")
	}
	if !r.WantsMedia() {
		if r.RootfsBaseURL != "" || r.MediaAuthority != "" {
			return errors.New("candidate target refuses media-only inputs")
		}
		return nil
	}
	return mediaBaseURL(r.RootfsBaseURL)
}

func (r Request) ValidateTarget() error {
	if err := r.validateDevelopmentTarget(); err != nil {
		return err
	}
	return r.validateMediaInputs()
}

func (r Request) WantsMedia() bool { return r.Target != "candidate" }
func (r Request) Purpose() string {
	if r.Development {
		return "development"
	}
	return "production"
}

func (r Request) RequestedTarget() string {
	if r.Development {
		return r.Target
	}
	return "release"
}

// recallLog keeps the last lines of every build command in memory and
// forwards everything to the build log once it opens. Admission commands
// run before the log file exists; without the ring their voice would be
// lost and failures would surface as a bare exit status.
type recallLog struct {
	lines []string
	frag  string
	file  io.Writer
}

func newRecallLog() *recallLog {
	return &recallLog{}
}

func (l *recallLog) Write(p []byte) (int, error) {
	parts := strings.Split(l.frag+string(p), "\n")
	l.frag = parts[len(parts)-1]
	l.lines = append(l.lines, parts[:len(parts)-1]...)
	if len(l.lines) > 20 {
		l.lines = l.lines[len(l.lines)-20:]
	}
	if l.file != nil {
		return l.file.Write(p)
	}
	return len(p), nil
}

// attach connects the build log file. Buffered admission output is flushed
// first so the file holds the whole attempt from the first command.
func (l *recallLog) attach(w io.Writer) {
	for _, line := range l.lines {
		// Best-effort admission replay; the build outcome never depends on it.
		_, _ = fmt.Fprintln(w, line)
	}
	l.file = w
}

// reason returns the last tool error line: the likeliest one-line cause.
// Command markers and secret-adjacent shell echoes never qualify.
func (l *recallLog) reason() string {
	if line := qualifyReason(l.frag); line != "" {
		return line
	}
	for i := len(l.lines) - 1; i >= 0; i-- {
		if line := qualifyReason(l.lines[i]); line != "" {
			return line
		}
	}
	return ""
}

func qualifyReason(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "COMMAND ") || strings.HasPrefix(line, "$ ") {
		return ""
	}
	return line
}

// Build is the single candidate execution owner. Qualification and protected
// delivery consume its unchanged bytes; this result is not a qualified release.
func Build(ctx context.Context, r Request, progress *build.BuildProgress) (Result, error) {
	ring := newRecallLog()
	var log io.Writer = ring
	capture := func(dir, name string, args ...string) (string, error) {
		return runBuildCommand(ctx, log, nil, dir, name, args...)
	}
	execute := func(dir, name string, args ...string) error {
		_, err := runBuildCommand(ctx, log, os.Stdout, dir, name, args...)
		return err
	}
	res, err := runBuild(ctx, r, progress, execute, capture, func(path string) (func() error, error) {
		f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if e != nil {
			return nil, e
		}
		ring.attach(f)
		if !r.WantsMedia() {
			return f.Close, nil
		}
		events, e := os.OpenFile(filepath.Join(filepath.Dir(path), "media-events.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if e != nil {
			return nil, errors.Join(e, f.Close())
		}
		ring.attach(&mediaEventWriter{log: f, events: events, start: time.Now()})
		return func() error { return errors.Join(f.Close(), events.Close()) }, nil
	})
	if err != nil {
		progress.NoteReason(failureReason(ring, err))
	}
	return res, err
}

// failureReason prefers the recall ring for tool failures, where it holds
// the failing command's own stderr. Local failures (decode, file,
// validation) must surface their own error instead: the ring then holds
// only progress noise from earlier, already successful commands.
func failureReason(ring *recallLog, err error) string {
	if reason := ring.reason(); reason != "" && isToolFailure(err) {
		return reason
	}
	line, _, _ := strings.Cut(strings.TrimSpace(err.Error()), "\n")
	return line
}

// isToolFailure reports whether err came from a build command rather than
// local admission or validation.
func isToolFailure(err error) bool {
	return err != nil && strings.Contains(err.Error(), "retain attempt and inspect build.log")
}

func verifyCheckoutSource(requestedSource string, capture build.BuildCapture) (string, error) {
	source, err := capture(requestedSource, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	if source != requestedSource || !filepath.IsAbs(source) {
		return "", errors.New("canonical checkout root required")
	}
	info, err := os.Lstat(filepath.Join(source, ".git"))
	if err != nil || !info.IsDir() {
		return "", errors.New("canonical checkout required; no worktree")
	}
	return source, nil
}

func verifyCommittedRevision(source, requestedRevision string, capture build.BuildCapture) (string, error) {
	status, err := capture(source, "git", "status", "--porcelain", "--untracked-files=normal")
	if err != nil {
		return "", err
	}
	if status != "" {
		return "", errors.New("clean committed source required")
	}
	revision, err := capture(source, "git", "rev-parse", "HEAD")
	if err != nil || !build.Revision(revision) {
		return "", errors.New("exact committed revision required")
	}
	if requestedRevision != "" && requestedRevision != revision {
		return "", errors.New("controller/source revision mismatch")
	}
	return revision, nil
}

func verifyCompiler(source string, capture build.BuildCapture) error {
	if runtime.Version() != "go1.26.7" {
		return errors.New("pinned Go 1.26.7 required")
	}
	goVersion, err := capture(source, "go", "env", "GOVERSION")
	if err != nil {
		return err
	}
	if goVersion != runtime.Version() {
		return errors.New("pinned Go compiler unavailable")
	}
	return nil
}

func admitBuildOutput(out, source string, wantsMedia bool, authority string) error {
	if !filepath.IsAbs(out) || !strings.HasPrefix(filepath.Clean(out), filepath.Join(source, ".artifacts/releases")+string(os.PathSeparator)) {
		return errors.New("fresh output must be below .artifacts/releases; parent must exist")
	}
	if wantsMedia {
		if err := deliver.PrivateFile(authority); err != nil {
			return errors.New("restricted media authority file required")
		}
	}
	return build.FreshDirectory(out)
}

func admitBuildInputs(r Request, capture build.BuildCapture) (string, error) {
	if err := r.ValidateTarget(); err != nil {
		return "", err
	}
	if err := build.RequireNative(r.Arch); err != nil {
		return "", err
	}
	if !deliver.ValidRepositoryPrefix(r.RepositoryPrefix) {
		return "", errors.New("explicit intended repository prefix required")
	}
	source, err := verifyCheckoutSource(r.Source, capture)
	if err != nil {
		return "", err
	}
	if err = verifyCompiler(source, capture); err != nil {
		return "", err
	}
	revision, err := verifyCommittedRevision(source, r.Revision, capture)
	if err != nil {
		return "", err
	}
	return revision, admitBuildOutput(r.Out, source, r.WantsMedia(), r.MediaAuthority)
}

func initBuildDirectories(out string) error {
	for _, dir := range []string{"inputs", "work", "artifacts", "evidence", "release", "logs"} {
		if err := os.Mkdir(filepath.Join(out, dir), 0o700); err != nil {
			return err
		}
	}
	return nil
}

func extractBuildSnapshot(source, out, revision string, execute build.BuildExec) (string, error) {
	snapshot := filepath.Join(out, "work/source")
	if err := os.Mkdir(snapshot, 0o700); err != nil {
		return "", err
	}
	archive := filepath.Join(out, "inputs/source.tar")
	if err := execute(source, "git", "archive", "--format=tar", "--output", archive, revision); err != nil {
		return "", err
	}
	if err := execute(snapshot, "tar", "--extract", "--file", archive, "--no-same-owner"); err != nil {
		return "", err
	}
	return snapshot, nil
}

func setupBuildWorkspace(r Request, revision string, progress *build.BuildProgress, openLog func(string) (func() error, error), execute build.BuildExec) (string, func() error, error) {
	if err := initBuildDirectories(r.Out); err != nil {
		return "", nil, err
	}
	if err := progress.CreateLog(filepath.Join(r.Out, "logs/timing.log")); err != nil {
		return "", nil, err
	}
	closeLog, err := openLog(filepath.Join(r.Out, "logs/build.log"))
	if err != nil {
		return "", nil, err
	}
	snapshot, err := extractBuildSnapshot(r.Source, r.Out, revision, execute)
	if err != nil {
		_ = closeLog()
		return "", nil, err
	}
	return snapshot, closeLog, nil
}

func freezeBaseImageConfig(snapshot, out, contextDir, arch, prefix, compression string, base Base, execute build.BuildExec, capture build.BuildCapture) error {
	platform, _ := build.OCIArchitecture(arch)
	pinned := base.Images[arch]
	if err := execute(snapshot, "podman", "--remote=false", "pull", "--platform=linux/"+platform, pinned); err != nil {
		return err
	}
	metadata, err := capture(snapshot, "podman", "--remote=false", "run", "--cidfile", filepath.Join(out, "evidence/base-config.cid"), "--network=none", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--entrypoint=/usr/bin/cat", pinned, "/usr/share/coreos-assembler/image.json")
	if err != nil {
		return err
	}
	var imageConfig map[string]json.RawMessage
	if err = json.Unmarshal([]byte(metadata), &imageConfig); err != nil {
		return err
	}
	if len(imageConfig) == 0 {
		return errors.New("missing upstream image configuration")
	}
	imageConfig["container-imgref"], _ = json.Marshal("ostree-image-signed:docker://" + prefix + "-host:candidate")
	imageConfig["bootc-install-to-fs"] = json.RawMessage("false")
	if err = setMediaCompression(imageConfig, compression); err != nil {
		return err
	}
	imageData, err := json.MarshalIndent(imageConfig, "", "  ")
	if err != nil {
		return err
	}
	return ownedWrite(filepath.Join(contextDir, "rootfs/usr/share/coreos-assembler/image.json"), append(imageData, '\n'), 0o644)
}

func prepareBuildHostContext(snapshot, out, arch, revision, prefix, compression, coreOSInputs string, p *build.Production, execute build.BuildExec, capture build.BuildCapture) (string, Base, string, error) {
	contextDir := filepath.Join(out, "work/host-context")
	var base Base
	var err error
	if coreOSInputs != "" {
		base, err = PrepareResolved(snapshot, contextDir, arch, revision, coreOSInputs)
	} else {
		base, err = Prepare(snapshot, contextDir, arch, revision)
	}
	if err != nil {
		return "", base, "", err
	}
	packageHash, err := LockHostPackages(snapshot, contextDir, arch, base)
	if err != nil {
		return "", base, "", err
	}
	if err = p.ResolveInputs(); err != nil {
		return "", base, "", err
	}
	return contextDir, base, packageHash, freezeBaseImageConfig(snapshot, out, contextDir, arch, prefix, compression, base, execute, capture)
}

// prepareBuildProduction shares one Production: ResolveInputs records the
// frozen image inputs on it, and later phases read them back from the same
// value. Passing Production by value here would strand the inputs on a copy
// and fail P4 with "image inputs must be frozen before production".
func prepareBuildProduction(p *build.Production, r Request, snapshot, revision string, execute build.BuildExec, capture build.BuildCapture, next func(string) error) (string, Base, string, mediaTools, mediaLock, error) {
	contextDir, base, packageHash, err := prepareBuildHostContext(snapshot, r.Out, r.Arch, revision, r.RepositoryPrefix, r.MediaCompression, r.CoreOSInputs, p, execute, capture)
	if err != nil {
		return "", base, "", mediaTools{}, mediaLock{}, err
	}
	if err = next("P2 / Verify and install frozen dependencies"); err != nil {
		return "", base, "", mediaTools{}, mediaLock{}, err
	}
	if err = p.Dependencies(); err != nil {
		return "", base, "", mediaTools{}, mediaLock{}, err
	}
	tooling, assembler, err := prepareBuildMedia(*p, r)
	return contextDir, base, packageHash, tooling, assembler, err
}

func compileSodaCommands(p build.Production, snapshot, contextDir string) error {
	names, err := build.SodaCommands(snapshot)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err = p.Compile(name, "./cmd/"+name, filepath.Join(contextDir, "rootfs/usr/libexec/soda", name)); err != nil {
			return err
		}
	}
	return nil
}

func recordToolFiles(tools, revision, arch, artifacts string) error {
	toolFiles := map[string]build.File{}
	entries, err := os.ReadDir(tools)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		hash, err := build.HashFile(filepath.Join(tools, entry.Name()))
		if err != nil {
			return err
		}
		toolFiles[entry.Name()] = build.File{SHA256: hash, Mode: 0o755}
	}
	toolData, err := json.MarshalIndent(struct {
		Revision, Architecture string
		Files                  map[string]build.File
	}{revision, arch, toolFiles}, "", "  ")
	if err != nil {
		return err
	}
	return build.WriteNew(filepath.Join(artifacts, "tools.json"), append(toolData, '\n'), 0o600)
}

func compileShippingTools(p build.Production, snapshot, contextDir, artifacts, revision, arch string) error {
	if err := compileSodaCommands(p, snapshot, contextDir); err != nil {
		return err
	}
	tools := filepath.Join(artifacts, "tools")
	if err := os.Mkdir(tools, 0o755); err != nil {
		return err
	}
	for _, tool := range []struct{ name, pkg string }{
		{"soda-installer", "./appliance/installer"},
		{"soda-artifacts", "./tools/soda-artifacts"},
		{"soda-acceptance", "./tools/soda-acceptance"},
	} {
		if err := p.Compile(tool.name, tool.pkg, filepath.Join(tools, tool.name)); err != nil {
			return err
		}
	}
	if err := os.Link(filepath.Join(tools, "soda-installer"), filepath.Join(contextDir, "rootfs/usr/libexec/soda/soda-install")); err != nil {
		return err
	}
	return recordToolFiles(tools, revision, arch, artifacts)
}

func buildHostCandidate(snapshot, contextDir, artifacts, arch, revision, prefix string, base Base, packageHash string, p build.Production, next func(string) error) error {
	if _, err := completeCandidate(snapshot, contextDir, artifacts, arch, revision, prefix, base, packageHash, p, next); err != nil {
		return err
	}
	if err := next("P5 / Build FCOS host candidate"); err != nil {
		return err
	}
	if err := Inventory(contextDir); err != nil {
		return err
	}
	return buildHost(contextDir, artifacts, arch, revision, prefix, base, p, next)
}

func executeBuildProduction(ctx context.Context, p build.Production, r Request, snapshot, contextDir, artifacts, revision string, base Base, packageHash string, mediaTooling mediaTools, assembler mediaLock, phase func(string) error) (Result, error) {
	if err := phase("P3 / Compile shipping programs and prepared tools"); err != nil {
		return Result{}, err
	}
	if err := compileShippingTools(p, snapshot, contextDir, artifacts, revision, r.Arch); err != nil {
		return Result{}, err
	}
	if err := buildHostCandidate(snapshot, contextDir, artifacts, r.Arch, revision, r.RepositoryPrefix, base, packageHash, p, phase); err != nil {
		return Result{}, err
	}
	if err := finishBuildMedia(ctx, p, r, mediaTooling, assembler, phase); err != nil {
		return Result{}, err
	}
	return recordBuildResult(p, r)
}

func finalizeBuild(progress *build.BuildProgress, result Result, err error) (Result, error) {
	if err == nil {
		err = errors.Join(progress.End(nil), progress.EndPhase(nil))
	}
	return result, err
}

func runBuild(ctx context.Context, r Request, progress *build.BuildProgress, execute build.BuildExec, capture build.BuildCapture, openLog func(string) (func() error, error)) (result Result, err error) {
	revision, err := admitBuildInputs(r, capture)
	if err != nil {
		return result, err
	}
	if err = progress.Phase("P1 / Admit and freeze inputs"); err != nil {
		return result, err
	}
	snapshot, closeLog, err := setupBuildWorkspace(r, revision, progress, openLog, execute)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, closeLog()) }()

	artifacts := filepath.Join(r.Out, "artifacts")
	p := build.Production{Source: snapshot, Native: filepath.Join(snapshot, ".artifacts/native", r.Arch), Out: artifacts, Arch: r.Arch, Revision: revision, Vendor: true, Execute: execute, Capture: capture, Next: progress.Next}

	contextDir, base, packageHash, mediaTooling, assembler, err := prepareBuildProduction(&p, r, snapshot, revision, execute, capture, progress.Phase)
	if err != nil {
		return result, err
	}
	res, err := executeBuildProduction(ctx, p, r, snapshot, contextDir, artifacts, revision, base, packageHash, mediaTooling, assembler, progress.Phase)
	return finalizeBuild(progress, res, err)
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

// resolveBuildTool resolves a build tool at run time. The GOTOOLCHAIN pin in
// buildEnvironment forces the exact compiler version; PATH decides which
// installation provides it. A build-time GOROOT would answer a run-time
// question with a stale path once the binary moves machines.
func resolveBuildTool(name string) string {
	if name == "go" {
		if path, err := exec.LookPath("go"); err == nil {
			return path
		}
	}
	return name
}

func runBuildCommand(ctx context.Context, log, output io.Writer, dir, name string, args ...string) (string, error) {
	name = resolveBuildTool(name)
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
	if err != nil {
		return strings.TrimSpace(data.String()), fmt.Errorf("%s failed; retain attempt and inspect build.log: %w", filepath.Base(name), err)
	}
	return strings.TrimSpace(data.String()), nil
}

func preparedChecks(p build.Production) error {
	// Existing suites consume these aliases, never rebuild shipping assets.
	for link, target := range map[string]string{
		filepath.Join(p.Source, ".artifacts/forgejo-js"):              filepath.Join(p.Native, "forgejo-js"),
		filepath.Join(p.Source, ".artifacts/browser-terminal/vendor"): filepath.Join(p.Native, "terminal-assets"),
	} {
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
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
