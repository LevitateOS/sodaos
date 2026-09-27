package main

import (
	"errors"
	"path/filepath"

	"github.com/levitateos/sodaos/internal/config"
	"github.com/levitateos/sodaos/internal/identity"
	identityforgejo "github.com/levitateos/sodaos/internal/identity/forgejo"
)

func configureForgejo(c settings, providers map[string]identity.Provider) error {
	if c.Forgejo == (identityforgejo.Config{}) && c.ForgejoSecretFile == "" {
		return nil
	}
	if !filepath.IsAbs(c.ForgejoSecretFile) {
		return errors.New("explicit Forgejo broker secret path required")
	}
	secret, err := config.Secret(c.ForgejoSecretFile)
	if err != nil {
		return err
	}
	provider, err := identityforgejo.New(c.Forgejo, secret)
	if err != nil {
		return err
	}
	providers[identity.Forgejo] = provider
	return nil
}
