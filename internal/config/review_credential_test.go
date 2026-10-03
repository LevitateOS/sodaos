package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReviewCredentialFile(t *testing.T) {
	c := Config{Listen: "127.0.0.1:8080", ForgejoURL: "https://forgejo.test/", ForgejoInternalURL: "http://127.0.0.1:3000", DatabaseDSNFile: "/etc/soda/postgres/soda.dsn", HostSocket: "/run/soda/host.sock", GrantKeyFile: "/etc/soda/grant-key", OperatorID: 1}
	path := filepath.Join(t.TempDir(), "config.json")
	save := func(c Config) {
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	save(c)
	if loaded, err := Load(path); err != nil || loaded.ForgejoReviewCredentialFile != "" {
		t.Fatal("unset review credential must load as unset", loaded, err)
	}
	host := uint32(1001)
	configured := c
	configured.ForgejoBackgroundSocket = "/run/soda/background.sock"
	configured.ForgejoBackgroundHostUID = &host
	configured.ForgejoBackgroundCredentialFile = "/etc/soda/background-pat"
	configured.ForgejoReviewCredentialFile = "/etc/soda/review-pat"
	save(configured)
	if loaded, err := Load(path); err != nil || loaded.ForgejoReviewCredentialFile != "/etc/soda/review-pat" {
		t.Fatal("review credential with background inputs must load", loaded, err)
	}
	relative := configured
	relative.ForgejoReviewCredentialFile = "review-pat"
	save(relative)
	if _, err := Load(path); err == nil {
		t.Fatal("relative review credential accepted")
	}
	unwired := c
	unwired.ForgejoReviewCredentialFile = "/etc/soda/review-pat"
	save(unwired)
	if _, err := Load(path); err == nil {
		t.Fatal("review credential without background inputs accepted")
	}
}
