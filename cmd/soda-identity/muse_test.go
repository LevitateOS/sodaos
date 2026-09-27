package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/identity/muse"
	"github.com/levitateos/sodaos/internal/store"
)

func TestMuseWorkerSettingsRequireDedicatedExplicitPaths(t *testing.T) {
	c := settings{AdminSocket: "/run/soda/admin.sock", RuntimeSocket: "/run/soda/runtime.sock", Muse: muse.Config{Binary: "/usr/local/bin/muse"}}
	if err := validateWorkerSettings(c); err != nil {
		t.Fatal("disabled workers require no runtime", err)
	}
	c.MuseWorkerRoot = "/run/soda-identity/workers"
	if err := validateWorkerSettings(c); err == nil {
		t.Fatal("partial worker configuration accepted")
	}
	c.MuseWorkerSocket = c.RuntimeSocket
	if err := validateWorkerSettings(c); err == nil {
		t.Fatal("broker credential socket exposed for launch")
	}
	c.MuseWorkerSocket = "/run/soda-identity/muse-launch.sock"
	if err := validateWorkerSettings(c); err != nil {
		t.Fatal(err)
	}
}

func TestMuseFactoryWithoutRuntimeCannotStopBaseContainer(t *testing.T) {
	lease, log := fakeFactoryRuntime(t, "still-running")
	lease.ProviderID = identity.Muse
	lease.Binding.Scope = "muse-factory"
	if err := (nativeRuntime{}).Stop(t.Context(), lease); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("missing independent runtime accepted", err)
	}
	if _, err := os.Stat(log); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("Muse retirement reached base OCI runtime", err)
	}
	if err := (nativeRuntime{}).Validate(t.Context(), lease); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("missing independent runtime validated", err)
	}
}

func TestMuseFactoryUsesOnlyCurrentParentLeaseAuthority(t *testing.T) {
	db, err := store.OpenEncrypted(filepath.Join(t.TempDir(), "broker.db"), bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	conn := identity.Connection{ID: "subscription", ProviderID: identity.Codex, OwnerID: 1, Generation: 1, State: identity.Ready}
	if err := db.IdentitySaveConnection(t.Context(), conn, []byte(`{"synthetic":"credential"}`)); err != nil {
		t.Fatal(err)
	}
	grant := identity.Grant{ID: "grant", ConnectionID: conn.ID, UserID: 2, ProjectID: "project", Revision: 1}
	if err := db.IdentitySaveGrant(t.Context(), grant); err != nil {
		t.Fatal(err)
	}
	lease := identity.Lease{ConnectionID: conn.ID, ProviderID: conn.ProviderID, Generation: 1, ActorID: 2, ProjectID: "project", GrantID: grant.ID, GrantRevision: 1, Deadline: time.Now().Add(time.Hour)}
	if err := factoryLeaseCurrent(t.Context(), db, lease); err != nil {
		t.Fatal(err)
	}
	if err := db.IdentityRevokeGrant(t.Context(), grant); err != nil {
		t.Fatal(err)
	}
	if err := factoryLeaseCurrent(t.Context(), db, lease); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("revoked parent grant admitted", err)
	}
	lease.GrantID = ""
	if err := factoryLeaseCurrent(t.Context(), db, lease); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("lease actor replaced owner", err)
	}
	lease.ActorID = conn.OwnerID
	lease.Deadline = time.Now().Add(-time.Second)
	if err := factoryLeaseCurrent(t.Context(), db, lease); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("expired parent execution admitted", err)
	}
}

func TestFactoryResolverRejectsRequestIdentityWithoutNativePeer(t *testing.T) {
	if _, err := factoryPeerNamespace(terminal.MusePeer{PID: -1, UID: uint32(os.Getuid())}); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("invalid kernel peer admitted", err)
	}
}
