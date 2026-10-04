package web

import (
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

func TestRootOnlyRedirectsToConfiguredForgejo(t *testing.T) {
	for _, base := range []string{"https://forgejo.example.test", "https://forgejo.example.test/"} {
		s := New(config.Config{ForgejoURL: base}, nil)
		for _, target := range []string{"/", "/?return_to=https://evil.example"} {
			w := httptest.NewRecorder()
			s.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
			want := base
			if want[len(want)-1] != '/' {
				want += "/"
			}
			if w.Code != 303 || w.Header().Get("Location") != want || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal(w.Code, w.Header())
			}
			if w.Header().Get("Content-Security-Policy") == "" {
				t.Fatal("missing CSP")
			}
		}
	}
}

func TestNoNativeDestinationDoesNotLoopOrRender(t *testing.T) {
	for _, base := range []string{"", "//evil.example", "javascript:alert(1)", "https://user:password@example.test", "https://forgejo.example.test?redirect=evil", "https://forgejo.example.test/native/", "http://forgejo.example.test"} {
		s := New(config.Config{ForgejoURL: base}, nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 503 || w.Header().Get("Location") != "" {
			t.Fatal(w.Code, w.Header())
		}
		w = httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
		if w.Code != 200 || w.Body.String() != "ok\n" {
			t.Fatal("health check depends on a frontend")
		}
	}
}

func TestCoordinatorLockLivesInFactoryState(t *testing.T) {
	root := t.TempDir()
	c := config.Config{DatabaseDSNFile: "/run/secrets/soda.dsn", FactoryPublicationRoot: root}
	if got, want := coordinatorLockPath(c), filepath.Join(root, "factory-coordinator.lock"); got != want {
		t.Fatalf("lock %q, want %q", got, want)
	}
	if got, want := coordinatorLockPath(config.Config{DatabaseDSNFile: "/run/secrets/soda.dsn"}), filepath.Join(defaultFactoryPublicationRoot, "factory-coordinator.lock"); got != want {
		t.Fatalf("default lock %q, want %q", got, want)
	}
}

func TestCoordinatorSharesBackendStoreAndClients(t *testing.T) {
	s := New(config.Config{ForgejoURL: "https://forgejo.example.test"}, nil)
	if s.Coordinator == nil || s.Coordinator.Store != s.Store {
		t.Fatal("coordinator does not own the backend database")
	}
	if s.Coordinator.Host != s.Host {
		t.Fatal("coordinator bypasses the backend host client")
	}
	if s.Coordinator.Broker == nil {
		t.Fatal("coordinator has no broker client")
	}
}
