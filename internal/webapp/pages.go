package webapp

import (
	"github.com/levitateos/sodaos/internal/webauth"
	"net/http"

	"github.com/levitateos/sodaos/internal/store"
)

// Bookmark entries carry no content or authority. Native login establishes the
// document actor; protected APIs authorize every inventory and operation.
func (s *API) spacesPage(w http.ResponseWriter, r *http.Request) {
	s.Auth.NativePageEntry(w, r, store.OAuthLogin{SpacesReturn: true})
}
func (s *API) runnersPage(w http.ResponseWriter, r *http.Request) {
	s.Auth.NativePageEntry(w, r, store.OAuthLogin{SettingsReturn: "runners"})
}
func (s *API) tailnetPage(w http.ResponseWriter, r *http.Request) {
	s.Auth.NativePageEntry(w, r, store.OAuthLogin{SettingsReturn: "tailnet"})
}
func (s *API) repositorySpacesPage(w http.ResponseWriter, r *http.Request) {
	id, ok := webauth.PositiveID(r.PathValue("repositoryID"))
	if !ok {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Error(w, "Invalid repository settings destination.", 400)
		return
	}
	s.Auth.NativePageEntry(w, r, store.OAuthLogin{RepositorySettingsReturn: true, RepositoryID: id})
}
