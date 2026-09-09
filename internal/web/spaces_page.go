package web

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
)

//go:embed templates/spaces.html
var spacesHTML string
var spacesTemplate = template.Must(template.New("spaces").Parse(spacesHTML))

// This is Soda HTML, not a Forgejo context or native-session impersonation. No
// project metadata is embedded: the shared workspace uses the protected collection.
func (s *Server) spacesPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	// Xterm creates measured styles. Inline CSS is page-only; scripts remain fixed
	// same-origin modules without inline execution/eval or native Forgejo globals.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		http.Error(w, "Spaces does not accept navigation parameters.", 400)
		return
	}
	if config.BaseURL(s.Config.ForgejoURL) != nil || !strings.HasPrefix(s.Config.ForgejoURL, "https://") {
		http.Error(w, "Native origin is not configured.", 503)
		return
	}
	data := struct {
		Origin, Actor, Login, Message string
		Authorized                    bool
	}{Origin: s.Config.ForgejoURL, Message: "Connect through Forgejo to open your Spaces. Soda and native Forgejo sign-in are separate."}
	status := http.StatusOK
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		http.Error(w, "Ambiguous Soda cookies; sign in again.", 400)
		return
	}
	if err == nil {
		if s.Store == nil {
			http.Error(w, "Soda session storage is unavailable.", 503)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		v, err := s.Store.Session(ctx, cookie.Value)
		if err == nil {
			data.Actor = strconv.FormatInt(v.User.ID, 10)
			data.Login = v.User.Login
			grant, grantErr := s.userGrant(r.WithContext(ctx), v)
			if grantErr == nil && forgejo.HasScope(grant.Scopes, "read:user") && forgejo.HasScope(grant.Scopes, "read:repository") {
				actor, actorErr := s.Forgejo.Current(ctx, grant.Access)
				current, currentErr := s.Store.Session(ctx, cookie.Value)
				data.Authorized = actorErr == nil && actor.ID == v.User.ID && currentErr == nil && current.ContextID == v.ContextID && current.CSRF == v.CSRF
			}
			if data.Authorized {
				data.Message = "Loading authorized Spaces; opening this page never creates or starts a project."
			} else {
				status = 503
				data.Message = "Soda authorization is unavailable. No complete workspace was inferred; reconnect explicitly or try this page later."
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			status = 503
			data.Message = "Soda session storage is unavailable. No workspace was inferred."
		}
	}
	var body bytes.Buffer
	if spacesTemplate.Execute(&body, data) != nil {
		http.Error(w, "Could not render Spaces.", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}
