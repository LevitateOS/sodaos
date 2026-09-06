package nativebuild

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCoreOSDownloadTLSRedirectBoundsAndNoOverwrite(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte("fixture"))
		case "/redirect":
			http.Redirect(w, r, "http://example.invalid/base", http.StatusFound)
		case "/query":
			http.Redirect(w, r, "/ok?token=must-not-follow", http.StatusFound)
		case "/missing":
			http.NotFound(w, r)
		default:
			http.Redirect(w, r, "/loop", http.StatusFound)
		}
	}))
	defer server.Close()
	client := coreOSClient(server.Client().Transport) // explicit test CA, never InsecureSkipVerify
	for _, path := range []string{"/redirect", "/query", "/missing", "/loop"} {
		out := filepath.Join(t.TempDir(), "download")
		if err := downloadHTTP(context.Background(), client, server.URL+path, out, 16); err == nil {
			t.Fatal("accepted unsafe/failed download")
		}
		if _, err := os.Lstat(out); !os.IsNotExist(err) {
			t.Fatal("rejected response created output")
		}
	}
	out := filepath.Join(t.TempDir(), "download")
	if err := downloadHTTP(context.Background(), client, server.URL+"/ok", out, 16); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil || string(raw) != "fixture" {
		t.Fatal("wrong download bytes")
	}
	if err := downloadHTTP(context.Background(), client, server.URL+"/ok", out, 16); err == nil {
		t.Fatal("overwrote download")
	}
	if err := downloadHTTP(context.Background(), client, server.URL+"/ok", filepath.Join(t.TempDir(), "limited"), 3); err == nil {
		t.Fatal("ignored size bound")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := downloadHTTP(ctx, client, server.URL+"/ok", filepath.Join(t.TempDir(), "cancelled"), 16); err == nil {
		t.Fatal("ignored cancellation")
	}
}
