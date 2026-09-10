package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/config"
)

func TestSetupBootstrapCredentialBoundary(t *testing.T) {
	const token = "synthetic-bootstrap-token-not-for-retention"
	const secret = "synthetic-oauth-secret"
	for _, name := range []string{"new", "legacy-file", "existing-config", "existing-secret", "non-admin", "current-denied", "application-denied", "malformed-application", "incomplete-application", "missing-token"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "soda")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			tokenPath := filepath.Join(root, "operator-input")
			if name != "missing-token" {
				if err := os.WriteFile(tokenPath, []byte(token+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			out := filepath.Join(dir, "dashboard.json")
			var retained string
			switch name {
			case "legacy-file":
				retained = filepath.Join(dir, "admin-token")
			case "existing-config":
				retained = out
			case "existing-secret":
				retained = filepath.Join(dir, "oauth-secret")
			}
			var before os.FileInfo
			if retained != "" {
				if err := os.WriteFile(retained, []byte("preserve original bytes\n"), 0600); err != nil {
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
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "login": "operator", "is_admin": name != "non-admin"})
				case "POST /api/v1/user/applications/oauth2":
					body, _ := io.ReadAll(r.Body)
					var got map[string]any
					if json.Unmarshal(body, &got) != nil || !reflect.DeepEqual(got, map[string]any{"name": "SodaOS dashboard", "redirect_uris": []any{"https://forgejo.test/-/soda/oauth/callback"}, "confidential_client": true}) {
						t.Error("unexpected OAuth application request")
					}
					if strings.Contains(string(body), token) {
						t.Error("token entered application payload")
					}
					switch name {
					case "application-denied":
						http.Error(w, token, http.StatusForbidden)
					case "malformed-application":
						_, _ = io.WriteString(w, token)
					case "incomplete-application":
						_, _ = io.WriteString(w, `{"client_id":"app"}`)
					default:
						w.WriteHeader(http.StatusCreated)
						_ = json.NewEncoder(w).Encode(map[string]string{"client_id": "app", "client_secret": secret})
					}
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
				setupErr = setup("https://forgejo.test/", server.URL, tokenPath, out)
			}()
			output, _ := os.ReadFile(log.Name())
			if strings.Contains(string(output), token) || (setupErr != nil && strings.Contains(setupErr.Error(), token)) {
				t.Fatal("bootstrap token escaped in output/error")
			}
			success := name == "new" || name == "legacy-file"
			if (setupErr == nil) != success {
				t.Fatalf("unexpected setup success=%v", setupErr == nil)
			}
			if name == "existing-secret" && !strings.Contains(setupErr.Error(), "OAuth application created, but credential write failed; inspect Forgejo applications before retrying") {
				t.Fatal("lost uncertain provider outcome guidance")
			}
			var gotCalls []string
			for len(calls) > 0 {
				gotCalls = append(gotCalls, <-calls)
			}
			var wantCalls []string
			if name != "existing-config" && name != "missing-token" {
				wantCalls = append(wantCalls, "GET /api/v1/user")
				if name != "non-admin" && name != "current-denied" {
					wantCalls = append(wantCalls, "POST /api/v1/user/applications/oauth2")
				}
			}
			if !reflect.DeepEqual(gotCalls, wantCalls) {
				t.Fatal("unexpected provider calls or automatic replay", gotCalls)
			}
			if retained != "" {
				data, err := os.ReadFile(retained)
				after, statErr := os.Stat(retained)
				if err != nil || statErr != nil || string(data) != "preserve original bytes\n" || !os.SameFile(before, after) || after.Mode().Perm() != 0600 {
					t.Fatal("existing operator file changed")
				}
			}
			if name != "missing-token" {
				data, err := os.ReadFile(tokenPath)
				st, statErr := os.Stat(tokenPath)
				if err != nil || statErr != nil || string(data) != token+"\n" || st.Mode().Perm() != 0600 {
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
			wantFiles := 3
			if name == "legacy-file" {
				wantFiles++
			}
			if len(entries) != wantFiles {
				t.Fatal("unexpected installed files")
			}
			data, _ := os.ReadFile(out)
			if strings.Contains(string(data), "admin_token_file") {
				t.Fatal("new config retained bootstrap reference")
			}
			c, err := config.Load(out)
			if err != nil || c.OperatorID != 42 || c.OAuthClientID != "app" || c.OAuthCallbackURL() != "https://forgejo.test/-/soda/oauth/callback" {
				t.Fatal("setup output did not load with original identity/callback")
			}
			if key, err := config.GrantKey(c.GrantKeyFile); err != nil || len(key) != 32 {
				t.Fatal("grant key missing or invalid")
			}
			if value, err := config.Secret(c.OAuthSecretFile); err != nil || value != secret {
				t.Fatal("OAuth secret not preserved")
			}
			for _, path := range []string{out, c.OAuthSecretFile, c.GrantKeyFile} {
				st, err := os.Stat(path)
				if err != nil || st.Mode().Perm() != 0600 {
					t.Fatal("setup file not restricted")
				}
			}
		})
	}
}
