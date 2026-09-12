package web

import (
	"net/http"

	"github.com/levitateos/sodaos/internal/store"
)

// Bookmark entries carry no content or authority. Native login establishes the
// document actor; protected APIs authorize every inventory and operation.
func (s *Server) spacesPage(w http.ResponseWriter, r *http.Request) {
	s.nativePageEntry(w, r, store.OAuthLogin{SpacesReturn: true})
}
func (s *Server) runnersPage(w http.ResponseWriter, r *http.Request) {
	s.nativePageEntry(w, r, store.OAuthLogin{SettingsReturn: "runners"})
}
func (s *Server) tailnetPage(w http.ResponseWriter, r *http.Request) {
	s.nativePageEntry(w, r, store.OAuthLogin{SettingsReturn: "tailnet"})
}
func (s *Server) repositorySpacesPage(w http.ResponseWriter, r *http.Request) {
	id, ok := positiveID(r.PathValue("repositoryID"))
	if !ok {
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Error(w, "Invalid repository settings destination.", 400)
		return
	}
	s.nativePageEntry(w, r, store.OAuthLogin{RepositorySettingsReturn: true, RepositoryID: id})
}
