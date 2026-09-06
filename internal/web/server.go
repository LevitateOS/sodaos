package web

import (
	"embed"
	"github.com/levitateos/sodaos/assets"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed templates/*.html static/*
var content embed.FS

type Server struct {
	Config    config.Config
	Store     *store.Store
	mux       *http.ServeMux
	templates *template.Template
}

func New(c config.Config, db *store.Store) *Server {
	s := &Server{Config: c, Store: db, mux: http.NewServeMux(), templates: template.Must(template.ParseFS(content, "templates/*.html"))}
	static, _ := fs.Sub(content, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	s.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.Files))))
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("ok\n"))
	})
	s.mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { s.render(w, "home", nil) })
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	s.mux.ServeHTTP(w, r)
}
func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		return
	}
}
