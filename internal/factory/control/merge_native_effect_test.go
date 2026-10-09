package control_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// TestNativeMergeCancelBeforeSubmit proves the first cancellation
// ordering through the production client: cancel before submit pins a
// tombstone, and the later submit stays cancelled without any effect.
func TestNativeMergeCancelBeforeSubmit(t *testing.T) {
	c := loadNativeST12(t)
	p, _ := nativeMergePublish(t, c, "cancelbefore")
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	merger := forgejo.NewMerger(bg, forgejo.New(c.FountainURL), c.TokenFile)
	id := "soda-st12-cancelbefore-" + factory.NewID()
	tombstone := nativeMergeCall(t, "cancel op", func(callCtx context.Context) (factory.OperationOutcome, error) {
		return merger.CancelOp(callCtx, id)
	})
	if tombstone.Effect != factory.OpEffectNotCommitted || tombstone.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("cancel before submit is not a tombstone: %+v", tombstone)
	}
	revision := nativeMergeIdle(t, bg)
	w := factory.MergeWork{
		MergeID: factory.NewID(), PublicationID: factory.NewID(), OperationID: id, AuthRevision: "st12-native-proof",
		HeadRef: "refs/heads/" + p.branch, BaseRef: c.BaseBranch, HeadOID: p.head, BaseOID: p.base,
		Repository: c.Repository, Issue: p.prNumber, PRNumber: p.prNumber, PRID: p.prID, IssueID: p.issueID,
		PRAuthorID: c.CreatorID, ReviewerID: c.ReviewerID, ActorID: c.ActorID,
		NativeRev: revision.Revision, NotAfter: time.Now().Add(5 * time.Minute).Unix(),
		AssessmentRevision: 1, ReviewID: 1,
	}
	// The tombstoned identity answers cancelled without evaluation; a
	// stale answer (bracket evaluated first) resubmits the same
	// identity, never a fresh one.
	var outcome factory.OperationOutcome
	for i := 0; i < 10; i++ {
		outcome = nativeMergeCall(t, "submit after cancel", func(callCtx context.Context) (factory.OperationOutcome, error) {
			return merger.SubmitMerge(callCtx, w)
		})
		if !nativeMergeStaleRefusal(outcome) {
			break
		}
		nativeMergeIdle(t, bg)
	}
	if outcome.Effect != factory.OpEffectNotCommitted || outcome.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("submit after cancel is not cancelled: %+v", outcome)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("cancelled merge moved the base")
	}
	nativeMergeReceipt(t, "cancelbefore-submit", outcome)
	nativeMergeReceipt(t, "cancelbefore-tombstone", tombstone)
}

