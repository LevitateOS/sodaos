package identity

import "strings"

const GitLaunchSocket = "/run/soda-git-interface/launch.sock"

// GitLaunchRequest describes a remote-helper invocation. Its four descriptors
// carry stdin, stdout, stderr and a guest loopback TCP listener, respectively.
// Actor, project and native execution identity are derived by the launcher.
type GitLaunchRequest struct {
	ConnectionID string `json:"connection_id,omitempty"`
	CWD          string `json:"cwd"`
	Remote       string `json:"remote"`
	Owner        string `json:"owner"`
	Repository   string `json:"repository"`
}

func (r GitLaunchRequest) Validate() error {
	if !launchAbsolutePath(r.CWD, false) || !launchText(r.ConnectionID, 128) || !gitLaunchPart(r.Remote) || !gitLaunchPart(r.Owner) || !gitLaunchPart(r.Repository) {
		return ErrDenied
	}
	return nil
}

func gitLaunchPart(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.HasPrefix(value, "-") && len(value) <= 255 && !strings.ContainsAny(value, "/\\\x00\r\n")
}
