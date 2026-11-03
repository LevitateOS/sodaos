package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/store"
)

type graphClientFixture struct {
	sdk.BackgroundClient
	revision    int64
	after       int64
	reads       int
	allowed     bool
	record      sdk.OperationRecord
	submitError error
	submits     int
}

func (f *graphClientFixture) ReadNativeRevision(context.Context) (sdk.NativeRevisionObservation, error) {
	f.reads++
	r := f.revision
	if f.reads > 1 && f.after > 0 {
		r = f.after
	}
	return sdk.NativeRevisionObservation{Revision: r, Idle: true}, nil
}

func (f *graphClientFixture) ReadSnapshot(_ context.Context, _ sdk.CredentialFile, req sdk.SnapshotRequest) (sdk.NativeSnapshot, error) {
	if req.RepositoryID != "42" || req.ActorID != "91" || len(req.WriterIDs) != 1 || req.WriterIDs[0] != "7" || len(req.Families) != 1 || req.Families[0] != sdk.SnapshotFamilyRepositoryWriters {
		return sdk.NativeSnapshot{}, errors.New("unexpected snapshot")
	}
	return sdk.NativeSnapshot{RepositoryID: req.RepositoryID, Writers: map[string]bool{req.WriterIDs[0]: f.allowed}}, nil
}

func (f *graphClientFixture) SubmitOperation(_ context.Context, _ sdk.CredentialFile, i sdk.OperationIntent) (sdk.OperationRecord, error) {
	f.submits++
	if i.Kind != sdk.OperationKindIssueGraph || i.RepositoryID != "42" || i.ActorID != "91" {
		return sdk.OperationRecord{}, errors.New("wrong native kind")
	}
	return f.record, f.submitError
}

func (f *graphClientFixture) GetOperation(_ context.Context, id string) (sdk.OperationLookup, error) {
	return sdk.OperationLookup{OperationID: id, Status: f.record.EffectState, Record: &f.record}, nil
}

func graphIdentityFixture(t *testing.T, repository, actor, owner int64) *FountainIdentities {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	db, _, cleanup, err := store.OpenEphemeral(t.Context(), admin)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	host, err := db.BindConfiguredFountainLedger(t.Context(), "https://graph-fixture.test")
	if err != nil {
		t.Fatal(err)
	}
	foreign := collaboration.Host{ID: collaboration.HostID(collaboration.NewID()), Provider: collaboration.GitHub, APIBase: "https://api.github-fixture.test"}
	if _, err := db.PutCollaborationHost(t.Context(), foreign); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ObserveCollaborationRepository(t.Context(), foreign.ID, strconv.FormatInt(repository, 10), "same/repository"); err != nil {
		t.Fatal(err)
	}
	for _, native := range []int64{actor, owner} {
		if _, err := db.ObserveCollaborationActor(t.Context(), foreign.ID, strconv.FormatInt(native, 10), "same-actor"); err != nil {
			t.Fatal(err)
		}
	}
	identities, err := NewFountainIdentities(t.Context(), db, host.ID)
	if err != nil {
		t.Fatal(err)
	}
	if local, err := identities.LocalRepository(t.Context(), repository); err != nil || int64(local) == repository {
		t.Fatal("graph fixture repository did not differ", local, err)
	}
	for _, native := range []int64{actor, owner} {
		if local, err := identities.LocalActor(t.Context(), strconv.FormatInt(native, 10)); err != nil || int64(local) == native {
			t.Fatal("graph fixture actor did not differ", local, err)
		}
	}
	return identities
}

func graphWriterFixture(t *testing.T, f *graphClientFixture) *IssueGraphWriter {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("graph adapter attempted ordinary REST mutation")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":91,"login":"soda-graph"}`))
	}))
	t.Cleanup(server.Close)
	path := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(path, []byte("synthetic-graph-credential"), 0o600); err != nil {
		t.Fatal(err)
	}
	return NewIssueGraphWriter(f, New(server.URL), path, graphIdentityFixture(t, 42, 91, 7))
}

func TestIssueGraphWriterChecksCurrentOwnerAtStableRevision(t *testing.T) {
	f := &graphClientFixture{revision: 8, allowed: true}
	w := graphWriterFixture(t, f)
	repository, _ := w.Identities.LocalRepository(t.Context(), 42)
	owner, _ := w.Identities.LocalActor(t.Context(), "7")
	actor, _ := w.Identities.LocalActor(t.Context(), "91")
	revision, err := w.CheckIntentOwner(t.Context(), int64(repository), int64(owner), int64(actor))
	if err != nil || revision != 8 {
		t.Fatal(revision, err)
	}
	before := f.reads
	for _, scope := range [][3]int64{{1, int64(owner), int64(actor)}, {42, int64(owner), int64(actor)}, {int64(repository), 2, int64(actor)}, {int64(repository), int64(owner), 1}, {int64(repository), int64(owner), 91}} {
		if _, err := w.CheckIntentOwner(t.Context(), scope[0], scope[1], scope[2]); err == nil || f.reads != before {
			t.Fatal("foreign or unmapped graph authority reached the native background", scope, err)
		}
	}
	f.reads = 0
	f.after = 9
	if _, err = w.CheckIntentOwner(t.Context(), int64(repository), int64(owner), int64(actor)); !errors.Is(err, ErrStaleSnapshot) {
		t.Fatal("authority race not rejected", err)
	}
	f.reads = 0
	f.after = 0
	f.allowed = false
	if _, err = w.CheckIntentOwner(t.Context(), int64(repository), int64(owner), int64(actor)); err == nil {
		t.Fatal("revoked human authority accepted")
	}
}

func TestIssueGraphWriterReconcilesExactOperationAfterLostResponse(t *testing.T) {
	payload, _ := json.Marshal(sdk.IssueGraphPayload{Creates: []sdk.IssueGraphCreate{{Key: "task", Title: "Objective", Body: "Outcome"}}})
	g := factory.IntentGeneration{OperationID: "intent-operation", ActorID: 91, NativeRev: 8, NotAfter: 2000000000, AuthRevision: "accepted-product-direction", Payload: payload}
	record := sdk.OperationRecord{OperationID: g.OperationID, Kind: sdk.OperationKindIssueGraph, ActorID: "91", RepositoryID: "42", EffectState: sdk.EffectCommitted, Receipt: json.RawMessage(`{"actor_id":91,"repository_id":42,"issues":[{"key":"task","issue_id":101,"issue_index":1,"content_version":0}],"dependencies":[]}`)}
	f := &graphClientFixture{record: record, submitError: errors.New("lost response")}
	w := graphWriterFixture(t, f)
	repository, _ := w.Identities.LocalRepository(t.Context(), 42)
	actor, _ := w.Identities.LocalActor(t.Context(), "91")
	g.ActorID = int64(actor)
	got, err := w.SubmitGraph(t.Context(), g, int64(repository))
	if err != nil || got.OperationID != g.OperationID || f.submits != 1 {
		t.Fatal(got, err, f.submits)
	}
	f.record.ActorID = "92"
	if _, err = w.SubmitGraph(t.Context(), g, int64(repository)); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal("foreign native receipt adopted", err)
	}
}
