package web

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/levitateos/sodaos/assets"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
)

//go:embed templates/*.html static/*
var content embed.FS

type Server struct {
	Config    config.Config
	Store     *store.Store
	Forgejo   *forgejo.Client
	Host      *host.Client
	mux       *http.ServeMux
	templates *template.Template
}

func New(c config.Config, db *store.Store) *Server {
	s := &Server{Config: c, Store: db, Forgejo: forgejo.New(c.ForgejoInternalURL), Host: host.NewClient(c.HostSocket), mux: http.NewServeMux(), templates: template.Must(template.ParseFS(content, "templates/*.html"))}
	static, _ := fs.Sub(content, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	s.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.Files))))
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})
	s.mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if c, err := r.Cookie("soda_session"); err == nil && s.Store != nil {
			if _, err = s.Store.Session(r.Context(), c.Value); err == nil {
				http.Redirect(w, r, "/projects", 303)
				return
			}
		}
		s.render(w, "home", nil)
	})
	s.authRoutes()
	s.apiRoutes()
	s.projectRoutes()
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; style-src-attr 'unsafe-inline'; img-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	// Refuse malformed API paths as JSON instead of ServeMux's HTML canonical
	// redirect. API callers must never follow a redirect into a page response.
	if (r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/")) && path.Clean(r.URL.Path) != r.URL.Path {
		jsonError(w, http.StatusNotFound, "not_found", "API route not found.")
		return
	}
	s.mux.ServeHTTP(w, r)
}
func (s *Server) render(w http.ResponseWriter, name string, data any) {
	var b bytes.Buffer
	if err := s.templates.ExecuteTemplate(&b, name, data); err != nil {
		http.Error(w, "Cannot render response", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b.Bytes())
}
