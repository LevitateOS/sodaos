package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

func TestSetupRevokesBootstrapTokenOnSuccess(t *testing.T) {
	const token = "synthetic-bootstrap-token-not-for-retention"
	for _, name := range []string{"success-revokes", "failure-keeps-retry-token"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "soda")
			if err := os.Mkdir(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			tokenPath := filepath.Join(root, "operator-input")
			if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(dir, "dashboard.json")
			revoked := false
			deletes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "token "+token {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				switch r.Method + " " + r.URL.Path {
				case "GET /api/v1/user":
					if revoked {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "login": "soda-tester", "is_admin": name != "failure-keeps-retry-token"})
				case "DELETE /api/v1/user/token":
					deletes++
					if revoked {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
					revoked = true
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Error("setup reached an unexpected provider endpoint")
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			setupErr := setup("https://forgejo.test/", server.URL, tokenPath, out, filepath.Join(root, "postgres"))
			if name == "success-revokes" {
				if setupErr != nil {
					t.Fatalf("setup failed: %v", setupErr)
				}
				if deletes != 1 {
					t.Fatalf("successful setup issued %d revocation calls, want 1", deletes)
				}
				if !revoked {
					t.Fatal("bootstrap token survives successful setup")
				}
				return
			}
			if setupErr == nil {
				t.Fatal("failing setup unexpectedly succeeded")
			}
			if deletes != 0 {
				t.Fatal("failed setup revoked the retry token")
			}
			if revoked {
				t.Fatal("failed setup must keep the bootstrap token usable for retry")
			}
		})
	}
}

func TestSetupProvisionsPostgresSecrets(t *testing.T) {
	const token = "synthetic-bootstrap-token-not-for-retention"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "login": "soda-tester", "is_admin": true})
		case "DELETE /api/v1/user/token":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Run("generates all four files", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "soda")
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		tokenPath := filepath.Join(root, "operator-input")
		if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		pgDir := filepath.Join(root, "postgres")
		if err := setup("https://forgejo.test/", server.URL, tokenPath, filepath.Join(dir, "dashboard.json"), pgDir); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		passwords := map[string]string{}
		for _, role := range []string{"super", "forgejo", "soda"} {
			data, err := os.ReadFile(filepath.Join(pgDir, role+".passwd"))
			st, statErr := os.Stat(filepath.Join(pgDir, role+".passwd"))
			if err != nil || statErr != nil || st.Mode().Perm() != 0o600 {
				t.Fatalf("%s password file missing or not restricted", role)
			}
			pw := strings.TrimSpace(string(data))
			if len(pw) != 64 {
				t.Fatalf("%s password has %d chars, want 64 hex", role, len(pw))
			}
			for _, c := range pw {
				if !strings.ContainsRune("0123456789abcdef", c) {
					t.Fatalf("%s password is not hex", role)
				}
			}
			passwords[role] = pw
		}
		if passwords["super"] == passwords["forgejo"] || passwords["forgejo"] == passwords["soda"] || passwords["super"] == passwords["soda"] {
			t.Fatal("role passwords are not distinct")
		}
		dsn, err := os.ReadFile(filepath.Join(pgDir, "soda.dsn"))
		st, statErr := os.Stat(filepath.Join(pgDir, "soda.dsn"))
		if err != nil || statErr != nil || st.Mode().Perm() != 0o600 {
			t.Fatal("soda.dsn missing or not restricted")
		}
		want := "postgres://soda:" + passwords["soda"] + "@soda-postgres:5432/soda?sslmode=disable"
		if strings.TrimSpace(string(dsn)) != want {
			t.Fatal("soda.dsn does not carry the soda role password")
		}
	})

	t.Run("preserves pre-existing secrets", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "soda")
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		tokenPath := filepath.Join(root, "operator-input")
		if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		pgDir := filepath.Join(root, "postgres")
		if err := os.MkdirAll(pgDir, 0o700); err != nil {
			t.Fatal(err)
		}
		prior := filepath.Join(pgDir, "super.passwd")
		if err := os.WriteFile(prior, []byte("operator-owned-secret\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		err := setup("https://forgejo.test/", server.URL, tokenPath, filepath.Join(dir, "dashboard.json"), pgDir)
		if err == nil {
			t.Fatal("setup overwrote pre-existing database secrets")
		}
		data, _ := os.ReadFile(prior)
		if string(data) != "operator-owned-secret\n" {
			t.Fatal("pre-existing secret changed")
		}
		if _, statErr := os.Stat(filepath.Join(dir, "grant-key")); !os.IsNotExist(statErr) {
			t.Fatal("failed setup left its own grant key behind")
		}
		if _, statErr := os.Stat(filepath.Join(dir, "dashboard.json")); !os.IsNotExist(statErr) {
			t.Fatal("failed setup published configuration")
		}
	})
}

