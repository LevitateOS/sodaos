package web

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
)

//go:embed templates/repository-spaces.html
var repositorySpacesHTML string
var repositorySpacesTemplate = template.Must(template.New("repository-spaces").Parse(repositorySpacesHTML))

func (s *Server) repositorySpacesPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	id, ok := positiveID(r.PathValue("repositoryID"))
	if !ok || r.URL.RawQuery != "" || r.URL.ForceQuery {
		http.Error(w, "Invalid repository settings destination.", 400)
		return
	}
	if config.BaseURL(s.Config.ForgejoURL) != nil || !strings.HasPrefix(s.Config.ForgejoURL, "https://") {
		http.Error(w, "Native origin unavailable.", 503)
		return
	}
	data := struct {
		Origin, RepositoryID, Actor, Login, Repository, RepositoryURL string
		Authorized                                                    bool
	}{Origin: s.Config.ForgejoURL, RepositoryID: strconv.FormatInt(id, 10)}
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
			access, err := s.visibleRepository(r, v, id)
			if err != nil {
				providerError(w, err)
				return
			}
			if err := s.requireCurrentSession(ctx, cookie.Value, v); err != nil {
				http.Error(w, "Soda context changed; reconnect.", 401)
				return
			}
			repo := access.repository
			if !validRepositoryPart(repo.Owner.Login) || !validRepositoryPart(repo.Name) {
				http.Error(w, "Native repository unavailable.", 503)
				return
			}
			data.Authorized = true
			data.Actor = strconv.FormatInt(v.User.ID, 10)
			data.Login = v.User.Login
			data.Repository = repo.Owner.Login + "/" + repo.Name
			data.RepositoryURL = s.Config.ForgejoURL + "/" + url.PathEscape(repo.Owner.Login) + "/" + url.PathEscape(repo.Name)
		}
	}
	if !data.Authorized {
		s.nativePageEntry(w, r, store.OAuthLogin{RepositorySettingsReturn: true, RepositoryID: id})
		return
	}
	if writePageTemplate(w, repositorySpacesTemplate, data, http.StatusOK) != nil {
		http.Error(w, "Cannot render settings.", 500)
		return
	}
}
