package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestMergePassReconcilesLostSubmitReply(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	calls := 0
	exec.submit = func(w factory.MergeWork) (factory.OperationOutcome, error) {
		calls++
		if calls == 1 {
			// The native side committed, but the reply never arrived.
			exec.ledger[w.OperationID] = factory.OperationOutcome{
				OperationID: w.OperationID, InstallationID: "test-installation", Kind: factory.OpMerge,
				ActorID: w.ActorID, RepositoryID: w.Repository,
				Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone,
				Completion: factory.OpCompletionComplete, Receipt: []byte(`{"pr_number":9}`),
			}
			return factory.OperationOutcome{NotObserved: true}, nil
		}
		return committedOutcome(`{"pr_number":9}`), nil
	}
	fx.wire(exec)
	first := fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeOpen || len(first.Waits) != 1 || first.Waits[0].Reason != "submit_unconfirmed" {
		t.Fatalf("lost reply misadvanced: %+v %+v", first, m)
	}
	intent := *m.Operation.Work
	second := fx.pass(t)
	m = fx.merge(t, p)
	if m.Stage != factory.MergeMerged || len(second.Merged) != 1 {
		t.Fatalf("lost reply unrecovered: %+v %+v", second, m)
	}
	if calls != 1 || *m.Operation.Work != intent {
		t.Fatal("lost reply resubmitted or rewrote its intent")
	}
}

func TestMergePassWaitsForNativeCompletion(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.submit = func(w factory.MergeWork) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionPending, Receipt: []byte(`{"pr_number":9}`)}, nil
	}
	fx.wire(exec)
	first := fx.pass(t)
	m := fx.merge(t, p)
	// A committed ref without finished bookkeeping never finishes.
	if m.Stage != factory.MergeOpen || len(first.Waits) != 1 || first.Waits[0].Reason != "native_completion_pending" {
		t.Fatalf("incomplete merge finished: %+v %+v", first, m)
	}
	for id, outcome := range exec.ledger {
		outcome.Completion = factory.OpCompletionComplete
		exec.ledger[id] = outcome
	}
	second := fx.pass(t)
	m = fx.merge(t, p)
	if m.Stage != factory.MergeMerged || len(second.Merged) != 1 || len(exec.submits) != 1 {
		t.Fatalf("completed merge unsettled: %+v %+v", second, m)
	}
}

func TestMergePassFencesUnconfirmedCompletion(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.confirm = func(w factory.MergeWork) (factory.MergeConfirmation, error) {
		return factory.MergeConfirmation{}, &factory.PublicationRefusal{Reason: "completion_unconfirmed"}
	}
	fx.wire(exec)
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeFenced || m.Reason != factory.MergeReasonUnattributed {
		t.Fatalf("unconfirmed completion merged: %+v", m)
	}
}

func TestMergeCompletionRequiresExactAncestryWitness(t *testing.T) {
	for _, tc := range []struct {
		name     string
		contains bool
		stage    string
	}{
		{name: "canonical true witness", contains: true, stage: factory.MergeMerged},
		{name: "false witness", contains: false, stage: factory.MergeFenced},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx, p, exec := completedOpenMerge(t)
			baseTip := strings.Repeat("c", 40)
			exec.confirm = func(w factory.MergeWork) (factory.MergeConfirmation, error) {
				now := time.Now().Unix()
				return factory.MergeConfirmation{
					MergedCommit: w.HeadOID, BaseTip: baseTip, BaseContainsMergedCommit: tc.contains,
					MergerID: w.ActorID, MergedUnix: now, ClosedUnix: now, NativeRev: 9,
					ObservedUnix: now, IssueClosed: true,
				}, nil
			}
			fx.wire(exec)
			fx.pass(t)
			merge := fx.merge(t, p)
			if merge.Stage != tc.stage {
				t.Fatalf("ancestry witness %v produced stage %s, want %s: %+v", tc.contains, merge.Stage, tc.stage, merge)
			}
			if tc.contains && (merge.MergedCommit != p.PRCreate.HeadOID || merge.Operation.MergedCommit != p.PRCreate.HeadOID) {
				t.Fatalf("ancestry-confirmed merge lost exact commit attribution: %+v", merge)
			}
		})
	}
}

func TestMergePassFencesIndeterminateEffect(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.submit = func(w factory.MergeWork) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{Effect: factory.OpEffectIndeterminate, Cancellation: factory.OpCancelNone}, nil
	}
	fx.wire(exec)
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeFenced || m.Reason != factory.MergeReasonFenced {
		t.Fatalf("indeterminate effect unfinished: %+v", m)
	}
}

