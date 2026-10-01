package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBackgroundServiceInputs(t *testing.T) {
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
	if loaded, err := Load(path); err != nil || loaded.BackgroundServiceConfigured() {
		t.Fatal("unset background inputs must load as unconfigured", loaded, err)
	}
	host := uint32(1001)
	configured := c
	configured.ForgejoBackgroundSocket = "/run/soda/background.sock"
	configured.ForgejoBackgroundHostUID = &host
	configured.ForgejoBackgroundCredentialFile = "/etc/soda/background-pat"
	save(configured)
	loaded, err := Load(path)
	if err != nil || !loaded.BackgroundServiceConfigured() || *loaded.ForgejoBackgroundHostUID != 1001 {
		t.Fatal("complete background inputs must load", loaded, err)
	}
	partial := configured
	partial.ForgejoBackgroundHostUID = nil
	save(partial)
	if _, err = Load(path); err == nil {
		t.Fatal("partial background inputs accepted")
	}
	relative := configured
	relative.ForgejoBackgroundSocket = "background.sock"
	save(relative)
	if _, err = Load(path); err == nil {
		t.Fatal("relative background socket accepted")
	}
}
