package publish

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
)

type fakeBackgroundOps struct {
	revision    extensions.NativeRevisionObservation
	revErr      error
	submit      extensions.OperationRecord
	submitErr   error
	lookup      extensions.OperationLookup
	lookupErr   error
	cancel      extensions.OperationRecord
	cancelErr   error
	pushEnv     []string
	pushErr     error
	submits     int
	pushRebinds []bool
	intents     []extensions.OperationIntent
}

func (f *fakeBackgroundOps) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	if f.revErr != nil {
		return extensions.NativeRevisionObservation{}, f.revErr
	}
	return f.revision, nil
}

func (f *fakeBackgroundOps) SubmitOperation(_ context.Context, _ extensions.CredentialFile, intent extensions.OperationIntent) (extensions.OperationRecord, error) {
	f.submits++
	f.intents = append(f.intents, intent)
	if f.submitErr != nil {
		return extensions.OperationRecord{}, f.submitErr
	}
	return f.submit, nil
}

func (f *fakeBackgroundOps) GetOperation(_ context.Context, _ string) (extensions.OperationLookup, error) {
	if f.lookupErr != nil {
		return extensions.OperationLookup{}, f.lookupErr
	}
	return f.lookup, nil
}

func (f *fakeBackgroundOps) CancelOperation(_ context.Context, _ string) (extensions.OperationRecord, error) {
	if f.cancelErr != nil {
		return extensions.OperationRecord{}, f.cancelErr
	}
	return f.cancel, nil
}

func (f *fakeBackgroundOps) PublishPushEnv(_ context.Context, _ string, rebind bool) ([]string, error) {
	f.pushRebinds = append(f.pushRebinds, rebind)
	if f.pushErr != nil {
		return nil, f.pushErr
	}
	return f.pushEnv, nil
}

func pendingRecord(opID string) extensions.OperationRecord {
	return extensions.OperationRecord{
		InstallationID: "install-1", OperationID: opID, Outcome: factory.OpEffectPending,
		ActorID: "5", RepositoryID: "7", Kind: factory.OpRefPublish,
		EffectState: factory.OpEffectPending, CancellationStatus: factory.OpCancelNone,
		CompletionState: factory.OpCompletionPending,
	}
}

func TestSubmitPublishMapsTerminalDispatchVerdicts(t *testing.T) {
	ctx := context.Background()
	c, _ := candidateFixture(t, "README.md")
	intent := func() PublishOpIntent {
		return PublishOpIntent{
			OperationID: "soda-test-publish-1", AuthRevision: "soda-assignment:a:publication:p:revision:0",
			Ref: "refs/heads/soda/factory/a", ExpectedOld: extensions.PublishExpectedOldAbsent,
			NewOID: strings.Repeat("2", 40), ComparisonRef: "refs/heads/main",
			ExpectedComparisonOID: strings.Repeat("1", 40),
			ActorID:               5, RepositoryID: 7, ExpectedRevision: 9,
			NotAfter: time.Now().Add(time.Hour).Unix(),
		}
	}
	for _, status := range []int{400} {
		ops := &fakeBackgroundOps{submitErr: &StatusError{Status: status}}
		if _, err := c.SubmitPublish(ctx, ops, intent()); err == nil {
			t.Fatalf("status %d accepted", status)
		} else {
			var refusal *Refusal
			if !errors.As(err, &refusal) {
				t.Fatalf("status %d: %v", status, err)
			}
		}
	}
	for _, status := range []int{401, 403} {
		ops := &fakeBackgroundOps{submitErr: &StatusError{Status: status}}
		if _, err := c.SubmitPublish(ctx, ops, intent()); err == nil {
			t.Fatalf("status %d accepted", status)
		} else {
			var wait *Wait
			if !errors.As(err, &wait) {
				t.Fatalf("status %d: %v", status, err)
			}
		}
	}
	ops := &fakeBackgroundOps{submitErr: &StatusError{Status: 409, Body: "intent_conflict"}}
	if _, err := c.SubmitPublish(ctx, ops, intent()); err == nil {
		t.Fatal("intent conflict accepted")
	} else {
		var refusal *Refusal
		if !errors.As(err, &refusal) || refusal.Reason != "intent_conflict" {
			t.Fatalf("intent conflict: %v", err)
		}
	}
}

