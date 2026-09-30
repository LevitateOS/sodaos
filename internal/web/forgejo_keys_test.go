package web

import (
	"crypto/ed25519"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"golang.org/x/crypto/ssh"
)

func TestOwnForgejoKeyReviewDoesNotSaveOrInstallAndIsBounded(t *testing.T) {
	public, err := ssh.NewPublicKey(ed25519.NewKeyFromSeed(make([]byte, 32)).Public())
	if err != nil {
		t.Fatal(err)
	}
	key := string(ssh.MarshalAuthorizedKey(public))
	for _, mode := range []string{"valid", "invalid-key", "unavailable", "invalid-id", "oversized", "null"} {
		t.Run(mode, func(t *testing.T) {
			s := apiTestServer(t)
			keyCalls := 0
			callback := func(in extensions.CallbackRequest) extensions.CallbackResponse {
				keyCalls++
				if in.Operation != extensions.OperationPublicSSHKeys {
					t.Error("unexpected native callback", in.Operation)
					return extensions.CallbackResponse{ErrorCode: "invalid_operation"}
				}
				if mode == "unavailable" {
					return extensions.CallbackResponse{ErrorCode: "unavailable"}
				}
				if mode == "null" {
					return extensions.CallbackResponse{}
				}
				count := 11
				if mode == "oversized" {
					count = 101
				}
				keys := make([]extensions.PublicKey, count)
				for i := range keys {
					keys[i] = extensions.PublicKey{ID: fmt.Sprint(i + 1), Key: key}
				}
				if mode == "invalid-id" {
					keys[10].ID = "0"
				}
				if mode == "invalid-key" {
					keys[10].Key = "PRIVATE KEY"
				}
				return extensions.CallbackResponse{PublicKeys: keys}
			}
			w := httptest.NewRecorder()
			nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/me/forgejo-keys?page=2", "", "alice"), callback)
			if mode == "valid" {
				if w.Code != 200 || !strings.Contains(w.Body.String(), ssh.FingerprintSHA256(public)) || !strings.Contains(w.Body.String(), `"page":2`) {
					t.Fatal(w.Code, w.Body.String())
				}
			} else if w.Code < 400 || strings.Contains(w.Body.String(), key) {
				t.Fatal(mode, w.Code, w.Body.String())
			}
			if keyCalls != 1 {
				t.Fatal("public-key lookup count", keyCalls)
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
	s := apiTestServer(t)
	callback := func(extensions.CallbackRequest) extensions.CallbackResponse {
		calls++
		return extensions.CallbackResponse{ErrorCode: "unexpected"}
	}
	for _, query := range []string{"page=0", "page=9", "page=1&page=1", "fingerprint=SHA256:abc", "username=bob", "page=01", "page="} {
		w := httptest.NewRecorder()
		nativeAPIServeWithCallback(t, s, w, apiTestRequest("GET", "/api/me/forgejo-keys?"+query, "", "alice"), callback)
		if w.Code != 400 || calls != 0 {
			t.Fatal(query, w.Code, calls)
		}
	}
}
