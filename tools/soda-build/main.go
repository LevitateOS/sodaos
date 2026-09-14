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
func run() (err error) {
	arch := flag.String("arch", "", "matching native x86_64 or aarch64")
	out := flag.String("out", "", "fresh absolute output below .artifacts/releases (parent must exist)")
	prefix := flag.String("repository-prefix", "ghcr.io/levitateos/sodaos", "intended immutable image repositories; no publication")
	rootfs := flag.String("rootfs-base-url", "", "public base URL for the exact hash-named rootfs file")
	authority := flag.String("media-authority", "", "restricted JSON naming Trust and Keys; never copied into build inputs")
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
	// No inherited child mode, false summary or unrelated installed-test activation.
	for _, key := range []string{"SODA_BUILD_START_NS", "SODA_BUILD_TIMING_LOG", "SODA_BUILD_CHILD"} {
		os.Unsetenv(key)
	}
	progress, e := nativebuild.NewBuildProgress("", "Soda release build")
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
	r := hostimage.Request{Source: source, Out: *out, Arch: *arch, RepositoryPrefix: *prefix, Revision: revision, RootfsBaseURL: *rootfs, MediaAuthority: *authority}
	result, e := hostimage.Build(ctx, r, progress)
	if e != nil {
		return e
	}
	fmt.Fprintln(os.Stderr, "MEDIA", result.Media, "(verified candidate-derived media; not a qualified release)")
	return incomplete{}
}
