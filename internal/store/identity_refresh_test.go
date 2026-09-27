package store

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
)

func TestForgejoRefreshCannotRestoreRevokedOrUnpreparedCustody(t *testing.T) {
	s, err := OpenEncrypted(filepath.Join(t.TempDir(), "identity.db"), bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	c := identity.Connection{ProviderID: identity.Forgejo, ID: "git-account", OwnerID: 1, Generation: 1, State: identity.Ready}
	seed := []byte(`{"access":"synthetic-old"}`)
	renewed := []byte(`{"access":"synthetic-renewed"}`)
	if err = s.IdentitySaveConnection(t.Context(), c, seed); err != nil {
		t.Fatal(err)
	}
	if err = s.IdentityRefresh(t.Context(), c, renewed); !errors.Is(err, identity.ErrStale) {
		t.Fatal("renewal bypassed durable admission withdrawal", err)
	}
	if err = s.IdentityState(t.Context(), c, identity.Reauth); err != nil {
		t.Fatal(err)
	}
	if _, err = s.IdentityCredential(t.Context(), c); err == nil {
		t.Fatal("uncertain old seed remained admissible")
	}
	if err = s.IdentityRefresh(t.Context(), c, renewed); err != nil {
		t.Fatal(err)
	}
	actual, err := s.IdentityCredential(t.Context(), c)
	if err != nil || !bytes.Equal(actual, renewed) {
		t.Fatal("verified renewal not retained", err)
	}
	if err = s.IdentityState(t.Context(), c, identity.Revoked); err != nil {
		t.Fatal(err)
	}
	if err = s.IdentityRefresh(t.Context(), c, renewed); !errors.Is(err, identity.ErrStale) {
		t.Fatal("renewal restored revoked custody", err)
	}
}
