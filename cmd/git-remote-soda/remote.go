package main

import (
	"net/url"
	"strings"

	"github.com/levitateos/sodaos/internal/identity"
)

// remoteRequest accepts only the transport selected by Soda's native-origin
// Git rewrite. The broker independently checks the requested repository.
func remoteRequest(args []string, cwd, connection string) (identity.GitLaunchRequest, error) {
	var out identity.GitLaunchRequest
	if len(args) != 2 {
		return out, identity.ErrDenied
	}
	u, err := url.Parse(args[1])
	if err != nil || !validRemoteURL(u) {
		return out, identity.ErrDenied
	}
	repository := strings.TrimPrefix(u.Path, "/")
	if !strings.HasSuffix(repository, ".git") {
		return out, identity.ErrDenied
	}
	out = identity.GitLaunchRequest{ConnectionID: connection, CWD: cwd, Remote: args[0], Owner: u.Host, Repository: strings.TrimSuffix(repository, ".git")}
	return out, out.Validate()
}

func validRemoteURL(u *url.URL) bool {
	return u.Scheme == "soda" && u.User == nil && u.Opaque == "" && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.RawPath == "" && u.Port() == ""
}
