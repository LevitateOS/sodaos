// Package web wires the dashboard HTTP root: namespace gates, health, avatars
// and native extension services. It does not own product handlers.
package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/forgejo/observation"
	"github.com/levitateos/sodaos/internal/host"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	"github.com/levitateos/sodaos/internal/web/api"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// Server is the dashboard HTTP facade. Handlers live in Auth and API.
type Server struct {
	Config       config.Config
	Store        *store.Store
	Forgejo      *forgejo.Client
	Host         *host.Client
	Auth         *auth.Service
	API          *api.API
	Coordinator  *control.Coordinator
	Application  factoryservice.Application
	Broker       *identityclient.Client
	FountainHost collaboration.Host
	mux          *http.ServeMux
}

// newServer constructs the shared transport and Project dependencies.
func newServer(c config.Config, db *store.Store) *Server {
	client := forgejo.New(c.ForgejoInternalURL)
	hostClient := host.NewClient(c.HostSocket)
	s := &Server{
		Config: c, Store: db, Forgejo: client, Host: hostClient,
		mux: http.NewServeMux(),
	}
	s.Auth = auth.New(&s.Config, db)
	s.API = api.New(&s.Config, db, client, hostClient, s.Auth)
	broker := identityclient.New(c.IdentitySocket)
	s.Broker = broker
	s.API.Identity = broker
	// Manager control uses the pinned service callback, without a factory
	// actor credential or background admission. Ordinary appliance activation
	// must support optional extensions before any factory actor is configured.
	callback := os.Getenv(extensions.ServiceCallbackEnv)
	peer, err := strconv.ParseUint(os.Getenv(extensions.ServiceCallbackPeerEnv), 10, 32)
	if filepath.IsAbs(callback) && err == nil {
		s.API.ExtensionManager = forgejo.NewServiceBackground(callback, uint32(peer), "")
	}

	// Health doubles as readiness: beyond "ok" it reports the two
	// dependencies admitted work needs — coordinator admission and the
	// store connection — without opening any new TCP surface. A 503
	// names the failing dependency so Caddy stops routing here.
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if s.Coordinator != nil && !s.Coordinator.Admitting() {
			http.Error(w, "shutting down\n", http.StatusServiceUnavailable)
			return
		}
		if s.Store != nil {
			ctx, stop := context.WithTimeout(r.Context(), 2*time.Second)
			defer stop()
			if err := s.Store.Ping(ctx); err != nil {
				http.Error(w, "store unavailable\n", http.StatusServiceUnavailable)
				return
			}
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	notFound := func(w http.ResponseWriter, r *http.Request) {
		auth.JSONError(w, http.StatusNotFound, "not_found", "API route not found.")
	}
	s.mux.HandleFunc("/api", notFound)
	s.mux.HandleFunc("/api/", notFound)
	return s
}

// NewFactory composes the appliance-hosted Factory product. Only this product
// constructor creates a coordinator or admits background work.
func NewFactory(c config.Config, db *store.Store) (*Server, error) {
	if db == nil {
		return nil, errors.New("Factory store is required")
	}
	// This constructor composes the appliance Factory. Binding the operator's
	// configured instance precedes all coordinator/background construction.
	fountain, err := db.BindConfiguredFountainLedger(context.Background(), c.ForgejoURL)
	if err != nil {
		return nil, fmt.Errorf("bind configured Fountain ledger: %w", err)
	}
	identities, err := forgejo.NewFountainIdentities(context.Background(), db, fountain.ID)
	if err != nil {
		return nil, fmt.Errorf("resolve configured Fountain namespace: %w", err)
	}
	s := newServer(c, db)
	s.FountainHost = fountain
	s.API.FountainIdentities = identities
	s.mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Soda Factory\n"))
	})
	client, hostClient, broker := s.Forgejo, s.Host, s.Broker
	factoryHost := &applianceFactoryHost{HostFactory: hostClient, identities: identities}
	view := s.API.FactoryApplication()
	ports := factoryservice.ApplicationPorts{Store: db, Host: factoryHost, Broker: host.NewFactoryObserver(hostClient, broker), CurrentChooser: view.Chooser, IssueEffects: view.IssueEffects}
	if c.TypeSafeAPIKeyFile != "" {
		ports.Planning = &control.TypeSafePriority{APIKeyFile: c.TypeSafeAPIKeyFile, Model: c.TypeSafeModel}
	}
	ports.Priority = &control.TypeSafePriority{APIKeyFile: c.TypeSafeAPIKeyFile, Model: c.TypeSafeModel}
	if c.BackgroundServiceConfigured() {
		background := forgejo.NewServiceBackground(c.ForgejoBackgroundSocket, *c.ForgejoBackgroundHostUID, "")
		observer := forgejo.NewServiceObserver(c.ForgejoBackgroundSocket, *c.ForgejoBackgroundHostUID,
			c.ForgejoBackgroundCredentialFile, client)
		observer.Identities = identities
		observer.ShareBackground(background)
		if c.ForgejoGraphCredentialFile != "" && c.ForgejoGraphActorID > 0 {
			graphActor, err := identities.LocalActor(context.Background(), strconv.FormatInt(c.ForgejoGraphActorID, 10))
			if err != nil {
				return nil, fmt.Errorf("resolve configured graph actor: %w", err)
			}
			ports.Graph = forgejo.NewIssueGraphWriter(background, client, c.ForgejoGraphCredentialFile, identities)
			ports.GraphActorID = int64(graphActor)
		}
		source := observation.NewServiceReadinessSource(observer, identities)
		ports.AcceptanceReads = source
		ports.Readiness = source
		ports.Creations = observer
		ports.DispatchReads = source
		completionSource := observation.NewCompletionSnapshotSource(observer, identities)
		completionSource.Store = db
		ports.CompletionEnumerator = completionSource
		ports.CompletionLocalCapture = completionSource.CaptureLocal
		ports.Publication = forgejo.NewPublisher(background, client, c.ForgejoInternalURL,
			publicationRoot(c), c.ForgejoBackgroundCredentialFile, identities)
		if c.ForgejoReviewCredentialFile != "" {
			ports.Reviews = forgejo.NewReviewer(background, client, c.ForgejoReviewCredentialFile, identities)
		}
		if c.ForgejoResolveCredentialFile != "" {
			ports.Resolutions = forgejo.NewResolver(background, client, c.ForgejoResolveCredentialFile, identities)
		}
		if c.ForgejoMergeCredentialFile != "" {
			ports.Merges = forgejo.NewMerger(background, client, c.ForgejoMergeCredentialFile, identities)
			ports.Checks = forgejo.NewCheckAssessor(background, client, c.ForgejoMergeCredentialFile, identities)
		}
	}
	application := factoryservice.NewApplication(ports)
	s.Application = application
	s.Coordinator, s.API.Coordinator = application.Coordinator, application.Coordinator
	s.mux.HandleFunc("POST /api/factory/intake", api.IntakeHandler{
		Coordinator: s.Coordinator, Store: db, Secret: loadIntakeSecret(c), Identities: identities,
	}.ServeHTTP)

	s.API.FactoryProjects = s.Coordinator
	return s, nil
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
	key, err := control.LoadFactoryMCPKey(publicationRoot(s.Config))
	if err != nil {
		return err
	}
	s.Coordinator.MCPKey = key
	if err := s.Application.Start(ctx, lockPath); err != nil {
		return err
	}
	return nil
}

// CloseCoordinator releases factory-ledger ownership after admitted work ends.
func (s *Server) CloseCoordinator() error { return s.Application.Close() }

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
