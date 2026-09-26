// Package codex uses the pinned Codex app-server managed enrollment protocol.
package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

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
	info, err := os.Lstat(c.Root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("enrollment root must be private")
	}
	if err = privateTmpfs(c.Root); err != nil {
		return nil, err
	}
	if err = checkBinary(c); err != nil {
		return nil, err
	}
	p := &Provider{config: c}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Binary, "--version")
	cmd.Env = p.environment(c.Root)
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "codex-cli "+c.Version {
		return nil, errors.New("codex version mismatch")
	}
	return p, nil
}

func (p *Provider) environment(root string) []string {
	out := []string{}
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "OPENAI") || strings.HasPrefix(name, "CODEX") || strings.HasPrefix(name, "CHATGPT") || strings.HasPrefix(name, "AWS") || name == "TMPDIR" || name == "XDG_CACHE_HOME" {
			continue
		}
		out = append(out, entry)
	}
	return append(out, "CODEX_HOME="+root, "TMPDIR="+root, "XDG_CACHE_HOME="+filepath.Join(root, "cache"))
}

func validateConfig(c Config) error {
	if !filepath.IsAbs(c.Binary) || !filepath.IsAbs(c.Root) || c.Version != "0.153.4" || len(c.SHA256) != 64 {
		return errors.New("invalid pinned Codex configuration")
	}
	return nil
}

func checkBinary(c Config) error {
	data, err := os.ReadFile(c.Binary)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != c.SHA256 {
		return errors.New("codex digest mismatch")
	}
	return nil
}
