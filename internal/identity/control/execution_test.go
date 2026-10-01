package control

import (
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	"github.com/levitateos/sodaos/internal/store"
)

func TestExecutionDuplicateAcquireReplaysSameLease(t *testing.T) {
	c, _, _, conn := controllerFixture(t)
	ctx := t.Context()
	first, err := c.Acquire(ctx, acquireInput(conn.ID, 1, "replay-exec"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.Acquire(ctx, acquireInput(conn.ID, 1, "replay-exec"))
	if err != nil || second.ID != first.ID {
		t.Fatal("duplicate acquire did not replay its lease", err)
	}
	got, err := c.GetExecution(ctx, identity.Factory, "replay-exec")
	if err != nil || got.State != identity.ExecutionLive || got.LeaseID != first.ID || got.Digest == "" {
		t.Fatal("execution identity not retained", err)
	}
}

func TestExecutionChangedRequestRefuses(t *testing.T) {
	c, _, _, conn := controllerFixture(t)
	ctx := t.Context()
	if _, err := c.Acquire(ctx, acquireInput(conn.ID, 1, "stable-exec")); err != nil {
		t.Fatal(err)
	}
	changed := acquireInput(conn.ID, 1, "stable-exec")
	changed.Role = "review"
	if _, err := c.Acquire(ctx, changed); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("changed request reused execution identity", err)
	}
}

func TestExecutionCloseBeforeAcquireBarsLaterUse(t *testing.T) {
	c, _, _, conn := controllerFixture(t)
	ctx := t.Context()
	if err := c.CloseExecution(ctx, identity.Factory, "early-close"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Acquire(ctx, acquireInput(conn.ID, 1, "early-close")); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("closed execution acquired", err)
	}
	got, err := c.GetExecution(ctx, identity.Factory, "early-close")
	if err != nil || got.State != identity.ExecutionTerminal {
		t.Fatal("close tombstone not retained", err)
	}
	if err := c.CloseExecution(ctx, identity.Factory, "early-close"); err != nil {
		t.Fatal("duplicate close failed", err)
	}
}

func TestExecutionCloseReconcilesLiveLeaseAndRegistrationStaysFenced(t *testing.T) {
	c, s, r, conn := controllerFixture(t)
	ctx := t.Context()
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 1, "withdraw-exec"))
	if err != nil {
		t.Fatal(err)
	}
	b := registerBinding(l)
	if _, err = c.Register(ctx, l.ID, b); err != nil {
		t.Fatal(err)
	}
	if err = c.CloseExecution(ctx, identity.Factory, "withdraw-exec"); err != nil {
		t.Fatal(err)
	}
	if r.stopped != 1 {
		t.Fatal("close did not retire native execution")
	}
	if _, err = s.IdentityLease(ctx, l.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("closed lease retained", err)
	}
	if _, err = c.Register(ctx, l.ID, b); err == nil {
		t.Fatal("closed execution registered again")
	}
	if _, err = c.Acquire(ctx, acquireInput(conn.ID, 1, "withdraw-exec")); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("closed execution reacquired", err)
	}
	got, err := c.GetExecution(ctx, identity.Factory, "withdraw-exec")
	if err != nil || got.State != identity.ExecutionTerminal || got.Binding == nil {
		t.Fatal("terminal identity not retained", err)
	}
}

func TestExecutionReturnRetainsTerminalIdentity(t *testing.T) {
	c, _, _, conn := controllerFixture(t)
	ctx := t.Context()
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 1, "return-exec"))
	if err != nil {
		t.Fatal(err)
	}
	b := registerBinding(l)
	d, err := c.Register(ctx, l.ID, b)
	if err != nil {
		t.Fatal(err)
	}
	_ = d
	maintained := []byte(`{"tokens":{"refresh_token":"synthetic-terminal"}}`)
	if err = c.Return(ctx, l.ID, b, maintained); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Acquire(ctx, acquireInput(conn.ID, 1, "return-exec")); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("returned execution reacquired", err)
	}
	got, err := c.GetExecution(ctx, identity.Factory, "return-exec")
	if err != nil || got.State != identity.ExecutionTerminal || got.LeaseID != l.ID {
		t.Fatal("returned identity not retained", err)
	}
}

func TestExecutionGetMissingIsNotFound(t *testing.T) {
	c, _, _, _ := controllerFixture(t)
	if _, err := c.GetExecution(t.Context(), identity.Factory, "absent-exec"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatal("missing execution did not report not-found", err)
	}
	if _, err := c.GetExecution(t.Context(), "unknown", "absent-exec"); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("unknown kind admitted", err)
	}
}

func TestExecutionTransportRoundTrip(t *testing.T) {
	c, _, _, conn := controllerFixture(t)
	ctx := t.Context()
	socket := filepath.Join(t.TempDir(), "broker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: c.Handler(true), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	client := identityclient.New(socket)
	l, err := client.Acquire(ctx, acquireInput(conn.ID, 1, "transport-exec"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.GetExecution(ctx, identity.Factory, "transport-exec")
	if err != nil || got.LeaseID != l.ID || got.State != identity.ExecutionLive {
		t.Fatal("transport lookup failed", err)
	}
	if _, err = client.GetExecution(ctx, identity.Factory, "absent-exec"); !errors.Is(err, identity.ErrNotFound) {
		t.Fatal("transport missing execution did not report not-found", err)
	}
	if err = client.CloseExecution(ctx, identity.Factory, "transport-exec"); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Acquire(ctx, acquireInput(conn.ID, 1, "transport-exec")); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("transport close did not fence acquire", err)
	}
}
