// Package web wires the dashboard HTTP root: namespace gates, health, avatars
// and registration of auth/api. It does not own OAuth or product handlers.
package web

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/levitateos/sodaos/internal/avatar"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Server is the dashboard HTTP facade. Handlers live in Auth and API.
type Server struct {
	Config  config.Config
	Store   *store.Store
	Forgejo *forgejo.Client
	Host    *host.Client
	Auth    *auth.Service
	API     *api.API
	mux     *http.ServeMux
}

// New constructs auth and product APIs and registers all dashboard routes.
func New(c config.Config, db *store.Store) *Server {
	client := forgejo.New(c.ForgejoInternalURL)
	hostClient := host.NewClient(c.HostSocket)
	s := &Server{
		Config: c, Store: db, Forgejo: client, Host: hostClient,
		mux: http.NewServeMux(),
	}
	s.Auth = auth.New(&s.Config, db, client)
	s.API = api.New(&s.Config, db, client, hostClient, s.Auth)
	s.Auth.SessionEndGate = s.API.TerminalLock()
	s.Auth.CancelTerminals = s.API.CancelTerminals

	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})
	s.mux.HandleFunc("GET /{$}", s.forgejoHome)
	s.mux.Handle(avatarPrefix, avatarHandler{render: avatar.Render})
	s.mux.Handle(strings.TrimSuffix(avatarPrefix, "/"), avatarHandler{render: avatar.Render})
	s.Auth.Register(s.mux)
	s.API.Register(s.mux)
	notFound := func(w http.ResponseWriter, r *http.Request) {
		auth.JSONError(w, http.StatusNotFound, "not_found", "API route not found.")
	}
	s.mux.HandleFunc("/api", notFound)
	s.mux.HandleFunc("/api/", notFound)
	return s
}

func (s *Server) forgejoHome(w http.ResponseWriter, r *http.Request) {
	s.Auth.ForgejoReturn(w, r, nil)
}

// CloseTerminals shuts down browser terminal peers before process exit.
func (s *Server) CloseTerminals() { s.API.CloseTerminals() }

// SetForgejo replaces the Forgejo client on the facade and both services.
func (s *Server) SetForgejo(client *forgejo.Client) {
	s.Forgejo = client
	s.Auth.Forgejo = client
	s.API.Forgejo = client
}

// SetHost replaces the host client on the facade and product API.
func (s *Server) SetHost(client *host.Client) {
	s.Host = client
	s.API.Host = client
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
		auth.JSONError(w, http.StatusNotFound, "not_found", "Soda route not found.")
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
