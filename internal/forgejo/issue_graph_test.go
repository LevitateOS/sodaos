package forgejo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
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
	if req.ActorID != "91" || len(req.Families) != 1 || req.Families[0] != sdk.SnapshotFamilyRepositoryWriters {
		return sdk.NativeSnapshot{}, errors.New("unexpected snapshot")
	}
	return sdk.NativeSnapshot{RepositoryID: req.RepositoryID, Writers: map[string]bool{req.WriterIDs[0]: f.allowed}}, nil
}

func (f *graphClientFixture) SubmitOperation(_ context.Context, _ sdk.CredentialFile, i sdk.OperationIntent) (sdk.OperationRecord, error) {
	f.submits++
	if i.Kind != sdk.OperationKindIssueGraph {
		return sdk.OperationRecord{}, errors.New("wrong native kind")
	}
	return f.record, f.submitError
}

func (f *graphClientFixture) GetOperation(_ context.Context, id string) (sdk.OperationLookup, error) {
	return sdk.OperationLookup{OperationID: id, Status: f.record.EffectState, Record: &f.record}, nil
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
	return NewIssueGraphWriter(f, New(server.URL), path)
}

func TestIssueGraphWriterChecksCurrentOwnerAtStableRevision(t *testing.T) {
	f := &graphClientFixture{revision: 8, allowed: true}
	w := graphWriterFixture(t, f)
	revision, err := w.CheckIntentOwner(t.Context(), 42, 7, 91)
	if err != nil || revision != 8 {
		t.Fatal(revision, err)
	}
	f.reads = 0
	f.after = 9
	if _, err = w.CheckIntentOwner(t.Context(), 42, 7, 91); !errors.Is(err, ErrStaleSnapshot) {
		t.Fatal("authority race not rejected", err)
	}
	f.reads = 0
	f.after = 0
	f.allowed = false
	if _, err = w.CheckIntentOwner(t.Context(), 42, 7, 91); err == nil {
		t.Fatal("revoked human authority accepted")
	}
}

func TestIssueGraphWriterReconcilesExactOperationAfterLostResponse(t *testing.T) {
	payload, _ := json.Marshal(sdk.IssueGraphPayload{Creates: []sdk.IssueGraphCreate{{Key: "task", Title: "Objective", Body: "Outcome"}}})
	g := factory.IntentGeneration{OperationID: "intent-operation", ActorID: 91, NativeRev: 8, NotAfter: 2000000000, AuthRevision: "accepted-product-direction", Payload: payload}
	record := sdk.OperationRecord{OperationID: g.OperationID, Kind: sdk.OperationKindIssueGraph, ActorID: "91", RepositoryID: "42", EffectState: sdk.EffectCommitted, Receipt: json.RawMessage(`{"actor_id":91,"repository_id":42,"issues":[{"key":"task","issue_id":101,"issue_index":1,"content_version":0}],"dependencies":[]}`)}
	f := &graphClientFixture{record: record, submitError: errors.New("lost response")}
	w := graphWriterFixture(t, f)
	got, err := w.SubmitGraph(t.Context(), g, 42)
	if err != nil || got.OperationID != g.OperationID || f.submits != 1 {
		t.Fatal(got, err, f.submits)
	}
	f.record.ActorID = "92"
	if _, err = w.SubmitGraph(t.Context(), g, 42); !errors.Is(err, ErrInvalidResponse) {
		t.Fatal("foreign native receipt adopted", err)
	}
}
