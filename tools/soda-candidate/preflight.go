// preflight checks the operator-side facts the controller also enforces
// before anything privileged runs. Checks are fast, local, and each names
// its fix; the controller re-admits everything.
package main

import (
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// validOutLeaf mirrors the worker admission rule: the output leaf becomes
// the worker unit name, which allows lowercase letters, digits, and dashes.
func validOutLeaf(leaf string) bool {
	if len(leaf) < 1 || len(leaf) > 48 {
		return false
	}
	for _, r := range leaf {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// suggestOut names a timestamped leaf below the isolated releases parent.
// Lowercase by worker rule; uppercase timestamps are refused at dispatch.
func suggestOut() string {
	cwd, _ := os.Getwd()
	leaf := time.Now().UTC().Format("20060102t150405z")
	return filepath.Join(cwd, ".artifacts", "releases", "isolated", leaf)
}

// workerPaths carries the worker-config fields the wrapper must know before
// dispatch. It never admits the config; the controller owns that.
type workerPaths struct {
	OutputParent string
	Runtime      string
	Tools        string
}

func readWorkerPaths(path string) (workerPaths, error) {
	var wp workerPaths
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return wp, errors.New("worker config not found; rerun the setup script")
		}
		return wp, fmt.Errorf("cannot read worker config (run the wrapper itself under sudo): %w", err)
	}
	if err := json.Unmarshal(raw, &wp); err != nil {
		return wp, fmt.Errorf("worker config is not valid JSON: %w", err)
	}
	if wp.Runtime == "" {
		return wp, errors.New("worker config names no runtime directory; rerun the setup script")
	}
	return wp, nil
}

func listRunningBuildUnits() (string, error) {
	out, err := exec.Command("systemctl", "list-units", "--type=service", "--state=running", "--no-legend", "--plain", "soda-build-*").Output()
	if err != nil {
		return "", fmt.Errorf("cannot list worker units (run the wrapper itself under sudo): %w", err)
	}
	return string(out), nil
}

// execRunner runs host commands for environment probes; tests stub it.
var execRunner = func(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

// pinnedGoVersion reads the toolchain pin from the checkout manifest.
func pinnedGoVersion() (string, error) {
	raw, err := os.ReadFile("go.mod")
	if err != nil {
		return "", errors.New("run soda-candidate from the checkout root (~/Projects/sodaos)")
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if v, ok := strings.CutPrefix(line, "go "); ok {
			if f, _, _ := strings.Cut(strings.TrimSpace(v), " "); f != "" {
				return f, nil
			}
		}
	}
	return "", errors.New("go.mod pins no Go version")
}

// controllerGoVersion reports the toolchain that built a binary, which is
// what the worker compares. `go version` output matches this string.
func controllerGoVersion(path string) (string, error) {
	bi, err := buildinfo.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read controller build stamp: %w", err)
	}
	return bi.GoVersion, nil
}

// pinnedGoRoot is the fixed host GOROOT provisioned by
// setup-soda-candidate.sh; mirrors tools/soda-build pinnedGoRoot.
const pinnedGoRoot = "/usr/local/lib/soda/pinned-go"

// toolsGoContext reports the SELinux type of the provisioned Go so a
// mislabeled toolchain is caught here instead of dying at dispatch.
func toolsGoContext() (string, error) {
	out, err := execRunner("stat", "-c", "%C", filepath.Join(pinnedGoRoot, "bin", "go"))
	if err != nil {
		return "", fmt.Errorf("cannot inspect provisioned Go: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(out), ":")
	if len(parts) != 4 {
		return "", fmt.Errorf("unexpected SELinux context %q", strings.TrimSpace(out))
	}
	return parts[2], nil
}

// probeWorkerSetpgid forks inside the worker sandbox and takes ownership of
// the child's process group, exactly like every worker child spawn. A stale
// or missing SELinux rule fails here in milliseconds instead of mid-build.
func probeWorkerSetpgid() error {
	unit := fmt.Sprintf("soda-candidate-probe-%d", time.Now().UnixNano())
	script := "import os\n" +
		"pid = os.fork()\n" +
		"if pid == 0:\n" +
		"    os.setpgrp()\n" +
		"    os._exit(0)\n" +
		"_, status = os.waitpid(pid, 0)\n" +
		"raise SystemExit(os.waitstatus_to_exitcode(status))\n"
	out, err := execRunner("/usr/bin/systemd-run",
		"--wait", "--pipe", "--service-type=exec", "--unit="+unit,
		"--property=User=soda-build-worker", "--property=Group=soda-build-worker",
		"--property=WorkingDirectory=/tmp",
		"--property=ProtectHome=tmpfs", "--property=ProtectSystem=strict",
		"--property=PrivateTmp=yes", "--property=PrivateMounts=yes",
		"/usr/bin/python3", "-c", script)
	if err != nil {
		return fmt.Errorf("worker process groups denied (install the setup SELinux module): %v\n%s", err, out)
	}
	return nil
}

// checkWorkerEnvironment verifies the machine facts each debug round so far
// has uncovered, every one of which used to die inside the build instead.
func checkWorkerEnvironment(o options, wp workerPaths) error {
	pin, err := pinnedGoVersion()
	if err != nil {
		return err
	}
	if o.controller != "" {
		got, err := controllerGoVersion(o.controller)
		if err != nil {
			return err
		}
		if got != "go"+pin {
			return fmt.Errorf("controller reports %s, want go%s: rebuild with the pinned toolchain and re-admit", got, pin)
		}
	}
	label, err := toolsGoContext()
	if err != nil {
		return err
	}
	if label != "lib_t" {
		return fmt.Errorf("provisioned Go carries label %q, want lib_t; rerun bash scripts/setup-soda-candidate.sh from the repo root", label)
	}
	out, err := execRunner("/usr/bin/git", "config", "--system", "--get-all", "safe.directory")
	if err != nil || !strings.Contains(out, "/run/soda-build-source") {
		return errors.New("system git lacks the worker source exception; rerun the setup script")
	}
	return probeWorkerSetpgid()
}

// prepareRuntime refuses a concurrent build and clears leftover session
// state. The runtime directory is per-attempt by definition: container
// runtime locks from a killed run must not poison the next one. Build caches
// live elsewhere and are never touched.
func prepareRuntime(o options, listUnits func() (string, error)) error {
	wp, err := readWorkerPaths(o.workerConfig)
	if err != nil {
		return err
	}
	st, err := os.Stat(wp.Runtime)
	if err != nil || !st.IsDir() {
		return errors.New("worker runtime directory missing; rerun the setup script")
	}
	if err := refuseConcurrentBuild(listUnits); err != nil {
		return err
	}
	if err := clearRuntimeDir(wp.Runtime); err != nil {
		return err
	}
	return checkWorkerEnvironment(o, wp)
}

func refuseConcurrentBuild(listUnits func() (string, error)) error {
	units, err := listUnits()
	if err != nil {
		return err
	}
	for _, line := range strings.Split(units, "\n") {
		if name, _, _ := strings.Cut(strings.TrimSpace(line), " "); strings.HasPrefix(name, "soda-build-") {
			return fmt.Errorf("worker unit %s is already running; refusing a concurrent build", name)
		}
	}
	return nil
}

func clearRuntimeDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("cannot inspect worker runtime directory: %w", err)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("cannot clear leftover worker runtime state: %w", err)
		}
	}
	return nil
}

// controllerArgs translates answers into the admitted flags.
func controllerArgs(o options) []string {
	args := []string{
		"--worker-config", o.workerConfig, "--arch", o.arch, "--out", o.out,
		"--repository-prefix", o.repoPrefix,
	}
	if o.mode == "production" {
		args = append(args, "--qualification-config", o.qualConfig)
		if o.signConfig != "" {
			args = append(args, "--signing-config", o.signConfig)
		}
		return args
	}
	target := o.mode
	if target == "" {
		target = "media"
	}
	args = append(args, "--development", "--target", target)
	if o.compression != "" {
		args = append(args, "--media-compression", o.compression)
	}
	if o.mode == "media" {
		args = append(args, "--rootfs-base-url", o.rootfsURL)
	}
	return args
}
