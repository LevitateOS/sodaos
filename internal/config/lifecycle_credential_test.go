package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLifecycleCredentialFile(t *testing.T) {
	c := Config{Listen: "127.0.0.1:8080", ForgejoURL: "https://forgejo.test/", ForgejoInternalURL: "http://127.0.0.1:3000", DatabaseDSNFile: "/etc/soda/postgres/soda.dsn", HostSocket: "/run/soda/host.sock", OperatorID: 1}
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
	if loaded, err := Load(path); err != nil || loaded.ForgejoLifecycleCredentialFile != "" {
		t.Fatal("unset lifecycle credential must load as unset", loaded, err)
	}
	host := uint32(1001)
	configured := c
	configured.ForgejoBackgroundSocket = "/run/soda/background.sock"
	configured.ForgejoBackgroundHostUID = &host
	configured.ForgejoBackgroundCredentialFile = "/etc/soda/background-pat"
	configured.ForgejoLifecycleCredentialFile = "/etc/soda/lifecycle-pat"
	save(configured)
	if loaded, err := Load(path); err != nil || loaded.ForgejoLifecycleCredentialFile != "/etc/soda/lifecycle-pat" {
		t.Fatal("lifecycle credential with background inputs must load", loaded, err)
	}
	relative := configured
	relative.ForgejoLifecycleCredentialFile = "lifecycle-pat"
	save(relative)
	if _, err := Load(path); err == nil {
		t.Fatal("relative lifecycle credential accepted")
	}
	unwired := c
	unwired.ForgejoLifecycleCredentialFile = "/etc/soda/lifecycle-pat"
	save(unwired)
	if _, err := Load(path); err == nil {
		t.Fatal("lifecycle credential without background inputs accepted")
	}
}
