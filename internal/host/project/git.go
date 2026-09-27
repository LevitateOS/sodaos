package project

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// configureGit selects only the appliance's configured native Forgejo origin.
// The slash confines Git's prefix rewrite to that origin's repository paths.
func (r *Runtime) configureGit(ctx context.Context, id string) error {
	if r.Config.GitSocket == "" {
		return nil
	}
	origin, err := gitOrigin(r.Config.ForgejoURL)
	if err != nil {
		return err
	}
	_, err = r.podman(ctx, nil, "exec", "soda-"+id, "/usr/bin/git", "config", "--system", "url.soda://.insteadOf", origin)
	return err
}

func gitOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || !rootGitOrigin(u) || strings.ContainsAny(raw, "\r\n\x00") {
		return "", errors.New("native git requires the configured HTTPS Forgejo origin")
	}
	return strings.TrimSuffix(raw, "/") + "/", nil
}

func rootGitOrigin(u *url.URL) bool {
	return u.RawPath == "" && u.Opaque == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && (u.Path == "" || u.Path == "/")
}
