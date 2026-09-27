package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func (c Config) validateMusePaths() error {
	if c.MuseToolsDirectory == "" && c.MuseSocket == "" && c.MuseCredentialRoot == "" {
		return nil
	}
	for _, path := range []string{c.MuseToolsDirectory, c.MuseSocket, c.MuseCredentialRoot} {
		if !filepath.IsAbs(path) || strings.ContainsAny(path, ":\x00\r\n") {
			return errors.New("muse tools, launch socket and credential root require absolute paths")
		}
	}
	return nil
}

// Remove only an empty run root; uncertain active credential directories remain.
func (w *Runtime) cleanupMuse(runID string) error {
	if w.Config.MuseCredentialRoot == "" {
		return nil
	}
	err := os.Remove(filepath.Join(w.Config.MuseCredentialRoot, runID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func validateMuseInterface(socket string) error {
	entries, err := os.ReadDir(filepath.Dir(socket))
	if err != nil || len(entries) != 1 || entries[0].Name() != "launch.sock" || filepath.Base(socket) != "launch.sock" {
		return errors.New("worker interface directory must contain only the launch socket")
	}
	info, err := os.Lstat(socket)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		return errors.New("worker launch socket required")
	}
	return nil
}
