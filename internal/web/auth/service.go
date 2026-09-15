// Package auth owns OAuth, session, provider grants and me-key proxies for
// native Forgejo pages. It is not the product environment or terminal API.
package auth

import (
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

// Service holds OAuth/session/provider state shared by protected product handlers.
type Service struct {
	Config  *config.Config
	Store   *store.Store
	Forgejo *forgejo.Client

	mux           *http.ServeMux
	providerLocks providerLocks

	// SessionEndGate and CancelTerminals are wired by web from the terminal
	// registry so login-context end stays serialized with peer cancellation.
	SessionEndGate  sync.Locker
	CancelTerminals func(contextID, token string)
}

// New constructs an auth service. Call Register after wiring CancelTerminals.
func New(cfg *config.Config, db *store.Store, client *forgejo.Client) *Service {
	return &Service{Config: cfg, Store: db, Forgejo: client}
}

// Register mounts OAuth, login cancel and session/me routes on mux.
func (s *Service) Register(mux *http.ServeMux) {
	s.mux = mux
	s.authRoutes()
	s.sessionRoutes()
}

func (s *Service) withSessionEndGate(fn func()) {
	if s.SessionEndGate != nil {
		s.SessionEndGate.Lock()
		defer s.SessionEndGate.Unlock()
	}
	fn()
}

func (s *Service) cancelTerminals(contextID, token string) {
	if s.CancelTerminals != nil {
		s.CancelTerminals(contextID, token)
	}
}

// ValidRepositoryPart rejects empty, relative or path-like repository name parts.
func ValidRepositoryPart(value string) bool {
	return value != "" && value != "." && value != ".." && len(value) <= 255 && !strings.ContainsAny(value, "/\\\x00\r\n")
}

// ForgejoReturn redirects to the configured native Forgejo origin, optionally
// deep-linking a verified repository Spaces fragment.
func (s *Service) ForgejoReturn(w http.ResponseWriter, r *http.Request, repo *forgejo.Repository) {
	w.Header().Set("Cache-Control", "no-store")
	if config.BaseURL(s.Config.ForgejoURL) != nil || !strings.HasPrefix(s.Config.ForgejoURL, "https://") {
		http.Error(w, "Native frontend is not configured.", http.StatusServiceUnavailable)
		return
	}
	u, _ := url.Parse(s.Config.ForgejoURL)
	u.Path, u.RawPath = "/", ""
	if repo != nil && ValidRepositoryPart(repo.Owner.Login) && ValidRepositoryPart(repo.Name) {
		u.Path += repo.Owner.Login + "/" + repo.Name
		u.Fragment = "sodaspaces"
	}
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}
