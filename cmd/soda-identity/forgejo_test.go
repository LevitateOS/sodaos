package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	identityforgejo "github.com/levitateos/sodaos/internal/identity/forgejo"
)

func TestForgejoConfigurationRequiresExplicitNativeApplication(t *testing.T) {
	providers := map[string]identity.Provider{}
	if err := configureForgejo(settings{}, providers); err != nil || len(providers) != 0 {
		t.Fatal("disabled provider altered authentication", err)
	}
	c := settings{Forgejo: identityforgejo.Config{Base: "https://forgejo.example.test"}}
	if err := configureForgejo(c, providers); err == nil {
		t.Fatal("partial native application configuration accepted")
	}
	c.ForgejoSecretFile = filepath.Join(t.TempDir(), "client-secret")
	if err := os.WriteFile(c.ForgejoSecretFile, []byte("synthetic-client-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	c.Forgejo.ClientID = "synthetic-client"
	c.Forgejo.RedirectURL = "https://forgejo.example.test/-/soda/identity/callback"
	if err := configureForgejo(c, providers); err != nil || providers[identity.Forgejo] == nil {
		t.Fatal("explicit native application not configured", err)
	}
}
