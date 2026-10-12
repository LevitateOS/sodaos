package forgejo

import (
	"context"
	"errors"
	"strings"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
)

type lifecycleFakeBackground struct {
	submit func(sdk.OperationIntent) (sdk.OperationRecord, error)
	get    func(string) (sdk.OperationLookup, error)
}

func (f *lifecycleFakeBackground) ReadNativeRevision(context.Context) (sdk.NativeRevisionObservation, error) {
	return sdk.NativeRevisionObservation{Revision: 1, Idle: true}, nil
}

func (f *lifecycleFakeBackground) ReadOperations(context.Context) (sdk.OperationsHandshake, error) {
	return sdk.OperationsHandshake{}, errors.New("unused")
}

func (f *lifecycleFakeBackground) ReadSnapshot(context.Context, sdk.CredentialFile, sdk.SnapshotRequest) (sdk.NativeSnapshot, error) {
	return sdk.NativeSnapshot{}, errors.New("unused")
}

func (f *lifecycleFakeBackground) ReadCreations(context.Context, sdk.CredentialFile, sdk.CreationRecoveryRequest) (sdk.CreationRecoveryPage, error) {
	return sdk.CreationRecoveryPage{}, errors.New("unused")
}

func (f *lifecycleFakeBackground) SubmitOperation(_ context.Context, _ sdk.CredentialFile, intent sdk.OperationIntent) (sdk.OperationRecord, error) {
	return f.submit(intent)
}

func (f *lifecycleFakeBackground) GetOperation(_ context.Context, id string) (sdk.OperationLookup, error) {
	return f.get(id)
}

func (f *lifecycleFakeBackground) CancelOperation(_ context.Context, id string) (sdk.OperationRecord, error) {
	return sdk.OperationRecord{OperationID: id}, nil
}

func TestLifecyclePayloadBuilders(t *testing.T) {
	if _, err := LifecycleIssuePayload(6, 3, 4); err != nil {
		t.Fatalf("issue payload rejected: %v", err)
	}
	if _, err := LifecycleReviewPayload(2, []int64{2}); err != nil {
		t.Fatalf("review payload rejected: %v", err)
	}
	if _, err := LifecycleNotePayload(5, "done", "pr/2@abc"); err != nil {
		t.Fatalf("note payload rejected: %v", err)
	}
	if _, err := LifecycleIssuePayload(6, 0, 0); err == nil {
		t.Fatal("empty issue payload accepted")
	}
	if _, err := LifecycleReviewPayload(2, []int64{1, 2, 3, 4}); err == nil {
		t.Fatal("oversize review payload accepted")
	}
	if _, err := LifecycleNotePayload(1, "x", ""); err == nil {
		t.Fatal("note without merge ref accepted")
	}
	if _, err := LifecycleNotePayload(5, strings.Repeat("x", factory.MaxCompletionNoteChars+1), "m"); err == nil {
		t.Fatal("oversize note accepted")
	}
}

func TestLifecycleKindsDistinct(t *testing.T) {
	kinds := map[string]bool{LifecycleKindIssue: true, LifecycleKindReview: true, LifecycleKindNote: true}
	if len(kinds) != 3 || LifecycleKindIssue == "" {
		t.Fatalf("kinds: %+v", kinds)
	}
}

func TestLifecycleSubmitReconcilesLostResponse(t *testing.T) {
	committed := sdk.OperationRecord{OperationID: "op-1", EffectState: "committed"}
	fake := &lifecycleFakeBackground{
		submit: func(sdk.OperationIntent) (sdk.OperationRecord, error) {
			return sdk.OperationRecord{}, errors.New("transport lost")
		},
		get: func(id string) (sdk.OperationLookup, error) {
			return sdk.OperationLookup{OperationID: id, Status: "committed", Record: &committed}, nil
		},
	}
	w := NewLifecycleWriter(fake, "token")
	payload, _ := LifecycleIssuePayload(6, 3, 0)
	record, err := w.SubmitAction(context.Background(), "op-1", 2, 3, LifecycleKindIssue, "rev-1", 1, 9999999999, payload)
	if err != nil {
		t.Fatalf("lost response not reconciled: %v", err)
	}
	if ClassifyLifecycleOutcome(record) != factory.LifecycleResultDelivered {
		t.Fatalf("state: %+v", record)
	}
}

func TestLifecycleSubmitUncertainStays(t *testing.T) {
	fake := &lifecycleFakeBackground{
		submit: func(sdk.OperationIntent) (sdk.OperationRecord, error) {
			return sdk.OperationRecord{}, errors.New("transport lost")
		},
		get: func(id string) (sdk.OperationLookup, error) {
			return sdk.OperationLookup{OperationID: id, Status: sdk.BackgroundOutcomeNotObserved}, nil
		},
	}
	w := NewLifecycleWriter(fake, "token")
	payload, _ := LifecycleIssuePayload(6, 3, 0)
	if _, err := w.SubmitAction(context.Background(), "op-2", 2, 3, LifecycleKindIssue, "rev-1", 1, 9999999999, payload); err == nil {
		t.Fatal("unobserved loss returned success")
	}
}

func TestClassifyLifecycleOutcome(t *testing.T) {
	cases := map[string]sdk.OperationRecord{
		factory.LifecycleResultDelivered: {EffectState: "committed"},
		factory.LifecycleResultFailed:    {EffectState: "not_committed", ReasonCode: "native_refused"},
		factory.LifecycleResultDenied:    {EffectState: "not_committed", ReasonCode: "authority_lost"},
		factory.LifecycleResultUncertain: {EffectState: "pending"},
	}
	for want, record := range cases {
		if got := ClassifyLifecycleOutcome(record); got != want {
			t.Fatalf("%+v classified %s", record, got)
		}
	}
	cancelled := sdk.OperationRecord{EffectState: "not_committed", ReasonCode: "cancelled_before_admission"}
	if got := ClassifyLifecycleOutcome(cancelled); got != factory.LifecycleResultDenied {
		t.Fatalf("cancelled classified %s", got)
	}
	if _, err := NewLifecycleWriter(nil, "").SubmitAction(context.Background(), "op", 1, 1, LifecycleKindIssue, "r", 1, 2, []byte(`{}`)); err == nil {
		t.Fatal("nil background accepted")
	}
}
