package store

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestForgejoReservationsRemainOwnerAndRepositoryBound(t *testing.T) {
	s, err := OpenEncrypted(filepath.Join(t.TempDir(), "git.db"), bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	c := identity.Connection{ProviderID: identity.Forgejo, ID: "forgejo-account", OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := s.IdentitySaveConnection(t.Context(), c, []byte(`{"access":"synthetic"}`)); err != nil {
		t.Fatal(err)
	}
	l := identity.Lease{ID: "git-a", ProviderID: identity.Forgejo, ConnectionID: c.ID, Generation: 1, ActorID: 1, ProjectID: "project", RepositoryID: 7, ExecutionID: "execution-a", Kind: identity.Terminal, Deadline: time.Now().Add(time.Hour)}
	if err := s.IdentityReserve(t.Context(), l); err != nil {
		t.Fatal(err)
	}
	l.ID, l.ExecutionID = "git-b", "execution-b"
	if err := s.IdentityReserve(t.Context(), l); err != nil {
		t.Fatal("independent owner session denied", err)
	}
	retained, err := s.IdentityLease(t.Context(), l.ID)
	if err != nil || retained.RepositoryID != 7 {
		t.Fatal("native repository binding lost", err)
	}
	l.ID, l.ActorID = "other-user", 2
	if err := s.IdentityReserve(t.Context(), l); err == nil {
		t.Fatal("another actor acquired native owner identity")
	}
	l.ID, l.ActorID, l.RepositoryID = "unbound", 1, 0
	if err := s.IdentityReserve(t.Context(), l); err == nil {
		t.Fatal("repository-free Forgejo lease admitted")
	}
	if err := s.IdentityForgetLease(t.Context(), "git-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IdentityLease(t.Context(), "git-b"); err != nil {
		t.Fatal("sibling reservation was removed", err)
	}
}
