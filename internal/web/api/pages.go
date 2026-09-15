package api

import (
	"net/http"
	"strings"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Bookmark entries carry no content or authority. Native login establishes the
// document actor; protected APIs authorize every inventory and operation.
func (s *API) spacesPage(w http.ResponseWriter, r *http.Request) {
	s.Auth.NativePageEntry(w, r, store.OAuthLogin{SpacesReturn: true})
}

func (s *API) workspacePage(w http.ResponseWriter, r *http.Request) {
	frame, ok := workspaceFrame(r)
	if !ok {
		rejectWorkspaceQuery(w)
		return
	}
	if s.serveWorkspaceShell(w, r, frame) {
		return
	}
	entry := r.Clone(r.Context())
	location := *r.URL
	location.RawQuery = ""
	location.ForceQuery = false
	entry.URL = &location
	s.Auth.NativePageEntry(w, entry, store.OAuthLogin{SpacesReturn: true})
}

func (s *API) serveWorkspaceShell(w http.ResponseWriter, r *http.Request, frame string) bool {
	if config.BaseURL(s.Config.ForgejoURL) != nil || !strings.HasPrefix(s.Config.ForgejoURL, "https://") {
		return false
	}
	if s.Store == nil {
		return false
	}
	session, err := s.Auth.BrowserSession(r)
	if err != nil {
		return false
	}
	s.writeWorkspaceShell(w, session, frame)
	return true
}

func rejectWorkspaceQuery(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Error(w, "Page entries do not accept navigation parameters.", 400)
}

func (s *API) runnersPage(w http.ResponseWriter, r *http.Request) {
	s.Auth.NativePageEntry(w, r, store.OAuthLogin{SettingsReturn: "runners"})
}

func (s *API) tailnetPage(w http.ResponseWriter, r *http.Request) {
	s.Auth.NativePageEntry(w, r, store.OAuthLogin{SettingsReturn: "tailnet"})
}

func (s *API) repositorySpacesPage(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.PositiveID(r.PathValue("repositoryID"))
	if !ok {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Error(w, "Invalid repository settings destination.", 400)
		return
	}
	s.Auth.NativePageEntry(w, r, store.OAuthLogin{RepositorySettingsReturn: true, RepositoryID: id})
}
