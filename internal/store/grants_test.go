package store

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestIdentityEncryptionKeyMustSurviveRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity.db")
	key := bytes.Repeat([]byte{7}, 32)
	s, err := OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	c := identity.Connection{ProviderID: identity.Codex, ID: "account", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err = s.IdentitySaveConnection(t.Context(), c, []byte(`{"tokens":{"refresh_token":"synthetic-secret"}}`)); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if opened, err := Open(path); !errors.Is(err, ErrGrantKey) {
		if opened != nil {
			_ = opened.Close()
		}
		t.Fatal("missing encryption key admitted", err)
	}
	if opened, err := OpenEncrypted(path, bytes.Repeat([]byte{8}, 32)); !errors.Is(err, ErrGrantKey) {
		if opened != nil {
			_ = opened.Close()
		}
		t.Fatal("incorrect encryption key admitted", err)
	}
	s, err = OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	credential, err := s.IdentityCredential(t.Context(), c)
	if err != nil || !bytes.Contains(credential, []byte("synthetic-secret")) {
		t.Fatal("identity credential did not survive restart", err)
	}
	if _, err = s.db.Exec(`DELETE FROM grant_key_check`); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if opened, err := OpenEncrypted(path, key); !errors.Is(err, ErrGrantKey) {
		if opened != nil {
			_ = opened.Close()
		}
		t.Fatal("encrypted identity admitted without key validation marker", err)
	}
}
