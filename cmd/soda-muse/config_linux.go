//go:build linux

package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// copyConfig always runs as the provisioned account, including symlink reads.
// Native provider credentials are excluded; custody supplies the single auth file.
func copyConfig(source, destination string) error {
	if !filepath.IsAbs(source) || !filepath.IsAbs(destination) {
		return errors.New("absolute config paths required")
	}
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		return copyConfigEntry(source, destination, path, entry, err)
	})
}

func copyConfigEntry(source, destination, path string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(source, path)
	if err != nil {
		return err
	}
	if entry.Name() == "auth.json" {
		return nil
	}
	target := filepath.Join(destination, relative)
	if entry.IsDir() {
		return os.MkdirAll(target, 0o700)
	}
	return copyConfigFile(path, target)
}

func copyConfigFile(path, target string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if info.Size() > 1<<20 {
		return errors.New("muse config file exceeds private view limit")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(target, body, 0o600)
}

// readConfig emits the upstream release's saved settings and trust records only.
// It executes as the invoking account, never as a privileged config reader.
func readConfig(source string) error {
	if !filepath.IsAbs(source) {
		return errors.New("absolute config path required")
	}
	view := map[string][]byte{}
	for _, name := range []string{"settings.json", "trust.json"} {
		file, err := os.Open(filepath.Join(source, name))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		body, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
		_ = file.Close()
		if err != nil {
			return err
		}
		if len(body) > 1<<20 {
			return errors.New("muse config file exceeds private view limit")
		}
		view[name] = body
	}
	return json.NewEncoder(os.Stdout).Encode(view)
}
