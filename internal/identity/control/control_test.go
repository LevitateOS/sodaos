package control

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/store"
)

type testProvider struct{ onStart func(int64) }

func (p testProvider) Start(_ context.Context, owner int64) (identity.EnrollmentSession, error) {
	if p.onStart != nil {
		p.onStart(owner)
	}
	return &testSession{}, nil
}

type testSession struct{}

func (*testSession) Snapshot() identity.Enrollment {
	return identity.Enrollment{ID: "enrollment", State: "completed"}
}

func (*testSession) Finish(context.Context) (identity.Connection, []byte, error) {
	return identity.Connection{ProviderID: identity.Codex, Email: "soda-tester@example.invalid", Plan: "plus"}, []byte(`{"tokens":{"refresh_token":"synthetic-private"}}`), nil
}
func (*testSession) Close() error { return nil }

type testRuntime struct {
	validateErr, stopErr, finishErr error
	validated, stopped, finished    int
	finishedData                    []byte
	beforeFinish                    func(context.Context, identity.Lease)
}

func (r *testRuntime) Validate(context.Context, identity.Lease) error {
	r.validated++
	return r.validateErr
}
func (r *testRuntime) Stop(context.Context, identity.Lease) error { r.stopped++; return r.stopErr }

func (r *testRuntime) Finish(ctx context.Context, l identity.Lease) ([]byte, error) {
	if r.beforeFinish != nil {
		r.beforeFinish(ctx, l)
	}
	r.finished++
	r.finishedData = []byte(`{"tokens":{"refresh_token":"synthetic-maintained"}}`)
	return r.finishedData, r.finishErr
}

func controllerFixture(t *testing.T) (*Controller, *store.Store, *testRuntime, identity.Connection) {
	t.Helper()
	s, err := store.OpenEncrypted(filepath.Join(t.TempDir(), "broker.db"), bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	r := &testRuntime{}
	c, err := New(s, map[string]identity.Provider{identity.Codex: testProvider{}, identity.Muse: testProvider{}}, r)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	e, err := c.StartEnrollment(ctx, 1, identity.Codex, "private subscription")
	if err != nil {
		t.Fatal(err)
	}
	e, err = c.Enrollment(ctx, 1, e.ID)
	if err != nil || e.Connection == nil {
		t.Fatal(err)
	}
	return c, s, r, *e.Connection
}

func acquireInput(id string, actor int64, exec ...string) identity.AcquireRequest {
	execution := "execution"
	if len(exec) > 0 {
		execution = exec[0]
	}
	return identity.AcquireRequest{ProviderID: identity.Codex, ConnectionID: id, ActorID: actor, ExecutionID: execution, ProjectID: "project", Kind: identity.Factory, Deadline: time.Now().Add(time.Hour)}
}

func registerBinding(l identity.Lease) identity.Binding {
	return identity.Binding{Kind: l.Kind, ID: "native-container", Generation: l.Generation}
}

func TestFactoryReservationRetainsAdmittedRepository(t *testing.T) {
	c, s, _, connection := controllerFixture(t)
	in := acquireInput(connection.ID, 1)
	in.RepositoryID = 7
	l, err := c.Acquire(t.Context(), in)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := s.IdentityLease(t.Context(), l.ID)
	if err != nil || retained.RepositoryID != 7 || retained.ActorID != 1 {
		t.Fatal("factory admission lost repository authority", err)
	}
}

func TestNamedGrantSerializationAndMaintainedReturn(t *testing.T) {
	c, s, r, conn := controllerFixture(t)
	ctx := t.Context()
	if _, err := c.Acquire(ctx, acquireInput(conn.ID, 2)); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("unnamed recipient admitted", err)
	}
	g, err := c.CreateGrant(ctx, 1, identity.GrantRequest{ConnectionID: conn.ID, UserID: 2, ProjectID: "project", ConfirmSubscription: true, ConfirmCredentialExposure: true})
	if err != nil {
		t.Fatal(err)
	}
	available, err := c.Available(ctx, 2, "project")
	if err != nil || len(available) != 1 || available[0].Email != "" {
		t.Fatal("delegated discovery leaked account identity", err)
	}
	wrong := acquireInput(conn.ID, 2, "wrong-project")
	wrong.ProjectID = "other"
	if _, err = c.Acquire(ctx, wrong); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("wrong project admitted", err)
	}
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 2))
	if err != nil || l.GrantID != g.ID {
		t.Fatal(err)
	}
	if _, err = c.Acquire(ctx, acquireInput(conn.ID, 1, "parallel-copy")); !errors.Is(err, identity.ErrBusy) {
		t.Fatal("parallel refresh copy admitted", err)
	}
	b := registerBinding(l)
	d, err := c.Register(ctx, l.ID, b)
	if err != nil || !bytes.Contains(d.Credential, []byte("synthetic-private")) || r.validated != 1 {
		t.Fatal("delivery preceded native attestation", err)
	}
	if _, err = c.Register(ctx, l.ID, b); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("delivery replay admitted", err)
	}
	maintained := []byte(`{"tokens":{"refresh_token":"synthetic-refreshed"}}`)
	if err = c.Return(ctx, l.ID, b, maintained); err != nil || r.stopped != 1 {
		t.Fatal(err)
	}
	current, err := s.IdentityConnection(ctx, conn.ID)
	if err != nil || current.Generation != conn.Generation+1 {
		t.Fatal("generation was not advanced", err)
	}
	data, err := s.IdentityCredential(ctx, current)
	if err != nil || !bytes.Equal(data, maintained) {
		t.Fatal("maintained credential lost", err)
	}
	if err = c.Return(ctx, l.ID, b, d.Credential); err == nil {
		t.Fatal("stale return replaced refreshed state")
	}
}

