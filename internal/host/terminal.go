package host

import (
	"github.com/levitateos/sodaos/internal/host/terminal"
)

type (
	// Terminal wire types are defined in host/terminal and re-exported here so
	// the Unix client surface stays in package host; web must not import the
	// privileged terminal executor.
	TerminalRequest = terminal.TerminalRequest
	TerminalState   = terminal.TerminalState
	TerminalFrame   = terminal.TerminalFrame
)

func ValidTerminalName(name string) bool { return terminal.ValidTerminalName(name) }
