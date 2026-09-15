package terminal

import (
	"encoding/base64"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	FrameLimit    = 131072 // bounded 64-row metadata; IO payload bounds remain smaller
	terminalLimit = 64
)

// TerminalRequest is private root:soda helper input. The web layer must resolve
// membership/login and authorize the real actor before using this operation.
type TerminalRequest struct {
	Action   string `json:"action"`
	ID       string `json:"id"`
	Project  string `json:"project"`
	Login    string `json:"login"`
	Identity int64  `json:"identity"`
	Cols     int    `json:"cols"`
	Rows     int    `json:"rows"`
	Expires  int64  `json:"expires"` // request/attachment deadline, never shell lifetime
	Name     string `json:"name"`
	Scope    string `json:"scope"` // opaque creation-context digest, never browser authority
}

type TerminalState struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	Ready     bool   `json:"ready"`
	Attached  bool   `json:"attached"`
	State     string `json:"state"`
}

type TerminalFrame struct {
	Type      string           `json:"type"`
	Data      string           `json:"data,omitempty"`
	Cols      int              `json:"cols,omitempty"`
	Rows      int              `json:"rows,omitempty"`
	Reason    string           `json:"reason,omitempty"`
	Terminals *[]TerminalState `json:"terminals,omitempty"`
}

func ValidTerminalName(name string) bool {
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) > 80 {
		return false
	}
	for _, c := range name {
		if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
			return false
		}
	}
	return true
}

func terminalDimensions(cols, rows int) bool {
	return cols >= 2 && cols <= 500 && rows >= 2 && rows <= 300
}

func validTerminalActor(in TerminalRequest) bool {
	return projectID.MatchString(in.Project) && loginName.MatchString(in.Login) && in.Login != "root" && in.Identity > 0 && ValidTerminalName(in.Name)
}

func validTerminalWindow(in TerminalRequest, now time.Time) bool {
	return in.Expires > now.Unix() && in.Expires <= now.Add(12*time.Hour).Unix()
}

func validTerminalScope(action, scope string) bool {
	creating := action == "reserve" || action == "create"
	if creating {
		return len(scope) == 64 && containerID.MatchString(scope)
	}
	return scope == ""
}

func validListRequest(in TerminalRequest) bool {
	return in.ID == "" && in.Cols == 0 && in.Rows == 0 && in.Name == ""
}

func validSizedTerminalAction(in TerminalRequest) bool {
	return terminalDimensions(in.Cols, in.Rows) && (in.Action != "attach" || in.Name == "")
}

func validIdleTerminalAction(in TerminalRequest) bool {
	return in.Cols == 0 && in.Rows == 0 && (in.Action == "rename" || in.Name == "")
}

func validTerminalAction(in TerminalRequest) bool {
	if in.Action == "list" {
		return validListRequest(in)
	}
	if !terminalID.MatchString(in.ID) {
		return false
	}
	switch in.Action {
	case "reserve", "create", "attach":
		return validSizedTerminalAction(in)
	case "inspect", "end", "rename":
		return validIdleTerminalAction(in)
	}
	return false
}

func (in TerminalRequest) Valid(now time.Time) bool {
	return validTerminalActor(in) && validTerminalWindow(in, now) && validTerminalScope(in.Action, in.Scope) && validTerminalAction(in)
}

func validTypedInput(f TerminalFrame) bool {
	data, err := base64.StdEncoding.Strict().DecodeString(f.Data)
	return err == nil && !strings.ContainsAny(f.Data, "\r\n") && len(data) > 0 && len(data) <= 16384 && f.Cols == 0 && f.Rows == 0
}

func validResizeInput(f TerminalFrame) bool {
	return f.Data == "" && terminalDimensions(f.Cols, f.Rows)
}

func validIdleInput(f TerminalFrame) bool {
	return f.Data == "" && f.Cols == 0 && f.Rows == 0
}

func (f TerminalFrame) InputValid() bool {
	if f.Reason != "" || f.Terminals != nil {
		return false
	}
	switch f.Type {
	case "input":
		return validTypedInput(f)
	case "resize":
		return validResizeInput(f)
	case "heartbeat", "close":
		return validIdleInput(f)
	}
	return false
}

func validOutputShape(f TerminalFrame) bool {
	if f.Cols != 0 || f.Rows != 0 {
		return false
	}
	return f.Type == "metadata" || f.Terminals == nil
}

func validTerminalState(state string) bool {
	switch state {
	case "ready", "opening", "ending", "ended":
		return true
	default:
		return false
	}
}

func validTerminalItemFlags(ready, attached bool, state string) bool {
	if ready != (state == "ready") {
		return false
	}
	return !attached || ready
}

func validTerminalItem(item TerminalState, seen map[string]bool) bool {
	if !terminalID.MatchString(item.ID) || seen[item.ID] || !ValidTerminalName(item.Name) {
		return false
	}
	if item.CreatedAt <= 0 || item.CreatedAt > 9007199254740991 {
		return false
	}
	return validTerminalItemFlags(item.Ready, item.Attached, item.State) && validTerminalState(item.State)
}

func validMetadataOutput(f TerminalFrame) bool {
	if f.Data != "" || f.Reason != "" || f.Terminals == nil || len(*f.Terminals) > terminalLimit {
		return false
	}
	seen := make(map[string]bool)
	for _, item := range *f.Terminals {
		if !validTerminalItem(item, seen) {
			return false
		}
		seen[item.ID] = true
	}
	return true
}

func validOutputData(f TerminalFrame) bool {
	if f.Reason != "" {
		return false
	}
	data, err := base64.StdEncoding.Strict().DecodeString(f.Data)
	return err == nil && len(data) > 0 && len(data) <= 4096
}

func validClosedReason(reason string) bool {
	switch reason {
	case "disconnected", "expired", "exited", "launch_failed", "stream_failed", "cleanup_unconfirmed":
		return true
	default:
		return false
	}
}

func validClosedOutput(f TerminalFrame) bool {
	return f.Data == "" && validClosedReason(f.Reason)
}

func (f TerminalFrame) OutputValid() bool {
	if !validOutputShape(f) {
		return false
	}
	switch f.Type {
	case "metadata":
		return validMetadataOutput(f)
	case "ready":
		return f.Data == "" && f.Reason == ""
	case "output":
		return validOutputData(f)
	case "closed":
		return validClosedOutput(f)
	default:
		return false
	}
}

var (
	projectID   = regexp.MustCompile(`^p[0-9a-f]{24}$`)
	loginName   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,30}$`)
	containerID = regexp.MustCompile(`^[0-9a-f]{64}$`)
	terminalID  = regexp.MustCompile(`^[0-9a-f]{32}$`)
)
