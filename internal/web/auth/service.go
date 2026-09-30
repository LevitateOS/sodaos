// Package auth owns the shared-credential callback and native extension profile
// resources. It does not authenticate product requests with Soda browser sessions.
package auth

import (
	"net/http"
	"strings"
	"sync"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
)

// Service holds shared-credential callback and native extension profile state.
type Service struct {
	Config *config.Config
	Store  *store.Store

	EnrollmentBroker EnrollmentBroker
	brokerMu         sync.Mutex
	brokerBindings   map[string]brokerBinding
}

// New constructs the broker callback service.
func New(cfg *config.Config, db *store.Store) *Service {
	return &Service{Config: cfg, Store: db}
}

// Register mounts the callback that returns shared Forgejo credential custody
// to the Soda broker. Product pages and API requests use native extensions.
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /identity/callback", s.brokerCallback)
}

// ValidRepositoryPart rejects empty, relative or path-like repository name parts.
func ValidRepositoryPart(value string) bool {
	return value != "" && value != "." && value != ".." && len(value) <= 255 && !strings.ContainsAny(value, "/\\\x00\r\n")
}
