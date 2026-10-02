package control_test

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// readEvidence reads one revision-bound evidence set through the same
// production source the coordinator verifies against.
func (fx *st15Fixture) readEvidence(index int64, commentIDs ...string) control.AcceptanceEvidence {
	fx.t.Helper()
	evidence, err := fx.coord.AcceptanceReads.ReadAcceptanceEvidence(fx.ctx,
		strconv.FormatInt(fx.cfg.Repository, 10), strconv.FormatInt(index, 10), commentIDs)
	if err != nil {
		fx.t.Fatalf("ST15 evidence #%d: %v", index, err)
	}
	return evidence
}

func (fx *st15Fixture) admitDecision(decision factory.Acceptance) string {
	fx.t.Helper()
	receipt, err := fx.coord.AdmitAcceptance(fx.ctx, factory.NewID(), "soda-maintainer", decision)
	if err != nil {
		fx.t.Fatalf("ST15 admit #%s: %v", decision.IssueIndex, err)
	}
	st15Receipt(fx.t, "accept-"+decision.IssueIndex+"-"+decision.ID, receipt)
	if receipt.DecisionID != decision.ID {
		fx.t.Fatalf("ST15 admit #%s admitted foreign decision %q", decision.IssueIndex, receipt.DecisionID)
	}
	return receipt.DecisionID
}

func (fx *st15Fixture) observeControl(index int64) factory.IssueControl {
	fx.t.Helper()
	ctrl, _, err := fx.coord.ObserveIssueEvent(fx.ctx, control.IntakeHint{
		Delivery: "st15-observe-" + st15RandHex(8), Repository: fx.cfg.Repository, Issue: index,
	})
	if err != nil {
		fx.t.Fatalf("ST15 observe #%d: %v", index, err)
	}
	return ctrl
}

func blockerCodes(ctrl factory.IssueControl) []string {
	codes := []string{}
	for _, b := range ctrl.Blockers {
		codes = append(codes, b.Code)
	}
	return codes
}

func requireBlocker(t interface {
	Helper()
	Fatalf(string, ...any)
}, ctrl factory.IssueControl, code string,
) {
	t.Helper()
	for _, have := range blockerCodes(ctrl) {
		if have == code {
			return
		}
	}
	t.Fatalf("ST15 #%d readiness %q lacks blocker %q (have %v)", ctrl.Issue, ctrl.Readiness, code, blockerCodes(ctrl))
}

// seedIssues creates the four journey issues and their dependency edges:
// P (question, blocks A by result), A (the work), B (blocked by A by
// code), C (decoy for the withdrawal leg).
func (fx *st15Fixture) seedIssues() error {
	issueP := fx.createIssue("ST15-P: confirm the widget API freeze",
		"Question for the maintainer: is the total() signature frozen, so the factory may fix its body without an API review?")
	issueA := fx.createIssue("ST15-A: fix total() to return the exact sum",
		"total() in widget.py adds a stray 1. Fix it to return the exact sum.\n\nQuestion for the maintainer: should empty input return 0?")
	issueB := fx.createIssue("ST15-B: document total() behavior",
		"Follow-up once ST15-A merges: document total() in the README.")
	issueC := fx.createIssue("ST15-C: decoy for withdrawal",
		"This issue is accepted and then withdrawn; it must never dispatch.")
	fx.issueP, fx.issuePIndex = issueP.ID, issueP.Number
	fx.issueA, fx.issueAIndex = issueA.ID, issueA.Number
	fx.issueB, fx.issueBIndex = issueB.ID, issueB.Number
	fx.issueC, fx.issueCIndex = issueC.ID, issueC.Number
	// Creation hints first: each pristine issue adopts its verified
	// initial acceptance automatically. Edges land after, so initials
	// never see a blocked creation.
	for _, issue := range []st15Issue{issueP, issueA, issueB, issueC} {
		fx.issueOpenedHint(issue)
	}
	// REST fixture creation does not establish verified original-creation
	// provenance (ST09 precedent): the best-effort initial adoption in the
	// creation hints refuses, and the legs below adopt the exact native
	// inputs through explicit maintainer decisions instead.
	for _, index := range []int64{issueP.Number, issueA.Number, issueB.Number, issueC.Number} {
		validity, err := fx.coord.AcceptanceStatus(fx.ctx, fx.cfg.Repository, strconv.FormatInt(index, 10))
		if err != nil {
			fx.t.Fatalf("ST15 #%d acceptance status: %v", index, err)
		}
		if validity.Valid || validity.Acceptance != nil {
			fx.t.Fatalf("ST15 #%d unexpectedly holds an acceptance: %+v", index, validity)
		}
		ctrl := fx.observeControl(index)
		if ctrl.Readiness == factory.ReadinessQueued || ctrl.Readiness == factory.ReadinessActive {
			fx.t.Fatalf("ST15 #%d runnable before acceptance: %+v", index, ctrl)
		}
	}
	fx.insertEdge(issueA.ID, issueP.ID)
	fx.insertEdge(issueB.ID, issueA.ID)
	st15Receipt(fx.t, "issues", map[string]any{
		"P": map[string]any{"id": issueP.ID, "index": issueP.Number},
		"A": map[string]any{"id": issueA.ID, "index": issueA.Number},
		"B": map[string]any{"id": issueB.ID, "index": issueB.Number},
		"C": map[string]any{"id": issueC.ID, "index": issueC.Number},
	})
	return nil
}

