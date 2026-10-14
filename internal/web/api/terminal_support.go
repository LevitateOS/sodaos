package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func terminalOperationError(w http.ResponseWriter, err error, fallback string) {
	var failure *host.TerminalError
	if errors.As(err, &failure) {
		messages := map[string]string{
			"terminal_support_mismatch":    "Terminal support needs an update. An appliance administrator must update this Project.",
			"terminal_support_unavailable": "Terminal support could not be inspected. Retry or ask an appliance administrator to inspect this Project.",
			"project_maintenance":          "Project maintenance is in progress. Managed access is temporarily unavailable.",
		}
		if message, ok := messages[failure.Code]; ok {
			auth.JSONError(w, 503, failure.Code, message)
			return
		}
	}
	auth.JSONError(w, 503, "terminal_unavailable", fallback)
}
