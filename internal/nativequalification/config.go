package nativequalification

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/acceptance"
	"github.com/levitateos/sodaos/internal/nativebuild"
	rd "github.com/levitateos/sodaos/internal/releasedelivery"
)

// Config is operator admission for the single same-base x86_64 scenario. It is
// never read from builder output and grants no account, trust or provider setup.
type Config struct {
	Executable, Work, Baseline, BaselineDisk, BaselineDiskSHA256                        string
	BaselineVariables, BaselineKey, BaselineKnownHosts, BaselineState, BaselinePassword string
	QEMU, Firmware, Variables, RegistryImage                                            string
}

func LoadConfig(path string) (Config, error) {
	var c Config
	if os.Geteuid() != 0 {
		return c, errors.New("protected qualification admission required")
	}
	if err := nativebuild.RequireNative("x86_64"); err != nil {
		return c, err
	}
	if err := rd.PrivateFile(path); err != nil {
		return c, err
	}
	if err := rd.ReadJSON(path, &c); err != nil {
		return c, err
	}
	if err := acceptance.TrustedExecutable(c.Executable); err != nil {
		return c, err
	}
	if !nativebuild.Digest(c.BaselineDiskSHA256) || !strings.HasPrefix(c.RegistryImage, "docker.io/library/registry@sha256:") || !nativebuild.Digest(strings.TrimPrefix(c.RegistryImage, "docker.io/library/registry@sha256:")) {
		return c, errors.New("admitted baseline hash and pinned upstream registry required")
	}
	for _, p := range []string{c.Baseline, c.BaselineDisk, c.BaselineVariables, c.BaselineKey, c.BaselineKnownHosts, c.BaselineState, c.BaselinePassword, c.QEMU, c.Firmware, c.Variables} {
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil || !filepath.IsAbs(p) || resolved != p || strings.ContainsAny(p, ":,\n\r\t %") {
			return c, errors.New("existing exact qualification input paths required")
		}
	}
	password, err := os.ReadFile(c.BaselinePassword)
	if err != nil || len(strings.TrimSpace(string(password))) == 0 {
		return c, errors.New("baseline console credential unavailable")
	}
	if err := nativebuild.PrivateDestination(c.Work); err != nil {
		return c, err
	}
	return c, nil
}

func writeNewJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return nativebuild.WriteNew(path, append(b, '\n'), 0600)
}