// acceptP resolves the prerequisite: close-without-answer first (closure
// alone resolves nothing), then the authorized answer, acceptance with
// resolutions, and close.
func (fx *st15Fixture) admitPLeg() error {
	t := fx.t
	ctrl := fx.observeControl(fx.issuePIndex)
	if ctrl.Readiness == factory.ReadinessQueued || ctrl.Readiness == factory.ReadinessActive {
		return fmt.Errorf("unaccepted P already runnable: %+v", ctrl)
	}
	// Refusal first: an acceptance that selects a missing answer refuses.
	evidence := fx.readEvidence(fx.issuePIndex)
	bogus := factory.Acceptance{
		ID: "d" + st15RandHex(12), IssueIndex: strconv.FormatInt(fx.issuePIndex, 10),
		TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest,
		Sources:    []factory.SelectedSource{{ID: "999999999", Digest: strings.Repeat("a", 64), ContentVersion: 1}},
		Repository: fx.cfg.Repository, Approver: fx.cfg.CreatorID, NativeRev: evidence.Revision, ContentVersion: evidence.Issue.ContentVer,
	}
	if _, err := fx.coord.AdmitAcceptance(fx.ctx, factory.NewID(), "soda-maintainer", bogus); err == nil {
		return errors.New("acceptance with a missing answer source admitted")
	} else {
		var refusal *control.AcceptanceRefusal
		if !errors.As(err, &refusal) || refusal.Reason != control.RefusalSourceMissing {
			return fmt.Errorf("missing-answer refusal: %v", err)
		}
		t.Logf("ST15 P: missing-answer acceptance refused (%s)", refusal.Reason)
	}
	// Closure without an answer resolves nothing.
	fx.api(fx.cfg.CreatorTokenFile, "PATCH", "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/"+strconv.FormatInt(fx.issuePIndex, 10),
		map[string]any{"state": "closed"}, &map[string]any{})
	fx.commentHint(fx.issuePIndex)
	ctrl = fx.observeControl(fx.issuePIndex)
	if ctrl.Readiness == factory.ReadinessActive {
		return fmt.Errorf("closed-unanswered P runnable: %+v", ctrl)
	}
	// The authorized answer reopens the question and resolves it.
	fx.api(fx.cfg.CreatorTokenFile, "PATCH", "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/"+strconv.FormatInt(fx.issuePIndex, 10),
		map[string]any{"state": "open"}, &map[string]any{})
	answerID := fx.commentIssue(fx.issuePIndex, "Yes: total() is frozen. Fix the body only.")
	fx.commentHint(fx.issuePIndex)
	evidence = fx.readEvidence(fx.issuePIndex, strconv.FormatInt(answerID, 10))
	answer := evidence.Comments[0]
	decision := factory.Acceptance{
		ID: "d" + st15RandHex(12), IssueIndex: strconv.FormatInt(fx.issuePIndex, 10),
		TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest,
		Sources:     []factory.SelectedSource{{ID: answer.ID, Digest: answer.Digest, ContentVersion: answer.ContentVer}},
		Resolutions: []factory.SelectedSource{{ID: answer.ID, Digest: answer.Digest, ContentVersion: answer.ContentVer}},
		Repository:  fx.cfg.Repository, Approver: fx.cfg.CreatorID, NativeRev: evidence.Revision, ContentVersion: evidence.Issue.ContentVer,
	}
	fx.acceptP = fx.admitDecision(decision)
	fx.api(fx.cfg.CreatorTokenFile, "PATCH", "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/"+strconv.FormatInt(fx.issuePIndex, 10),
		map[string]any{"state": "closed"}, &map[string]any{})
	fx.commentHint(fx.issuePIndex)
	st15Receipt(fx.t, "accept-P", map[string]any{"decision": fx.acceptP, "answer": answerID})
	return nil
}

