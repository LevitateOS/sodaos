package store

import (
	"context"
	"errors"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

type loopFakeBackground struct {
	submit func(extensions.OperationIntent) (extensions.OperationRecord, error)
	get    func(string) (extensions.OperationLookup, error)
}

func (f *loopFakeBackground) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	return extensions.NativeRevisionObservation{Revision: 1, Idle: true}, nil
}

func (f *loopFakeBackground) ReadSnapshot(context.Context, extensions.CredentialFile, extensions.SnapshotRequest) (extensions.NativeSnapshot, error) {
	return extensions.NativeSnapshot{}, errors.New("unused")
}

func (f *loopFakeBackground) ReadCreations(context.Context, extensions.CredentialFile, extensions.CreationRecoveryRequest) (extensions.CreationRecoveryPage, error) {
	return extensions.CreationRecoveryPage{}, errors.New("unused")
}

func (f *loopFakeBackground) SubmitOperation(_ context.Context, _ extensions.CredentialFile, intent extensions.OperationIntent) (extensions.OperationRecord, error) {
	return f.submit(intent)
}

func (f *loopFakeBackground) GetOperation(_ context.Context, id string) (extensions.OperationLookup, error) {
	return f.get(id)
}

func (f *loopFakeBackground) CancelOperation(_ context.Context, id string) (extensions.OperationRecord, error) {
	return extensions.OperationRecord{OperationID: id}, nil
}

func loopAction(id string) factory.LifecycleAction {
	return factory.LifecycleAction{ID: id, Event: factory.LifecycleEventIssueCreated, Action: factory.LifecycleActionIssueLabelAssignee, ActorID: 7, RepositoryID: 9, IssueID: 42, LabelID: 3, AuthRevision: "rev-1"}
}

