package workspace

import (
	"context"
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
)

func (c Config) validateGitPaths() error {
	if c.GitSocket == "" && c.GitToolsDirectory == "" {
		return nil
	}
	for _, path := range []string{c.GitSocket, c.GitToolsDirectory} {
		if !filepath.IsAbs(path) || strings.ContainsAny(path, ":\x00\r\n") {
			return errors.New("git tools and launch socket require absolute paths")
		}
	}
	if c.GitSocket == c.MuseSocket || filepath.Dir(c.GitSocket) == filepath.Dir(c.MuseSocket) {
		return errors.New("git and Muse launch interfaces must be separate")
	}
	return nil
}

// gitRemote accepts the one repository selected by trusted factory admission.
func gitRemote(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || !validFactoryRemoteURL(u) {
		return "", errors.New("exact HTTPS factory Git remote required")
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 2 || !strings.HasSuffix(parts[1], ".git") {
		return "", errors.New("exact HTTPS factory Git remote required")
	}
	name := strings.TrimSuffix(parts[1], ".git")
	request := identity.GitLaunchRequest{CWD: "/workspace/repo", Remote: "origin", Owner: parts[0], Repository: name}
	if request.Validate() != nil {
		return "", errors.New("invalid factory Git repository")
	}
	return "soda://" + parts[0] + "/" + parts[1], nil
}

func validFactoryRemoteURL(u *url.URL) bool {
	return u.Scheme == "https" && u.Host != "" && u.User == nil && u.RawPath == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.Opaque == ""
}

func (w *Runtime) configureGit(ctx context.Context, id string) error {
	if w.Config.GitSocket == "" {
		return nil
	}
	local, err := gitRemote(w.GitRemote)
	if err != nil {
		return err
	}
	if _, err := w.Exec.Run(ctx, nil, "podman", "exec", id, "git", "config", "--global", "url."+local+".insteadOf", w.GitRemote); err != nil {
		return err
	}
	_, err = w.Exec.Run(ctx, nil, "podman", "exec", id, "git", "-C", "/workspace/repo", "remote", "set-url", "origin", w.GitRemote)
	return err
}
