package publish

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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

func TestDecodePublishReceiptRefusesLookalikes(t *testing.T) {
	receipt := PublishReceipt{
		Ref: "refs/heads/soda/factory/a", OldOID: strings.Repeat("0", 40),
		NewOID: strings.Repeat("2", 40), ComparisonRef: "refs/heads/main",
		ComparisonOID: strings.Repeat("1", 40), ActorID: 5, RepositoryID: 7,
	}
	raw, _ := json.Marshal(receipt)
	outcome := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: raw}
	branch, err := DecodePublishReceipt(outcome, receipt.Ref, extensions.PublishExpectedOldAbsent, receipt.NewOID, receipt.ComparisonRef, receipt.ComparisonOID, 5, 7)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if branch.NewOID != receipt.NewOID || branch.Comparison != receipt.ComparisonOID {
		t.Fatalf("branch: %+v", branch)
	}
	for name, mutate := range map[string]func(*PublishReceipt){
		"ref":            func(r *PublishReceipt) { r.Ref = "refs/heads/other" },
		"new":            func(r *PublishReceipt) { r.NewOID = strings.Repeat("3", 40) },
		"comparison":     func(r *PublishReceipt) { r.ComparisonRef = "refs/heads/other" },
		"comparison_oid": func(r *PublishReceipt) { r.ComparisonOID = strings.Repeat("3", 40) },
		"actor":          func(r *PublishReceipt) { r.ActorID = 6 },
		"repo":           func(r *PublishReceipt) { r.RepositoryID = 8 },
		"old":            func(r *PublishReceipt) { r.OldOID = strings.Repeat("4", 40) },
	} {
		mutated := receipt
		mutate(&mutated)
		raw, _ := json.Marshal(mutated)
		bad := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: raw}
		if _, err := DecodePublishReceipt(bad, receipt.Ref, extensions.PublishExpectedOldAbsent, receipt.NewOID, receipt.ComparisonRef, receipt.ComparisonOID, 5, 7); err == nil {
			t.Errorf("case %s adopted", name)
		}
	}
	pending := factory.OperationOutcome{Effect: factory.OpEffectPending, Receipt: raw}
	if _, err := DecodePublishReceipt(pending, receipt.Ref, extensions.PublishExpectedOldAbsent, receipt.NewOID, receipt.ComparisonRef, receipt.ComparisonOID, 5, 7); err == nil {
		t.Fatal("pending outcome decoded")
	}
	missing := factory.OperationOutcome{Effect: factory.OpEffectCommitted}
	if _, err := DecodePublishReceipt(missing, receipt.Ref, extensions.PublishExpectedOldAbsent, receipt.NewOID, receipt.ComparisonRef, receipt.ComparisonOID, 5, 7); err == nil {
		t.Fatal("missing receipt decoded")
	}
}

func TestDecodePRCreateReceiptRefusesLookalikes(t *testing.T) {
	receipt := PRCreateReceipt{
		HeadRef: "refs/heads/soda/factory/a", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		AuthorID: 5, RepositoryID: 7, PRID: 8, IssueID: 10, PRNumber: 9,
	}
	raw, _ := json.Marshal(receipt)
	outcome := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: raw}
	created, err := DecodePRCreateReceipt(outcome, receipt.HeadRef, receipt.BaseRef, receipt.HeadOID, receipt.BaseOID, 5, 7)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.PRNumber != 9 || created.PRID != 8 || created.IssueID != 10 {
		t.Fatalf("created: %+v", created)
	}
	for name, mutate := range map[string]func(*PRCreateReceipt){
		"head":   func(r *PRCreateReceipt) { r.HeadRef = "refs/heads/other" },
		"base":   func(r *PRCreateReceipt) { r.BaseOID = strings.Repeat("3", 40) },
		"repo":   func(r *PRCreateReceipt) { r.RepositoryID = 8 },
		"author": func(r *PRCreateReceipt) { r.AuthorID = 6 },
		"number": func(r *PRCreateReceipt) { r.PRNumber = 0 },
	} {
		mutated := receipt
		mutate(&mutated)
		raw, _ := json.Marshal(mutated)
		bad := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: raw}
		if _, err := DecodePRCreateReceipt(bad, receipt.HeadRef, receipt.BaseRef, receipt.HeadOID, receipt.BaseOID, 5, 7); err == nil {
			t.Errorf("case %s adopted", name)
		}
	}
}

func TestPushBranchMovesExactlyItsTarget(t *testing.T) {
	c, r := candidateFixture(t, "README.md")
	remote := filepath.Join(c.Root, "remote.git")
	command := exec.Command("git", "init", "--bare", remote)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bare: %v %s", err, out)
	}
	c.Remote = servePublicationGit(t, c.Root) + "/remote.git"
	validated, err := c.PrepareValidated(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer validated.Close()
	target := "refs/heads/soda/factory/testpush"
	ops := &fakeBackgroundOps{}
	if err := validated.PushBranch(context.Background(), c, ops, "soda-test-publish-1", target); err != nil {
		t.Fatalf("push: %v", err)
	}
	if len(ops.pushRebinds) != 1 || ops.pushRebinds[0] {
		t.Fatalf("push env: %+v", ops.pushRebinds)
	}
	out, err := exec.Command("git", "--git-dir="+remote, "for-each-ref", "--format=%(refname) %(objectname)").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != target+" "+r.Commit {
		t.Fatalf("remote refs: %s", out)
	}
	if err := validated.PushBranch(context.Background(), c, ops, "bad id", target); err == nil {
		t.Fatal("malformed operation pushed")
	} else {
		var refusal *Refusal
		if !errors.As(err, &refusal) {
			t.Fatalf("operation: %v", err)
		}
	}
	if err := validated.PushBranch(context.Background(), c, ops, "soda-test-publish-1", "main"); err == nil {
		t.Fatal("malformed ref pushed")
	}
}

