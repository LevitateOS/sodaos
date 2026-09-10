package web

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestOwnForgejoKeyReviewDoesNotSaveOrInstallAndIsBounded(t *testing.T) {
	public, err := ssh.NewPublicKey(ed25519.NewKeyFromSeed(make([]byte, 32)).Public())
	if err != nil {
		t.Fatal(err)
	}
	key := string(ssh.MarshalAuthorizedKey(public))
	for _, mode := range []string{"valid", "foreign-owner", "deploy", "missing-type", "invalid-key", "unavailable", "identity", "duplicate", "null"} {
		t.Run(mode, func(t *testing.T) {
			keyCalls := 0
			s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.Header.Get("Authorization") != "token acting-alice" {
					t.Error("unexpected mutation or actor")
				}
				if r.URL.Path == "/api/v1/user" {
					id := 1
					if mode == "identity" {
						id = 2
					}
					fmt.Fprintf(w, `{"id":%d,"login":"alice"}`, id)
					return
				}
				keyCalls++
				if r.URL.Path != "/api/v1/user/keys" || r.URL.RawQuery != "limit=10&page=2" {
					t.Error("unexpected key selector", r.URL.String())
				}
				owner := 1
				kind := "user"
				value := key
				if mode == "foreign-owner" {
					owner = 2
				}
				if mode == "deploy" {
					kind = "deploy"
				}
				if mode == "missing-type" {
					kind = ""
				}
				if mode == "invalid-key" {
					value = "PRIVATE KEY"
				}
				if mode == "unavailable" {
					w.WriteHeader(503)
					return
				}
				if mode == "null" {
					fmt.Fprint(w, "null")
					return
				}
				row := map[string]any{"id": 7, "user": map[string]int{"id": owner}, "key_type": kind, "key": value, "title": "Laptop <not HTML>"}
				rows := []any{row}
				if mode == "duplicate" {
					rows = append(rows, row)
				}
				_ = json.NewEncoder(w).Encode(rows)
			})
			w := httptest.NewRecorder()
			s.ServeHTTP(w, apiTestRequest("GET", "/api/me/forgejo-keys?page=2", "", "alice"))
			if mode == "valid" {
				if w.Code != 200 || !strings.Contains(w.Body.String(), ssh.FingerprintSHA256(public)) || !strings.Contains(w.Body.String(), `"page":2`) {
					t.Fatal(w.Code, w.Body.String())
				}
			} else if w.Code < 400 || strings.Contains(w.Body.String(), key) {
				t.Fatal(mode, w.Code, w.Body.String())
			}
			if mode == "identity" && keyCalls != 0 {
				t.Fatal("mismatched actor listed keys")
			}
			keys, err := s.Store.Keys(t.Context(), 1)
			if err != nil || len(keys) != 0 {
				t.Fatal("review mutated Soda preferences")
			}
		})
	}
}
func TestForgejoKeyPickerRejectsGlobalQueries(t *testing.T) {
	calls := 0
	s := grantedTestServer(t, func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(503) })
	for _, query := range []string{"page=0", "page=9", "page=1&page=1", "fingerprint=SHA256:abc", "username=bob", "page=01", "page="} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, apiTestRequest("GET", "/api/me/forgejo-keys?"+query, "", "alice"))
		if w.Code != 400 || calls != 0 {
			t.Fatal(query, w.Code, calls)
		}
	}
}
