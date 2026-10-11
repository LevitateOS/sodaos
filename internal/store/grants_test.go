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
	if err := s.db.QueryRowContext(t.Context(), `SELECT credential FROM identity_connections WHERE id=$1`, c.ID).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	credential, err := s.grants.open(encrypted, identityBinding(c))
	if err != nil || !bytes.Contains(credential, []byte("synthetic-secret")) {
		t.Fatal("identity credential did not survive restart", err)
	}
	if _, err = s.db.ExecContext(t.Context(), `DELETE FROM grant_key_check`); err != nil {
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

func TestDashboardStoreDoesNotReadOrMutateBrokerKeyState(t *testing.T) {
	key := bytes.Repeat([]byte{9}, 32)
	broker, dsn := postgresFixture(t, key)
	connection := identity.Connection{ProviderID: identity.Codex, ID: "dashboard-boundary", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := broker.SeedIdentityConnection(t.Context(), connection, []byte(`{"tokens":{"refresh_token":"synthetic"}}`)); err != nil {
		t.Fatal(err)
	}
	var before []byte
	if err := broker.db.QueryRowContext(t.Context(), `SELECT ciphertext FROM grant_key_check WHERE id=1`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err := broker.Close(); err != nil {
		t.Fatal(err)
	}

	dashboard, err := OpenDashboard(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	if dashboard.grants != nil {
		t.Fatal("dashboard received broker encryption authority")
	}
	if err := dashboard.SeedIdentityConnection(t.Context(), connection, []byte(`{"tokens":{"refresh_token":"changed"}}`)); !errors.Is(err, ErrGrantKey) {
		t.Fatalf("dashboard fixture helper accepted credential write: %v", err)
	}
	var after []byte
	if err := dashboard.db.QueryRowContext(t.Context(), `SELECT ciphertext FROM grant_key_check WHERE id=1`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("dashboard open changed broker key marker")
	}
	var stored []byte
	if err := dashboard.db.QueryRowContext(t.Context(), `SELECT credential FROM identity_connections WHERE id=$1`, connection.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stored, []byte("synthetic")) || bytes.Contains(stored, []byte("changed")) {
		t.Fatal("dashboard exposed or replaced encrypted broker credential")
	}
}
