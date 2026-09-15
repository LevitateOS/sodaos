// soda-build owns source-to-media production. B4/B5 connect native qualification
// and protected final release evidence; media output is not a qualified release.
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

	"github.com/levitateos/sodaos/internal/hostimage"
	"github.com/levitateos/sodaos/internal/nativebuild"
	"golang.org/x/sys/unix"
)

type interrupted int

func (e interrupted) Error() string { return "build interrupted" }
func (e interrupted) ExitCode() int { return int(e) }

// Incomplete is not a release success or permission to sign this candidate.
type incomplete struct{}

func (incomplete) Error() string {
	return "P1-P8 media verified; B4-B5 qualification/final signing are not connected; no qualified release"
}
func (incomplete) ExitCode() int { return 2 }
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(nativebuild.BuildExitCode(err))
	}
}

func parseBuildFlags() (r hostimage.Request, workerBuild bool, workerConfigPath string, err error) {
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
	flag.Parse()
	if flag.NArg() != 0 {
		return r, false, "", errors.New("unexpected positional arguments")
	}
	r = hostimage.Request{Out: *out, Arch: *arch, RepositoryPrefix: *prefix, RootfsBaseURL: *rootfs, MediaAuthority: *authority, Development: *development, Target: *target, MediaCompression: *compression}
	if err := r.ValidateTarget(); err != nil {
		return r, false, "", err
	}
	return r, *build, *configPath, nil
}

func admitBuildDispatch(workerBuild bool, workerConfigPath, authority string) error {
	if workerBuild {
		if workerConfigPath != "" {
			return errors.New("build stage cannot select qualification authority")
		}
		return buildWorkerIdentity()
	}
	if workerConfigPath == "" || authority != "" {
		return errors.New("root-owned --worker-config required; media authority belongs to the isolated worker")
	}
	return nil
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

func reportBuildResult(result hostimage.Result, r hostimage.Request) error {
	fmt.Fprintln(os.Stderr, "CANDIDATE", result.Candidate)
	if result.Media != "" {
		fmt.Fprintln(os.Stderr, "MEDIA", result.Media)
	}
	fmt.Fprintln(os.Stderr, result.Scope)
	return completion(r)
}

func runParentBuild(ctx context.Context, workerConfigPath string, r hostimage.Request, progress *nativebuild.BuildProgress) error {
	config, e := loadWorkerConfig(workerConfigPath, r)
	if e != nil {
		return e
	}
	result, e := runBuildWorker(ctx, config, r, progress)
	if e != nil {
		return e
	}
	return reportBuildResult(result, r)
}

func run() (err error) {
	r, workerBuild, workerConfigPath, err := parseBuildFlags()
	if err != nil {
		return err
	}
	if err := admitBuildDispatch(workerBuild, workerConfigPath, r.MediaAuthority); err != nil {
		return err
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	defer watchBuildSignals(ctx, cancel)()
	sanitizeBuildEnv(workerBuild)
	progress, e := startBuildProgress(r)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(context.Cause(ctx), err); err = errors.Join(err, progress.Finish(err)) }()
	if e = bindBuildSource(&r); e != nil {
		return e
	}
	if workerBuild {
		_, e := hostimage.Build(ctx, r, progress)
		return e // stage completion only; the trusted parent owns qualification
	}
	return runParentBuild(ctx, workerConfigPath, r, progress)
}

func completion(r hostimage.Request) error {
	if r.Development {
		return nil // success for the explicit target, never release qualification
	}
	return incomplete{}
}