func TestRevocationDeniesBeforeUncertainTermination(t *testing.T) {
	c, s, r, conn := controllerFixture(t)
	ctx := t.Context()
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Register(ctx, l.ID, registerBinding(l)); err != nil {
		t.Fatal(err)
	}
	r.stopErr = errors.New("native stop could not be established")
	if err = c.Revoke(ctx, 1, conn.ID); !errors.Is(err, identity.ErrUncertain) {
		t.Fatal(err)
	}
	current, err := s.IdentityConnection(ctx, conn.ID)
	if err != nil || current.State != identity.Revoked {
		t.Fatal("revocation was not durable before native stop", err)
	}
	if _, err = c.Acquire(ctx, acquireInput(conn.ID, 1)); !errors.Is(err, identity.ErrUncertain) {
		t.Fatal("revoked stream reused", err)
	}
	if _, err = s.IdentityLease(ctx, l.ID); err != nil {
		t.Fatal("uncertain binding forgotten", err)
	}
	r.stopErr = nil
	if err = c.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = s.IdentityLease(ctx, l.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("revoked cleanup was not retried before deadline", err)
	}
	current, err = s.IdentityConnection(ctx, conn.ID)
	if err != nil || current.State != identity.Revoked {
		t.Fatal("cleanup revived revoked connection", err)
	}
}

func TestRestartAndNativeAttestationFailClosed(t *testing.T) {
	c, s, r, conn := controllerFixture(t)
	ctx := t.Context()
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 1))
	if err != nil {
		t.Fatal(err)
	}
	r.validateErr = errors.New("resource mismatch")
	if _, err = c.Register(ctx, l.ID, registerBinding(l)); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("unattested resource received bytes", err)
	}
	restarted, err := New(s, map[string]identity.Provider{identity.Codex: testProvider{}, identity.Muse: testProvider{}}, r)
	if err != nil {
		t.Fatal(err)
	}
	if err = restarted.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = restarted.Acquire(ctx, acquireInput(conn.ID, 1)); !errors.Is(err, identity.ErrUncertain) {
		t.Fatal("interrupted stream reused", err)
	}
}

