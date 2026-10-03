package store

import (
	"bytes"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestIdentityCipherBindingAndLeaseReturnAtomicity(t *testing.T) {
	s, _ := postgresFixture(t, bytes.Repeat([]byte{4}, 32))
	ctx := t.Context()
	var err error
	c := identity.Connection{ProviderID: identity.Codex, ID: "subscription", OwnerID: 1, Generation: 1, State: identity.Ready}
	credential := []byte(`{"tokens":{"refresh_token":"synthetic-secret"}}`)
	if err = s.IdentitySaveConnection(ctx, c, credential); err != nil {
		t.Fatal(err)
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
	if _, err = s.grants.open(encrypted, identityBinding(wrong)); err == nil {
		t.Fatal("ciphertext not bound to connection generation")
	}
	l := identity.Lease{ProviderID: identity.Codex, ID: "lease", ConnectionID: c.ID, Generation: 1, ActorID: 1, ExecutionID: "execution", Kind: identity.Factory, Deadline: time.Now().Add(time.Hour)}
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

func TestIdentityExecutionAdmitsOnceAndGuardsDigest(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := t.Context()
	var err error
	first := identity.Execution{Kind: identity.Factory, ExecutionID: "run", Digest: "digest", State: identity.ExecutionPending}
	if _, created, err := s.IdentityAdmitExecution(ctx, first); err != nil || !created {
		t.Fatal("execution admission failed", err)
	}
	repeated, created, err := s.IdentityAdmitExecution(ctx, first)
	if err != nil || created || repeated.Digest != "digest" {
		t.Fatal("execution re-admission failed", err)
	}
	live := first
	live.State, live.LeaseID = identity.ExecutionLive, "lease"
	if err = s.IdentityObserveExecution(ctx, live); err != nil {
		t.Fatal(err)
	}
	changed := live
	changed.Digest = "other"
	if err = s.IdentityObserveExecution(ctx, changed); err == nil {
		t.Fatal("execution digest mutated")
	}
	terminal := live
	terminal.State = identity.ExecutionTerminal
	if err = s.IdentityObserveExecution(ctx, terminal); err != nil {
		t.Fatal(err)
	}
	if err = s.IdentityObserveExecution(ctx, live); err == nil {
		t.Fatal("terminal execution revived")
	}
	if _, err = s.IdentityExecution(ctx, identity.Factory, "absent"); err != ErrNotFound {
		t.Fatal("missing execution misreported", err)
	}
}

func TestIdentityAuditFailureRollsBackAdmission(t *testing.T) {
	s, _ := postgresFixture(t, bytes.Repeat([]byte{4}, 32))
	ctx := t.Context()
	c := identity.Connection{ProviderID: identity.Codex, ID: "subscription", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := s.IdentitySaveConnection(ctx, c, []byte(`{"tokens":{"refresh_token":"synthetic"}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE OR REPLACE FUNCTION fail_identity_audit() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'audit unavailable'; END $$ LANGUAGE plpgsql`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER failed_identity_audit BEFORE INSERT ON identity_events FOR EACH ROW EXECUTE FUNCTION fail_identity_audit()`); err != nil {
		t.Fatal(err)
	}
	l := identity.Lease{ProviderID: identity.Codex, ID: "lease", ConnectionID: c.ID, Generation: 1, ActorID: 1, ExecutionID: "execution", Kind: identity.Factory, Deadline: time.Now().Add(time.Hour)}
	if err := s.IdentityReserve(ctx, l); err == nil {
		t.Fatal("unaudited credential reservation committed")
	}
	if _, err := s.IdentityLease(ctx, l.ID); err != ErrNotFound {
		t.Fatal("audit failure left committed reservation", err)
	}
}
