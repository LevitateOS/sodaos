// Package web wires the dashboard HTTP root: namespace gates, health, avatars
// and native extension services. It does not own product handlers.
package web

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/levitateos/sodaos/internal/avatar"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/api"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Server is the dashboard HTTP facade. Handlers live in Auth and API.
type Server struct {
	Config      config.Config
	Store       *store.Store
	Forgejo     *forgejo.Client
	Host        *host.Client
	Auth        *auth.Service
	API         *api.API
	Coordinator *control.Coordinator
	mux         *http.ServeMux
}

// New constructs auth and product APIs and registers all dashboard routes.
func New(c config.Config, db *store.Store) *Server {
	client := forgejo.New(c.ForgejoInternalURL)
	hostClient := host.NewClient(c.HostSocket)
	s := &Server{
		Config: c, Store: db, Forgejo: client, Host: hostClient,
		mux: http.NewServeMux(),
	}
	s.Auth = auth.New(&s.Config, db)
	s.API = api.New(&s.Config, db, client, hostClient, s.Auth)
	broker := identityclient.New(c.IdentitySocket)
	s.API.Identity = broker
	s.Coordinator = control.NewCoordinator(db, hostClient, broker)
	s.API.Coordinator = s.Coordinator
	if c.BackgroundServiceConfigured() {
		background := forgejo.NewServiceBackground(c.ForgejoBackgroundSocket, *c.ForgejoBackgroundHostUID, "")
		observer := forgejo.NewServiceObserver(c.ForgejoBackgroundSocket, *c.ForgejoBackgroundHostUID,
			c.ForgejoBackgroundCredentialFile, client)
		observer.ShareBackground(background)
		source := api.NewServiceReadinessSource(observer)
		s.Coordinator.AcceptanceReads = source
		s.Coordinator.Readiness = source
		s.Coordinator.DispatchReads = source
		s.Coordinator.Publication = forgejo.NewPublisher(background, client, c.ForgejoInternalURL,
			publicationRoot(c), c.ForgejoBackgroundCredentialFile)
		if c.ForgejoReviewCredentialFile != "" {
			s.Coordinator.Reviews = forgejo.NewReviewer(background, client, c.ForgejoReviewCredentialFile)
		}
		if c.ForgejoMergeCredentialFile != "" {
			s.Coordinator.Merges = forgejo.NewMerger(background, client, c.ForgejoMergeCredentialFile)
			s.Coordinator.Checks = forgejo.NewCheckAssessor(background, client, c.ForgejoMergeCredentialFile)
		}
	}
	s.mux.HandleFunc("POST /api/factory/intake", api.IntakeHandler{
		Coordinator: s.Coordinator, Secret: loadIntakeSecret(c),
	}.ServeHTTP)

	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	s.mux.HandleFunc("GET /{$}", s.forgejoHome)
	s.mux.Handle(avatarPrefix, avatarHandler{render: avatar.Render})
	s.mux.Handle(strings.TrimSuffix(avatarPrefix, "/"), avatarHandler{render: avatar.Render})
	notFound := func(w http.ResponseWriter, r *http.Request) {
		auth.JSONError(w, http.StatusNotFound, "not_found", "API route not found.")
	}
	s.mux.HandleFunc("/api", notFound)
	s.mux.HandleFunc("/api/", notFound)
	return s
}

func (s *Server) forgejoHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	native, err := url.Parse(s.Config.ForgejoURL)
	if err != nil || config.BaseURL(s.Config.ForgejoURL) != nil || native.Scheme != "https" {
		http.Error(w, "Native Forgejo is not configured.", http.StatusServiceUnavailable)
		return
	}
	native.Path, native.RawPath = "/", ""
	http.Redirect(w, r, native.String(), http.StatusSeeOther)
}

// CloseTerminals shuts down native terminal peers before process exit.
func (s *Server) CloseTerminals() { s.API.CloseTerminals() }

// defaultFactoryPublicationRoot is the private backend publication
// workspace created by tmpfiles: validated bundles and push workspaces
// live here, never inside a Project.
const defaultFactoryPublicationRoot = "/var/lib/soda/dashboard/publication"

// publicationRoot resolves the configured publication workspace root,
// defaulting to the private backend data directory.
func publicationRoot(c config.Config) string {
	if c.FactoryPublicationRoot != "" {
		return c.FactoryPublicationRoot
	}
	return defaultFactoryPublicationRoot
}

// loadIntakeSecret reads the optional native webhook intake secret. An
// unset or unreadable secret disables intake: deliveries refuse as
// unavailable instead of bypassing authentication.
func loadIntakeSecret(c config.Config) []byte {
	if c.FactoryIntakeSecretFile == "" {
		return nil
	}
	secret, err := config.Secret(c.FactoryIntakeSecretFile)
	if err != nil {
		return nil
	}
	return []byte(secret)
}

// coordinatorLockPath keeps the factory-ledger lock in writable factory
// state. It must not derive from the DSN file: secrets arrive on read-only
// mounts, and the lock needs O_CREATE.
func coordinatorLockPath(c config.Config) string {
	return filepath.Join(publicationRoot(c), "factory-coordinator.lock")
}

// StartCoordinator takes exclusive factory-ledger ownership and settles
// outstanding runs before the operator endpoint serves. Only the serving
// backend calls this; request handling never starts a coordinator.
func (s *Server) StartCoordinator(ctx context.Context) error {
	lockPath := coordinatorLockPath(s.Config)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return err
	}
	return s.Coordinator.Start(ctx, lockPath)
}

// CloseCoordinator releases factory-ledger ownership before process exit.
func (s *Server) CloseCoordinator() { _ = s.Coordinator.Close() }

// SetForgejo replaces the Forgejo client on the facade and product API.
func (s *Server) SetForgejo(client *forgejo.Client) {
	s.Forgejo = client
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