// TestLifecycleJournalLoop proves the Soda durable side of UX-L07: the
// journaled action, finite attempts, denial/revocation stops and
// uncertain-result reconciliation compose through the real store with a
// scripted native transport. Real native effects are proved by the
// Fountain round-trip tests; no execution-provider call is required.
func TestLifecycleJournalLoop(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()

	committed := func(intent extensions.OperationIntent) (extensions.OperationRecord, error) {
		return extensions.OperationRecord{OperationID: intent.OperationID, EffectState: "committed"}, nil
	}
	lookupCommitted := func(record extensions.OperationRecord) func(string) (extensions.OperationLookup, error) {
		return func(id string) (extensions.OperationLookup, error) {
			return extensions.OperationLookup{OperationID: id, Status: "committed", Record: &record}, nil
		}
	}

	a, err := s.RecordAction(ctx, loopAction("act-loop-1"))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if _, err := s.RecordAction(ctx, loopAction("act-loop-1")); err != nil {
		t.Fatalf("duplicate event replay: %v", err)
	}
	if !a.CanAttempt("label", 1) {
		t.Fatal("first attempt refused")
	}
	a = a.RecordAttempt("label", "op-loop-1")
	if _, err := s.UpdateAction(ctx, a, 0); err != nil {
		t.Fatalf("attempt update: %v", err)
	}
	w := forgejo.NewLifecycleWriter(&loopFakeBackground{submit: committed, get: lookupCommitted(extensions.OperationRecord{OperationID: "op-loop-1", EffectState: "committed"})}, "token")
	payload, err := forgejo.LifecycleIssuePayload(42, 3, 0)
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	record, err := w.SubmitAction(ctx, "op-loop-1", 7, 9, forgejo.LifecycleKindIssue, "rev-1", 1, 9999999999, payload)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	a = a.RecordResult("label", forgejo.ClassifyLifecycleOutcome(record))
	if _, err := s.UpdateAction(ctx, a, 1); err != nil {
		t.Fatalf("result update: %v", err)
	}
	if !a.Settled() {
		t.Fatal("delivered action unsettled")
	}

	b, err := s.RecordAction(ctx, loopAction("act-loop-2"))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	b = b.RecordAttempt("label", "op-loop-2")
	if _, err := s.UpdateAction(ctx, b, 0); err != nil {
		t.Fatalf("attempt update: %v", err)
	}
	refused := extensions.OperationRecord{OperationID: "op-loop-2", EffectState: "not_committed", ReasonCode: "native_refused"}
	w2 := forgejo.NewLifecycleWriter(&loopFakeBackground{submit: func(id extensions.OperationIntent) (extensions.OperationRecord, error) { return refused, nil }, get: lookupCommitted(refused)}, "token")
	record2, err := w2.SubmitAction(ctx, "op-loop-2", 7, 9, forgejo.LifecycleKindIssue, "rev-1", 1, 9999999999, payload)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	b = b.RecordResult("label", forgejo.ClassifyLifecycleOutcome(record2))
	if _, err := s.UpdateAction(ctx, b, 1); err != nil {
		t.Fatalf("result update: %v", err)
	}
	if !b.CanAttempt("label", 2) {
		t.Fatal("retry refused after definitive no-effect")
	}
	b = b.RecordAttempt("label", "op-loop-3")
	if _, err := s.UpdateAction(ctx, b, 1); err != nil {
		t.Fatalf("retry update: %v", err)
	}
	record3, err := w.SubmitAction(ctx, "op-loop-3", 7, 9, forgejo.LifecycleKindIssue, "rev-1", 1, 9999999999, payload)
	if err != nil {
		t.Fatalf("retry submit: %v", err)
	}
	b = b.RecordResult("label", forgejo.ClassifyLifecycleOutcome(record3))
	if _, err := s.UpdateAction(ctx, b, 2); err != nil {
		t.Fatalf("retry result update: %v", err)
	}
	if b.CanAttempt("label", 3) {
		t.Fatal("third attempt admitted")
	}

	c, err := s.RecordAction(ctx, loopAction("act-loop-3"))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	c = c.RecordAttempt("label", "op-loop-4")
	if _, err := s.UpdateAction(ctx, c, 0); err != nil {
		t.Fatalf("attempt update: %v", err)
	}
	denied := extensions.OperationRecord{OperationID: "op-loop-4", EffectState: "not_committed", ReasonCode: "authority_lost"}
	w3 := forgejo.NewLifecycleWriter(&loopFakeBackground{submit: func(extensions.OperationIntent) (extensions.OperationRecord, error) { return denied, nil }, get: lookupCommitted(denied)}, "token")
	record4, err := w3.SubmitAction(ctx, "op-loop-4", 7, 9, forgejo.LifecycleKindIssue, "rev-1", 1, 9999999999, payload)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	c = c.RecordResult("label", forgejo.ClassifyLifecycleOutcome(record4))
	if _, err := s.UpdateAction(ctx, c, 1); err != nil {
		t.Fatalf("result update: %v", err)
	}
	if c.CanAttempt("label", 2) {
		t.Fatal("retry admitted after denial")
	}

	d, err := s.RecordAction(ctx, loopAction("act-loop-4"))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	d = d.RecordAttempt("label", "op-loop-5")
	if _, err := s.UpdateAction(ctx, d, 0); err != nil {
		t.Fatalf("attempt update: %v", err)
	}
	lost := extensions.OperationRecord{OperationID: "op-loop-5", EffectState: "committed"}
	w4 := forgejo.NewLifecycleWriter(&loopFakeBackground{
		submit: func(extensions.OperationIntent) (extensions.OperationRecord, error) {
			return extensions.OperationRecord{}, errors.New("transport lost")
		},
		get: lookupCommitted(lost),
	}, "token")
	record5, err := w4.SubmitAction(ctx, "op-loop-5", 7, 9, forgejo.LifecycleKindIssue, "rev-1", 1, 9999999999, payload)
	if err != nil {
		t.Fatalf("lost response not reconciled: %v", err)
	}
	d = d.RecordResult("label", forgejo.ClassifyLifecycleOutcome(record5))
	if _, err := s.UpdateAction(ctx, d, 1); err != nil {
		t.Fatalf("result update: %v", err)
	}
	if !d.Settled() {
		t.Fatal("reconciled action unsettled")
	}

	e, err := s.RecordAction(ctx, loopAction("act-loop-5"))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	e = e.RecordAttempt("label", "op-loop-6")
	if _, err := s.UpdateAction(ctx, e, 0); err != nil {
		t.Fatalf("attempt update: %v", err)
	}
	w5 := forgejo.NewLifecycleWriter(&loopFakeBackground{
		submit: func(extensions.OperationIntent) (extensions.OperationRecord, error) {
			return extensions.OperationRecord{}, errors.New("transport lost")
		},
		get: func(id string) (extensions.OperationLookup, error) {
			return extensions.OperationLookup{OperationID: id, Status: extensions.BackgroundOutcomeNotObserved}, nil
		},
	}, "token")
	if _, err := w5.SubmitAction(ctx, "op-loop-6", 7, 9, forgejo.LifecycleKindIssue, "rev-1", 1, 9999999999, payload); err == nil {
		t.Fatal("unobserved loss returned success")
	}
	stored, err := s.LifecycleAction(ctx, "act-loop-5")
	if err != nil || stored.Results["label"] != factory.LifecycleResultUncertain {
		t.Fatalf("uncertain not retained: %+v %v", stored, err)
	}
	if stored.CanAttempt("label", 2) {
		t.Fatal("retry admitted while uncertain")
	}
	if _, err := s.RevokeAction(ctx, "act-loop-5"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	revoked, err := s.LifecycleAction(ctx, "act-loop-5")
	if err != nil || revoked.Applicable() {
		t.Fatalf("revoked still applicable: %+v %v", revoked, err)
	}

	empty, err := s.ListActions(ctx, 9, 4242, "", 64)
	if err != nil || len(empty) != 0 {
		t.Fatalf("default issue lists %d actions", len(empty))
	}
}
