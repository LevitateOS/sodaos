package store

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestIdentityCipherBindingAndLeaseReturnAtomicity(t *testing.T) {
	s, err := OpenEncrypted(filepath.Join(t.TempDir(), "identity.db"), bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ctx := t.Context()
	c := identity.Connection{ID: "subscription", OwnerID: 1, Generation: 1, State: identity.Ready}
	credential := []byte(`{"tokens":{"refresh_token":"synthetic-secret"}}`)
	if err = s.IdentitySaveConnection(ctx, c, credential); err != nil {
		t.Fatal(err)
	}
	var encrypted []byte
	if err = s.db.QueryRow(`SELECT credential FROM identity_connections WHERE id=?`, c.ID).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte("synthetic-secret")) {
		t.Fatal("plaintext persisted")
	}
	wrong := c
	wrong.Generation++
	if _, err = s.grants.open(encrypted, identityBinding(wrong)); err == nil {
		t.Fatal("ciphertext not bound to connection generation")
	}
	l := identity.Lease{ID: "lease", ConnectionID: c.ID, Generation: 1, ActorID: 1, ExecutionID: "execution", Kind: identity.Factory, Deadline: time.Now().Add(time.Hour)}
	if err = s.IdentityReserve(ctx, l); err != nil {
		t.Fatal(err)
	}
	stale := l
	stale.ID = "wrong-lease"
	if err = s.IdentityReturn(ctx, stale, []byte(`{"tokens":{"refresh_token":"new"}}`)); err == nil {
		t.Fatal("unrelated return admitted")
	}
	current, err := s.IdentityConnection(ctx, c.ID)
	if err != nil || current.Generation != 1 {
		t.Fatal("failed return changed generation", err)
	}
	if _, err = s.IdentityLease(ctx, l.ID); err != nil {
		t.Fatal("failed return released active lease", err)
	}
	if err = s.IdentityReturn(ctx, l, []byte(`{"tokens":{"refresh_token":"new"}}`)); err != nil {
		t.Fatal(err)
	}
	if err = s.IdentityReturn(ctx, l, credential); err == nil {
		t.Fatal("stale generation admitted")
	}
	events, err := s.IdentityEvents(ctx, 1, c.ID)
	if err != nil || len(events) != 3 {
		t.Fatal("live lease deletion lost immutable attribution", err)
	}
	returned := events[0]
	if returned.Action != "returned" || returned.ExecutionID != l.ExecutionID || returned.ActorID != 1 || returned.LeaseID != l.ID || returned.Generation != l.Generation {
		t.Fatal("incorrect execution attribution")
	}
	other, err := s.IdentityEvents(ctx, 2, c.ID)
	if err != nil || len(other) != 0 {
		t.Fatal("other owner read audit", err)
	}
	if _, err = s.db.Exec(`UPDATE identity_events SET data='{}'`); err == nil {
		t.Fatal("audit mutated")
	}
	if _, err = s.db.Exec(`DELETE FROM identity_events`); err == nil {
		t.Fatal("audit deleted")
	}
	var leaked int
	if err = s.db.QueryRow(`SELECT count(*) FROM identity_events WHERE instr(data,'synthetic-secret')>0`).Scan(&leaked); err != nil || leaked != 0 {
		t.Fatal("credentials entered audit", err)
	}
}

func TestIdentityAuditFailureRollsBackAdmission(t *testing.T) {
	s, err := OpenEncrypted(filepath.Join(t.TempDir(), "identity.db"), bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	ctx := t.Context()
	c := identity.Connection{ID: "subscription", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err = s.IdentitySaveConnection(ctx, c, []byte(`{"tokens":{"refresh_token":"synthetic"}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`CREATE TRIGGER failed_identity_audit BEFORE INSERT ON identity_events BEGIN SELECT RAISE(ABORT,'audit unavailable'); END;`); err != nil {
		t.Fatal(err)
	}
	l := identity.Lease{ID: "lease", ConnectionID: c.ID, Generation: 1, ActorID: 1, ExecutionID: "execution", Kind: identity.Factory, Deadline: time.Now().Add(time.Hour)}
	if err = s.IdentityReserve(ctx, l); err == nil {
		t.Fatal("unaudited credential reservation committed")
	}
	if _, err = s.IdentityLease(ctx, l.ID); err != ErrNotFound {
		t.Fatal("audit failure left committed reservation", err)
	}
}
