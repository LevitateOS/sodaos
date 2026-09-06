package web

import (
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/levitateos/sodaos/internal/config"
)

func frontendFixture() fstest.MapFS {
	return fstest.MapFS{
		"index.html":          &fstest.MapFile{Data: []byte(`<html><div id="root"></div><script src="/app/assets/main-123.js"></script></html>`)},
		"LICENSES.txt":        &fstest.MapFile{Data: []byte("fixture license")},
		".vite/manifest.json": &fstest.MapFile{Data: []byte(`{"index.html":{"file":"assets/main-123.js","isEntry":true}}`)},
		"assets/main-123.js":  &fstest.MapFile{Data: []byte(`console.log("fixture");`)},
	}
}

func TestFrontendDeliveryAndAPIBoundary(t *testing.T) {
	files := frontendFixture()
	files["assets/unlisted.txt"] = &fstest.MapFile{Data: []byte("not a public asset")}
	frontend, err := LoadFrontend(files)
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{}, nil)
	s.MountFrontend(frontend)
	for _, tc := range []struct {
		path        string
		status      int
		body, cache string
	}{
		{"/app/profile", 200, `id="root"`, "no-cache"},
		{"/app/assets/main-123.js", 200, "console.log", "public, max-age=31536000, immutable"},
		{"/app/assets/missing.js", 404, "404", ""},
		{"/app/assets/", 404, "404", ""},
		{"/app/assets", 404, "404", ""},
		{"/app/assets/unlisted.txt", 404, "404", ""},
		{"/app/.vite/manifest.json", 404, "404", ""},
		{"/api/unknown", 404, `"not_found"`, "no-store"},
		{"/api//session", 404, `"not_found"`, "no-store"},
		{"/api/session", 401, `"unauthenticated"`, "no-store"},
	} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.body) || w.Header().Get("Cache-Control") != tc.cache {
			t.Fatalf("%s: %d %s %s", tc.path, w.Code, w.Header().Get("Cache-Control"), w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 || strings.Contains(w.Body.String(), `id="root"`) {
		t.Fatal("legacy home replaced")
	}
}

func TestFrontendRejectsIncompleteBundle(t *testing.T) {
	for _, missing := range []string{"index.html", "LICENSES.txt", ".vite/manifest.json", "assets/main-123.js"} {
		files := frontendFixture()
		delete(files, missing)
		if _, err := LoadFrontend(files); err == nil {
			t.Fatal("accepted missing", missing)
		}
	}
	for _, manifest := range []string{
		`null`, `{}`, `not json`,
		`{"index.html":{"file":"../secret","isEntry":true}}`,
		`{"index.html":{"file":"assets/main-123.js"}}`,
		`{"index.html":{"file":"assets/main-123.js","isEntry":true,"imports":["missing"]}}`,
	} {
		files := frontendFixture()
		files[".vite/manifest.json"].Data = []byte(manifest)
		if _, err := LoadFrontend(files); err == nil {
			t.Fatal("accepted invalid manifest", manifest)
		}
	}
}
