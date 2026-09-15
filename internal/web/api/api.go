// Package api serves product HTTP and WebSocket APIs for environments,
// spaces, terminals, runners and Tailnet settings. It is not the OAuth owner.
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

// API owns product handlers and the browser terminal peer registry.
type API struct {
	Config  *config.Config
	Store   *store.Store
	Forgejo *forgejo.Client
	Host    *host.Client
	Auth    *auth.Service

	mux              *http.ServeMux
	terminalMu       sync.Mutex
	SpacesSlots      chan struct{}
	RepositorySlots  chan struct{}
	TerminalPeers    map[*http.Request]*TerminalPeer
	TerminalStopping map[string]bool
	terminalClosed   bool
	terminalWG       sync.WaitGroup
}

// TerminalPeer is a browser websocket peer tracked for logout cancellation.
type TerminalPeer struct {
	Token, ContextID, Project, ID string
	Cancel                        context.CancelFunc
}

// New constructs the product API. Auth must already exist; web wires the
// session-end bridge onto Auth after construction.
func New(cfg *config.Config, db *store.Store, client *forgejo.Client, hostClient *host.Client, auth *auth.Service) *API {
	return &API{
		Config: cfg, Store: db, Forgejo: client, Host: hostClient, Auth: auth,
		SpacesSlots: make(chan struct{}, 4), RepositorySlots: make(chan struct{}, 4),
	}
}

// Register mounts product routes. Call after Auth.Register so session guards exist.
func (s *API) Register(mux *http.ServeMux) {
	s.mux = mux
	s.environmentRoutes()
	s.runnerRoutes()
	s.tailnetRoutes()
	s.pageRoutes()
}

// TerminalLock exposes the peer-registry lock for auth session-end serialization.
func (s *API) TerminalLock() *sync.Mutex { return &s.terminalMu }

// CancelTerminals cancels matching peers; callers must hold TerminalLock.
func (s *API) CancelTerminals(contextID, token string) {
	for _, peer := range s.TerminalPeers {
		if (contextID != "" && peer.ContextID == contextID) || (token != "" && peer.Token == token) {
			peer.Cancel()
		}
	}
}

func (s *API) pageRoutes() {
	s.mux.HandleFunc("GET /spaces", s.spacesPage)
	s.mux.HandleFunc("GET /workspace", s.workspacePage)
	s.mux.HandleFunc("GET /repositories/{repositoryID}/settings/spaces", s.repositorySpacesPage)
}