// acceptA accepts the work issue once its own question is answered: first
// the acceptance is impossible (missing answer), then the blocker set
// shows the result prerequisite, then the answer admits the decision.
func (fx *st15Fixture) admitALeg() error {
	t := fx.t
	evidence := fx.readEvidence(fx.issueAIndex)
	var edge control.AcceptanceEdge
	for _, candidate := range evidence.Dependencies {
		if candidate.Visible {
			edge = candidate
		}
	}
	if edge.Occurrence == "" {
		return fmt.Errorf("A edge to P not observed: %+v", evidence.Dependencies)
	}
	bogus := factory.Acceptance{
		ID: "d" + st15RandHex(12), IssueIndex: strconv.FormatInt(fx.issueAIndex, 10),
		TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest,
		Sources: []factory.SelectedSource{{ID: "999999998", Digest: strings.Repeat("b", 64), ContentVersion: 1}},
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: edge.Occurrence, DependsOn: edge.DependsOn,
			EndpointRepo: fx.cfg.Repository, EndpointIssue: fx.issuePIndex, Outcome: factory.PrereqResult,
		}},
		Repository: fx.cfg.Repository, Approver: fx.cfg.CreatorID, NativeRev: evidence.Revision, ContentVersion: evidence.Issue.ContentVer,
	}
	if _, err := fx.coord.AdmitAcceptance(fx.ctx, factory.NewID(), "soda-maintainer", bogus); err == nil {
		return errors.New("A acceptance with a missing answer admitted")
	} else {
		var refusal *control.AcceptanceRefusal
		if !errors.As(err, &refusal) || refusal.Reason != control.RefusalSourceMissing {
			return fmt.Errorf("A missing-answer refusal: %v", err)
		}
		t.Logf("ST15 A: missing-answer acceptance refused (%s)", refusal.Reason)
	}
	answerID := fx.commentIssue(fx.issueAIndex, "Yes: empty input returns 0. Fix the sum and keep the frozen signature.")
	fx.answerA = answerID
	fx.commentHint(fx.issueAIndex)
	evidence = fx.readEvidence(fx.issueAIndex, strconv.FormatInt(answerID, 10))
	for _, candidate := range evidence.Dependencies {
		if candidate.Visible {
			edge = candidate
		}
	}
	answer := evidence.Comments[0]
	decision := factory.Acceptance{
		ID: "d" + st15RandHex(12), IssueIndex: strconv.FormatInt(fx.issueAIndex, 10),
		TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest,
		Sources: []factory.SelectedSource{{ID: answer.ID, Digest: answer.Digest, ContentVersion: answer.ContentVer}},
		// The dependent records the accepted resolution: P's closure
		// under this occurrence satisfies the result prerequisite.
		Resolutions: []factory.SelectedSource{{ID: answer.ID, Digest: answer.Digest, ContentVersion: answer.ContentVer}},
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: edge.Occurrence, DependsOn: edge.DependsOn,
			EndpointRepo: fx.cfg.Repository, EndpointIssue: fx.issuePIndex, Outcome: factory.PrereqResult,
		}},
		Repository: fx.cfg.Repository, Approver: fx.cfg.CreatorID, NativeRev: evidence.Revision, ContentVersion: evidence.Issue.ContentVer,
	}
	fx.acceptA = fx.admitDecision(decision)
	st15Receipt(fx.t, "accept-A", map[string]any{"decision": fx.acceptA, "answer": answerID})
	return nil
}

// acceptBAndC accepts the dependant (code prerequisite on A, blocked
// until A merges) and the decoy (withdrawn before it can dispatch).
func (fx *st15Fixture) admitBCLegs() error {
	evidence := fx.readEvidence(fx.issueBIndex)
	var edge control.AcceptanceEdge
	for _, candidate := range evidence.Dependencies {
		if candidate.Visible {
			edge = candidate
		}
	}
	if edge.Occurrence == "" {
		return fmt.Errorf("B edge to A not observed: %+v", evidence.Dependencies)
	}
	decision := factory.Acceptance{
		ID: "d" + st15RandHex(12), IssueIndex: strconv.FormatInt(fx.issueBIndex, 10),
		TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest,
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: edge.Occurrence, DependsOn: edge.DependsOn, PrereqAcceptance: fx.acceptA,
			EndpointRepo: fx.cfg.Repository, EndpointIssue: fx.issueAIndex, Outcome: factory.PrereqCode,
		}},
		Repository: fx.cfg.Repository, Approver: fx.cfg.CreatorID, NativeRev: evidence.Revision, ContentVersion: evidence.Issue.ContentVer,
	}
	fx.acceptB = fx.admitDecision(decision)
	ctrl := fx.observeControl(fx.issueBIndex)
	requireBlocker(fx.t, ctrl, factory.BlockerCodePending)

	evidenceC := fx.readEvidence(fx.issueCIndex)
	decC := factory.Acceptance{
		ID: "d" + st15RandHex(12), IssueIndex: strconv.FormatInt(fx.issueCIndex, 10),
		TitleDigest: evidenceC.Issue.TitleDigest, ContentDigest: evidenceC.Issue.ContentDigest,
		Repository: fx.cfg.Repository, Approver: fx.cfg.CreatorID, NativeRev: evidenceC.Revision, ContentVersion: evidenceC.Issue.ContentVer,
	}
	admittedC := fx.admitDecision(decC)
	if _, err := fx.coord.WithdrawAcceptance(fx.ctx, factory.NewID(), "soda-maintainer", fx.cfg.Repository, fx.issueCIndex, admittedC, fx.cfg.CreatorID); err != nil {
		return fmt.Errorf("C withdrawal: %w", err)
	}
	ctrlC := fx.observeControl(fx.issueCIndex)
	if ctrlC.Readiness == factory.ReadinessQueued || ctrlC.Readiness == factory.ReadinessActive {
		return fmt.Errorf("withdrawn C runnable: %+v", ctrlC)
	}
	st15Receipt(fx.t, "accept-BC", map[string]any{"B": fx.acceptB, "B_blockers": blockerCodes(ctrl), "C": admittedC, "C_readiness": ctrlC.Readiness})
	return nil
}

