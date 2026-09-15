// soda-build owns source-to-native-qualification production. B5 final signing
// remains disconnected; native qualification alone is not a qualified release.
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

	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"github.com/levitateos/sodaos/internal/nativequalification"
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
	Request             hostimage.Request
	WorkerBuild         bool
	WorkerConfig        string
	QualificationConfig string
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
	workerQualify := flag.Bool("worker-qualify", false, "internal protected native qualification worker")
	guestAction := flag.String("qualification-state", "", "internal native guest observation/state action")
	expectedPayload := flag.String("expected-payload", "", "internal exact guest payload identity")
	flag.Parse()
	if flag.NArg() != 0 {
		return f, errors.New("unexpected positional arguments")
	}
	f.Request = hostimage.Request{Out: *out, Arch: *arch, RepositoryPrefix: *prefix, RootfsBaseURL: *rootfs, MediaAuthority: *authority, Development: *development, Target: *target, MediaCompression: *compression}
	f.WorkerBuild, f.WorkerConfig = *build, *configPath
	f.QualificationConfig, f.WorkerQualify = *qualificationConfig, *workerQualify
	f.GuestAction, f.ExpectedPayload = *guestAction, *expectedPayload
	if f.GuestAction != "" || f.WorkerQualify {
		return f, nil
	}
	return f, f.Request.ValidateTarget()
}

func admitWorkerBuild(f buildFlags) error {
	if f.WorkerConfig != "" || f.QualificationConfig != "" {
		return errors.New("build stage cannot select qualification authority")
	}
	return buildWorkerIdentity()
}

func admitParentDispatch(f buildFlags) error {
	if f.WorkerConfig == "" || f.Request.MediaAuthority != "" {
		return errors.New("root-owned --worker-config required; media authority belongs to the isolated worker")
	}
	if f.Request.Development && f.QualificationConfig != "" {
		return errors.New("development build cannot request protected qualification")
	}
	if !f.Request.Development && (f.QualificationConfig == "" || f.Request.Arch != "x86_64") {
		return errors.New("production requires --qualification-config and the selected native x86_64 scenario")
	}
	return nil
}

func admitBuildDispatch(f buildFlags) error {
	if f.WorkerBuild {
		return admitWorkerBuild(f)
	}
	if f.WorkerQualify || f.GuestAction != "" {
		return nil
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

func startBuildProgress(r hostimage.Request) (*nativebuild.BuildProgress, error) {
	title := "Soda release build"
	if r.Development {
		title = "Soda development " + r.Target + " (not release-qualified)"
	}
	return nativebuild.NewBuildProgress("", title)
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
	if !nativebuild.Revision(revision) {
		return "", errors.New("controller needs build VCS metadata; compile tools/soda-build from the committed checkout")
	}
	return revision, nil
}

func bindBuildSource(r *hostimage.Request) error {
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
	state, err := nativequalification.GuestState(ctx, action, expectedPayload)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(state)
}

func printBuildArtifacts(result hostimage.Result) {
	fmt.Fprintln(os.Stderr, "CANDIDATE", result.Candidate)
	if result.Media != "" {
		fmt.Fprintln(os.Stderr, "MEDIA", result.Media)
	}
	fmt.Fprintln(os.Stderr, result.Scope)
}

func runProtectedQualification(ctx context.Context, progress *nativebuild.BuildProgress, result hostimage.Result, r hostimage.Request, qualification nativequalification.Config) error {
	if err := progress.Phase("P9 / Protected native install-update-recovery"); err != nil {
		return err
	}
	if err := nativequalification.Dispatch(ctx, qualification, filepath.Dir(result.Candidate), r.Revision, filepath.Join(r.Source, ".artifacts/b4-qualification/controller-runs", filepath.Base(r.Out)), os.Stderr); err != nil {
		return err
	}
	return errors.Join(progress.End(nil), progress.EndPhase(nil))
}

func loadQualification(path string, configExecutable string, development bool) (nativequalification.Config, error) {
	var qualification nativequalification.Config
	if development {
		return qualification, nil
	}
	qualification, err := nativequalification.LoadConfig(path)
	if err != nil {
		return qualification, err
	}
	if qualification.Executable != configExecutable {
		return qualification, errors.New("qualification must use this admitted controller")
	}
	return qualification, nil
}

func reportBuildResult(ctx context.Context, progress *nativebuild.BuildProgress, result hostimage.Result, r hostimage.Request, qualification nativequalification.Config) error {
	printBuildArtifacts(result)
	if !r.Development {
		if err := runProtectedQualification(ctx, progress, result, r, qualification); err != nil {
			return err
		}
	}
	return completion(r)
}

func runParentBuild(ctx context.Context, workerConfigPath, qualificationConfig string, r hostimage.Request, progress *nativebuild.BuildProgress) error {
	config, err := loadWorkerConfig(workerConfigPath, r)
	if err != nil {
		return err
	}
	qualification, err := loadQualification(qualificationConfig, config.Executable, r.Development)
	if err != nil {
		return err
	}
	result, err := runBuildWorker(ctx, config, r, progress)
	if err != nil {
		return err
	}
	return reportBuildResult(ctx, progress, result, r, qualification)
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
		return nativequalification.Run(ctx, f.QualificationConfig)
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
		_, e := hostimage.Build(ctx, f.Request, progress)
		return e // stage completion only; the trusted parent owns qualification
	}
	return runParentBuild(ctx, f.WorkerConfig, f.QualificationConfig, f.Request, progress)
}

func completion(r hostimage.Request) error {
	if r.Development {
		return nil // success for the explicit target, never release qualification
	}
	return incomplete{}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(nativebuild.BuildExitCode(err))
	}
}
