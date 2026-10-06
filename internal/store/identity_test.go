package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestIdentityConnectionRoundTripAndValidation(t *testing.T) {
	s, _ := postgresFixture(t, bytes.Repeat([]byte{4}, 32))
	ctx := t.Context()
	c := identity.Connection{ProviderID: identity.Codex, ID: "subscription", OwnerID: 1, Generation: 1, State: identity.Ready}
	credential := []byte(`{"tokens":{"refresh_token":"synthetic-secret"}}`)
	if err := s.SeedIdentityConnection(ctx, c, credential); err != nil {
		t.Fatal(err)
	}
	var raw []byte
	if err := s.queryRow(ctx, `SELECT data FROM identity_connections WHERE id=?`, c.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var got identity.Connection
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != c.ID || got.Generation != 1 || got.State != identity.Ready {
		t.Fatal("saved connection unreadable")
	}
	var encrypted []byte
	if err := s.queryRow(ctx, `SELECT credential FROM identity_connections WHERE id=?`, c.ID).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte("synthetic-secret")) {
		t.Fatal("plaintext persisted")
	}
	wrong := c
	wrong.Generation++
	var err error
	if _, err = s.grants.open(encrypted, identityBinding(wrong)); err == nil {
		t.Fatal("ciphertext not bound to connection generation")
	}
	bad := c
	bad.ProviderID = "unknown"
	if err = s.SeedIdentityConnection(ctx, bad, credential); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("unknown provider admitted", err)
	}
	empty := c
	empty.ID = "empty-credential"
	if err = s.SeedIdentityConnection(ctx, empty, []byte(`not json`)); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("malformed credential admitted", err)
	}
	unkeyed, _ := postgresFixture(t, nil)
	if err = unkeyed.SeedIdentityConnection(ctx, c, credential); !errors.Is(err, ErrGrantKey) {
		t.Fatal("unkeyed save admitted", err)
	}
}

func TestIdentityGrantRoundTrip(t *testing.T) {
	s, _ := postgresFixture(t, bytes.Repeat([]byte{4}, 32))
	ctx := t.Context()
	c := identity.Connection{ProviderID: identity.Codex, ID: "subscription", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := s.SeedIdentityConnection(ctx, c, []byte(`{"tokens":{"refresh_token":"synthetic"}}`)); err != nil {
		t.Fatal(err)
	}
	g := identity.Grant{ID: "grant-1", ConnectionID: c.ID, UserID: 2, ProjectID: "project"}
	if err := s.SeedIdentityGrant(ctx, g); err != nil {
		t.Fatal(err)
	}
	var raw []byte
	if err := s.queryRow(ctx, `SELECT data FROM identity_grants WHERE id=?`, g.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var got identity.Grant
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.ConnectionID != c.ID || got.UserID != 2 {
		t.Fatal("saved grant unreadable")
	}
}

func TestIdentityAuditTrailIsImmutableAndCredentialFree(t *testing.T) {
	s, _ := postgresFixture(t, bytes.Repeat([]byte{4}, 32))
	ctx := t.Context()
	c := identity.Connection{ProviderID: identity.Codex, ID: "subscription", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := s.SeedIdentityConnection(ctx, c, []byte(`{"tokens":{"refresh_token":"synthetic-secret"}}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedIdentityGrant(ctx, identity.Grant{ID: "grant-1", ConnectionID: c.ID, UserID: 2, ProjectID: "project"}); err != nil {
		t.Fatal(err)
	}
	var actions string
	if err := s.queryRow(ctx, `SELECT string_agg(data->>'action', ',' ORDER BY id) FROM identity_events WHERE connection_id=?`, c.ID).Scan(&actions); err != nil || actions != "connected,grant_created" {
		t.Fatal("audit trail incomplete", actions, err)
	}
	if _, err := s.exec(ctx, `UPDATE identity_events SET data='{}'`); err == nil {
		t.Fatal("audit mutated")
	}
	if _, err := s.exec(ctx, `DELETE FROM identity_events`); err == nil {
		t.Fatal("audit deleted")
	}
	var leaked int
	if err := s.queryRow(ctx, `SELECT count(*) FROM identity_events WHERE strpos(data::text,'synthetic-secret')>0`).Scan(&leaked); err != nil || leaked != 0 {
		t.Fatal("credentials entered audit", err)
	}
}

func TestIdentityAuditFailureRollsBackSave(t *testing.T) {
	s, _ := postgresFixture(t, bytes.Repeat([]byte{4}, 32))
	ctx := t.Context()
	c := identity.Connection{ProviderID: identity.Codex, ID: "subscription", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := s.SeedIdentityConnection(ctx, c, []byte(`{"tokens":{"refresh_token":"synthetic"}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE OR REPLACE FUNCTION fail_identity_audit() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'audit unavailable'; END $$ LANGUAGE plpgsql`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER failed_identity_audit BEFORE INSERT ON identity_events FOR EACH ROW EXECUTE FUNCTION fail_identity_audit()`); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedIdentityGrant(ctx, identity.Grant{ID: "grant-1", ConnectionID: c.ID, UserID: 2, ProjectID: "project"}); err == nil {
		t.Fatal("unaudited grant committed")
	}
	var raw []byte
	if err := s.queryRow(ctx, `SELECT data FROM identity_grants WHERE id=?`, "grant-1").Scan(&raw); err != ErrNotFound {
		t.Fatal("audit failure left committed grant", err)
	}
}