func TestObserveTipParsesExactAdvertisement(t *testing.T) {
	c, _ := candidateFixture(t, "README.md")
	remote := filepath.Join(c.Root, "remote.git")
	command := exec.Command("git", "init", "--bare", remote)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bare: %v %s", err, out)
	}
	seed := filepath.Join(c.Root, "seed")
	if err := os.Mkdir(seed, 0o700); err != nil {
		t.Fatal(err)
	}
	run := func(dir string, args ...string) string {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = dir
		out, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("seed Git: %v %s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run(seed, "init", "--initial-branch=main")
	run(seed, "config", "user.name", "soda-tester")
	run(seed, "config", "user.email", "soda-tester@localhost")
	if err := os.WriteFile(filepath.Join(seed, "file"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(seed, "add", ".")
	run(seed, "commit", "-m", "seed")
	tip := run(seed, "rev-parse", "HEAD")
	run(seed, "remote", "add", "origin", remote)
	run(seed, "push", "origin", "main")
	git, cleanup, err := c.sourceRepository(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	got, err := observeTip(context.Background(), git, remote, "refs/heads/main", false)
	if err != nil || got != tip {
		t.Fatalf("tip: %q %v", got, err)
	}
	got, err = observeTip(context.Background(), git, remote, "refs/heads/absent", true)
	if err != nil || got != "" {
		t.Fatalf("absent: %q %v", got, err)
	}
	if _, err := observeTip(context.Background(), git, remote, "refs/heads/absent", false); err == nil {
		t.Fatal("missing comparison accepted")
	} else {
		var refusal *Refusal
		if !errors.As(err, &refusal) {
			t.Fatalf("comparison: %v", err)
		}
	}
}

func TestObserveForPublishRefusesBeforeAnyCall(t *testing.T) {
	c, _ := candidateFixture(t, "README.md")
	ops := &fakeBackgroundOps{revision: extensions.NativeRevisionObservation{Revision: 9, Idle: true}}
	if _, err := c.ObserveForPublish(context.Background(), ops, "main", "refs/heads/main"); err == nil {
		t.Fatal("malformed refs accepted")
	}
	if _, err := c.ObserveForPublish(context.Background(), ops, "refs/heads/main", "refs/heads/main"); err == nil {
		t.Fatal("identical refs accepted")
	}
	if _, err := c.ObserveForPublish(context.Background(), nil, "refs/heads/a", "refs/heads/main"); err == nil {
		t.Fatal("missing transport accepted")
	}
	ops = &fakeBackgroundOps{revision: extensions.NativeRevisionObservation{Revision: 9, Idle: false}}
	if _, err := c.ObserveForPublish(context.Background(), ops, "refs/heads/a", "refs/heads/main"); err == nil {
		t.Fatal("busy revision accepted")
	} else {
		var wait *Wait
		if !errors.As(err, &wait) {
			t.Fatalf("busy: %v", err)
		}
	}
}

// Serve real smart HTTP for the publisher's Git transport test. The fixture
// only proves the exact refspec; native operation enforcement needs Fountain.
func servePublicationGit(t *testing.T, root string) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(&cgi.Handler{
		Path: git, Args: []string{"http-backend"}, Dir: root,
		Env: []string{"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1", "REMOTE_USER=soda-tester"},
	})
	t.Cleanup(server.Close)
	return server.URL
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

func TestReceiptRejectsTrailingDocument(t *testing.T) {
	branch := PublishReceipt{Ref: "refs/heads/soda/factory/a", OldOID: strings.Repeat("0", 40), NewOID: strings.Repeat("2", 40), ComparisonRef: "refs/heads/main", ComparisonOID: strings.Repeat("1", 40), ActorID: 5, RepositoryID: 7}
	raw, _ := json.Marshal(branch)
	outcome := factory.OperationOutcome{Effect: factory.OpEffectCommitted, Receipt: append(raw, []byte(" {}")...)}
	if _, err := DecodePublishReceipt(outcome, branch.Ref, "absent", branch.NewOID, branch.ComparisonRef, branch.ComparisonOID, 5, 7); err == nil {
		t.Fatal("trailing branch document adopted")
	}
	pr := PRCreateReceipt{HeadRef: branch.Ref, BaseRef: branch.ComparisonRef, HeadOID: branch.NewOID, BaseOID: branch.ComparisonOID, AuthorID: 5, RepositoryID: 7, PRID: 8, IssueID: 10, PRNumber: 9}
	raw, _ = json.Marshal(pr)
	outcome.Receipt = append(raw, []byte(" {}")...)
	if _, err := DecodePRCreateReceipt(outcome, pr.HeadRef, pr.BaseRef, pr.HeadOID, pr.BaseOID, 5, 7); err == nil {
		t.Fatal("trailing PR document adopted")
	}
}
