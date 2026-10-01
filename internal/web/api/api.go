// Package api serves product HTTP and WebSocket APIs for environments,
// spaces, terminals and Tailnet settings.
package api

import (
	"context"
	"net/http"
	"sync"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// API owns native-extension product handlers and its terminal peer registry.
type API struct {
	Config   *config.Config
	Store    *store.Store
	Forgejo  *forgejo.Client
	Host     *host.Client
	Auth     *auth.Service
	Identity IdentityClient

	terminalMu       sync.Mutex
	SpacesSlots      chan struct{}
	RepositorySlots  chan struct{}
	TerminalPeers    map[*http.Request]*TerminalPeer
	TerminalStopping map[string]bool
	terminalClosed   bool
	terminalWG       sync.WaitGroup
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
	}
}
