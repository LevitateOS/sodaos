// soda-candidate wraps the admitted soda-build controller with preflight checks
// and a step-by-step progress display. Build choices are asked in the TUI;
// flags only pre-seed answers for scripting. The wrapper never admits
// workers, signs, or publishes; those authorities stay with the controller
// and the release configs.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// exitError carries a child exit code through run to main.
type exitError struct{ code int }

func (e exitError) Error() string {
	if e.code == 2 {
		return "controller incomplete: no qualified release without final signing"
	}
	return fmt.Sprintf("controller exited %d", e.code)
}

func (e exitError) ExitCode() int { return e.code }

type options struct {
	controller     string
	workerConfig   string
	arch           string
	out            string
	mode           string
	compression    string
	rootfsURL      string
	rootfsDir      string
	repoPrefix     string
	qualConfig     string
	signConfig     string
	nonInteractive bool
}

func parseOptions(args []string) (options, error) {
	var o options
	native, err := nativeArch()
	if err != nil {
		return o, err
	}
	fs := flag.NewFlagSet("soda-candidate", flag.ContinueOnError)
	fs.StringVar(&o.controller, "controller", "", "admitted soda-build executable (asked when empty)")
	fs.StringVar(&o.workerConfig, "worker-config", "", "restricted worker configuration (asked when empty)")
	fs.StringVar(&o.arch, "arch", native, "matching native x86_64 or aarch64")
	fs.StringVar(&o.out, "out", "", "fresh output below .artifacts/releases (asked when empty)")
	fs.StringVar(&o.mode, "mode", "", "candidate, media, or production (asked when empty)")
	fs.StringVar(&o.compression, "media-compression", "", "fast: development media only")
	fs.StringVar(&o.rootfsURL, "rootfs-base-url", "", "public base URL for the hash-named rootfs file")
	fs.StringVar(&o.rootfsDir, "rootfs-dir", fixtureRootfsDir, "pickup folder served for loopback development media")
	fs.StringVar(&o.repoPrefix, "repository-prefix", "ghcr.io/levitateos/sodaos", "intended image repositories; no publication")
	fs.StringVar(&o.qualConfig, "qualification-config", "", "production qualification configuration")
	fs.StringVar(&o.signConfig, "signing-config", "", "production final signing configuration")
	fs.BoolVar(&o.nonInteractive, "non-interactive", false, "require all flags; timestamped log output")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	if fs.NArg() != 0 {
		return o, errors.New("unexpected positional arguments")
	}
	if o.mode != "" && o.mode != "candidate" && o.mode != "media" && o.mode != "production" {
		return o, errors.New("--mode accepts candidate, media, or production")
	}
	if o.arch != "x86_64" && o.arch != "aarch64" {
		return o, errors.New("matching native x86_64 or aarch64 required")
	}
	return o, nil
}

func nativeArch() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64", nil
	case "arm64":
		return "aarch64", nil
	default:
		return "", fmt.Errorf("unsupported native %s: build requires matching x86_64 or aarch64", runtime.GOARCH)
	}
}

// validateResolved checks the final answers in the controller's own
// vocabulary. The controller re-admits everything; this never widens it.
func validateResolved(o *options) error {
	if o.controller == "" || !filepath.IsAbs(o.controller) {
		return errors.New("absolute controller path required")
	}
	if o.workerConfig == "" || !filepath.IsAbs(o.workerConfig) {
		return errors.New("absolute worker config path required")
	}
	if o.out == "" || !filepath.IsAbs(o.out) {
		return errors.New("absolute --out required")
	}
	if !validOutLeaf(filepath.Base(o.out)) {
		return errors.New("output name must be lowercase letters, digits, or dashes (worker name rule)")
	}
	switch o.mode {
	case "candidate":
		if o.rootfsURL != "" || o.compression != "" {
			return errors.New("candidate refuses media-only inputs")
		}
		if o.qualConfig != "" || o.signConfig != "" {
			return errors.New("development cannot request protected qualification or final signing")
		}
		return nil
	case "media":
		if o.qualConfig != "" || o.signConfig != "" {
			return errors.New("development cannot request protected qualification or final signing")
		}
		if o.compression != "" && o.compression != "fast" {
			return errors.New("--media-compression accepts only fast with development media")
		}
		if o.rootfsURL == "" {
			return errors.New("media requires the rootfs base URL")
		}
		return nil
	case "production":
		if o.qualConfig == "" {
			return errors.New("production requires the qualification config")
		}
		if o.compression != "" {
			return errors.New("media compression is development-only")
		}
		if o.rootfsURL == "" {
			return errors.New("production media requires the rootfs base URL")
		}
		return nil
	default:
		return errors.New("choose a build mode: candidate, media, or production")
	}
}

