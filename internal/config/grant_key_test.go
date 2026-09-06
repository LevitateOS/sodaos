package config

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestGrantKeyRestrictedAndExact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "key")
	value := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	if _, err := GrantKey(path); err == nil {
		t.Fatal("missing key accepted")
	}
	if err := os.WriteFile(path, []byte(value+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	key, err := GrantKey(path)
	if err != nil || len(key) != 32 {
		t.Fatal(err)
	}
	if err = os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = GrantKey(path); err == nil {
		t.Fatal("public key file accepted")
	}
	if err = os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("not-base64"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = GrantKey(path); err == nil {
		t.Fatal("malformed key accepted")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err = os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err = GrantKey(link); err == nil {
		t.Fatal("symlink accepted")
	}
}
