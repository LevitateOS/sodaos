package identity

import (
	"path/filepath"
	"strings"
)

const MuseLaunchSocket = "/run/soda-muse-interface/launch.sock"

type (
	NestedRegistration struct {
		ChildID        string `json:"child_id"`
		ActorID        int64  `json:"actor_id,string"`
		RegistrationID string `json:"registration_id"`
		Muse           bool   `json:"muse"`
	}
	// LaunchRequest carries invocation preferences, never caller authority or credentials.
	LaunchRequest struct {
		Home         string              `json:"home,omitempty"`
		Register     *NestedRegistration `json:"register,omitempty"`
		ConfigHome   string              `json:"config_home,omitempty"`
		Term         string              `json:"term,omitempty"`
		ConnectionID string              `json:"connection_id"`
		CWD          string              `json:"cwd"`
		Args         []string            `json:"args"`
		TTY          bool                `json:"tty"`
		Cols         uint16              `json:"cols"`
		Rows         uint16              `json:"rows"`
	}
)

func (r LaunchRequest) Validate() error {
	if r.Register != nil {
		return r.registrationValid()
	}
	if !launchAbsolutePath(r.CWD, false) || !r.configPathsValid() {
		return ErrDenied
	}
	if !r.launchSizesValid() {
		return ErrDenied
	}
	if r.TTY && (r.Cols == 0 || r.Rows == 0) {
		return ErrDenied
	}
	return launchArgumentsValid(r.Args)
}

func (r LaunchRequest) registrationValid() error {
	if r.CWD != "" || len(r.Args) != 0 || r.ConnectionID != "" || r.TTY || r.Register.ActorID <= 0 || !r.Register.Muse {
		return ErrDenied
	}
	return nil
}

func launchAbsolutePath(value string, optional bool) bool {
	if optional && value == "" {
		return true
	}
	return filepath.IsAbs(value) && launchText(value, 4096)
}

func launchText(value string, limit int) bool {
	return len(value) <= limit && !strings.ContainsRune(value, 0)
}

func launchArgumentsValid(args []string) error {
	size := 0
	for _, arg := range args {
		size += len(arg)
		if !launchText(arg, 32768) {
			return ErrDenied
		}
	}
	if size > 32768 {
		return ErrDenied
	}
	return nil
}

// LaunchControl is the bounded live shell control protocol.
type (
	LaunchControl struct {
		Signal int    `json:"signal,omitempty"`
		Cols   uint16 `json:"cols,omitempty"`
		Rows   uint16 `json:"rows,omitempty"`
	}
	LaunchExit struct {
		Code  int    `json:"code"`
		Error string `json:"error,omitempty"`
	}
)

// MuseArguments preserves ordinary invocation bytes under the subscription provider.
func MuseArguments(args []string) ([]string, error) {
	for _, arg := range args {
		if museAuthOverride(arg) {
			return nil, ErrDenied
		}
	}
	positional := musePositional(args)
	if positional == "auth" || positional == "login" || positional == "logout" {
		return nil, ErrDenied
	}
	return museProviderArguments(args), nil
}

// The pinned native dispatcher recognizes subcommands only in argv[0].
func museProviderArguments(args []string) []string {
	if len(args) == 0 {
		return []string{"--provider", "meta"}
	}
	switch args[0] {
	case "exec", "resume", "serve":
		return append([]string{args[0], "--provider", "meta"}, args[1:]...)
	case "config", "export", "trace", "skills", "sandbox", "schema", "session-message", "mcp", "init":
		return append([]string{}, args...)
	}
	return append([]string{"--provider", "meta"}, args...)
}

func museAuthOverride(arg string) bool {
	name, _, _ := strings.Cut(arg, "=")
	return name == "--provider" || name == "--base-url"
}

func musePositional(args []string) string {
	skip := false
	for _, arg := range args {
		if skip {
			skip = false
			continue
		}
		if strings.HasPrefix(arg, "-") {
			skip = museValueFlag(arg)
			continue
		}
		return arg
	}
	return ""
}

func museValueFlag(flag string) bool {
	switch flag {
	case "--model", "--reasoning-effort", "--agents", "--preset", "--image", "--workspace", "--worktree-base", "--worktree-existing", "--approval-mode", "--permission-profile", "--approval-judge", "--sandbox-network", "--echo-delay-ms":
		return true
	}
	return false
}

func (r LaunchRequest) configPathsValid() bool {
	return launchAbsolutePath(r.ConfigHome, true) && launchAbsolutePath(r.Home, true)
}

func (r LaunchRequest) launchSizesValid() bool {
	return launchText(r.Term, 128) && len(r.ConnectionID) <= 128 && len(r.Args) <= 256
}
