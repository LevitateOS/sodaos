package web

import (
	"context"
	"net/http"

	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/webapp"
	"github.com/levitateos/sodaos/internal/webauth"
)

// Test-facing aliases keep package-web integration tests compiling against the facade.
const (
	sessionCookie      = webauth.SessionCookie
	oauthCookie        = webauth.OAuthCookie
	expectedUserHeader = webauth.ExpectedUserHeader
	apiBodyLimit       = webauth.APIBodyLimit
)

func requestCookie(r *http.Request, name string) (*http.Cookie, error) {
	return webauth.RequestCookie(r, name)
}

func positiveID(value string) (int64, bool) { return webauth.PositiveID(value) }

func jsonError(w http.ResponseWriter, status int, code, message string) {
	webauth.JSONError(w, status, code, message)
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	webauth.JSONResponse(w, status, value)
}

type terminalView = webapp.TerminalView
type spacesView = webapp.SpacesView
type terminalPeer = webapp.TerminalPeer
type environmentView = webapp.EnvironmentView

func environmentDTO(p store.Project) environmentView { return webapp.EnvironmentDTO(p) }

func (s *Server) nativeOAuthReturn(login store.OAuthLogin) string {
	return s.Auth.NativeOAuthReturn(login)
}

func (s *Server) apiProtected(next func(http.ResponseWriter, *http.Request, store.Session), methods ...string) http.HandlerFunc {
	return s.Auth.Protected(next, methods...)
}

func (s *Server) requireCurrentSession(ctx context.Context, token string, original store.Session) error {
	return s.Auth.RequireCurrentSession(ctx, token, original)
}

func (s *Server) userGrant(r *http.Request, v store.Session) (store.Grant, error) {
	return s.Auth.UserGrant(r, v)
}

var BrowserTerminalID = webapp.BrowserTerminalID

func TerminalCreationScope(v store.Session) string {
	return webapp.TerminalCreationScope(v)
}
