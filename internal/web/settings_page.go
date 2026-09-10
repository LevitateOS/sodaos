package web

import (
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
	"github.com/levitateos/sodaos/internal/store"
)

//go:embed templates/runners.html
var runnersHTML string
var runnersTemplate = template.Must(template.New("runners").Parse(runnersHTML))

func (s *Server) runnersPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		http.Error(w, "Settings do not accept navigation parameters.", 400)
		return
	}
	if config.BaseURL(s.Config.ForgejoURL) != nil || !strings.HasPrefix(s.Config.ForgejoURL, "https://") {
		http.Error(w, "Native origin is not configured.", 503)
		return
	}
	data := struct {
		Origin, Actor, Login, Message string
		Authorized                    bool
	}{Origin: s.Config.ForgejoURL, Message: "Connect through Forgejo as the configured Soda operator to manage local runners."}
	status := http.StatusOK
	cookie, err := requestCookie(r, sessionCookie)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		http.Error(w, "Ambiguous Soda cookies.", 400)
		return
	}
	if err == nil {
		if s.Store == nil {
			http.Error(w, "Soda storage unavailable.", 503)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		v, err := s.Store.Session(ctx, cookie.Value)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Soda storage unavailable.", 503)
			return
		}
		if err == nil {
			if err := s.operatorAuthorization(r, v); err != nil {
				var providerFailure *forgejo.HTTPError
				status = http.StatusServiceUnavailable
				data.Message = "Soda authorization is unavailable. Reconnect explicitly or try again later."
				if errors.Is(err, errOperatorRequired) {
					status = http.StatusForbidden
					data.Message = "This Soda session is not the configured Soda operator. Forgejo site administration alone does not grant appliance access."
				} else if errors.Is(err, store.ErrGrantUnavailable) || errors.Is(err, store.ErrGrantKey) || errors.Is(err, errProviderIdentity) {
					status = http.StatusUnauthorized
				} else if errors.Is(err, errRepositoryConsent) {
					status = http.StatusForbidden
				} else if errors.As(err, &providerFailure) && (providerFailure.Status == http.StatusUnauthorized || providerFailure.Status == http.StatusForbidden) {
					status = providerFailure.Status
				}
			} else {
				data.Actor = strconv.FormatInt(v.User.ID, 10)
				data.Login = v.User.Login
				data.Authorized = true
			}
		}
	}
	if !data.Authorized && status == http.StatusOK {
		s.nativePageEntry(w, r, store.OAuthLogin{SettingsReturn: "runners"})
		return
	}
	if writePageTemplate(w, runnersTemplate, data, status) != nil {
		http.Error(w, "Cannot render settings.", 500)
		return
	}
}
