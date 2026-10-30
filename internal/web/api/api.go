// Package api serves product HTTP and WebSocket APIs for environments,
// spaces, terminals and Tailnet settings.
package api

import (
	"context"
	"net/http"
	"sync"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// API owns native-extension product handlers and its terminal peer registry.
type API struct {
	IssueMakerNative IssueMakerNative
	Config           *config.Config
	Store            *store.Store
	Forgejo          *forgejo.Client
	Host             *host.Client
	Auth             *auth.Service
	Identity         IdentityClient
	Coordinator      *control.Coordinator
	ExtensionManager *forgejo.ServiceBackground
	// IntentExploreChooser overrides the TypeSafe judgment service for
	// intent exploration when set. Production leaves it nil.
	IntentExploreChooser control.ProductJudgment
	// ProjectCreationAdmission holds native maintenance exclusion before any
	// durable reservation. Tests with a fake native host supply an explicit gate.
	ProjectCreationAdmission func() (func(), error)
	// MemberAccessFence publishes the retained startup fence before access
	// changes. Tests substitute an isolated real filesystem root.
	MemberAccessFence func(string, int64, bool) error

	terminalMu         sync.Mutex
	memberAccessMu     sync.Mutex
	applicationMu      sync.Mutex
	applicationPending map[string]bool
	SpacesSlots        chan struct{}
	RepositorySlots    chan struct{}
	TerminalPeers      map[*http.Request]*TerminalPeer
	TerminalStopping   map[string]bool
	terminalClosed     bool
	terminalWG         sync.WaitGroup
}

// TerminalPeer is a native extension operation or stream tracked for lifecycle cancellation.
type TerminalPeer struct {
	Token, ContextID, Project, ID string
	Cancel                        context.CancelFunc
}

// New constructs the native extension product API. Auth must already exist.
func New(cfg *config.Config, db *store.Store, client *forgejo.Client, hostClient *host.Client, auth *auth.Service) *API {
	return &API{
		Config: cfg, Store: db, Forgejo: client, Host: hostClient, Auth: auth,
		SpacesSlots: make(chan struct{}, 4), RepositorySlots: make(chan struct{}, 4),
		ProjectCreationAdmission: host.AcquireProjectCreation,
		MemberAccessFence:        host.SetMemberAccessFence,
	}
}
