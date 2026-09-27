package control

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

type forgejoRefresherFake struct {
	testProvider
	checks, calls int
	failure       bool
	result        identity.Connection
	before        func()
	old, renewed  []byte
}

func (p *forgejoRefresherFake) CredentialExpiry(owner int64, data []byte) (time.Time, error) {
	p.checks++
	if owner != 1 {
		return time.Time{}, identity.ErrDenied
	}
	switch string(data) {
	case `"soon"`:
		return time.Now().Add(30 * time.Second), nil
	case `"expired"`:
		return time.Now().Add(-time.Minute), nil
	case `"fresh"`:
		return time.Now().Add(time.Hour), nil
	default:
		return time.Time{}, identity.ErrDenied
	}
}

func (p *forgejoRefresherFake) Refresh(_ context.Context, _ int64, old []byte) (identity.Connection, []byte, error) {
	p.calls++
	p.old = old
	if p.before != nil {
		p.before()
	}
	p.renewed = []byte(`"fresh"`)
	if p.failure {
		return identity.Connection{}, p.renewed, errors.New("sensitive native credential failure")
	}
	return p.result, p.renewed, nil
}

func forgejoCustodyFixture(t *testing.T, seed string) (*Controller, *store.Store, *forgejoRefresherFake, identity.Connection) {
	t.Helper()
	c, s, _, _ := controllerFixture(t)
	conn := identity.Connection{ID: "git-account", ProviderID: identity.Forgejo, OwnerID: 1, Generation: 3, State: identity.Ready, Label: "Git account", Email: "old@example.test"}
	if err := s.IdentitySaveConnection(t.Context(), conn, []byte(seed)); err != nil {
		t.Fatal(err)
	}
	p := &forgejoRefresherFake{result: identity.Connection{ProviderID: identity.Forgejo, OwnerID: 1, State: identity.Ready, Email: "verified@example.test"}}
	c.providers[identity.Forgejo] = p
	return c, s, p, conn
}

func TestForgejoCustodySerializesNativeRenewalOnce(t *testing.T) {
	c, s, p, conn := forgejoCustodyFixture(t, `"expired"`)
	p.before = func() {
		current, err := s.IdentityConnection(t.Context(), conn.ID)
		if err != nil || current.State != identity.Reauth {
			t.Error("native renewal preceded durable withdrawal")
		}
		if _, err := s.IdentityCredential(t.Context(), conn); err == nil {
			t.Error("old custody remained readable")
		}
	}
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Go(func() {
			actual, data, err := c.forgejoCredential(t.Context(), 1, conn.ID)
			defer clear(data)
			if err != nil || string(data) != `"fresh"` || actual.ID != conn.ID || actual.Generation != conn.Generation || actual.Label != conn.Label || actual.OwnerID != conn.OwnerID || actual.Email != p.result.Email {
				t.Error("renewed custody metadata or seed mismatch")
			}
		})
	}
	group.Wait()
	if p.calls != 1 || !bytes.Equal(p.old, make([]byte, len(p.old))) || !bytes.Equal(p.renewed, make([]byte, len(p.renewed))) {
		t.Fatal("renewal repeated or plaintext buffers retained")
	}
}

func TestForgejoCustodyUsesFreshSeedAndRejectsUnauthorizedAdmission(t *testing.T) {
	c, _, p, conn := forgejoCustodyFixture(t, `"fresh"`)
	if _, data, err := c.forgejoCredential(t.Context(), 2, conn.ID); !errors.Is(err, identity.ErrDenied) || data != nil || p.checks != 0 {
		t.Fatal("foreign owner accessed custody")
	}
	actual, data, err := c.forgejoCredential(t.Context(), 1, conn.ID)
	defer clear(data)
	if err != nil || actual != conn || string(data) != `"fresh"` || p.calls != 0 {
		t.Fatal("unexpired custody unnecessarily renewed")
	}
}

