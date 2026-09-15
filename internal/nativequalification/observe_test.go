package nativequalification

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestFinalizationLockUsesExactNativeDeployment(t *testing.T) {
	d := deployment{Osname: "fedora-coreos", Checksum: strings.Repeat("a", 64), Serial: 2, Staged: true}
	locked := "  fedora-coreos " + d.Checksum + ".2 (finalization locked)\n"
	if !finalizationLocked(locked, d) {
		t.Fatal("native lock not recognized")
	}
	for _, text := range []string{strings.ReplaceAll(locked, ".2", ".0"), strings.ReplaceAll(locked, "a", "b"), strings.ReplaceAll(locked, " (finalization locked)", ""), "*" + strings.TrimSpace(locked)} {
		if finalizationLocked(text, d) {
			t.Fatal("different or booted deployment accepted", text)
		}
	}
}

func TestConsoleSSHKeyComesFromNativeSerial(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
	if got := consoleHostKey([]byte("boot banner\r\n" + want + " soda-tester\r\nP9_PUBLIC_PADDING\n")); got != want {
		t.Fatalf("got %q", got)
	}
	if consoleHostKey([]byte("ssh-ed25519 invalid\n")) != "" {
		t.Fatal("malformed key accepted")
	}
}

func TestRequiredContentFaultLeavesNativeRegistryInControl(t *testing.T) {
	f := fixture{blob: "sha256:test", blocked: true}
	upstreamCalls := 0
	handler := f.registryHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { upstreamCalls++; w.WriteHeader(http.StatusOK) }))
	for _, tc := range []struct {
		path   string
		status int
	}{{"/v2/repo/manifests/candidate", 200}, {"/v2/repo/blobs/sha256:test", 503}, {"/v2/repo/blobs/sha256:other", 200}} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("GET", tc.path, nil))
		if response.Code != tc.status {
			t.Fatal(tc.path, response.Code)
		}
	}
	if upstreamCalls != 2 || f.hits != 1 || f.payloadRequests != 2 {
		t.Fatal("fault observation differs")
	}
	f.blocked = false
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/v2/repo/blobs/sha256:test", nil))
	if response.Code != 200 || upstreamCalls != 3 {
		t.Fatal("native serving not restored")
	}
}

func TestLaterWritesMustPreserveEarlierState(t *testing.T) {
	before := map[string]any{"project": "same", "forgejo_repository": "same", "machine_settings_public_keys": "same", "schema": 10, "project_files": map[string]any{"a.txt": "a"}, "forgejo_ref": "a", "user": "profile-a"}
	after := map[string]any{"project": "same", "forgejo_repository": "same", "machine_settings_public_keys": "same", "schema": 10, "project_files": map[string]any{"a.txt": "a", "b.txt": "b"}, "forgejo_ref": "b", "user": "profile-b"}
	if err := laterState(before, after); err != nil {
		t.Fatal(err)
	}
	after["user"] = "profile-a"
	if laterState(before, after) == nil {
		t.Fatal("no-op profile write accepted")
	}
	after["user"] = "profile-b"
	after["project_files"] = map[string]any{"a.txt": "overwritten", "b.txt": "b"}
	if laterState(before, after) == nil {
		t.Fatal("earlier data loss accepted")
	}
}
