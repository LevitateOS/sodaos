package host

import (
	"net/http"

	"github.com/levitateos/sodaos/internal/host/terminal"
)

type (
	TerminalRequest = hostterminal.TerminalRequest
	TerminalState   = hostterminal.TerminalState
	TerminalFrame   = hostterminal.TerminalFrame
)

func ValidTerminalName(name string) bool { return hostterminal.ValidTerminalName(name) }

// Terminal implements hostterminal attach launching for the native executor.
func (Native) Terminal(container string, in hostterminal.TerminalRequest) (hostterminal.Process, error) {
	return hostterminal.AttachNative(container, in)
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