// proveClosureNotOutcome closes A without a merge while B watches: B's
// code prerequisite stays pending, and A's acceptance survives the
// close/reopen cycle.
func (fx *st15Fixture) proveClosureNotOutcome() error {
	fx.api(fx.cfg.CreatorTokenFile, "PATCH", "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/"+strconv.FormatInt(fx.issueAIndex, 10),
		map[string]any{"state": "closed"}, &map[string]any{})
	fx.commentHint(fx.issueBIndex)
	ctrl := fx.observeControl(fx.issueBIndex)
	requireBlocker(fx.t, ctrl, factory.BlockerCodePending)
	fx.api(fx.cfg.CreatorTokenFile, "PATCH", "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/issues/"+strconv.FormatInt(fx.issueAIndex, 10),
		map[string]any{"state": "open"}, &map[string]any{})
	fx.commentHint(fx.issueAIndex)
	validity, err := fx.coord.AcceptanceStatus(fx.ctx, fx.cfg.Repository, strconv.FormatInt(fx.issueAIndex, 10))
	if err != nil {
		return fmt.Errorf("A acceptance status: %w", err)
	}
	if !validity.Valid || validity.Acceptance == nil || validity.Acceptance.ID != fx.acceptA {
		return fmt.Errorf("A acceptance did not survive close/reopen: %+v", validity)
	}
	ctrlA := fx.observeControl(fx.issueAIndex)
	// The environment grant stays revoked until the provider gate, so
	// reopened A waits on authority — but the result prerequisite is
	// satisfied (P closed with the accepted resolution) and the
	// acceptance head survived the close/reopen cycle.
	requireBlocker(fx.t, ctrlA, factory.BlockerAuthorityMissing)
	for _, code := range blockerCodes(ctrlA) {
		if code == factory.BlockerResultPending {
			return fmt.Errorf("reopened A still result-blocked: %+v", ctrlA)
		}
	}
	st15Receipt(fx.t, "closure-not-outcome", map[string]any{
		"B_blockers_while_closed": blockerCodes(ctrl), "A_decision": validity.Acceptance.ID, "A_readiness": ctrlA.Readiness,
	})
	return nil
}

// proveGrantWithdrawal shows dispatch waiting while the environment
// grant is revoked. The grant stays revoked until the provider gate, so
// every intake auto-dispatch safely no-ops through the input legs.
func (fx *st15Fixture) proveGrantWithdrawal() error {
	grant, err := fx.db.EnvironmentGrant(fx.ctx, fx.cfg.Repository)
	if err != nil {
		return err
	}
	if grant.Active {
		grant.Active = false
		if err := fx.db.SaveEnvironmentGrant(fx.ctx, grant); err != nil {
			return err
		}
	}
	report := fx.coord.Dispatch(fx.ctx)
	if len(report.Launched) != 0 {
		return fmt.Errorf("dispatch advanced without its environment grant: %+v", report)
	}
	st15Receipt(fx.t, "grant-withdrawal", map[string]any{"dispatched_while_revoked": 0})
	return nil
}

// activateAuthority restores the environment grant after the provider
// gate, so the journey proceeds to real execution.
func (fx *st15Fixture) activateAuthority() error {
	grant, err := fx.db.EnvironmentGrant(fx.ctx, fx.cfg.Repository)
	if err != nil {
		return err
	}
	grant.Active = true
	return fx.db.SaveEnvironmentGrant(fx.ctx, grant)
}

var _ = forgejo.ContentDigest
