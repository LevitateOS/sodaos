package control_test

import (
	"context"
	"testing"
)

func (fx *st15Fixture) writeFinalReceipt() {
	st15Receipt(fx.t, "journey", map[string]any{
		"status": "passed", "qualification": "development-only",
		"checks":      fx.checks,
		"issues":      map[string]any{"P": fx.issuePIndex, "A": fx.issueAIndex, "B": fx.issueBIndex, "C": fx.issueCIndex},
		"acceptances": map[string]any{"P": fx.acceptP, "A": fx.acceptA, "B": fx.acceptB},
		"assignments": map[string]any{"A": fx.assignA, "B": fx.assignB},
		"publication": fx.pubA, "pr": fx.prNumber, "head1": fx.head1, "head2": fx.head2,
		"reviews": map[string]any{"r1": fx.review1.ReviewID, "v1": fx.verdict1, "r2": fx.review2.ReviewID, "v2": fx.verdict2},
		"ci":      map[string]any{"fail_rev": fx.ciFailRev, "pass_rev": fx.ciPassRev},
	})
}

func TestST15ComposedDemo(t *testing.T) {
	cfg := loadST15(t)
	t.Setenv("ST09_RECEIPT_DIR", st15ReceiptDir(t))
	fx := &st15Fixture{t: t, ctx: context.Background(), cfg: cfg}
	fx.check("host-stack", fx.setupHostStack())
	fx.check("factory-store", fx.openFactoryStore())
	fx.check("wire-coordinator", fx.wireCoordinator())
	fx.check("project", fx.setupProject())
	fx.check("seed-repository", fx.seedRepository())
	fx.check("prepare-roles", fx.prepareRoles())
	fx.check("human-work", fx.seedHumanWork())
	fx.stallForBrowserDiag()
	fx.check("grant-withdrawal", fx.proveGrantWithdrawal())
	fx.check("seed-issues", fx.seedIssues())
	fx.check("accept-P", fx.admitPLeg())
	fx.check("accept-A", fx.admitALeg())
	fx.check("accept-BC", fx.admitBCLegs())
	fx.check("closure-not-outcome", fx.proveClosureNotOutcome())
	fx.check("provider-gate", fx.providerGate())
	fx.check("activate-authority", fx.activateAuthority())
	fx.check("dispatch-A", fx.dispatchA())
	fx.check("publish-A", fx.publishA())
	fx.check("review-1", fx.reviewLeg1())
	fx.check("ci-fail", fx.ciFail())
	fx.check("correct-A", fx.correctA())
	fx.check("review-2", fx.reviewLeg2())
	fx.check("ci-pass", fx.ciPass())
	fx.check("merge-and-dependants", fx.mergeAndDependants())
	fx.check("retention", fx.proveRetention())
	fx.writeFinalReceipt()
}