func TestSetupBootstrapCredentialBoundary(t *testing.T) {
	const token = "synthetic-bootstrap-token-not-for-retention"
	for _, name := range []string{"new", "unrelated-file", "existing-config", "existing-key", "non-admin", "current-denied", "missing-token"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "soda")
			if err := os.Mkdir(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			tokenPath := filepath.Join(root, "operator-input")
			if name != "missing-token" {
				if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			out := filepath.Join(dir, "dashboard.json")
			var retained string
			switch name {
			case "unrelated-file":
				retained = filepath.Join(dir, "operator-notes")
			case "existing-config":
				retained = out
			case "existing-key":
				retained = filepath.Join(dir, "grant-key")
			}
			var before os.FileInfo
			if retained != "" {
				if err := os.WriteFile(retained, []byte("preserve original bytes\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				before, _ = os.Stat(retained)
			}
			calls := make(chan string, 8)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls <- r.Method + " " + r.URL.Path
				if r.Header.Get("Authorization") != "token "+token || r.URL.RawQuery != "" {
					t.Error("bootstrap token must be confined to the Authorization header")
				}
				switch r.Method + " " + r.URL.Path {
				case "GET /api/v1/user":
					if name == "current-denied" {
						http.Error(w, token, http.StatusForbidden)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "login": "soda-tester", "is_admin": name != "non-admin"})
				case "DELETE /api/v1/user/token":
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Error("setup reached an unexpected provider endpoint")
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			// Capture the real success message, without a new production logging seam.
			log, err := os.CreateTemp(root, "stdout-")
			if err != nil {
				t.Fatal(err)
			}
			defer log.Close()
			original := os.Stdout
			var setupErr error
			func() {
				os.Stdout = log
				defer func() { os.Stdout = original }()
				setupErr = setup("https://forgejo.test/", server.URL, tokenPath, out, filepath.Join(root, "postgres"))
			}()
			output, _ := os.ReadFile(log.Name())
			if strings.Contains(string(output), token) || (setupErr != nil && strings.Contains(setupErr.Error(), token)) {
				t.Fatal("bootstrap token escaped in output/error")
			}
			success := name == "new" || name == "unrelated-file"
			if (setupErr == nil) != success {
				t.Fatalf("unexpected setup success=%v", setupErr == nil)
			}
			var gotCalls []string
			for len(calls) > 0 {
				gotCalls = append(gotCalls, <-calls)
			}
			var wantCalls []string
			if name != "existing-config" && name != "missing-token" {
				wantCalls = append(wantCalls, "GET /api/v1/user")
			}
			if success {
				wantCalls = append(wantCalls, "DELETE /api/v1/user/token")
			}
			if !reflect.DeepEqual(gotCalls, wantCalls) {
				t.Fatal("unexpected provider calls or automatic replay", gotCalls)
			}
			if retained != "" {
				data, err := os.ReadFile(retained)
				after, statErr := os.Stat(retained)
				if err != nil || statErr != nil || string(data) != "preserve original bytes\n" || !os.SameFile(before, after) || after.Mode().Perm() != 0o600 {
					t.Fatal("existing operator file changed")
				}
			}
			if name != "missing-token" {
				data, err := os.ReadFile(tokenPath)
				st, statErr := os.Stat(tokenPath)
				if err != nil || statErr != nil || string(data) != token+"\n" || st.Mode().Perm() != 0o600 {
					t.Fatal("operator input changed")
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
				if err != nil || strings.Contains(string(data), token) {
					t.Fatal("bootstrap token retained in setup output")
				}
			}
			if !success {
				if name != "existing-config" {
					if _, err := os.Stat(out); !os.IsNotExist(err) {
						t.Fatal("failed setup published configuration")
					}
				}
				return
			}
			wantFiles := 2
			if name == "unrelated-file" {
				wantFiles++
			}
			if len(entries) != wantFiles {
				t.Fatal("unexpected installed files")
			}
			c, err := config.Load(out)
			if err != nil || c.OperatorID != 42 || c.ForgejoURL != "https://forgejo.test" {
				t.Fatal("setup output did not load with native operator identity")
			}
			if key, err := config.GrantKey(c.GrantKeyFile); err != nil || len(key) != 32 {
				t.Fatal("grant key missing or invalid")
			}
			for _, path := range []string{out, c.GrantKeyFile} {
				st, err := os.Stat(path)
				if err != nil || st.Mode().Perm() != 0o600 {
					t.Fatal("setup file not restricted")
				}
			}
		})
	}
}
