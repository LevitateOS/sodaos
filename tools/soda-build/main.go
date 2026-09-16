// soda-build owns source-to-native-qualification production and connects
// protected final signing when --signing-config is admitted.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
	"github.com/levitateos/sodaos/internal/release/image"
	"github.com/levitateos/sodaos/internal/release/qualify"
	"golang.org/x/sys/unix"
)

type interrupted int

func (e interrupted) Error() string { return "build interrupted" }
func (e interrupted) ExitCode() int { return int(e) }

// Incomplete is not a release success or permission to sign this candidate.
type incomplete struct{}

func (incomplete) Error() string {
	return "P9 native qualification passed; B5 final protected signing is not connected; no qualified release"
}
func (incomplete) ExitCode() int { return 2 }

type buildFlags struct {
	Request             image.Request
	WorkerBuild         bool
	WorkerConfig        string
	QualificationConfig string
	SigningConfig       string
	WorkerQualify       bool
	GuestAction         string
	ExpectedPayload     string
}

func parseBuildFlags() (buildFlags, error) {
	var f buildFlags
	development := flag.Bool("development", false, "explicit development-only run; never release-qualified")
	target := flag.String("target", "", "development boundary: candidate or media (requires --development)")
	compression := flag.String("media-compression", "", "fast: development media only; changes host compression metadata (default: upstream)")
	arch := flag.String("arch", "", "matching native x86_64 or aarch64")
	out := flag.String("out", "", "fresh absolute output below .artifacts/releases (parent must exist)")
	prefix := flag.String("repository-prefix", "ghcr.io/levitateos/sodaos", "intended immutable image repositories; no publication")
	rootfs := flag.String("rootfs-base-url", "", "public base URL for the exact hash-named rootfs file")
	authority := flag.String("media-authority", "", "worker-local fixture authority; not release custody")
	configPath := flag.String("worker-config", "", "root-owned configuration for isolated worker dispatch")
	build := flag.Bool("worker-build", false, "internal build stage; requires the isolated build identity")
	qualificationConfig := flag.String("qualification-config", "", "root-owned qualification configuration; required for production")
	signingConfig := flag.String("signing-config", "", "root-owned final signing configuration; required for a qualified release")
	workerQualify := flag.Bool("worker-qualify", false, "internal protected native qualification worker")
	guestAction := flag.String("qualification-state", "", "internal native guest observation/state action")
	expectedPayload := flag.String("expected-payload", "", "internal exact guest payload identity")
	flag.Parse()
	if flag.NArg() != 0 {
		return f, errors.New("unexpected positional arguments")
	}
	f.Request = image.Request{Out: *out, Arch: *arch, RepositoryPrefix: *prefix, RootfsBaseURL: *rootfs, MediaAuthority: *authority, Development: *development, Target: *target, MediaCompression: *compression}
	f.WorkerBuild, f.WorkerConfig = *build, *configPath
	f.QualificationConfig, f.SigningConfig = *qualificationConfig, *signingConfig
	f.WorkerQualify = *workerQualify
	f.GuestAction, f.ExpectedPayload = *guestAction, *expectedPayload
	if f.GuestAction != "" || f.WorkerQualify {
		return f, nil
	}
	return f, f.Request.ValidateTarget()
}

func admitWorkerBuild(f buildFlags) error {
	if f.WorkerConfig != "" || f.QualificationConfig != "" || f.SigningConfig != "" {
		return errors.New("build stage cannot select qualification or signing authority")
	}
	return buildWorkerIdentity()
}

func admitParentDispatch(f buildFlags) error {
	if f.WorkerConfig == "" || f.Request.MediaAuthority != "" {
		return errors.New("root-owned --worker-config required; media authority belongs to the isolated worker")
	}
	if f.Request.Development && (f.QualificationConfig != "" || f.SigningConfig != "") {
		return errors.New("development build cannot request protected qualification or final signing")
	}
	if !f.Request.Development && (f.QualificationConfig == "" || f.Request.Arch != "x86_64") {
		return errors.New("production requires --qualification-config and the selected native x86_64 scenario")
	}
	return nil
}