// TestNativeMergeCancelAfterCommit proves the second cancellation
// ordering through the production client: cancel after commit reports
// too-late, the committed effect stands, and lookup confirms both.
func TestNativeMergeCancelAfterCommit(t *testing.T) {
	c := loadNativeST12(t)
	p, _ := nativeMergePublish(t, c, "cancelafter")
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	merger := forgejo.NewMerger(bg, forgejo.New(c.FountainURL), c.TokenFile)
	submitted := nativeMergeSubmitMergeCommitted(t, merger, bg, factory.MergeWork{
		MergeID: factory.NewID(), PublicationID: factory.NewID(),
		AuthRevision: "st12-native-proof",
		HeadRef:      "refs/heads/" + p.branch, BaseRef: c.BaseBranch, HeadOID: p.head, BaseOID: p.base,
		Repository: c.Repository, Issue: p.prNumber, PRNumber: p.prNumber, PRID: p.prID, IssueID: p.issueID,
		PRAuthorID: c.CreatorID, ReviewerID: c.ReviewerID, ActorID: c.ActorID,
		AssessmentRevision: 1, ReviewID: 1,
	})
	cancelled := nativeMergeCall(t, "cancel after commit", func(callCtx context.Context) (factory.OperationOutcome, error) {
		return merger.CancelOp(callCtx, submitted.OperationID)
	})
	if cancelled.Effect != factory.OpEffectCommitted || cancelled.Cancellation != factory.OpCancelTooLate {
		t.Fatalf("cancel after commit is not too-late: %+v", cancelled)
	}
	looked := nativeMergeCall(t, "cancel lookup", func(callCtx context.Context) (factory.OperationOutcome, error) {
		return merger.LookupOp(callCtx, submitted.OperationID)
	})
	if looked.Effect != factory.OpEffectCommitted || looked.Cancellation != factory.OpCancelTooLate {
		t.Fatalf("lookup does not confirm the standing effect: %+v", looked)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p.head {
		t.Fatal("committed merge did not move the base")
	}
	nativeMergeReceipt(t, "cancelafter-cancel", cancelled)
}

// dropOnceMerger drops the first submit reply after the native side
// already committed it: the pass must reconcile by lookup instead of
// resubmitting.
type dropOnceMerger struct {
	control.MergeExecutor
	submits int
	dropped bool
}

func (d *dropOnceMerger) SubmitMerge(ctx context.Context, w factory.MergeWork) (factory.OperationOutcome, error) {
	d.submits++
	outcome, err := d.MergeExecutor.SubmitMerge(ctx, w)
	if err != nil {
		return outcome, err
	}
	if !d.dropped {
		d.dropped = true
		return factory.OperationOutcome{}, errors.New("ST12 proof dropped the submit reply")
	}
	return outcome, nil
}

// countMerger counts submits without changing any verdict.
type countMerger struct {
	control.MergeExecutor
	submits int
}

func (l *countMerger) SubmitMerge(ctx context.Context, w factory.MergeWork) (factory.OperationOutcome, error) {
	l.submits++
	return l.MergeExecutor.SubmitMerge(ctx, w)
}

// lagOnceMerger reports the first committed outcome as
// completion-pending, simulating the native lag between effect and
// completion through the exact production wait path. The submit,
// commit and confirmation stay native; only the timing is injected,
// and only when the host never exhibits the lag itself.
type lagOnceMerger struct {
	control.MergeExecutor
	submits int
	lagged  bool
}

func (l *lagOnceMerger) lag(outcome factory.OperationOutcome) factory.OperationOutcome {
	if !l.lagged && outcome.Effect == factory.OpEffectCommitted && outcome.Completion == factory.OpCompletionComplete {
		l.lagged = true
		outcome.Completion = ""
	}
	return outcome
}

func (l *lagOnceMerger) SubmitMerge(ctx context.Context, w factory.MergeWork) (factory.OperationOutcome, error) {
	l.submits++
	outcome, err := l.MergeExecutor.SubmitMerge(ctx, w)
	if err != nil {
		return outcome, err
	}
	return l.lag(outcome), nil
}

func (l *lagOnceMerger) LookupOp(ctx context.Context, id string) (factory.OperationOutcome, error) {
	outcome, err := l.MergeExecutor.LookupOp(ctx, id)
	if err != nil {
		return outcome, err
	}
	return l.lag(outcome), nil
}

// TestNativeMergeLostReply proves a lost submit reply reconciles by
// lookup: one submit, no replay, same intent, confirmed completion.
func TestNativeMergeLostReply(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeLostReplyAttempt(t, c) {
			return
		}
		t.Logf("lost reply reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("lost reply never observed its scenario outcome")
}

// nativeMergeLostReplyAttempt runs one lost-reply scenario: false means
// host churn burned the proof submit, and only then may the caller
// reseed.
func nativeMergeLostReplyAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeApprove(t, c, p, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx, p); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	dropped := &dropOnceMerger{MergeExecutor: fx.coord.Merges}
	fx.coord.Merges = dropped
	m, _ := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m.Stage != factory.MergeMerged || !dropped.dropped || dropped.submits != 1 {
		t.Fatalf("lost reply unrecovered: %+v submits=%d", m, dropped.submits)
	}
	if m.Operation.Work == nil || m.Operation.Work.OperationID != factory.MergeOperationID(p.ID, 1) {
		t.Fatalf("lost reply rewrote its intent: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p.PRCreate.HeadOID {
		t.Fatal("reconciled merge did not move the base")
	}
	nativeMergeReceipt(t, "lostreply-merge", m)
	return true
}

// TestNativeMergeCommittedIncomplete proves a committed ref never
// finishes bookkeeping: the merge waits for native completion under
// its single identity, then confirms without resubmitting. It first
// polls for the natural lag window; only when the host reports
// completion synchronously does it inject one lagged pass through
// the exact production wait path.
func TestNativeMergeCommittedIncomplete(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeCommittedIncompleteAttempt(t, c) {
			return
		}
		t.Logf("committed-incomplete reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("committed-incomplete never observed its scenario outcome")
}

func nativeMergeCommittedIncompleteAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	ctx := context.Background()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeApprove(t, c, p, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx, p); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	counted := &countMerger{MergeExecutor: fx.coord.Merges}
	fx.coord.Merges = counted
	for i := 0; i < 40; i++ {
		report := fx.coord.MergePass(ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("merge errors: %+v", report)
			}
		}
		m, err := fx.db.MergeByPublication(ctx, p.ID)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if m.Operation.Effect == factory.OpEffectCommitted && m.Operation.Completion != factory.OpCompletionComplete &&
			(m.Stage == factory.MergeOpen || m.Stage == factory.MergeFenced) {
			t.Logf("natural committed-but-incomplete window observed")
			nativeMergeReceipt(t, "committedincomplete-caught", m)
			m, _ = nativeMergeDrive(t, c, fx, p.ID)
			if m.Stage != factory.MergeMerged || counted.submits != 1 || m.Operation.Attempts != 1 ||
				m.Operation.Work == nil || m.Operation.Work.OperationID != factory.MergeOperationID(p.ID, 1) {
				t.Fatalf("caught merge misreconciled: %+v submits=%d", m, counted.submits)
			}
			nativeMergeReceipt(t, "committedincomplete-merge", m)
			return true
		}
		if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
			return false
		}
		if m.Stage != factory.MergeOpen && m.Stage != factory.MergeFenced {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	m, err := fx.db.MergeByPublication(ctx, p.ID)
	nativeMust(t, err)
	if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m.Stage != factory.MergeMerged {
		t.Fatalf("natural scenario misadvanced: %+v", m)
	}
	t.Logf("completion synchronous; proving the wait path by injected lag")
	fx2, n2 := nativeMergeSetup(t, c)
	p2, _ := nativeMergeSeedPublication(t, c, fx2, n2)
	nativeMergeApprove(t, c, p2, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p2.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx2, p2); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	lagged := &lagOnceMerger{MergeExecutor: fx2.coord.Merges}
	fx2.coord.Merges = lagged
	sawWait := false
	for i := 0; i < 40; i++ {
		report := fx2.coord.MergePass(ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("merge errors: %+v", report)
			}
		}
		for _, wait := range report.Waits {
			if wait.Reason == "native_completion_pending" {
				sawWait = true
			}
		}
		m2, err := fx2.db.MergeByPublication(ctx, p2.ID)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if m2.Stage != factory.MergeOpen && m2.Stage != factory.MergeFenced {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	m2, err := fx2.db.MergeByPublication(ctx, p2.ID)
	nativeMust(t, err)
	if m2.Stage == factory.MergeFailed && m2.Reason == factory.MergeReasonRefused && m2.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m2.Stage != factory.MergeMerged || !sawWait || !lagged.lagged || lagged.submits != 1 ||
		m2.Operation.Attempts != 1 || m2.Operation.Work == nil || m2.Operation.Work.OperationID != factory.MergeOperationID(p2.ID, 1) {
		t.Fatalf("lagged merge misreconciled: %+v sawWait=%v lagged=%v submits=%d", m2, sawWait, lagged.lagged, lagged.submits)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p2.PRCreate.HeadOID {
		t.Fatal("reconciled merge did not move the base")
	}
	nativeMergeReceipt(t, "committedincomplete-merge", m2)
	return true
}

// TestNativeMergeProtectionRefuses runs LAST: native branch protection
// refuses the merge without any Soda bypass, and the refusal is
// recorded exactly. It removes its protection before finishing.
func TestNativeMergeProtectionRefuses(t *testing.T) {
	c := loadNativeST12(t)
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	var protection struct {
		BranchName        string `json:"branch_name"`
		RequiredApprovals int64  `json:"required_approvals"`
	}
	nativeMergeAPI(t, c, http.MethodPost, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/branch_protections",
		map[string]any{"branch_name": "main", "required_approvals": 2}, &protection)
	if protection.BranchName != "main" || protection.RequiredApprovals != 2 {
		t.Fatalf("native protection unconfirmed: %+v", protection)
	}
	t.Cleanup(func() {
		nativeMergeAPI(t, c, http.MethodDelete, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/branch_protections/main", nil, nil)
	})
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeProtectionAttempt(t, c, before) {
			return
		}
		t.Logf("protection reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("protection never observed its scenario outcome")
}

// nativeMergeProtectionAttempt runs one protection scenario: false
// means host churn burned the proof submit (stale instead of the
// protection refusal), and only then may the caller reseed.
func nativeMergeProtectionAttempt(t *testing.T, c nativeST12Config, before string) bool {
	t.Helper()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeApprove(t, c, p, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx, p); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	m, _ := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m.Stage != factory.MergeFailed || m.Reason != factory.MergeReasonRefused || m.Operation.Effect != factory.OpEffectNotCommitted {
		t.Fatalf("protected merge misadvanced: %+v", m)
	}
	if m.Operation.Reason != "native_refused" || m.Operation.Attempts != 1 {
		t.Fatalf("refusal not recorded exactly: %+v", m.Operation)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("refused merge moved the base")
	}
	var issue struct {
		State string `json:"state"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(p.PRNumber, 10), nil, &issue)
	if issue.State != "open" {
		t.Fatalf("refused PR did not stay open: %q", issue.State)
	}
	nativeMergeReceipt(t, "protection-merge", m)
	return true
}