func TestAdminListenerCannotDeliverCredentials(t *testing.T) {
	c, _, _, _ := controllerFixture(t)
	for _, path := range []string{"/acquire", "/register", "/return", "/register?alias=connections"} {
		r := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"owner_id":"1"}`))
		w := httptest.NewRecorder()
		c.Handler(false).ServeHTTP(w, r)
		if w.Code != http.StatusForbidden || bytes.Contains(w.Body.Bytes(), []byte("synthetic-private")) {
			t.Fatal("admin delivery allowed", path, w.Code)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/connections", bytes.NewBufferString(`{"owner_id":"1"}`))
	r.Header.Set("Origin", "https://browser.invalid")
	w := httptest.NewRecorder()
	c.Handler(true).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatal("browser origin reached private protocol")
	}
}

func TestGrantRevocationDeniesBeforeRetirementAndPreservesOtherGrant(t *testing.T) {
	c, s, r, conn := controllerFixture(t)
	ctx := t.Context()
	grantInput := identity.GrantRequest{ConnectionID: conn.ID, UserID: 2, ProjectID: "project", ConfirmSubscription: true, ConfirmCredentialExposure: true}
	g, err := c.CreateGrant(ctx, 1, grantInput)
	if err != nil {
		t.Fatal(err)
	}
	grantInput.UserID = 3
	other, err := c.CreateGrant(ctx, 1, grantInput)
	if err != nil {
		t.Fatal(err)
	}
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 2))
	if err != nil {
		t.Fatal(err)
	}
	b := registerBinding(l)
	original, err := c.Register(ctx, l.ID, b)
	if err != nil {
		t.Fatal(err)
	}
	r.beforeFinish = func(ctx context.Context, finishing identity.Lease) {
		revoked, err := s.IdentityGrant(ctx, g.ID)
		if err != nil || !revoked.Revoked || revoked.Revision != 2 {
			t.Fatal("native capture began before durable grant withdrawal", err)
		}
		if _, err = c.registrationAuthority(ctx, finishing); !errors.Is(err, identity.ErrDenied) {
			t.Fatal("withdrawn grant still authorizes delivery during retirement", err)
		}
	}
	if err = c.RevokeGrant(ctx, 1, g.ID); err != nil {
		t.Fatal(err)
	}
	r.beforeFinish = nil
	current, err := s.IdentityConnection(ctx, conn.ID)
	if err != nil || current.State != identity.Ready || current.Generation != conn.Generation+1 || r.finished != 1 || r.stopped != 1 {
		t.Fatal("verified grant retirement invalidated sponsoring connection", err)
	}
	maintained := []byte(`{"tokens":{"refresh_token":"synthetic-maintained"}}`)
	data, err := s.IdentityCredential(ctx, current)
	if err != nil || !bytes.Equal(data, maintained) {
		t.Fatal("grant retirement lost refreshed credential", err)
	}
	if _, err = c.Acquire(ctx, acquireInput(conn.ID, 2)); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("revoked delegate reacquired sponsorship", err)
	}
	stillAuthorized, err := s.IdentityGrant(ctx, other.ID)
	if err != nil || stillAuthorized.Revoked || stillAuthorized.Revision != other.Revision {
		t.Fatal("unrelated grant changed", err)
	}
	next, err := c.Acquire(ctx, acquireInput(conn.ID, 3, "other-grant"))
	if err != nil {
		t.Fatal("other authorized grant cannot reuse sponsorship", err)
	}
	delivery, err := c.Register(ctx, next.ID, registerBinding(next))
	if err != nil || !bytes.Equal(delivery.Credential, maintained) {
		t.Fatal("next workload did not receive maintained credential", err)
	}
	if err = c.Return(ctx, l.ID, b, original.Credential); err == nil {
		t.Fatal("late retired workload returned stale credentials")
	}
	after, err := s.IdentityConnection(ctx, conn.ID)
	if err != nil || after.State != identity.Ready || after.Generation != current.Generation || r.stopped != 1 {
		t.Fatal("late return changed current execution or connection", err)
	}
	data, err = s.IdentityCredential(ctx, after)
	if err != nil || !bytes.Equal(data, maintained) {
		t.Fatal("late return replaced maintained credential", err)
	}
	revoked, err := s.IdentityGrant(ctx, g.ID)
	if err != nil || !revoked.Revoked || revoked.Revision != 2 {
		t.Fatal("credential preservation restored revoked grant", err)
	}
}

func TestGrantCaptureFailureStillAttemptsStopAndKeepsUncertainLease(t *testing.T) {
	c, s, r, conn := controllerFixture(t)
	ctx := t.Context()
	g, err := c.CreateGrant(ctx, 1, identity.GrantRequest{ConnectionID: conn.ID, UserID: 2, ProjectID: "project", ConfirmSubscription: true, ConfirmCredentialExposure: true})
	if err != nil {
		t.Fatal(err)
	}
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 2))
	if err != nil {
		t.Fatal(err)
	}
	b := registerBinding(l)
	if _, err = c.Register(ctx, l.ID, b); err != nil {
		t.Fatal(err)
	}
	r.finishErr = errors.New("native credential capture failed")
	if err = c.RevokeGrant(ctx, 1, g.ID); !errors.Is(err, identity.ErrUncertain) {
		t.Fatal("failed capture accepted as preserved credentials", err)
	}
	if r.finished != 1 || r.stopped != 1 {
		t.Fatal("capture failure skipped native shutdown")
	}
	current, err := s.IdentityConnection(ctx, conn.ID)
	if err != nil || current.State != identity.Reauth || current.Generation != conn.Generation {
		t.Fatal("failed capture advanced credential stream", err)
	}
	retained, err := s.IdentityLease(ctx, l.ID)
	if err != nil || retained.Binding == nil || *retained.Binding != b {
		t.Fatal("failed capture forgot native boundary", err)
	}
	revoked, err := s.IdentityGrant(ctx, g.ID)
	if err != nil || !revoked.Revoked {
		t.Fatal("failed capture revived grant", err)
	}
	if _, err = c.Acquire(ctx, acquireInput(conn.ID, 1, "sponsor-retry")); !errors.Is(err, identity.ErrUncertain) {
		t.Fatal("uncertain sponsor was reusable", err)
	}
	for _, value := range r.finishedData {
		if value != 0 {
			t.Fatal("failed capture left credential bytes live")
		}
	}
}

func TestDelegateEndRetainsConnectionAndUnrelatedActorCannotEnd(t *testing.T) {
	c, s, r, conn := controllerFixture(t)
	ctx := t.Context()
	_, err := c.CreateGrant(ctx, 1, identity.GrantRequest{ConnectionID: conn.ID, UserID: 2, ProjectID: "project", ConfirmSubscription: true, ConfirmCredentialExposure: true})
	if err != nil {
		t.Fatal(err)
	}
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 2))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Register(ctx, l.ID, registerBinding(l)); err != nil {
		t.Fatal(err)
	}
	if err = c.EndLease(ctx, 3, l.ID); !errors.Is(err, identity.ErrDenied) {
		t.Fatal("unrelated actor ended sponsored execution", err)
	}
	leases, err := c.Leases(ctx, 2, conn.ID)
	if err != nil || len(leases) != 1 || leases[0].Binding != nil {
		t.Fatal("delegate inventory leaked native binding", err)
	}
	if err = c.EndLease(ctx, 2, l.ID); err != nil {
		t.Fatal(err)
	}
	current, err := s.IdentityConnection(ctx, conn.ID)
	if err != nil || current.State != identity.Ready || r.stopped != 1 {
		t.Fatal("ordinary end required reconnection", err)
	}
	if _, err = c.Acquire(ctx, acquireInput(conn.ID, 2)); err != nil {
		t.Fatal("maintained subscription could not be reused", err)
	}
}

func TestReservationRecoveryDoesNotInvalidateUndeliveredCredential(t *testing.T) {
	c, _, _, conn := controllerFixture(t)
	ctx := t.Context()
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 1))
	if err != nil {
		t.Fatal(err)
	}
	if err = c.ReconcileLease(ctx, l.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Acquire(ctx, acquireInput(conn.ID, 1)); err != nil {
		t.Fatal("undelivered reservation invalidated credential", err)
	}
}

func TestFinishCannotReleaseUntilNativeRetirementConfirmed(t *testing.T) {
	c, s, r, conn := controllerFixture(t)
	ctx := t.Context()
	l, err := c.Acquire(ctx, acquireInput(conn.ID, 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Register(ctx, l.ID, registerBinding(l)); err != nil {
		t.Fatal(err)
	}
	r.stopErr = errors.New("credential boundary retirement failed")
	if err = c.EndLease(ctx, 1, l.ID); !errors.Is(err, identity.ErrUncertain) {
		t.Fatal("retirement failure accepted credential return", err)
	}
	current, err := s.IdentityConnection(ctx, conn.ID)
	if err != nil || current.State != identity.Reauth || current.Generation != conn.Generation {
		t.Fatal("uncertain retirement advanced credential stream", err)
	}
	if _, err = s.IdentityLease(ctx, l.ID); err != nil {
		t.Fatal("uncertain native binding forgotten", err)
	}
	if r.finished != 1 || r.stopped != 1 {
		t.Fatal("finish did not require separate retirement confirmation")
	}
	for _, b := range r.finishedData {
		if b != 0 {
			t.Fatal("failed return left credential bytes live")
		}
	}
}

func TestEnrollmentSuppliesTrustedOwner(t *testing.T) {
	c, _, _, _ := controllerFixture(t)
	var observed int64
	c.providers[identity.Codex] = testProvider{onStart: func(owner int64) { observed = owner }}
	if _, err := c.StartEnrollment(t.Context(), 42, identity.Codex, "native account"); err != nil {
		t.Fatal(err)
	}
	if observed != 42 {
		t.Fatalf("provider received owner %d", observed)
	}
}
