package web

import (
	"bytes"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/frontend"
)

// Frontend is a validated build output. Load it before opening/migrating the
// database so a missing bundle cannot leave a partially upgraded application.
type Frontend = frontend.Bundle

func LoadFrontend(files fs.FS) (*Frontend, error) { return frontend.Load(files) }

func (s *Server) MountFrontend(frontend *Frontend) {
	s.mux.HandleFunc("GET /app/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/app/")
		if name == "assets" || strings.HasPrefix(name, "assets/") || name == "LICENSES.txt" {
			if !fs.ValidPath(name) || (name != "LICENSES.txt" && !frontend.Assets[name]) {
				http.NotFound(w, r)
				return
			}
			info, err := fs.Stat(frontend.Files, name)
			if err != nil || !info.Mode().IsRegular() {
				http.NotFound(w, r)
				return
			}
			body, err := fs.ReadFile(frontend.Files, name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			if name == "LICENSES.txt" {
				w.Header().Set("Cache-Control", "no-cache")
			}
			http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(body))
			return
		}
		if strings.HasPrefix(name, ".") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(frontend.Index))
	})
}
