//go:build linux

package workspace

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/factory"
	"golang.org/x/sys/unix"
)

func (w *Runtime) museArguments(args []string, runID string) ([]string, error) {
	if w.Config.MuseSocket == "" {
		return args, nil
	}
	if err := validateMuseInterface(w.Config.MuseSocket); err != nil {
		return nil, err
	}
	if !factory.ValidID(runID) {
		return nil, errors.New("invalid Muse workspace run")
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(w.Config.MuseCredentialRoot, &fs); err != nil {
		return nil, err
	}
	if fs.Type != unix.TMPFS_MAGIC {
		return nil, errors.New("muse worker credentials require tmpfs")
	}
	root := filepath.Join(w.Config.MuseCredentialRoot, runID)
	if err := os.Mkdir(root, 0o700); err != nil {
		return nil, err
	}
	for source, destination := range map[string]string{
		"muse":        "/usr/local/bin/muse",
		"muse-native": "/usr/local/libexec/soda/muse",
	} {
		args = append(args, "--volume", filepath.Join(w.Config.MuseToolsDirectory, source)+":"+destination+":ro,z")
	}
	return append(args, "--volume", filepath.Dir(w.Config.MuseSocket)+":/run/soda-muse-interface:ro,z", "--volume", root+":/run/soda-muse/credentials:ro,z"), nil
}
