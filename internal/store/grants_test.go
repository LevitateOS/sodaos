package store

import (
	"bytes"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestIdentityEncryptionKeyMustSurviveRestart(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	s, dsn := postgresFixture(t, key)
	c := identity.Connection{ProviderID: identity.Codex, ID: "account", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := s.SeedIdentityConnection(t.Context(), c, []byte(`{"tokens":{"refresh_token":"synthetic-secret"}}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if opened, err := Open(dsn); !errors.Is(err, ErrGrantKey) {
		if opened != nil {
			_ = opened.Close()
		}
		t.Fatal("missing encryption key admitted", err)
	}
	if opened, err := OpenEncrypted(dsn, bytes.Repeat([]byte{8}, 32)); !errors.Is(err, ErrGrantKey) {
		if opened != nil {
			_ = opened.Close()
		}
		t.Fatal("incorrect encryption key admitted", err)
	}
	s, err := OpenEncrypted(dsn, key)
	if err != nil {
		t.Fatal(err)
	}
	var encrypted []byte
	if err := s.queryRow(t.Context(), `SELECT credential FROM identity_connections WHERE id=?`, c.ID).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	credential, err := s.grants.open(encrypted, identityBinding(c))
	if err != nil || !bytes.Contains(credential, []byte("synthetic-secret")) {
		t.Fatal("identity credential did not survive restart", err)
	}
	if _, err = s.exec(t.Context(), `DELETE FROM grant_key_check`); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if opened, err := OpenEncrypted(dsn, key); !errors.Is(err, ErrGrantKey) {
		if opened != nil {
			_ = opened.Close()
		}
		t.Fatal("encrypted identity admitted without key validation marker", err)
	}
}
