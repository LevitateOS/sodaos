package web

import (
	"context"
	"net/http"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/web/api"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Extension wires the separate Forgejo extension process. The service socket is
// a dedicated private listener, not the dashboard's public browser listener.
func Extension(socket string) (extensions.Application, func()) {
	handler, closeTransport := api.Extension(socket)
	return extensions.Application{HTTP: handler, Policies: map[string]extensions.PolicyHandler{
		extensions.PolicyForgejoUsername: forgejoUsernamePolicy,
	}}, closeTransport
}

func forgejoUsernamePolicy(_ context.Context, request extensions.PolicyRequest) (extensions.PolicyDecision, error) {
	switch request.Operation {
	case "create":
		if request.UserID != "" {
			return extensions.PolicyDecision{ReasonCode: "invalid_request"}, nil
		}
	case "rename":
		if _, valid := auth.PositiveID(request.UserID); !valid {
			return extensions.PolicyDecision{ReasonCode: "invalid_request"}, nil
		}
	default:
		return extensions.PolicyDecision{ReasonCode: "invalid_request"}, nil
	}
	if !project.ValidLogin(request.Username) || request.Username == "root" {
		return extensions.PolicyDecision{ReasonCode: "unsupported_linux_login"}, nil
	}
	return extensions.PolicyDecision{Allowed: true}, nil
}

// ExtensionHandler is mounted only on the dedicated private service socket.
func (s *Server) ExtensionHandler() http.Handler { return s.API.ExtensionHandler() }