func admitWorkerQualify(f buildFlags) error {
	if f.WorkerBuild || f.GuestAction != "" || f.WorkerConfig != "" || f.SigningConfig != "" {
		return errors.New("qualification worker cannot select build, guest or signing authority")
	}
	if f.QualificationConfig != "/run/soda-p9-input/config.json" {
		return errors.New("qualification worker requires the mounted qualifier inputs")
	}
	return qualifierIdentity()
}

func admitGuestAction(f buildFlags) error {
	if f.WorkerBuild || f.WorkerQualify || f.WorkerConfig != "" || f.QualificationConfig != "" || f.SigningConfig != "" {
		return errors.New("guest observation cannot select worker or release authority")
	}
	if f.ExpectedPayload == "" {
		return errors.New("guest observation requires the exact payload identity")
	}
	if f.GuestAction != "snapshot" && f.GuestAction != "later" && f.GuestAction != "content" {
		return errors.New("guest reseeding refused")
	}
	return nil
}

func admitBuildDispatch(f buildFlags) error {
	if f.WorkerBuild {
		return admitWorkerBuild(f)
	}
	if f.GuestAction != "" {
		return admitGuestAction(f)
	}
	if f.WorkerQualify {
		return admitWorkerQualify(f)
	}
	return admitParentDispatch(f)
}

func watchBuildSignals(ctx context.Context, cancel context.CancelCauseFunc) func() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case s := <-signals:
			cancel(interrupted(128 + int(s.(syscall.Signal))))
		case <-ctx.Done():
		}
	}()
	return func() { signal.Stop(signals) }
}

func sanitizeBuildEnv(workerBuild bool) {
	// No inherited child mode, false summary or unrelated installed-test activation.
	for _, key := range []string{"SODA_BUILD_TIMING_LOG", "SODA_BUILD_CHILD"} {
		os.Unsetenv(key)
	}
	if !workerBuild {
		os.Unsetenv("SODA_BUILD_START_NS")
	}
}

func startBuildProgress(r image.Request) (*build.BuildProgress, error) {
	title := "Soda release build"
	if r.Development {
		title = "Soda development " + r.Target + " (not release-qualified)"
	}
	return build.NewBuildProgress("", title)
}

func controllerRevision() (string, error) {
	var revision string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
			if setting.Key == "vcs.modified" && setting.Value == "true" {
				return "", errors.New("controller must be compiled from committed source")
			}
		}
	}
	if !build.Revision(revision) {
		return "", errors.New("controller needs build VCS metadata; compile tools/soda-build from the committed checkout")
	}
	return revision, nil
}

func bindBuildSource(r *image.Request) error {
	if e := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); e != nil {
		return e
	}
	source, e := os.Getwd()
	if e != nil {
		return e
	}
	revision, e := controllerRevision()
	if e != nil {
		return e
	}
	r.Source, r.Revision = source, revision
	return nil
}

func runGuestAction(ctx context.Context, action, expectedPayload string) error {
	if action != "snapshot" && action != "later" && action != "content" {
		return errors.New("guest reseeding refused")
	}
	state, err := qualify.GuestState(ctx, action, expectedPayload)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(state)
}

func printBuildArtifacts(result image.Result) {
	fmt.Fprintln(os.Stderr, "CANDIDATE", result.Candidate)
	if result.Media != "" {
		fmt.Fprintln(os.Stderr, "MEDIA", result.Media)
	}
	fmt.Fprintln(os.Stderr, result.Scope)
}