func TestSubmitPublishReconcilesLostReplies(t *testing.T) {
	ctx := context.Background()
	c, _ := candidateFixture(t, "README.md")
	in := PublishOpIntent{
		OperationID: "soda-test-publish-1", AuthRevision: "soda-assignment:a:publication:p:revision:0",
		Ref: "refs/heads/soda/factory/a", ExpectedOld: extensions.PublishExpectedOldAbsent,
		NewOID: strings.Repeat("2", 40), ComparisonRef: "refs/heads/main",
		ExpectedComparisonOID: strings.Repeat("1", 40),
		ActorID:               5, RepositoryID: 7, ExpectedRevision: 9,
		NotAfter: time.Now().Add(time.Hour).Unix(),
	}
	record := pendingRecord(in.OperationID)
	ops := &fakeBackgroundOps{
		submitErr: errors.New("connection reset"),
		lookup: extensions.OperationLookup{
			InstallationID: "install-1", OperationID: in.OperationID,
			Status: factory.OpEffectPending, Record: &record,
		},
	}
	_, err := c.SubmitPublish(ctx, ops, in)
	var adopted *Adopted
	if !errors.As(err, &adopted) {
		t.Fatalf("lost reply: %v", err)
	}
	if adopted.Outcome.Effect != factory.OpEffectPending || ops.submits != 1 {
		t.Fatalf("adopted: %+v submits=%d", adopted.Outcome, ops.submits)
	}
	ops = &fakeBackgroundOps{
		submitErr: &StatusError{Status: 503},
		lookup: extensions.OperationLookup{
			InstallationID: "install-1", OperationID: in.OperationID,
			Status: extensions.BackgroundOutcomeNotObserved,
		},
	}
	_, err = c.SubmitPublish(ctx, ops, in)
	if adopted == nil || !errors.As(err, &adopted) {
		t.Fatalf("unobserved: %v", err)
	}
	if !adopted.Outcome.NotObserved {
		t.Fatalf("unobserved: %+v", adopted.Outcome)
	}
}

