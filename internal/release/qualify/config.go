package qualify

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/release/build"
	"github.com/levitateos/sodaos/internal/release/deliver"
)

// Config is operator admission for the single same-base x86_64 scenario. It is
// never read from builder output and grants no account, trust or provider setup.
type Config struct {
	Executable, Work, Baseline, BaselineDisk, BaselineDiskSHA256                        string
	BaselineVariables, BaselineKey, BaselineKnownHosts, BaselineState, BaselinePassword string
	QEMU, Firmware, Variables, RegistryImage                                            string
}

func pinnedRegistry(image string) bool {
	return strings.HasPrefix(image, "docker.io/library/registry@sha256:") && build.Digest(strings.TrimPrefix(image, "docker.io/library/registry@sha256:"))
}

func exactQualificationPath(p string) bool {
	resolved, err := filepath.EvalSymlinks(p)
	return err == nil && filepath.IsAbs(p) && resolved == p && !strings.ContainsAny(p, ":,\n\r\t %")
}

func requireExactPaths(paths []string) error {
	for _, p := range paths {
		if !exactQualificationPath(p) {
			return errors.New("existing exact qualification input paths required")
		}
	}
	return nil
}

func requireBaselinePassword(path string) error {
	password, err := os.ReadFile(path)
	if err != nil || len(strings.TrimSpace(string(password))) == 0 {
		return errors.New("baseline console credential unavailable")
	}
	return nil
}

func admitConfig(path string) (Config, error) {
	var c Config
	if os.Geteuid() != 0 {
		return c, errors.New("protected qualification admission required")
	}
	if err := build.RequireNative("x86_64"); err != nil {
		return c, err
	}
	if err := deliver.PrivateFile(path); err != nil {
		return c, err
	}
	if err := deliver.ReadJSON(path, &c); err != nil {
		return c, err
	}
	if err := acceptance.TrustedExecutable(c.Executable); err != nil {
		return c, err
	}
	return c, nil
}

func LoadConfig(path string) (Config, error) {
	c, err := admitConfig(path)
	if err != nil {
		return c, err
	}
	if !build.Digest(c.BaselineDiskSHA256) || !pinnedRegistry(c.RegistryImage) {
		return c, errors.New("admitted baseline hash and pinned upstream registry required")
	}
	if err := requireExactPaths([]string{c.Baseline, c.BaselineDisk, c.BaselineVariables, c.BaselineKey, c.BaselineKnownHosts, c.BaselineState, c.BaselinePassword, c.QEMU, c.Firmware, c.Variables}); err != nil {
		return c, err
	}
	if err := requireBaselinePassword(c.BaselinePassword); err != nil {
		return c, err
	}
	if err := build.PrivateDestination(c.Work); err != nil {
		return c, err
	}
	return c, nil
}

func writeNewJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return build.WriteNew(path, append(b, '\n'), 0o600)
}
