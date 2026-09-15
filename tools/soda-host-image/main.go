// Retained writable-layout producer for the old installer only. Host-candidate
// production has one owner: tools/soda-build. Remove this legacy lane at B6.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/levitateos/sodaos/internal/nativebuild"
)

func main() {
	// Only the retiring shell/media lane retains Python group supervision.
	if os.Getenv("SODA_BUILD_SUPERVISED") != "1" {
		source, e := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		self, e := os.Executable()
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		python, e := exec.LookPath("python3")
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		args := append([]string{python, filepath.Join(strings.TrimSpace(string(source)), "scripts/build_progress.py"), "supervise", self}, os.Args[1:]...)
		if e = syscall.Exec(python, args, os.Environ()); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(nativebuild.BuildExitCode(err))
	}
}

func gitOutput(args ...string) (string, error) {
	data, e := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(data)), e
}

func admitLegacyNative(arch, out string) (source, revision string, err error) {
	if e := nativebuild.RequireNative(arch); e != nil {
		return "", "", e
	}
	if runtime.Version() != "go1.26.7" {
		return "", "", errors.New("pinned Go 1.26.7 required")
	}
	source, e := gitOutput("rev-parse", "--show-toplevel")
	if e != nil {
		return "", "", e
	}
	status, e := gitOutput("status", "--porcelain", "--untracked-files=normal")
	if e != nil {
		return "", "", e
	}
	if status != "" {
		return "", "", errors.New("clean committed source required")
	}
	revision, e = gitOutput("rev-parse", "HEAD")
	if e != nil {
		return "", "", e
	}
	if out != filepath.Join(source, ".artifacts/native", arch) {
		return "", "", errors.New("legacy output must use its native stage path")
	}
	return source, revision, nil
}

func run() (err error) {
	arch := flag.String("arch", "", "native architecture")
	out := flag.String("out", "", "fresh legacy native output")
	legacy := flag.Bool("legacy-native", false, "retained writable installer only")
	flag.Parse()
	if !*legacy || flag.NArg() != 0 {
		return errors.New("host producer retired; use soda-build; this command requires --legacy-native")
	}
	source, revision, e := admitLegacyNative(*arch, *out)
	if e != nil {
		return e
	}
	if e = nativebuild.FreshDirectory(*out); e != nil {
		return e
	}
	progress, e := nativebuild.NewBuildProgress(source, "Legacy native components")
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, progress.Finish(err)) }()
	if e = progress.CreateLog(filepath.Join(*out, "timing.log")); e != nil {
		return e
	}
	log, e := os.OpenFile(filepath.Join(*out, "build.log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if e != nil {
		return e
	}
	defer log.Close()
	execution := nativebuild.BuildExecution{Context: context.Background(), Log: log, Output: os.Stdout}
	return legacyNative(source, *out, *arch, revision, execution, progress.Next)
}