func TestSubmitPublishAdoptsTerminalReplay(t *testing.T) {
	ctx := context.Background()
	c, _ := candidateFixture(t, "README.md")
	in := PublishOpIntent{
		OperationID: "soda-test-publish-1", AuthRevision: "soda-assignment:a:publication:p:revision:0",
		Ref: "refs/heads/soda/factory/a", ExpectedOld: extensions.PublishExpectedOldAbsent,
		NewOID: strings.Repeat("2", 40), ComparisonRef: "refs/heads/main",
		ExpectedComparisonOID: strings.Repeat("1", 40),
		ActorID:               5, RepositoryID: 7, ExpectedRevision: 9,
		NotAfter: time.Now().Add(time.Hour).Unix(),
	}
	record := pendingRecord(in.OperationID)
	record.Outcome, record.EffectState, record.ReasonCode = factory.OpEffectNotCommitted, factory.OpEffectNotCommitted, "stale_head"
	ops := &fakeBackgroundOps{submit: record}
	outcome, err := c.SubmitPublish(ctx, ops, in)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if outcome.Effect != factory.OpEffectNotCommitted || outcome.Reason != "stale_head" {
		t.Fatalf("replay: %+v", outcome)
	}
	if len(ops.intents) != 1 || ops.intents[0].Kind != factory.OpRefPublish {
		t.Fatalf("intent: %+v", ops.intents)
	}
	var payload extensions.PublishPayload
	if err := json.Unmarshal(ops.intents[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Ref != in.Ref || payload.NewOID != in.NewOID || payload.Correction != nil {
		t.Fatalf("payload: %+v", payload)
	}
}

func TestSubmitPublishRejectsMalformedIntents(t *testing.T) {
	ctx := context.Background()
	c, _ := candidateFixture(t, "README.md")
	base := PublishOpIntent{
		OperationID: "soda-test-publish-1", AuthRevision: "soda-assignment:a:publication:p:revision:0",
		Ref: "refs/heads/soda/factory/a", ExpectedOld: extensions.PublishExpectedOldAbsent,
		NewOID: strings.Repeat("2", 40), ComparisonRef: "refs/heads/main",
		ExpectedComparisonOID: strings.Repeat("1", 40),
		ActorID:               5, RepositoryID: 7, ExpectedRevision: 9,
		NotAfter: time.Now().Add(time.Hour).Unix(),
	}
	cases := map[string]func(*PublishOpIntent){
		"operation":  func(i *PublishOpIntent) { i.OperationID = "bad id" },
		"actor":      func(i *PublishOpIntent) { i.ActorID = 0 },
		"auth":       func(i *PublishOpIntent) { i.AuthRevision = "" },
		"revision":   func(i *PublishOpIntent) { i.ExpectedRevision = 0 },
		"ref":        func(i *PublishOpIntent) { i.Ref = "main" },
		"sametuple":  func(i *PublishOpIntent) { i.ComparisonRef = i.Ref },
		"correction": func(i *PublishOpIntent) { i.CorrectionNumber = 3 },
	}
	for name, mutate := range cases {
		in := base
		mutate(&in)
		ops := &fakeBackgroundOps{submit: pendingRecord(in.OperationID)}
		if _, err := c.SubmitPublish(ctx, ops, in); err == nil {
			t.Errorf("case %s accepted", name)
		} else if ops.submits != 0 {
			t.Errorf("case %s submitted", name)
		}
	}
	in := base
	in.NotAfter = time.Now().Add(-time.Minute).Unix()
	ops := &fakeBackgroundOps{submit: pendingRecord(in.OperationID)}
	if _, err := c.SubmitPublish(ctx, ops, in); err == nil {
		t.Fatal("expired observation accepted")
	} else {
		var wait *Wait
		if !errors.As(err, &wait) || ops.submits != 0 {
			t.Fatalf("expired: %v submits=%d", err, ops.submits)
		}
	}
}

func TestSubmitPRCreateBuildsExactIntent(t *testing.T) {
	ctx := context.Background()
	c, _ := candidateFixture(t, "README.md")
	in := PRCreateOpIntent{
		OperationID: "soda-test-prcreate-1", AuthRevision: "soda-assignment:a:publication:p:revision:0",
		HeadRef: "refs/heads/soda/factory/a", BaseRef: "refs/heads/main",
		ExpectedHead: strings.Repeat("2", 40), ExpectedBase: strings.Repeat("1", 40),
		Title: "candidate", Body: "body",
		ActorID: 5, RepositoryID: 7, ExpectedRevision: 9,
		NotAfter: time.Now().Add(time.Hour).Unix(),
	}
	record := pendingRecord(in.OperationID)
	record.Kind = factory.OpPRCreate
	ops := &fakeBackgroundOps{submit: record}
	outcome, err := c.SubmitPRCreate(ctx, ops, in)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if outcome.Effect != factory.OpEffectPending {
		t.Fatalf("outcome: %+v", outcome)
	}
	var payload extensions.PRCreatePayload
	if err := json.Unmarshal(ops.intents[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.HeadRepositoryID != 7 || payload.HeadRef != in.HeadRef || payload.BaseRef != in.BaseRef ||
		payload.ExpectedHeadOID != in.ExpectedHead || payload.Title != "candidate" || payload.AllowMaintainerEdit {
		t.Fatalf("payload: %+v", payload)
	}
	in.Title = "   "
	ops = &fakeBackgroundOps{submit: record}
	if _, err := c.SubmitPRCreate(ctx, ops, in); err == nil || ops.submits != 0 {
		t.Fatal("blank title submitted")
	}
}

func TestLookupAndCancelMapHonestly(t *testing.T) {
	ctx := context.Background()
	c, _ := candidateFixture(t, "README.md")
	ops := &fakeBackgroundOps{
		lookup: extensions.OperationLookup{
			InstallationID: "install-1", OperationID: "soda-test-publish-1",
			Status: extensions.BackgroundOutcomeNotObserved,
		},
	}
	outcome, err := c.LookupOperation(ctx, ops, "soda-test-publish-1")
	if err != nil || !outcome.NotObserved {
		t.Fatalf("not observed: %+v %v", outcome, err)
	}
	record := pendingRecord("soda-test-publish-1")
	record.CancellationStatus = factory.OpCancelCancelled
	record.EffectState, record.Outcome = factory.OpEffectNotCommitted, factory.OpEffectNotCommitted
	ops = &fakeBackgroundOps{cancel: record}
	outcome, err = c.CancelOperation(ctx, ops, "soda-test-publish-1")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if outcome.Effect != factory.OpEffectNotCommitted || outcome.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("cancel: %+v", outcome)
	}
	ops = &fakeBackgroundOps{cancelErr: &StatusError{Status: 401}}
	if _, err := c.CancelOperation(ctx, ops, "soda-test-publish-1"); err == nil {
		t.Fatal("unauthorized cancel accepted")
	} else {
		var wait *Wait
		if !errors.As(err, &wait) {
			t.Fatalf("cancel wait: %v", err)
		}
	}
	if _, err := c.LookupOperation(ctx, ops, "bad id"); err == nil {
		t.Fatal("malformed lookup accepted")
	}
}

func TestPublicationRecordRejectsForeignScope(t *testing.T) {
	base := pendingRecord("soda-test-publish-1")
	intent := extensions.OperationIntent{
		OperationID: base.OperationID, ActorID: base.ActorID, RepositoryID: base.RepositoryID, Kind: base.Kind,
	}
	for name, change := range map[string]func(*extensions.OperationRecord){
		"operation":  func(r *extensions.OperationRecord) { r.OperationID = "other" },
		"actor":      func(r *extensions.OperationRecord) { r.ActorID = "6" },
		"repository": func(r *extensions.OperationRecord) { r.RepositoryID = "8" },
		"kind":       func(r *extensions.OperationRecord) { r.Kind = factory.OpPRCreate },
	} {
		t.Run(name, func(t *testing.T) {
			record := base
			change(&record)
			if _, err := submitOperation(context.Background(), &fakeBackgroundOps{submit: record}, "", intent); err == nil {
				t.Fatal("foreign record adopted")
			}
		})
	}
	record := base
	record.InstallationID = "other-installation"
	ops := &fakeBackgroundOps{lookup: extensions.OperationLookup{
		OperationID: base.OperationID, InstallationID: base.InstallationID, Status: base.EffectState, Record: &record,
	}}
	if _, err := lookupOperation(context.Background(), ops, base.OperationID); err == nil {
		t.Fatal("foreign nested installation adopted")
	}
}
