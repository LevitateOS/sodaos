package host

import (
	"net/http"

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

// Terminal implements terminal attach launching for the native executor.
func (Native) Terminal(container string, in terminal.TerminalRequest) (terminal.Process, error) {
	return terminal.AttachNative(container, in)
}

func (d *Daemon) terminalHandler(w http.ResponseWriter, r *http.Request) {
	if d.Terminal == nil {
		http.Error(w, "terminal unavailable", http.StatusServiceUnavailable)
		return
	}
	d.Terminal.Handler(w, r)
}

// CloseTerminals closes both pending and live streams; HTTP Shutdown alone does
// not close hijacked WebSockets. No native mutation/global user process lock is held.
func (d *Daemon) CloseTerminals() {
	if d.Terminal != nil {
		d.Terminal.Close()
	}
}
