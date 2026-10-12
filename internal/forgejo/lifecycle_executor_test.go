package forgejo

import (
	"context"
	"errors"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
)

type consumeFakeBackground struct {
	revision int64
	submit   func(sdk.OperationIntent) (sdk.OperationRecord, error)
	get      func(string) (sdk.OperationLookup, error)
}

func (f *consumeFakeBackground) ReadOperations(context.Context) (sdk.OperationsHandshake, error) {
	return sdk.OperationsHandshake{}, errors.New("unused")
}

func (f *consumeFakeBackground) ReadNativeRevision(context.Context) (sdk.NativeRevisionObservation, error) {
	return sdk.NativeRevisionObservation{Revision: f.revision, Idle: true}, nil
}

func (f *consumeFakeBackground) ReadSnapshot(context.Context, sdk.CredentialFile, sdk.SnapshotRequest) (sdk.NativeSnapshot, error) {
	return sdk.NativeSnapshot{}, errors.New("unused")
}

func (f *consumeFakeBackground) ReadCreations(context.Context, sdk.CredentialFile, sdk.CreationRecoveryRequest) (sdk.CreationRecoveryPage, error) {
	return sdk.CreationRecoveryPage{}, errors.New("unused")
}

func (f *consumeFakeBackground) SubmitOperation(_ context.Context, _ sdk.CredentialFile, intent sdk.OperationIntent) (sdk.OperationRecord, error) {
	return f.submit(intent)
}

func (f *consumeFakeBackground) GetOperation(_ context.Context, id string) (sdk.OperationLookup, error) {
	return f.get(id)
}

func (f *consumeFakeBackground) CancelOperation(_ context.Context, id string) (sdk.OperationRecord, error) {
	return sdk.OperationRecord{OperationID: id}, nil
}

func TestLifecycleExecutorRevision(t *testing.T) {
	e := NewLifecycleExecutor(&consumeFakeBackground{revision: 7}, nil, "token")
	rev, err := e.ObserveLifecycleRevision(context.Background(), 9)
	if err != nil || rev != 7 {
		t.Fatalf("revision: %d %v", rev, err)
	}
	if _, err := e.ObserveLifecycleRevision(context.Background(), 0); err == nil {
		t.Fatal("bad repository accepted")
	}
	if _, err := (*LifecycleExecutor)(nil).ObserveLifecycleRevision(context.Background(), 9); err == nil {
		t.Fatal("nil executor accepted")
	}
}

func TestLifecycleExecutorSubmitLookup(t *testing.T) {
	ctx := context.Background()
	committed := func(intent sdk.OperationIntent) (sdk.OperationRecord, error) {
		return sdk.OperationRecord{OperationID: intent.OperationID, EffectState: "committed", Kind: intent.Kind, ActorID: intent.ActorID, RepositoryID: intent.RepositoryID}, nil
	}
	seen := map[string]sdk.OperationIntent{}
	e := NewLifecycleExecutor(&consumeFakeBackground{revision: 3, submit: func(intent sdk.OperationIntent) (sdk.OperationRecord, error) {
		seen[intent.OperationID] = intent
		return committed(intent)
	}, get: func(id string) (sdk.OperationLookup, error) {
		record, err := committed(sdk.OperationIntent{OperationID: id})
		if err != nil {
			return sdk.OperationLookup{}, err
		}
		return sdk.OperationLookup{OperationID: id, Status: "committed", Record: &record}, nil
	}}, nil, "token")
	work := factory.LifecycleWork{OperationID: "op-1", Event: factory.LifecycleEventIssueCreated, ActorID: 7, RepositoryID: 9, IssueID: 42, LabelID: 3, AuthRevision: "rev-1", NativeRev: 3, NotAfter: 9999999999}
	outcome, err := e.SubmitLifecycle(ctx, work)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if outcome.Effect != factory.OpEffectCommitted || outcome.OperationID != "op-1" {
		t.Fatalf("outcome: %+v", outcome)
	}
	intent := seen["op-1"]
	if intent.Kind != factory.LifecycleKindIssue || intent.AuthorizationRevision != "rev-1" || intent.ExpectedNativeRevision != 3 {
		t.Fatalf("intent: %+v", intent)
	}
	lookup, err := e.LookupLifecycle(ctx, "op-1")
	if err != nil || lookup.Effect != factory.OpEffectCommitted {
		t.Fatalf("lookup: %+v %v", lookup, err)
	}
	absent := NewLifecycleExecutor(&consumeFakeBackground{get: func(id string) (sdk.OperationLookup, error) {
		return sdk.OperationLookup{OperationID: id, Status: sdk.BackgroundOutcomeNotObserved}, nil
	}}, nil, "token")
	missing, err := absent.LookupLifecycle(ctx, "op-absent")
	if err != nil || !missing.NotObserved {
		t.Fatalf("absent: %+v %v", missing, err)
	}
	denied, err := e.SubmitLifecycle(ctx, factory.LifecycleWork{OperationID: "op-2", Event: factory.LifecycleEventPRCreated, ActorID: 7, RepositoryID: 9, IssueID: 11, ReviewerIDs: []int64{5}, AuthRevision: "rev-1", NativeRev: 3, NotAfter: 9999999999})
	if err != nil {
		t.Fatalf("review submit: %v", err)
	}
	if denied.Effect != factory.OpEffectCommitted || seen["op-2"].Kind != factory.LifecycleKindReview {
		t.Fatalf("review: %+v", denied)
	}
	if _, err := e.SubmitLifecycle(ctx, factory.LifecycleWork{OperationID: "op-3", Event: "bogus"}); err == nil {
		t.Fatal("bogus event accepted")
	}
	if _, err := e.LookupLifecycle(ctx, ""); err == nil {
		t.Fatal("empty lookup accepted")
	}
}

func TestLifecycleExecutorForeignRecord(t *testing.T) {
	e := NewLifecycleExecutor(&consumeFakeBackground{submit: func(intent sdk.OperationIntent) (sdk.OperationRecord, error) {
		return sdk.OperationRecord{OperationID: "other", EffectState: "committed"}, nil
	}}, nil, "token")
	work := factory.LifecycleWork{OperationID: "op-1", Event: factory.LifecycleEventIssueCreated, ActorID: 7, RepositoryID: 9, IssueID: 42, LabelID: 3, AuthRevision: "rev-1", NativeRev: 1, NotAfter: 9999999999}
	if _, err := e.SubmitLifecycle(context.Background(), work); err == nil {
		t.Fatal("foreign record adopted")
	}
}
