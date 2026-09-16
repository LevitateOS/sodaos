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
	"path/filepath"
	"runtime"
	"strings"

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
	if err := validateModeFlag(o.mode); err != nil {
		return o, err
	}
	return o, validateArchFlag(o.arch)
}

func validateModeFlag(mode string) error {
	if mode != "" && mode != "candidate" && mode != "media" && mode != "production" {
		return errors.New("--mode accepts candidate, media, or production")
	}
	return nil
}

func validateArchFlag(arch string) error {
	if arch != "x86_64" && arch != "aarch64" {
		return errors.New("matching native x86_64 or aarch64 required")
	}
	return nil
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
	if err := validateCommonPaths(o); err != nil {
		return err
	}
	switch o.mode {
	case "candidate":
		return validateCandidate(o)
	case "media":
		return validateMedia(o)
	case "production":
		return validateProduction(o)
	default:
		return errors.New("choose a build mode: candidate, media, or production")
	}
}

func validateCommonPaths(o *options) error {
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
	return nil
}

func validateCandidate(o *options) error {
	if o.rootfsURL != "" || o.compression != "" {
		return errors.New("candidate refuses media-only inputs")
	}
	if o.qualConfig != "" || o.signConfig != "" {
		return errors.New("development cannot request protected qualification or final signing")
	}
	return nil
}

func validateMedia(o *options) error {
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
}

func validateProduction(o *options) error {
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
}

// preflight checks the operator-side facts the controller also enforces:
// checkout root, native arch, clean source revision, and a fresh output
// directory. The controller binds its source to this working directory, so
// starting anywhere else fails late and confusingly without this check.
func preflight(o options) error {
	if err := checkCheckoutRoot(); err != nil {
		return err
	}
	native, err := nativeArch()
	if err != nil {
		return err
	}
	if o.arch != native {
		return fmt.Errorf("arch %s is not this native %s host", o.arch, native)
	}
	if err := checkCleanTree(); err != nil {
		return err
	}
	return checkFreshOut(o.out)
}

func checkCheckoutRoot() error {
	if st, err := os.Stat("go.mod"); err != nil || st.IsDir() {
		return errors.New("run soda-candidate from the checkout root (~/Projects/sodaos)")
	}
	return nil
}

func checkCleanTree() error {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir, _ = os.Getwd()
	out, err := cmd.Output()
	if err != nil {
		return errors.New("source must be a clean git checkout")
	}
	if lines := dirtyFiles(out); len(lines) != 0 {
		shown := lines
		suffix := ""
		if len(lines) > 10 {
			shown = lines[:10]
			suffix = fmt.Sprintf("\n... and %d more", len(lines)-10)
		}
		return fmt.Errorf(
			"controller requires committed source (%d dirty file(s)); commit or stash first:\n%s%s",
			len(lines), strings.Join(shown, "\n"), suffix)
	}
	return nil
}

// dirtyFiles parses git status porcelain output into per-file status lines.
func dirtyFiles(out []byte) []string {
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func checkFreshOut(out string) error {
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		return fmt.Errorf("output %s exists; choose a fresh --out per attempt", out)
	}
	if st, err := os.Stat(filepath.Dir(out)); err != nil || !st.IsDir() {
		return fmt.Errorf("output parent %s must already exist", filepath.Dir(out))
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

// preseedRuntime fails fast on a preseeded config before asking anything;
// the prepareRuntime call in run covers a path typed on the screen. Both
// are cheap and idempotent: refuse a running worker, then drop idle
// session state.
func preseedRuntime(o options) error {
	if o.workerConfig == "" {
		return nil
	}
	return prepareRuntime(o, listRunningBuildUnits)
}

func resolveOptions(args []string, stdin, stderr *os.File) (options, error) {
	o, err := parseOptions(args)
	if err != nil {
		return o, err
	}
	if err := preseedRuntime(o); err != nil {
		return o, err
	}
	if !isTerminal(stdin) || !isTerminal(stderr) || o.nonInteractive {
		if o.mode == "" {
			return o, errors.New("choose --mode candidate, media, or production (or run on a terminal)")
		}
		return o, nil
	}
	p := newPrompter(bufio.NewReader(stdin), stderr)
	if err := p.overview(&o, suggestOut); err != nil {
		return o, err
	}
	return o, nil
}

func run(args []string, stdin, stdout, stderr *os.File) error {
	o, err := resolveOptions(args, stdin, stderr)
	if err != nil {
		return err
	}
	if err := validateResolved(&o); err != nil {
		return err
	}
	if err := preflight(o); err != nil {
		return err
	}
	if err := prepareRuntime(o, listRunningBuildUnits); err != nil {
		return err
	}
	stop, err := maybeServeFixture(o)
	if err != nil {
		return err
	}
	defer stop()
	r, err := startControllerRun(o, stdin, stdout, stderr)
	if err != nil {
		return err
	}
	code, err := r.wait()
	if err != nil {
		return err
	}
	if code != 0 {
		return exitError{code: code}
	}
	// File the built image where the installer was told to fetch it, so the
	// run ends with a bootable ISO and a served rootfs, not homework.
	return fileBuiltRootfs(o, stderr)
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
