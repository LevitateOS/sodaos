package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFactoryIntakeSecretFile(t *testing.T) {
	c := Config{Listen: "127.0.0.1:8080", ForgejoURL: "https://forgejo.test/", ForgejoInternalURL: "http://127.0.0.1:3000", Database: "/var/lib/soda/dashboard/soda.db", HostSocket: "/run/soda/host.sock", GrantKeyFile: "/etc/soda/grant-key", OperatorID: 1}
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
	if loaded, err := Load(path); err != nil || loaded.FactoryIntakeSecretFile != "" {
		t.Fatal("unset intake secret must load empty", loaded, err)
	}
	configured := c
	configured.FactoryIntakeSecretFile = "/etc/soda/factory-intake-secret"
	save(configured)
	loaded, err := Load(path)
	if err != nil || loaded.FactoryIntakeSecretFile != "/etc/soda/factory-intake-secret" {
		t.Fatal("absolute intake secret must load", loaded, err)
	}
	relative := configured
	relative.FactoryIntakeSecretFile = "intake-secret"
	save(relative)
	if _, err = Load(path); err == nil {
		t.Fatal("relative intake secret accepted")
	}
}
