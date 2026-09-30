// Package auth owns native extension profile resources. It does not authenticate
// product requests with Soda browser sessions.
package auth

import (
	"strings"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/store"
)

// Service holds native extension profile state.
type Service struct {
	Config *config.Config
	Store  *store.Store
}

// New constructs the native extension profile service.
func New(cfg *config.Config, db *store.Store) *Service {
	return &Service{Config: cfg, Store: db}
}

// ValidRepositoryPart rejects empty, relative or path-like repository name parts.
func ValidRepositoryPart(value string) bool {
	return value != "" && value != "." && value != ".." && len(value) <= 255 && !strings.ContainsAny(value, "/\\\x00\r\n")
}