func runProtectedQualification(ctx context.Context, progress *build.BuildProgress, result image.Result, r image.Request, qualification qualify.Config) (string, error) {
	if err := progress.Phase("P9 / Protected native install-update-recovery"); err != nil {
		return "", err
	}
	custody := filepath.Join(r.Source, ".artifacts/b4-qualification/controller-runs", filepath.Base(r.Out))
	if err := qualify.Dispatch(ctx, qualification, filepath.Dir(result.Candidate), r.Revision, custody, os.Stderr); err != nil {
		return "", err
	}
	return custody, errors.Join(progress.End(nil), progress.EndPhase(nil))
}

func loadSigning(path string, development bool) (deliver.Config, error) {
	var signing deliver.Config
	if development || path == "" {
		return signing, nil
	}
	return deliver.LoadConfig(path)
}

func runProtectedFinalization(ctx context.Context, progress *build.BuildProgress, result image.Result, custody string, signing deliver.Config) error {
	if signing.Trust == "" {
		return incomplete{}
	}
	if err := progress.Phase("P10 / Finalize and sign release metadata"); err != nil {
		return err
	}
	if result.Media == "" {
		return errors.New("final signing requires sealed media.json from P8")
	}
	evidence := filepath.Join(custody, "evidence/qualification.json")
	out := filepath.Join(custody, "final")
	ref, err := deliver.Finalize(ctx, deliver.Native{Home: out}, signing, filepath.Dir(result.Candidate), result.Media, evidence, out)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "FINAL", ref)
	return errors.Join(progress.End(nil), progress.EndPhase(nil))
}

func reportBuildResult(ctx context.Context, progress *build.BuildProgress, result image.Result, r image.Request, qualification qualify.Config, signing deliver.Config) error {
	printBuildArtifacts(result)
	if r.Development {
		return nil
	}
	custody, err := runProtectedQualification(ctx, progress, result, r, qualification)
	if err != nil {
		return err
	}
	return runProtectedFinalization(ctx, progress, result, custody, signing)
}

func loadQualification(path string, configExecutable string, development bool) (qualify.Config, error) {
	var qualification qualify.Config
	if development {
		return qualification, nil
	}
	qualification, err := qualify.LoadConfig(path)
	if err != nil {
		return qualification, err
	}
	if qualification.Executable != configExecutable {
		return qualification, errors.New("qualification must use this admitted controller")
	}
	return qualification, nil
}

func runParentBuild(ctx context.Context, workerConfigPath, qualificationConfig, signingConfig string, r image.Request, progress *build.BuildProgress) error {
	config, err := loadWorkerConfig(workerConfigPath, r)
	if err != nil {
		return err
	}
	qualification, err := loadQualification(qualificationConfig, config.Executable, r.Development)
	if err != nil {
		return err
	}
	signing, err := loadSigning(signingConfig, r.Development)
	if err != nil {
		return err
	}
	result, err := runBuildWorker(ctx, config, r, progress)
	if err != nil {
		return err
	}
	return reportBuildResult(ctx, progress, result, r, qualification, signing)
}

func run() (err error) {
	f, err := parseBuildFlags()
	if err != nil {
		return err
	}
	if err := admitBuildDispatch(f); err != nil {
		return err
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	defer watchBuildSignals(ctx, cancel)()
	if f.GuestAction != "" {
		return runGuestAction(ctx, f.GuestAction, f.ExpectedPayload)
	}
	if f.WorkerQualify {
		return qualify.Run(ctx, f.QualificationConfig)
	}
	sanitizeBuildEnv(f.WorkerBuild)
	progress, e := startBuildProgress(f.Request)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(context.Cause(ctx), err); err = errors.Join(err, progress.Finish(err)) }()
	if e = bindBuildSource(&f.Request); e != nil {
		return e
	}
	if f.WorkerBuild {
		_, e := image.Build(ctx, f.Request, progress)
		return e // stage completion only; the trusted parent owns qualification
	}
	return runParentBuild(ctx, f.WorkerConfig, f.QualificationConfig, f.SigningConfig, f.Request, progress)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(build.BuildExitCode(err))
	}
}
