// preflight holds the wrapper-side answer checks: output naming and
// controller argument translation. Worker admission, environment checks,
// and runtime state belong to the controller; the wrapper never probes or
// mutates them.
package main

import (
	"os"
	"path/filepath"
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

// controllerArgs translates answers into the admitted flags.
func controllerArgs(o options) []string {
	args := []string{
		"--worker-config", o.workerConfig, "--arch", o.arch, "--out", o.out,
		"--repository-prefix", o.repoPrefix,
	}
	target := o.mode
	if target == "" {
		target = "media"
	}
	args = append(args, "--development", "--target", target)
	if o.compression != "" {
		args = append(args, "--media-compression", o.compression)
	}
	if target == "media" {
		args = append(args, "--rootfs-base-url", o.rootfsURL)
	}
	return args
}
