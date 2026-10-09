package control_test

// Native check-assessment proof (ST11, S-checks). Fixture candidates, PRs
// and ordinary commit statuses are prepared through native Git and REST.
// Assessment reads bracketed native snapshots and disclosed accepted-policy
// records; agent narrative never enters the verdict. Workflow cases also
// require Actions enabled without a runner, and read genuine Actions-produced
// pending evidence through native snapshots and ordinary run/status APIs.
// Absence of SODA_ST11_NATIVE skips native work and never counts as passing
// evidence. Ordinary-status and workflow producer scopes remain distinct.

import (
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestNativeCheckPass(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-build", "st11-review-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "pass")
	nativePostStatus(t, c, p.head, "st11-build", "success")
	nativePostStatus(t, c, p.head, "st11-review-gate", "success")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "pass", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckPass || assessment.Reason != factory.CheckReasonPass {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Logf("PASS exact head/base satisfies every configured required check (native rev %d, policy rev %d)", assessment.NativeRev, assessment.PolicyRevision)
}

func TestNativeCheckPending(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "pending")
	nativePostStatus(t, c, p.head, "st11-gate", "pending")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "pending", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckPending || assessment.Reason != factory.CheckReasonPending {
		t.Fatalf("verdict: %+v", assessment)
	}
	if len(assessment.Results) != 1 || assessment.Results[0].State != "pending" {
		t.Fatalf("pending state not observed: %+v", assessment.Results)
	}
	t.Log("PASS pending check waits without failing")
}

func TestNativeCheckMissing(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-present", "st11-absent"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "missing")
	nativePostStatus(t, c, p.head, "st11-present", "success")
	nativePostStatus(t, c, p.head, "st11-unrequired", "success")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "missing", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckPending || assessment.Reason != factory.CheckReasonPending {
		t.Fatalf("verdict: %+v", assessment)
	}
	if assessment.Results[1].State != "" || assessment.Results[1].Passed {
		t.Fatalf("missing check not reported: %+v", assessment.Results)
	}
	t.Log("PASS missing required check waits while unrequired success is ignored")
}

func TestNativeCheckFailed(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "failed")
	nativePostStatus(t, c, p.head, "st11-gate", "failure")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "failed", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckFailed || assessment.Reason != factory.CheckReasonFailed {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS failed check fails the assessment")
}

func TestNativeCheckCancelled(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "cancelled")
	nativePostStatus(t, c, p.head, "st11-gate", "cancelled")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "cancelled", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckFailed || assessment.Reason != factory.CheckReasonCancelled {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS cancelled check fails the assessment")
}

func TestNativeCheckSkipped(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "skipped")
	nativePostStatus(t, c, p.head, "st11-gate", "skipped")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "skipped", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckFailed || assessment.Reason != factory.CheckReasonSkipped {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS skipped check fails the assessment")
}

func TestNativeCheckUnknownState(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "unknown")
	// The native API stores arbitrary state strings; only exact success
	// passes, so an unrecognized state fails closed.
	nativePostStatus(t, c, p.head, "st11-gate", "bogus")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "unknown-state", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckFailed || assessment.Reason != factory.CheckReasonUnknownState {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS unrecognized native state fails closed")
}

func TestNativeCheckLatestWins(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "latest")
	nativePostStatus(t, c, p.head, "st11-gate", "failure")
	nativePostStatus(t, c, p.head, "st11-gate", "success")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	assessment := nativeVerifyChecks(t, db, "latest-wins", p.target(c), nativeCheckAdopted(policy), policy, observed)
	if assessment.Verdict != factory.CheckPass {
		t.Fatalf("verdict: %+v", assessment)
	}
	if len(assessment.Results) != 1 || assessment.Results[0].State != "success" {
		t.Fatalf("latest status did not win: %+v", assessment.Results)
	}
	t.Log("PASS latest status per context governs, matching combined-status semantics")
}

func TestNativeCheckPolicyEmpty(t *testing.T) {
	c := loadNativeST11(t)
	db := nativeCheckDB(t)
	policy := nativeCheckPolicy(t, db, c, []string{"st11-gate"})
	p := nativeCheckPublish(t, c, nativeNewCandidate(t, c), "empty")
	nativePostStatus(t, c, p.head, "st11-gate", "success")
	observed := nativeObserveChecks(t, nativeCheckAssessor(t, c), p.target(c), c.ActorID)
	// A missing policy arrives as the zero record: zero configured checks
	// can never authorize a merge, however green the head looks.
	var missing factory.RepositoryPolicy
	assessment := nativeVerifyChecks(t, db, "policy-empty", p.target(c), nativeCheckAdopted(policy), missing, observed)
	if assessment.Verdict != factory.CheckRefused || assessment.Reason != factory.CheckReasonPolicyEmpty {
		t.Fatalf("verdict: %+v", assessment)
	}
	t.Log("PASS zero configured checks refuse despite green evidence")
}
