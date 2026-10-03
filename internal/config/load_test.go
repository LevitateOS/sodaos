package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsTrailingDataPastSizeLimit(t *testing.T) {
	c := Config{Listen: "127.0.0.1:8080", ForgejoURL: "https://forgejo.test/", ForgejoInternalURL: "http://127.0.0.1:3000", DatabaseDSNFile: "/etc/soda/postgres/soda.dsn", HostSocket: "/run/soda/host.sock", GrantKeyFile: "/etc/soda/grant-key", OperatorID: 1}
	valid, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	write := func(contents string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(string(valid) + `{"forgejo_url":"https://evil.test"}`)
	if _, err = Load(path); err == nil || !strings.Contains(err.Error(), "one JSON object") {
		t.Fatal("small trailing object was not rejected", err)
	}
	// Trailing bytes pushed past the 64 KiB window must not read as a clean
	// EOF: the file is oversized, not a single object.
	write(string(valid) + strings.Repeat(" ", 65537-len(valid)) + `{"forgejo_url":"https://evil.test"}`)
	if _, err = Load(path); err == nil || !strings.Contains(err.Error(), "exceeds 64 KiB") {
		t.Fatal("trailing data past the size limit was accepted", err)
	}
	write(string(valid))
	if _, err = Load(path); err != nil {
		t.Fatal("valid configuration rejected", err)
	}
}

func TestPrivateHTTPSDeploymentBoundary(t *testing.T) {
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
	loaded, err := Load(path)
	if err != nil || loaded.ForgejoURL != "https://forgejo.test" {
		t.Fatal(loaded, err)
	}
	insecure := c
	insecure.ForgejoURL = "http://forgejo.test"
	save(insecure)
	if _, err = Load(path); err == nil {
		t.Fatal("insecure browser accepted")
	}
	if err = os.WriteFile(path, []byte(`{"public_url":"https://legacy.example"}`), 0o600); err != nil {
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
