package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

// Frontend is a validated build output. Load it before opening/migrating the
// database so a missing bundle cannot leave a partially upgraded application.
type Frontend struct {
	files  fs.FS
	index  []byte
	assets map[string]bool
}

func LoadFrontend(files fs.FS) (*Frontend, error) {
	if err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.Type()&fs.ModeSymlink != 0 || (!entry.IsDir() && (!info.Mode().IsRegular() || info.Size() > 8<<20)) {
			return errors.New("frontend contains a link, special file or oversized asset")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	index, err := fs.ReadFile(files, "index.html")
	if err != nil || len(index) == 0 || len(index) > 1<<20 {
		return nil, errors.New("frontend index is missing or invalid")
	}
	if license, err := fs.ReadFile(files, "LICENSES.txt"); err != nil || len(license) == 0 {
		return nil, errors.New("frontend licenses are missing")
	}
	var manifest map[string]struct {
		File           string   `json:"file"`
		CSS            []string `json:"css"`
		Assets         []string `json:"assets"`
		Imports        []string `json:"imports"`
		DynamicImports []string `json:"dynamicImports"`
		IsEntry        bool     `json:"isEntry"`
	}
	raw, err := fs.ReadFile(files, ".vite/manifest.json")
	if err != nil || json.Unmarshal(raw, &manifest) != nil || len(manifest) == 0 {
		return nil, errors.New("frontend manifest is missing or invalid")
	}
	entryFound := false
	assets := make(map[string]bool)
	for _, chunk := range manifest {
		entryFound = entryFound || chunk.IsEntry
		if chunk.IsEntry && !bytes.Contains(index, []byte(`src="/app/`+chunk.File+`"`)) {
			return nil, errors.New("frontend index does not match its entry manifest")
		}
		for _, name := range append(append([]string{chunk.File}, chunk.CSS...), chunk.Assets...) {
			if !fs.ValidPath(name) || !strings.HasPrefix(name, "assets/") {
				return nil, errors.New("frontend manifest contains an invalid asset path")
			}
			if info, err := fs.Stat(files, name); err != nil || !info.Mode().IsRegular() {
				return nil, errors.New("frontend manifest asset is missing")
			}
			assets[name] = true
		}
		for _, name := range append(chunk.Imports, chunk.DynamicImports...) {
			if _, ok := manifest[name]; !ok {
				return nil, errors.New("frontend manifest import is missing")
			}
		}
	}
	if !entryFound {
		return nil, errors.New("frontend manifest has no entry")
	}
	return &Frontend{files: files, index: index, assets: assets}, nil
}

func (s *Server) MountFrontend(frontend *Frontend) {
	s.mux.HandleFunc("GET /app/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/app/")
		if name == "assets" || strings.HasPrefix(name, "assets/") || name == "LICENSES.txt" {
			if !fs.ValidPath(name) || (name != "LICENSES.txt" && !frontend.assets[name]) {
				http.NotFound(w, r)
				return
			}
			info, err := fs.Stat(frontend.files, name)
			if err != nil || !info.Mode().IsRegular() {
				http.NotFound(w, r)
				return
			}
			body, err := fs.ReadFile(frontend.files, name)
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
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(frontend.index))
	})
}
