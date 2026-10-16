package control

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func resolutionTestAdmission() CorrectionAdmission {
	return CorrectionAdmission{
		CycleID: strings.Repeat("7", 32), RunID: strings.Repeat("c", 32),
		SetDigest:    strings.Repeat("e", 64),
		NewCandidate: strings.Repeat("d", 40),
		Fixed:        []string{correctionTestKey(1), correctionTestKey(2)},
	}
}

func resolutionTestDigest() string { return strings.Repeat("a", 64) }

func resolutionTestPlan(t *testing.T, admission CorrectionAdmission) []AcceptedResolution {
	t.Helper()
	publication := factory.Publication{ID: strings.Repeat("9", 32)}
	mappings := []factory.ReviewCommentMapping{{Key: correctionTestKey(1), CommentID: 11}}
	evidence := map[int64]factory.ThreadEvidence{
		11: {Digest: resolutionTestDigest(), Repository: 7, PRNumber: 3, RootCommentID: 11},
	}
	binding := factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpThreadResolve}
	planned, err := PlanAcceptedResolutions(publication, admission, mappings, evidence, binding, 1,
		"refs/heads/fix", "refs/heads/main", admission.NewCandidate, strings.Repeat("b", 40), 7, 3, 5)
	if err != nil {
		t.Fatal(err)
	}
	return planned
}

func TestAcceptedResolveOperationIDStable(t *testing.T) {
	first := AcceptedResolveOperationID(strings.Repeat("9", 32), strings.Repeat("7", 32), correctionTestKey(1))
	second := AcceptedResolveOperationID(strings.Repeat("9", 32), strings.Repeat("7", 32), correctionTestKey(1))
	if first != second || !factory.ValidID(first) {
		t.Fatalf("resolve operation identity unstable: %q %q", first, second)
	}
	if other := AcceptedResolveOperationID(strings.Repeat("9", 32), strings.Repeat("7", 32), correctionTestKey(2)); other == first {
		t.Fatal("resolve operation identity ignores its finding key")
	}
}

func TestPlanAcceptedResolutionsBindsEvidenceBeforeWrites(t *testing.T) {
	planned := resolutionTestPlan(t, resolutionTestAdmission())
	if len(planned) != 2 {
		t.Fatalf("planned resolutions omit an accepted fix: %+v", planned)
	}
	var threaded, vacuous *AcceptedResolution
	for i := range planned {
		if planned[i].NoNativeThread {
			vacuous = &planned[i]
		} else {
			threaded = &planned[i]
		}
	}
	if threaded == nil || vacuous == nil {
		t.Fatalf("threaded and threadless fixes misplanned: %+v", planned)
	}
	if threaded.FindingKey != correctionTestKey(1) || threaded.Request.RootCommentID != 11 ||
		threaded.Request.Digest != resolutionTestDigest() || !threaded.Request.Resolved {
		t.Fatalf("resolve request differs from its evidence: %+v", threaded)
	}
	if threaded.OperationID == "" || vacuous.OperationID == "" || threaded.OperationID == vacuous.OperationID {
		t.Fatalf("resolve operation identities missing: %+v", planned)
	}
}

func TestPlanAcceptedResolutionsRefuses(t *testing.T) {
	admission := resolutionTestAdmission()
	publication := factory.Publication{ID: strings.Repeat("9", 32)}
	mappings := []factory.ReviewCommentMapping{{Key: correctionTestKey(1), CommentID: 11}}
	goodEvidence := map[int64]factory.ThreadEvidence{
		11: {Digest: resolutionTestDigest(), Repository: 7, PRNumber: 3, RootCommentID: 11},
	}
	binding := factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpThreadResolve}
	plan := func(evidence map[int64]factory.ThreadEvidence, b factory.ActorBindingRef, author int64, head string) error {
		_, err := PlanAcceptedResolutions(publication, admission, mappings, evidence, b, author,
			"refs/heads/fix", "refs/heads/main", head, strings.Repeat("b", 40), 7, 3, 5)
		return err
	}
	if err := plan(nil, binding, 1, admission.NewCandidate); err == nil {
		t.Fatal("missing thread evidence accepted")
	}
	deleted := map[int64]factory.ThreadEvidence{
		11: {Digest: resolutionTestDigest(), Repository: 7, PRNumber: 3, RootCommentID: 11, Deleted: true},
	}
	if err := plan(deleted, binding, 1, admission.NewCandidate); err == nil {
		t.Fatal("deleted resolve root accepted")
	}
	if err := plan(goodEvidence, factory.ActorBindingRef{}, 1, admission.NewCandidate); err == nil {
		t.Fatal("missing resolve binding accepted")
	}
	if err := plan(goodEvidence, binding, 6, admission.NewCandidate); err == nil {
		t.Fatal("PR author resolving its own thread accepted")
	}
	if err := plan(goodEvidence, binding, 1, strings.Repeat("f", 40)); err == nil {
		t.Fatal("resolutions for a superseded head accepted")
	}
}

func TestReconcileAcceptedResolutionsNeedsJudgmentAcceptance(t *testing.T) {
	admission := resolutionTestAdmission()
	planned := resolutionTestPlan(t, admission)
	confirmed := map[string]bool{correctionTestKey(1): true}
	// Native resolved state alone is never semantic proof: even fully
	// confirmed readback without a bound acceptance refuses.
	if _, err := ReconcileAcceptedResolutions(JudgmentAcceptance{}, admission, planned, confirmed); err == nil {
		t.Fatal("resolution without judgment acceptance reconciled")
	}
	stale := JudgmentAcceptance{DecisionID: strings.Repeat("d", 32), FindingDigest: strings.Repeat("f", 64)}
	if _, err := ReconcileAcceptedResolutions(stale, admission, planned, confirmed); err == nil {
		t.Fatal("stale judgment acceptance reconciled")
	}
}

func TestReconcileAcceptedResolutionsAccountsEveryFixOnce(t *testing.T) {
	admission := resolutionTestAdmission()
	planned := resolutionTestPlan(t, admission)
	acceptance := JudgmentAcceptance{DecisionID: strings.Repeat("d", 32), FindingDigest: admission.SetDigest}
	full := map[string]bool{correctionTestKey(1): true}
	outcome, err := ReconcileAcceptedResolutions(acceptance, admission, planned, full)
	if err != nil {
		t.Fatalf("confirmed reconciliation refused: %v", err)
	}
	if len(outcome.Resolved) != 2 || len(outcome.Waits) != 0 {
		t.Fatalf("confirmed reconciliation incomplete: %+v", outcome)
	}
	partial, err := ReconcileAcceptedResolutions(acceptance, admission, planned, map[string]bool{})
	if err != nil {
		t.Fatalf("pending reconciliation refused: %v", err)
	}
	if len(partial.Waits) != 1 || partial.Waits[0] != correctionTestKey(1) {
		t.Fatalf("pending resolution not waited: %+v", partial)
	}
	short := planned[:1]
	if _, err := ReconcileAcceptedResolutions(acceptance, admission, short, full); err == nil {
		t.Fatal("resolutions omitting an accepted fix reconciled")
	}
	foreign := append(append([]AcceptedResolution{}, planned...), AcceptedResolution{
		FindingKey: correctionTestKey(77), NoNativeThread: true,
	})
	if _, err := ReconcileAcceptedResolutions(acceptance, admission, foreign, full); err == nil {
		t.Fatal("resolution for an unaccepted finding reconciled")
	}
}
