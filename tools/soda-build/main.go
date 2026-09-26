// soda-build dispatches development candidate and media builds to the
// isolated worker. Qualification, signing and production paths are removed;
// development output is never a release.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/image"
	"golang.org/x/sys/unix"
)

type interrupted int

func (e interrupted) Error() string { return "build interrupted" }
func (e interrupted) ExitCode() int { return int(e) }

type buildFlags struct {
	Request      image.Request
	WorkerBuild  bool
	WorkerConfig string
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
	liveInputs := flag.String("live-inputs", "", "internal controller-resolved live inputs file")
	configPath := flag.String("worker-config", "", "root-owned configuration for isolated worker dispatch")
	build := flag.Bool("worker-build", false, "internal build stage; requires the isolated build identity")
	flag.Parse()
	if flag.NArg() != 0 {
		return f, errors.New("unexpected positional arguments")
	}
	f.Request = image.Request{Out: *out, Arch: *arch, RepositoryPrefix: *prefix, RootfsBaseURL: *rootfs, MediaAuthority: *authority, Development: *development, Target: *target, MediaCompression: *compression, LiveInputs: *liveInputs}
	f.WorkerBuild, f.WorkerConfig = *build, *configPath
	return f, f.Request.ValidateTarget()
}

func admitWorkerBuild(f buildFlags) error {
	if f.WorkerConfig != "" {
		return errors.New("build stage cannot select worker configuration")
	}
	if f.Request.LiveInputs == "" {
		return errors.New("isolated worker requires controller-resolved live inputs; it never fetches")
	}
	return buildWorkerIdentity()
}

func admitParentDispatch(f buildFlags) error {
	if f.WorkerConfig == "" || f.Request.MediaAuthority != "" {
		return errors.New("root-owned --worker-config required; media authority belongs to the isolated worker")
	}
	if f.Request.LiveInputs != "" {
		return errors.New("controller resolves live inputs per attempt; operator selection refused")
	}
	if !f.Request.Development {
		return errors.New("production builds removed; development only")
	}
	return nil
}

func admitBuildDispatch(f buildFlags) error {
	if f.WorkerBuild {
		return admitWorkerBuild(f)
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

func printBuildArtifacts(result image.Result) {
	fmt.Fprintln(os.Stderr, "CANDIDATE", result.Candidate)
	if result.Media != "" {
		fmt.Fprintln(os.Stderr, "MEDIA", result.Media)
	}
	fmt.Fprintln(os.Stderr, result.Scope)
}

func runParentBuild(ctx context.Context, workerConfigPath string, r image.Request, progress *build.BuildProgress) error {
	config, err := loadWorkerConfig(workerConfigPath, r)
	if err != nil {
		return err
	}
	result, err := runBuildWorker(ctx, config, r, progress)
	if err != nil {
		return err
	}
	printBuildArtifacts(result)
	return nil
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
		return e // stage completion only
	}
	return runParentBuild(ctx, f.WorkerConfig, f.Request, progress)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(build.BuildExitCode(err))
	}
}
