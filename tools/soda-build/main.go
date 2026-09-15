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
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(nativebuild.BuildExitCode(err))
	}
}
func run() (err error) {
	development := flag.Bool("development", false, "explicit development-only run; never release-qualified")
	target := flag.String("target", "", "development boundary: candidate or media (requires --development)")
	compression := flag.String("media-compression", "", "fast: development media only; changes host compression metadata (default: upstream)")
	arch := flag.String("arch", "", "matching native x86_64 or aarch64")
	out := flag.String("out", "", "fresh absolute output below .artifacts/releases (parent must exist)")
	prefix := flag.String("repository-prefix", "ghcr.io/levitateos/sodaos", "intended immutable image repositories; no publication")
	rootfs := flag.String("rootfs-base-url", "", "public base URL for the exact hash-named rootfs file")
	authority := flag.String("media-authority", "", "worker-local fixture authority; not release custody")
	workerConfigPath := flag.String("worker-config", "", "root-owned configuration for isolated worker dispatch")
	workerBuild := flag.Bool("worker-build", false, "internal build stage; requires the isolated build identity")
	qualificationConfig := flag.String("qualification-config", "", "root-owned qualification configuration; required for production")
	workerQualify := flag.Bool("worker-qualify", false, "internal protected native qualification worker")
	guestAction := flag.String("qualification-state", "", "internal native guest observation/state action")
	expectedPayload := flag.String("expected-payload", "", "internal exact guest payload identity")
	flag.Parse()
	if flag.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		select {
		case s := <-signals:
			cancel(interrupted(128 + int(s.(syscall.Signal))))
		case <-ctx.Done():
		}
	}()
	if *guestAction != "" {
		if *guestAction != "snapshot" && *guestAction != "later" && *guestAction != "content" {
			return errors.New("guest reseeding refused")
		}
		state, e := nativequalification.GuestState(ctx, *guestAction, *expectedPayload)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(state)
	}
	if *workerQualify {
		return nativequalification.Run(ctx, *qualificationConfig)
	}
	r := hostimage.Request{Out: *out, Arch: *arch, RepositoryPrefix: *prefix, RootfsBaseURL: *rootfs, MediaAuthority: *authority, Development: *development, Target: *target, MediaCompression: *compression}
	if err := r.ValidateTarget(); err != nil {
		return err
	}
	if *workerBuild {
		if *workerConfigPath != "" || *qualificationConfig != "" {
			return errors.New("build stage cannot select qualification authority")
		}
		if err := buildWorkerIdentity(); err != nil {
			return err
		}
	} else if *workerConfigPath == "" || *authority != "" {
		return errors.New("root-owned --worker-config required; media authority belongs to the isolated worker")
	}
	if r.Development && *qualificationConfig != "" {
		return errors.New("development build cannot request protected qualification")
	}
	if !r.Development && !*workerBuild && (*qualificationConfig == "" || r.Arch != "x86_64") {
		return errors.New("production requires --qualification-config and the selected native x86_64 scenario")
	}
	// No inherited child mode, false summary or unrelated installed-test activation.
	for _, key := range []string{"SODA_BUILD_TIMING_LOG", "SODA_BUILD_CHILD"} {
		os.Unsetenv(key)
	}
	if !*workerBuild {
		os.Unsetenv("SODA_BUILD_START_NS")
	}
	title := "Soda release build"
	if r.Development {
		title = "Soda development " + r.Target + " (not release-qualified)"
	}
	progress, e := nativebuild.NewBuildProgress("", title)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(context.Cause(ctx), err); err = errors.Join(err, progress.Finish(err)) }()
	if e = unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); e != nil {
		return e
	}
	source, e := os.Getwd()
	if e != nil {
		return e
	}
	var revision string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = setting.Value
			}
			if setting.Key == "vcs.modified" && setting.Value == "true" {
				return errors.New("controller must be compiled from committed source")
			}
		}
	}
	if !nativebuild.Revision(revision) {
		return errors.New("controller needs build VCS metadata; compile tools/soda-build from the committed checkout")
	}
	r.Source, r.Revision = source, revision
	if *workerBuild {
		_, e := hostimage.Build(ctx, r, progress)
		return e // stage completion only; the trusted parent owns qualification
	}
	config, e := loadWorkerConfig(*workerConfigPath, r)
	if e != nil {
		return e
	}
	var qualification nativequalification.Config
	if !r.Development {
		qualification, e = nativequalification.LoadConfig(*qualificationConfig)
		if e != nil {
			return e
		}
		if qualification.Executable != config.Executable {
			return errors.New("qualification must use this admitted controller")
		}
	}
	result, e := runBuildWorker(ctx, config, r, progress)
	if e != nil {
		return e
	}
	fmt.Fprintln(os.Stderr, "CANDIDATE", result.Candidate)
	if result.Media != "" {
		fmt.Fprintln(os.Stderr, "MEDIA", result.Media)
	}
	fmt.Fprintln(os.Stderr, result.Scope)
	if !r.Development {
		if e = progress.Phase("P9 / Protected native install-update-recovery"); e != nil {
			return e
		}
		e = nativequalification.Dispatch(ctx, qualification, filepath.Dir(result.Candidate), r.Revision, filepath.Join(r.Source, ".artifacts/b4-qualification/controller-runs", filepath.Base(r.Out)), os.Stderr)
		if e != nil {
			return e
		}
		if e = errors.Join(progress.End(nil), progress.EndPhase(nil)); e != nil {
			return e
		}
	}
	return completion(r)
}

func completion(r hostimage.Request) error {
	if r.Development {
		return nil // success for the explicit target, never release qualification
	}
	return incomplete{}
}
