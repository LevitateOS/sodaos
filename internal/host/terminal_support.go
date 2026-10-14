package host

import (
	"context"
	"errors"
	"time"
)

// TerminalSupport describes current installed bytes separately from Project runtime.
// Only State is exposed to members; digests remain native operator diagnostics.
type TerminalSupport struct {
	State string `json:"state"`
}

func validSupportState(state string) bool {
	switch state {
	case "ready", "support_mismatch", "inspection_unavailable", "maintenance_in_progress":
		return true
	}
	return false
}

func (c *Client) TerminalSupport(ctx context.Context, project string) TerminalSupport {
	out := TerminalSupport{State: "inspection_unavailable"}
	if !projectID.MatchString(project) {
		return out
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var reply TerminalSupport
	if c.callLimit(ctx, "/terminal-support", map[string]string{"id": project}, &reply, 1024) == nil && validSupportState(reply.State) {
		return reply
	}
	return out
}

// TerminalError carries a bounded reason across the private terminal stream.
// It contains no stderr or arbitrary native exception text.
type TerminalError struct{ Code string }

func (e *TerminalError) Error() string { return e.Code }

func terminalFailure(reason string) error {
	switch reason {
	case "terminal_support_mismatch", "terminal_support_unavailable", "project_maintenance":
		return &TerminalError{Code: reason}
	}
	return errors.New("native terminal outcome unavailable")
}
