package web

import (
	"net/http"
	"strings"

	"github.com/levitateos/factory/pkg/factory"
	factoryclient "github.com/levitateos/factory/pkg/factory/client"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/avatar"
	"github.com/levitateos/sodaos/internal/config"
)

// NewForge constructs Project, terminal, application and native extension
// administration without a coordinator, worker MCP listener or background work.
func NewForge(c config.Config, db *store.Store) *Server {
	s := newServer(c, db)
	s.mux.HandleFunc("GET /{$}", s.forgejoHome)
	s.mux.Handle(avatarPrefix, avatarHandler{render: avatar.Render})
	s.mux.Handle(strings.TrimSuffix(avatarPrefix, "/"), avatarHandler{render: avatar.Render})
	if c.FactoryControlSocket != "" {
		s.API.FactoryProjects = factoryclient.New(c.FactoryControlSocket)
	}
	return s
}

// ExtensionHandler exposes only this product's routes. The Fountain adapter
// selects the product endpoint; Forge never proxies or executes Factory APIs.
func (s *Server) ExtensionHandler() http.Handler {
	local := s.API.ExtensionHandler()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owned := factory.OwnsAPIPath(r.URL.Path)
		if s.Coordinator != nil {
			if r.URL.Path == "/api/factory/intake" {
				s.mux.ServeHTTP(w, r)
				return
			}
			if !owned {
				http.NotFound(w, r)
				return
			}
		} else if owned {
			http.NotFound(w, r)
			return
		}
		local.ServeHTTP(w, r)
	})
}
