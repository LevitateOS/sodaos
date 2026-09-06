package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateHTTPSDeploymentBoundary(t *testing.T) {
	c := Config{Listen: "127.0.0.1:8080", PublicURL: "https://soda.test/", ForgejoURL: "https://forgejo.test/", ForgejoInternalURL: "http://127.0.0.1:3000", Database: "/var/lib/soda/dashboard/soda.db", HostSocket: "/run/soda/host.sock", OAuthClientID: "app", OAuthSecretFile: "/etc/soda/oauth-secret", AdminTokenFile: "/etc/soda/admin-token", OperatorID: 1}
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
	if err != nil || loaded.PublicURL != "https://soda.test" {
		t.Fatal(loaded, err)
	}
	insecure := c
	insecure.PublicURL = "http://soda.test"
	save(insecure)
	if _, err = Load(path); err == nil {
		t.Fatal("insecure browser accepted")
	}
	exposed := c
	exposed.Listen = "0.0.0.0:8080"
	save(exposed)
	if _, err = Load(path); err == nil {
		t.Fatal("public plaintext backend accepted")
	}
}
