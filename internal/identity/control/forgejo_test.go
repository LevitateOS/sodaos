package control

import (
	"errors"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestForgejoRefusesCredentialDelivery(t *testing.T) {
	c, s, runtime, _ := controllerFixture(t)
	connection := identity.Connection{ProviderID: identity.Forgejo, ID: "git-account", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := s.IdentitySaveConnection(t.Context(), connection, []byte(`{"access":"synthetic-private"}`)); err != nil {
		t.Fatal(err)
	}
	request := identity.AcquireRequest{ProviderID: identity.Forgejo, ActorID: 1, ConnectionID: connection.ID, ExecutionID: "git-execution", Kind: identity.Terminal, ProjectID: "project", Deadline: time.Now().Add(time.Hour)}
	if _, err := c.Acquire(t.Context(), request); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("Git account admitted to raw credential acquisition")
	}
	lease := identity.Lease{ID: "git-lease", ProviderID: identity.Forgejo, ConnectionID: connection.ID, Generation: 1, ActorID: 1, ProjectID: "project", ExecutionID: "git-execution", Kind: identity.Terminal, Deadline: request.Deadline}
	if err := s.IdentityReserve(t.Context(), lease); err != nil {
		t.Fatal(err)
	}
	binding := identity.Binding{Kind: identity.Terminal, ID: "git-unit", Project: "project", Login: "soda-tester", Generation: 1}
	delivery, err := c.Register(t.Context(), lease.ID, binding)
	if !errors.Is(err, identity.ErrDenied) || len(delivery.Credential) != 0 || runtime.validated != 0 {
		t.Fatal("Git account entered runtime credential delivery")
	}
}

func TestForgejoAccountCannotBeSharedOrEnrolledTwice(t *testing.T) {
	c, s, _, _ := controllerFixture(t)
	c.providers[identity.Forgejo] = testProvider{}
	connection := identity.Connection{ProviderID: identity.Forgejo, ID: "git-account", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := s.IdentitySaveConnection(t.Context(), connection, []byte(`{"access":"synthetic-private"}`)); err != nil {
		t.Fatal(err)
	}
	grant := identity.GrantRequest{ConnectionID: connection.ID, UserID: 2, ProjectID: "project", ConfirmSubscription: true, ConfirmCredentialExposure: true}
	if _, err := c.CreateGrant(t.Context(), 1, grant); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("another user could adopt the owner's Forgejo identity")
	}
	if _, err := c.StartEnrollment(t.Context(), 1, identity.Forgejo, "duplicate"); !errors.Is(err, identity.ErrBusy) {
		t.Fatal("duplicate native refresh custodian admitted")
	}
	if err := s.IdentityState(t.Context(), connection, identity.Revoked); err != nil {
		t.Fatal(err)
	}
	if _, err := c.StartEnrollment(t.Context(), 1, identity.Forgejo, "reconnection"); err != nil {
		t.Fatal("retired connection prevented native reconnection", err)
	}
	if _, err := c.StartEnrollment(t.Context(), 1, identity.Forgejo, "unretained duplicate"); !errors.Is(err, identity.ErrBusy) {
		t.Fatal("unretained completed enrollment admitted another refresh custodian")
	}
}
