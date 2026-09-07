package web

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

type Server struct {
	Config        config.Config
	Store         *store.Store
	Forgejo       *forgejo.Client
	Host          *host.Client
	mux           *http.ServeMux
	providerLocks providerLocks
}

func New(c config.Config, db *store.Store) *Server {
	s := &Server{Config: c, Store: db, Forgejo: forgejo.New(c.ForgejoInternalURL), Host: host.NewClient(c.HostSocket), mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})
	s.mux.HandleFunc("GET /{$}", s.forgejoHome)
	s.authRoutes()
	s.apiRoutes()
	return s
}

// No standalone Soda UI remains. Only the configured native frontend is a
// browser destination; caller parameters and historical OAuth return paths are
// never redirect authority. This does not create or transfer a native session.
func (s *Server) forgejoHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	u, err := url.Parse(s.Config.ForgejoURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		http.Error(w, "Native frontend is not configured.", http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, strings.TrimRight(u.String(), "/")+"/", http.StatusSeeOther)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	// Refuse malformed API paths as JSON instead of ServeMux's HTML canonical
	// redirect. API callers must never follow a redirect into a page response.
	if (r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/")) && path.Clean(r.URL.Path) != r.URL.Path {
		jsonError(w, http.StatusNotFound, "not_found", "API route not found.")
		return
	}
	s.mux.ServeHTTP(w, r)
}
