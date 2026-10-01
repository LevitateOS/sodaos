package control

import (
	"bytes"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

func museConnection(t *testing.T, c *Controller) identity.Connection {
	t.Helper()
	e, err := c.StartEnrollment(t.Context(), 1, identity.Muse, "subscription")
	if err != nil {
		t.Fatal(err)
	}
	e, err = c.Enrollment(t.Context(), 1, e.ID)
	if err != nil || e.Connection == nil || e.ProviderID != identity.Muse {
		t.Fatal("provider identity lost", err)
	}
	return *e.Connection
}

func TestMuseConcurrentImmutableCustody(t *testing.T) {
	c, s, r, _ := controllerFixture(t)
	conn := museConnection(t, c)
	first, err := c.Acquire(t.Context(), museAcquireInput(conn.ID, 1))
	if err != nil {
		t.Fatal(err)
	}
	secondInput := museAcquireInput(conn.ID, 1)
	secondInput.ExecutionID = "independent-execution"
	second, err := c.Acquire(t.Context(), secondInput)
	if err != nil {
		t.Fatal("concurrent Muse use refused", err)
	}
	before, err := s.IdentityCredential(t.Context(), conn)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []identity.Lease{first, second} {
		b := registerBinding(l)
		b.ID = l.ExecutionID
		if _, err := c.Register(t.Context(), l.ID, b); err != nil {
			t.Fatal(err)
		}
	}
	// Even valid changed JSON cannot replace the shared subscription snapshot.
	if err := c.Return(t.Context(), first.ID, identity.Binding{Kind: first.Kind, ID: first.ExecutionID, Generation: first.Generation}, []byte(`{"replacement":"synthetic"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IdentityLease(t.Context(), second.ID); err != nil {
		t.Fatal("sibling lease lost", err)
	}
	if err := c.EndLease(t.Context(), 1, second.ID); err != nil {
		t.Fatal(err)
	}
	current, err := s.IdentityConnection(t.Context(), conn.ID)
	if err != nil || current.Generation != conn.Generation || current.State != identity.Ready {
		t.Fatal("immutable generation changed", err)
	}
	after, err := s.IdentityCredential(t.Context(), current)
	if err != nil || !bytes.Equal(before, after) || r.finished != 0 || r.stopped != 2 {
		t.Fatal("immutable credential captured or replaced", err)
	}
}

func TestMuseGrantRevocationRetiresOnlyGrantedLease(t *testing.T) {
	c, s, r, _ := controllerFixture(t)
	conn := museConnection(t, c)
	grant, err := c.CreateGrant(t.Context(), 1, identity.GrantRequest{ConnectionID: conn.ID, UserID: 2, ProjectID: "project", ConfirmSubscription: true, ConfirmCredentialExposure: true})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := c.Acquire(t.Context(), museAcquireInput(conn.ID, 1))
	if err != nil {
		t.Fatal(err)
	}
	delegated, err := c.Acquire(t.Context(), museAcquireInput(conn.ID, 2, "delegated-execution"))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []identity.Lease{owner, delegated} {
		if _, err := c.Register(t.Context(), l.ID, registerBinding(l)); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.RevokeGrant(t.Context(), 1, grant.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IdentityLease(t.Context(), delegated.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("revoked lease retained", err)
	}
	if _, err := s.IdentityLease(t.Context(), owner.ID); err != nil {
		t.Fatal("owner lease retired", err)
	}
	if r.stopped != 1 {
		t.Fatal("unexpected native retirement")
	}
	current, err := s.IdentityConnection(t.Context(), conn.ID)
	if err != nil || current.State != identity.Ready {
		t.Fatal("unrelated subscription disabled", err)
	}
}

func TestMuseRejectionAndFailedDisconnectRetireEveryLease(t *testing.T) {
	for _, reject := range []bool{true, false} {
		c, s, r, _ := controllerFixture(t)
		conn := museConnection(t, c)
		var leases []identity.Lease
		for _, execID := range []string{"rejection-a", "rejection-b"} {
			l, err := c.Acquire(t.Context(), museAcquireInput(conn.ID, 1, execID))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Register(t.Context(), l.ID, registerBinding(l)); err != nil {
				t.Fatal(err)
			}
			leases = append(leases, l)
		}
		r.stopErr = errors.New("retirement unconfirmed")
		var err error
		if reject {
			err = c.Reject(t.Context(), leases[0].ID, registerBinding(leases[0]))
		} else {
			err = c.Revoke(t.Context(), 1, conn.ID)
		}
		if !errors.Is(err, identity.ErrUncertain) || r.stopped != 2 {
			t.Fatal("did not attempt every lease", err, r.stopped)
		}
		if _, err := c.Acquire(t.Context(), museAcquireInput(conn.ID, 1)); !errors.Is(err, identity.ErrUncertain) {
			t.Fatal("failed retirement remained usable", err)
		}
		for _, l := range leases {
			if _, err := s.IdentityLease(t.Context(), l.ID); err != nil {
				t.Fatal("uncertain execution forgotten", err)
			}
		}
	}
}

func museAcquireInput(id string, actor int64, exec ...string) identity.AcquireRequest {
	in := acquireInput(id, actor, exec...)
	in.ProviderID = identity.Muse
	return in
}