func TestMergePassWithdrawsBeforeSubmit(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.observe = func(w factory.MergeWork) (factory.MergeObservation, error) {
		return factory.MergeObservation{}, &factory.PublicationWait{Reason: "native_busy"}
	}
	fx.wire(exec)
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeOpen || m.Operation.Work != nil {
		t.Fatalf("seed observed despite busy native: %+v", m)
	}
	if _, err := fx.publish.db.WithdrawDispatch(context.Background(), p.Repository, "test withdrawal", "soda-tester"); err != nil {
		t.Fatal(err)
	}
	withdrawal := fx.publish.coord.cancelRepositoryMerges(context.Background(), p.Repository)
	m = fx.merge(t, p)
	if withdrawal.Pending || m.Stage != factory.MergeWithdrawn || len(exec.submits) != 0 {
		t.Fatalf("pre-submit withdrawal unconfirmed: %+v %+v", withdrawal, m)
	}
}

func TestMergePassWithdrawalAfterCommitConfirmsCompletion(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	submitted := false
	exec.submit = func(w factory.MergeWork) (factory.OperationOutcome, error) {
		submitted = true
		return factory.OperationOutcome{Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionPending, Receipt: []byte(`{"pr_number":9}`)}, nil
	}
	fx.wire(exec)
	fx.pass(t)
	if !submitted {
		t.Fatal("seed merge never submitted")
	}
	if _, err := fx.publish.db.WithdrawDispatch(context.Background(), p.Repository, "test withdrawal", "soda-tester"); err != nil {
		t.Fatal(err)
	}
	// Completion still pending: withdrawal cannot finish yet.
	withdrawal := fx.publish.coord.cancelRepositoryMerges(context.Background(), p.Repository)
	m := fx.merge(t, p)
	if !withdrawal.Pending || m.Stage != factory.MergeOpen {
		t.Fatalf("pending completion presented as withdrawn: %+v %+v", withdrawal, m)
	}
	for id, outcome := range exec.ledger {
		outcome.Completion = factory.OpCompletionComplete
		exec.ledger[id] = outcome
	}
	withdrawal = fx.publish.coord.cancelRepositoryMerges(context.Background(), p.Repository)
	m = fx.merge(t, p)
	// The effect stands: withdrawal finishes the bookkeeping honestly.
	if withdrawal.Pending || m.Stage != factory.MergeMerged {
		t.Fatalf("committed merge not confirmed under withdrawal: %+v %+v", withdrawal, m)
	}
}

func TestMergeCompletionReleasesCodeDependant(t *testing.T) {
	fx, p := mergeSeed(t, 3)
	endpoint := fx.publish.seed.decision[p.Issue]
	dependant := factory.Acceptance{
		ID: "d" + strings.Repeat("f", 24), Repository: p.Repository, IssueIndex: "4", Approver: 5, NativeRev: 41,
		TitleDigest: strings.Repeat("a", 64), ContentDigest: strings.Repeat("b", 64), ContentVersion: 2,
		Sources: []factory.SelectedSource{{ID: "11", ContentVersion: 0, Digest: strings.Repeat("c", 64)}},
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: "21", DependsOn: "9", EndpointRepo: p.Repository, EndpointIssue: p.Issue,
			Outcome: factory.PrereqCode, PrereqAcceptance: endpoint.ID,
		}},
	}
	ctx := context.Background()
	if err := fx.publish.db.AdmitAcceptanceDecision(ctx, dependant); err != nil {
		t.Fatal(err)
	}
	fx.publish.reads.evidence["7/4"] = AcceptanceEvidence{
		Revision:     9,
		Issue:        AcceptanceIssueView{Index: "4", TitleDigest: dependant.TitleDigest, ContentDigest: dependant.ContentDigest, ContentVer: 2, Visible: true},
		Comments:     []AcceptanceComment{{ID: "11", Digest: dependant.Sources[0].Digest, ContentVer: 0, Visible: true}},
		Dependencies: []AcceptanceEdge{{Occurrence: "21", DependsOn: "9", Visible: true}},
	}
	before, err := fx.publish.coord.assessOne(ctx, p.Repository, 4)
	if err != nil {
		t.Fatal(err)
	}
	if before.control.Readiness != factory.ReadinessBlocked {
		t.Fatalf("dependant runnable before completion: %+v", before.control)
	}
	fx.wire(happyMerger())
	fx.pass(t)
	m := fx.merge(t, p)
	if m.Stage != factory.MergeMerged {
		t.Fatalf("seed merge unsettled: %+v", m)
	}
	after, err := fx.publish.db.IssueControl(ctx, p.Repository, 4)
	if err != nil {
		t.Fatal(err)
	}
	if after.Readiness != factory.ReadinessQueued || after.Reason != factory.ReasonEligible {
		t.Fatalf("confirmed completion did not release the dependant: %+v", after)
	}
	if _, err := fx.publish.db.IssueMergeCompletion(ctx, p.Repository, p.Issue); err != nil {
		t.Fatalf("completion record unreadable: %v", err)
	}
}
