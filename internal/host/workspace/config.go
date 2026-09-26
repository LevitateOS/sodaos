// Package workspace executes disposable rootless OCI workspaces. It does not
// admit work, hold Forgejo authority or manage persistent human Projects.
package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

// Config is trusted operator configuration, never input from a coding agent.
// HarnessDirectory contains the complete pinned Codex package, not auth state.
type Config struct {
	Root             string   `json:"root"`
	HarnessDirectory string   `json:"harness_directory"`
	CredentialHome   string   `json:"credential_home"`
	Image            string   `json:"image"`
	ProxyImage       string   `json:"proxy_image"`
	HarnessVersion   string   `json:"harness_version"`
	Model            string   `json:"model"`
	AllowedDomains   []string `json:"allowed_domains"`
	CPUs             int      `json:"cpus"`
	MemoryBytes      int64    `json:"memory_bytes"`
	WritableBytes    int64    `json:"writable_bytes"`
	PIDs             int      `json:"pids"`
}

var hostname = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$`)

func pinned(image string) bool {
	digest, ok := strings.CutPrefix(image, "sha256:")
	return ok && factory.ValidDigest(digest)
}

func (c Config) Validate() error {
	if !pinned(c.Image) || !pinned(c.ProxyImage) {
		return errors.New("workspace and proxy images must be pinned")
	}
	if err := c.validateLimits(); err != nil {
		return err
	}
	if err := c.validateDomains(); err != nil {
		return err
	}
	if c.HarnessVersion == "" || c.Model == "" {
		return errors.New("a qualified Codex version and model are required")
	}
	return c.validatePaths()
}

func (c Config) validatePaths() error {
	for _, path := range []string{c.Root, c.HarnessDirectory, c.CredentialHome} {
		if !filepath.IsAbs(path) || strings.ContainsAny(path, ":\x00\r\n") {
			return errors.New("workspace paths must be explicit absolute paths")
		}
	}
	return nil
}

func (c Config) validateLimits() error {
	if c.CPUs < 1 || c.PIDs < 16 || c.PIDs > 4096 {
		return errors.New("invalid CPU or process limit")
	}
	if c.MemoryBytes < 256<<20 || c.WritableBytes < 80<<20 || c.WritableBytes > c.MemoryBytes/2 {
		return errors.New("tmpfs workspace must fit within its enforced memory limit")
	}
	return nil
}

func (c Config) validateDomains() error {
	if len(c.AllowedDomains) == 0 || len(c.AllowedDomains) > 64 {
		return errors.New("an explicit egress profile is required")
	}
	for _, name := range c.AllowedDomains {
		if len(name) > 253 || !hostname.MatchString(name) || strings.Contains(name, "..") {
			return errors.New("invalid allowed hostname")
		}
	}
	return nil
}

// Open admits existing private operator inputs; it never creates a login or
// replaces a credential directory. CLI-maintained auth state is reused in place.
func Open(c Config) (*Runtime, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if os.Geteuid() == 0 {
		return nil, errors.New("factory runtime must run as an unprivileged user")
	}
	if err := c.validateInputs(); err != nil {
		return nil, err
	}
	return &Runtime{Config: c, Exec: Native{}}, nil
}

func (c Config) validateInputs() error {
	for _, path := range []string{c.Root, c.CredentialHome} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
			return errors.New("factory state and credentials must be private directories")
		}
	}
	for _, file := range []string{"bin/codex", "bin/codex-code-mode-host", "codex-resources/bwrap", "codex-path/rg"} {
		info, err := os.Stat(filepath.Join(c.HarnessDirectory, file))
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			return errors.New("complete executable Codex package required")
		}
	}
	return c.validateCredentialState()
}

func (c Config) Bind(r *factory.Run) {
	r.Image, r.Harness, r.Model = c.Image, "codex-"+c.HarnessVersion, c.Model
	for _, kind := range []string{"network", "proxy", "workspace"} {
		r.Resources = append(r.Resources, factory.Resource{Kind: kind, Name: factory.ResourceName(r.ID, kind)})
	}
}

// Persistent conversation state cannot be shared between implementation and review.
func (c Config) validateCredentialState() error {
	entries, err := os.ReadDir(c.CredentialHome)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.Contains(name, ".sqlite") || name == "sessions" || name == "archived_sessions" || name == "memories" {
			return errors.New("credential home contains conversation state; enroll a dedicated clean factory login")
		}
	}
	return nil
}
