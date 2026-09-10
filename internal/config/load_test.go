package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrivateHTTPSDeploymentBoundary(t *testing.T) {
	c := Config{Listen: "127.0.0.1:8080", ForgejoURL: "https://forgejo.test/", ForgejoInternalURL: "http://127.0.0.1:3000", Database: "/var/lib/soda/dashboard/soda.db", HostSocket: "/run/soda/host.sock", OAuthClientID: "app", OAuthSecretFile: "/etc/soda/oauth-secret", GrantKeyFile: "/etc/soda/grant-key", AdminTokenFile: "/etc/soda/admin-token", OperatorID: 1}
	path := filepath.Join(t.TempDir(), "config.json")
	save := func(c Config) {
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	save(c)
	loaded, err := Load(path)
	if err != nil || loaded.ForgejoURL != "https://forgejo.test" || loaded.OAuthCallbackURL() != "https://forgejo.test/-/soda/oauth/callback" {
		t.Fatal(loaded, err)
	}
	// Retired bootstrap references are parse-only, even when unusable. Loading
	// must not open the path or alter an existing operator file.
	retired := filepath.Join(t.TempDir(), "retired-token")
	if err := os.WriteFile(retired, []byte("synthetic-retained-input"), 0000); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(retired)
	if err != nil {
		t.Fatal(err)
	}
	for _, legacy := range []string{"", "relative-unused-path", retired, retired + "-missing"} {
		t.Run("legacy="+legacy, func(t *testing.T) {
			candidate := c
			candidate.AdminTokenFile = legacy
			save(candidate)
			loaded, err := Load(path)
			if err != nil || loaded.OperatorID != c.OperatorID || loaded.GrantKeyFile != c.GrantKeyFile || loaded.OAuthSecretFile != c.OAuthSecretFile {
				t.Fatal("retired credential reference affected configuration loading", err)
			}
			if legacy == "" {
				data, err := os.ReadFile(path)
				if err != nil || strings.Contains(string(data), "admin_token_file") {
					t.Fatal("new configuration includes retired field")
				}
			}
		})
	}
	after, err := os.Lstat(retired)
	if err != nil || !os.SameFile(before, after) || after.Mode().Perm() != 0000 || !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("retired credential changed")
	}
	insecure := c
	insecure.ForgejoURL = "http://forgejo.test"
	save(insecure)
	if _, err = Load(path); err == nil {
		t.Fatal("insecure browser accepted")
	}
	if err = os.WriteFile(path, []byte(`{"public_url":"https://legacy.example"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(path); err == nil || !strings.Contains(err.Error(), `unknown field "public_url"`) {
		t.Fatal("legacy standalone configuration was not explicitly rejected", err)
	}
	exposed := c
	exposed.Listen = "0.0.0.0:8080"
	save(exposed)
	if _, err = Load(path); err == nil {
		t.Fatal("public plaintext backend accepted")
	}
}
