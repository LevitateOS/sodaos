// Package muse confines pinned native Muse subscription enrollment and its file store.
package muse

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const Version = "1.4.0-R4161.1"

type Config struct {
	Binary  string `json:"binary"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Root    string `json:"root"`
}
type Provider struct{ config Config }

func New(c Config) (*Provider, error) {
	if err := validateConfig(c); err != nil {
		return nil, err
	}
	if err := checkBinary(c); err != nil {
		return nil, err
	}
	if err := checkVersion(c); err != nil {
		return nil, err
	}
	return &Provider{config: c}, nil
}

func checkBinary(c Config) error {
	f, err := os.Open(c.Binary)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(h, f)
	_ = f.Close()
	if err != nil {
		return err
	}
	if hex.EncodeToString(h.Sum(nil)) != c.SHA256 {
		return errors.New("muse digest mismatch")
	}
	return nil
}

func checkVersion(c Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Binary, "--version")
	cmd.Env = environment(c.Root)
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "Muse Code 1.4.0 ("+Version+")" {
		return errors.New("muse version mismatch")
	}
	return nil
}

// Enrollment inherits no user-selected key, provider endpoint or account state.
func environment(root string) []string {
	return []string{"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=" + root, "XDG_CONFIG_HOME=" + filepath.Join(root, "config"), "XDG_DATA_HOME=" + filepath.Join(root, "data"), "XDG_STATE_HOME=" + filepath.Join(root, "state"), "XDG_CACHE_HOME=" + filepath.Join(root, "cache"), "TMPDIR=" + root, "TBH_CREDENTIAL_BACKEND=file", "NO_COLOR=1"}
}

func validateConfig(c Config) error {
	if !filepath.IsAbs(c.Binary) || !filepath.IsAbs(c.Root) || c.Version != Version || len(c.SHA256) != 64 {
		return errors.New("invalid pinned Muse configuration")
	}
	info, err := os.Lstat(c.Root)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return errors.New("muse enrollment root must be private")
	}
	if err := privateTmpfs(c.Root); err != nil {
		return err
	}
	return nil
}