// preflight checks the operator-side facts the controller also enforces:
// checkout root, native arch, clean source revision, and a fresh output
// directory. The controller binds its source to this working directory, so
// starting anywhere else fails late and confusingly without this check.
func preflight(o options) error {
	if st, err := os.Stat("go.mod"); err != nil || st.IsDir() {
		return errors.New("run soda-candidate from the checkout root (~/Projects/sodaos)")
	}
	native, err := nativeArch()
	if err != nil {
		return err
	}
	if o.arch != native {
		return fmt.Errorf("arch %s is not this native %s host", o.arch, native)
	}
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=no")
	cmd.Dir, _ = os.Getwd()
	out, err := cmd.Output()
	if err != nil {
		return errors.New("source must be a clean git checkout")
	}
	if len(out) != 0 {
		return errors.New("controller requires committed source; commit or stash first")
	}
	if _, err := os.Stat(o.out); !os.IsNotExist(err) {
		return fmt.Errorf("output %s exists; choose a fresh --out per attempt", o.out)
	}
	if st, err := os.Stat(filepath.Dir(o.out)); err != nil || !st.IsDir() {
		return fmt.Errorf("output parent %s must already exist", filepath.Dir(o.out))
	}
	return nil
}

// monotonicNS shares the CLOCK_MONOTONIC epoch the controller totals use.
func monotonicNS() (int64, error) {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return 0, err
	}
	return ts.Nano(), nil
}

func describe(o options) string {
	return fmt.Sprintf("mode %s | arch %s | out %s | run %s", modeLabel(o.mode), o.arch, o.out, o.controller)
}

func run(args []string, stdin, stdout, stderr *os.File) error {
	o, err := parseOptions(args)
	if err != nil {
		return err
	}
	// Fail fast on a preseeded config before asking anything; the second
	// call below covers a path typed on the screen. Both are cheap and
	// idempotent: refuse a running worker, then drop idle session state.
	if o.workerConfig != "" {
		if err := prepareRuntime(o.workerConfig, listRunningBuildUnits); err != nil {
			return err
		}
	}
	interactive := isTerminal(stdin) && isTerminal(stderr) && !o.nonInteractive
	if interactive {
		p := newPrompter(bufio.NewReader(stdin), stderr)
		if err := p.overview(&o, suggestOut); err != nil {
			return err
		}
	} else if o.mode == "" {
		return errors.New("choose --mode candidate, media, or production (or run on a terminal)")
	}
	if err := validateResolved(&o); err != nil {
		return err
	}
	if err := preflight(o); err != nil {
		return err
	}
	if err := prepareRuntime(o.workerConfig, listRunningBuildUnits); err != nil {
		return err
	}
	// Loopback development media serves itself: the installer needs a live
	// pickup address, and the operator should never hand-run a file server.
	if fixtureWanted(o.mode, o.rootfsURL) {
		addr, err := fixtureAddr(o.rootfsURL)
		if err != nil {
			return err
		}
		stop, _, err := serveFixture(addr, o.rootfsDir)
		if err != nil {
			return err
		}
		defer stop()
	}
	ns, err := monotonicNS()
	if err != nil {
		return err
	}
	env := os.Environ()
	if os.Getenv("SODA_BUILD_START_NS") == "" {
		env = append(env, fmt.Sprintf("SODA_BUILD_START_NS=%d", ns))
	}
	argv := append([]string{o.controller}, controllerArgs(o)...)
	if os.Geteuid() != 0 {
		argv = append([]string{"sudo", fmt.Sprintf("SODA_BUILD_START_NS=%d", ns)}, argv...)
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir, _ = os.Getwd()
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	childErr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	tty := isTerminal(stderr) && !o.nonInteractive
	view := newRenderer(stderr, tty, termWidth(stderr))
	if err := view.note("soda-candidate: " + describe(o)); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	relay := make(chan os.Signal, 1)
	signal.Notify(relay, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(relay)
	go func() {
		for s := range relay {
			if sig, ok := s.(syscall.Signal); ok {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()
	if tty {
		view.startTicker()
	}
	sc := bufio.NewScanner(childErr)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	var feedErr error
	for sc.Scan() {
		if err := view.feed(strings.TrimRight(sc.Text(), "\r")); err != nil && feedErr == nil {
			feedErr = err
		}
	}
	waitErr := cmd.Wait()
	if tty {
		view.stopTicker()
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if feedErr != nil {
		return feedErr
	}
	code := 0
	var exit *exec.ExitError
	if errors.As(waitErr, &exit) {
		code = exit.ExitCode()
	} else if waitErr != nil {
		return waitErr
	}
	if err := view.finish(code); err != nil {
		return err
	}
	if code != 0 {
		return exitError{code: code}
	}
	// File the built image where the installer was told to fetch it, so the
	// run ends with a bootable ISO and a served rootfs, not homework.
	if fixtureWanted(o.mode, o.rootfsURL) {
		name, err := copyBuiltRootfs(o.out, o.rootfsDir)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(stderr, "soda-candidate: serving "+name+" from "+strings.TrimSuffix(o.rootfsURL, "/")+"/"+name); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "soda-candidate:", err)
		var ee exitError
		if errors.As(err, &ee) {
			os.Exit(ee.ExitCode())
		}
		os.Exit(1)
	}
}
