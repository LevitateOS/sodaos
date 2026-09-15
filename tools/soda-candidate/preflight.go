// preflight checks the operator-side facts the controller also enforces
// before anything privileged runs. Checks are fast, local, and each names
// its fix; the controller re-admits everything.
package main

import (
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

// prepareRuntime refuses a concurrent build and clears leftover session
// state. The runtime directory is per-attempt by definition: container
// runtime locks from a killed run must not poison the next one. Build caches
// live elsewhere and are never touched.
func prepareRuntime(workerConfigPath string, listUnits func() (string, error)) error {
	wp, err := readWorkerPaths(workerConfigPath)
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
	return clearRuntimeDir(wp.Runtime)
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
