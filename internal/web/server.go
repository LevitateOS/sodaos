package web

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/levitateos/sodaos/internal/avatar"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

type Server struct {
	Config           config.Config
	Store            *store.Store
	Forgejo          *forgejo.Client
	Host             *host.Client
	mux              *http.ServeMux
	providerLocks    providerLocks
	terminalMu       sync.Mutex
	spacesSlots      chan struct{}
	repositorySlots  chan struct{}
	terminalPeers    map[*http.Request]*terminalPeer
	terminalStopping map[string]bool
	terminalClosed   bool
	terminalWG       sync.WaitGroup
}

func New(c config.Config, db *store.Store) *Server {
	s := &Server{Config: c, Store: db, Forgejo: forgejo.New(c.ForgejoInternalURL), Host: host.NewClient(c.HostSocket), mux: http.NewServeMux(), spacesSlots: make(chan struct{}, 4), repositorySlots: make(chan struct{}, 4)}
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})
	s.mux.HandleFunc("GET /{$}", s.forgejoHome)
	s.mux.Handle(avatarPrefix, avatarHandler{render: avatar.Render})
	s.mux.Handle(strings.TrimSuffix(avatarPrefix, "/"), avatarHandler{render: avatar.Render})
	s.mux.HandleFunc("GET /spaces", s.spacesPage)
	s.mux.HandleFunc("GET /repositories/{repositoryID}/settings/spaces", s.repositorySpacesPage)
	s.authRoutes()
	s.apiRoutes()
	s.runnerRoutes()
	s.tailnetRoutes()
	return s
}

func (s *Server) forgejoHome(w http.ResponseWriter, r *http.Request) {
	s.forgejoReturn(w, r, nil)
}

// Only a freshly resolved repository can select a path under the configured
// native origin. Neither provider URLs nor caller return paths are accepted.
// A redirect does not create or transfer a native Forgejo session.
func (s *Server) forgejoReturn(w http.ResponseWriter, r *http.Request, repo *forgejo.Repository) {
	w.Header().Set("Cache-Control", "no-store")
	if config.BaseURL(s.Config.ForgejoURL) != nil || !strings.HasPrefix(s.Config.ForgejoURL, "https://") {
		http.Error(w, "Native frontend is not configured.", http.StatusServiceUnavailable)
		return
	}
	u, _ := url.Parse(s.Config.ForgejoURL)
	u.Path, u.RawPath = "/", ""
	if repo != nil && validRepositoryPart(repo.Owner.Login) && validRepositoryPart(repo.Name) {
		u.Path += repo.Owner.Login + "/" + repo.Name
		u.Fragment = "sodaspaces"
	}
	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

func publicAvatarPath(p string) bool {
	return p == strings.TrimSuffix(avatarPrefix, "/") || strings.HasPrefix(p, avatarPrefix)
}

func canonicalAvatarPath(u *url.URL) bool {
	return path.Clean(u.Path) == u.Path && u.EscapedPath() == u.Path && !strings.Contains(u.Path, "\\")
}

func sodaPublicProbe(p string) bool {
	return p == "/" || p == "/healthz"
}

func canonicalSodaPath(u *url.URL) bool {
	return u.RawPath == "" && !strings.Contains(u.Path, "\\") &&
		(u.Path == config.SodaPath+"/" || path.Clean(u.Path) == u.Path)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	// Public avatars retain their full namespaced path and never enter session
	// routing. Refuse aliases before ServeMux can canonicalize them.
	if publicAvatarPath(r.URL.Path) {
		if !canonicalAvatarPath(r.URL) {
			avatarError(w, r, http.StatusNotFound, "Avatar route not found.")
			return
		}
		s.mux.ServeHTTP(w, r)
		return
	}
	// Only the fixed namespace is public through Caddy. Root/health remain direct
	// backend probes, not aliases for browser API or authentication routes.
	if sodaPublicProbe(r.URL.Path) {
		if r.URL.RawPath != "" {
			http.NotFound(w, r)
			return
		}
		s.mux.ServeHTTP(w, r)
		return
	}
	if !strings.HasPrefix(r.URL.Path, config.SodaPath+"/") {
		http.NotFound(w, r)
		return
	}
	// Reject encoded aliases and canonicalization instead of redirecting API
	// requests (especially mutations) into another route or the native frontend.
	if !canonicalSodaPath(r.URL) {
		jsonError(w, http.StatusNotFound, "not_found", "Soda route not found.")
		return
	}
	mounted := r.Clone(r.Context())
	mounted.URL.Path = strings.TrimPrefix(r.URL.Path, config.SodaPath)
	if mounted.URL.Path == "/healthz" {
		http.NotFound(w, r)
		return
	}
	s.mux.ServeHTTP(w, mounted)
}
