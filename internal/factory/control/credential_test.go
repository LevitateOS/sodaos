package control

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCredentialPreservesEnrollment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	original := []byte(`{"tokens":{"refresh_token":"synthetic-original"}}`)
	renewed := []byte(`{"tokens":{"refresh_token":"synthetic-renewed"}}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveCredential(path, credentialSHA(original), renewed); err != nil {
		t.Fatal(err)
	}
	if err := saveCredential(path, credentialSHA(original), original); err == nil {
		t.Fatal("overwrote changed enrollment")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(renewed) {
		t.Fatal("lost maintained credential state")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatal("credential permissions widened")
	}
	if err := saveCredential(path, credentialSHA(renewed), []byte("invalid")); err == nil {
		t.Fatal("accepted invalid state")
	}
}
