//go:build linux

package workspace

import (
	"errors"
	"os"
	"path/filepath"
)

func (w *Runtime) gitArguments(args []string) ([]string, error) {
	if w.Config.GitSocket == "" {
		return args, nil
	}
	if err := validateMuseInterface(w.Config.GitSocket); err != nil {
		return nil, err
	}
	tool := filepath.Join(w.Config.GitToolsDirectory, "git-remote-soda")
	info, err := os.Lstat(tool)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return nil, errors.New("public Git remote helper required")
	}
	return append(args, "--volume", tool+":/usr/local/bin/git-remote-soda:ro,z", "--volume", filepath.Dir(w.Config.GitSocket)+":/run/soda-git-interface:ro,z"), nil
}