func TestForgejoUncertainRenewalCannotReplayAfterRestart(t *testing.T) {
	c, s, p, conn := forgejoCustodyFixture(t, `"expired"`)
	p.failure = true
	if _, data, err := c.forgejoCredential(t.Context(), 1, conn.ID); !errors.Is(err, identity.ErrUncertain) || data != nil {
		t.Fatal("uncertain native renewal admitted")
	}
	current, err := s.IdentityConnection(t.Context(), conn.ID)
	if err != nil || current.State != identity.Reauth {
		t.Fatal("uncertain refresh restored old custody")
	}
	restarted, err := New(s, map[string]identity.Provider{identity.Forgejo: p}, &testRuntime{})
	if err != nil {
		t.Fatal(err)
	}
	for _, controller := range []*Controller{c, restarted} {
		if _, data, err := controller.forgejoCredential(t.Context(), 1, conn.ID); !errors.Is(err, identity.ErrDenied) || data != nil {
			t.Fatal("reauth seed replayed")
		}
	}
	if p.calls != 1 {
		t.Fatal("native refresh seed reused")
	}
	if err := c.Revoke(t.Context(), 1, conn.ID); err != nil {
		t.Fatal(err)
	}
	if _, data, err := c.forgejoCredential(t.Context(), 1, conn.ID); !errors.Is(err, identity.ErrDenied) || data != nil || p.calls != 1 {
		t.Fatal("revoked seed replayed")
	}
}

func TestForgejoRenewalRejectsInvalidNativeMetadata(t *testing.T) {
	for _, field := range []string{"owner", "provider", "email", "state", "plan"} {
		t.Run(field, func(t *testing.T) {
			c, s, p, conn := forgejoCustodyFixture(t, `"expired"`)
			switch field {
			case "owner":
				p.result.OwnerID = 2
			case "provider":
				p.result.ProviderID = identity.Muse
			case "email":
				p.result.Email = ""
			case "state":
				p.result.State = identity.Reauth
			case "plan":
				p.result.Plan = "unexpected"
			}
			if _, data, err := c.forgejoCredential(t.Context(), 1, conn.ID); !errors.Is(err, identity.ErrUncertain) || data != nil {
				t.Fatal("invalid native renewal admitted")
			}
			current, err := s.IdentityConnection(t.Context(), conn.ID)
			if err != nil || current.State != identity.Reauth || current.OwnerID != conn.OwnerID || current.Email != conn.Email {
				t.Fatal("invalid native metadata overwrote custody")
			}
		})
	}
}

func TestForgejoRenewalCannotResurrectRevocation(t *testing.T) {
	c, s, p, conn := forgejoCustodyFixture(t, `"expired"`)
	p.before = func() {
		if err := s.IdentityState(t.Context(), conn, identity.Revoked); err != nil {
			t.Fatal(err)
		}
	}
	if _, data, err := c.forgejoCredential(t.Context(), 1, conn.ID); !errors.Is(err, identity.ErrUncertain) || data != nil {
		t.Fatal("renewal overwrote concurrent revocation")
	}
	current, err := s.IdentityConnection(t.Context(), conn.ID)
	if err != nil || current.State != identity.Revoked {
		t.Fatal("revocation lost")
	}
	if _, data, err := c.forgejoCredential(t.Context(), 1, conn.ID); !errors.Is(err, identity.ErrDenied) || data != nil || p.calls != 1 {
		t.Fatal("revoked refresh replayed")
	}
}

func TestForgejoCustodyRenewsWithinMargin(t *testing.T) {
	c, _, p, conn := forgejoCustodyFixture(t, `"soon"`)
	_, data, err := c.forgejoCredential(t.Context(), 1, conn.ID)
	defer clear(data)
	if err != nil || p.calls != 1 || string(data) != `"fresh"` {
		t.Fatal("near-expiry credential admitted without renewal")
	}
}
