package control

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func clearTestIDs() (publicationID, attemptRoot, reviewer, head, base, diffBase string) {
	return strings.Repeat("7", 32), strings.Repeat("b", 32), strings.Repeat("4", 32),
		strings.Repeat("2", 40), strings.Repeat("1", 40), strings.Repeat("3", 40)
}

func clearTestCycle(t *testing.T, publicationID, attemptRoot, reviewer, head, base string, round int, carried []factory.Finding) factory.ReviewCycle {
	t.Helper()
	keys := make([]string, 0, len(carried))
	for _, finding := range carried {
		keys = append(keys, finding.Key)
	}
	digest, err := factory.FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	return factory.ReviewCycle{
		PublicationID: publicationID, AttemptRoot: attemptRoot,
		ID:                 factory.ReviewCycleID(publicationID, attemptRoot, round, "review"),
		Phase:              "review",
		TriggerDecision:    factory.DeliveryReviewAgain,
		TriggerReason:      "round needs its independent inspection",
		Head:               head,
		BaseOID:            base,
		FindingDigest:      digest,
		ReviewerAssignment: reviewer,
		Round:              round,
	}
}

// clearTestReviewer builds the finished phased reviewer assignment behind
// one clear admission: the recorded typed verdict is the single source the
// consumer binds, never a parallel report or a faked candidate.
func clearTestReviewer(t *testing.T, reviewer, head, base string, cycle factory.ReviewCycle, assessments []factory.FindingAssessment, carried []factory.Finding) factory.Assignment {
	t.Helper()
	content, err := factory.FindingContentDigest(carried)
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := factory.ClearCoverageDigest(head, base, content)
	if err != nil {
		t.Fatal(err)
	}
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	a.ID, a.SourceCommit = reviewer, head
	// The reviewer is a child of the attempt root: its lineage binds
	// the recorded publication owner and cycle attempt root.
	a.ParentAssignment = cycle.AttemptRoot
	a.AttemptRoot, a.PublicationAssignment = cycle.AttemptRoot, cycle.AttemptRoot
	a.Stage, a.Outcome, a.Reason, a.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, 1700000100
	a.Result = &factory.AssignmentResult{
		AssignmentID: reviewer, RunID: a.Run, Phase: project.ExecutionReview,
		Status: "completed", Summary: "verdict recorded", Reported: true, RecordedUnix: 1700000100,
		PhaseResult: &factory.PhaseResult{
			Version: factory.PhaseResultVersion, Phase: project.ExecutionReview, Status: "completed",
			Summary: "verdict recorded",
			Output: &factory.PhaseOutput{Clear: &factory.ClearReviewReport{
				Assessments: assessments, CycleID: cycle.ID, Conclusion: factory.ClearReviewConclusion,
				Head: head, BaseOID: base, CoverageDigest: coverage,
			}},
		},
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("clear reviewer invalid: %v", err)
	}
	return a
}

func clearTestInput(t *testing.T, publicationID, attemptRoot, reviewer, head, base, diffBase string, round int, carried []factory.Finding, assessments []factory.FindingAssessment) ClearAdmissionInput {
	t.Helper()
	cycle := clearTestCycle(t, publicationID, attemptRoot, reviewer, head, base, round, carried)
	scope, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	check, err := factory.FindingContentDigest([]factory.Finding{correctionTestFinding(9, false, factory.FindingFixVerification)})
	if err != nil {
		t.Fatal(err)
	}
	return ClearAdmissionInput{
		Publication: factory.Publication{
			ID: publicationID, Candidate: head,
			AssignmentID: attemptRoot,
			PRCreate:     factory.PublicationOperation{BaseOID: base},
			ReviewCycles: []factory.ReviewCycle{cycle},
		},
		Cycle:          cycle,
		Reviewer:       clearTestReviewer(t, reviewer, head, base, cycle, assessments, carried),
		Carried:        carried,
		ScopeDigest:    scope,
		PolicyRevision: "policy-revision-9", CheckDigest: check,
		DiffBase: diffBase,
	}
}

func TestAdmitClearReviewFirstAndLater(t *testing.T) {
	publicationID, attemptRoot, reviewer, head, base, diffBase := clearTestIDs()
	first, acceptance, err := AdmitClearReview(clearTestInput(t, publicationID, attemptRoot, reviewer, head, base, diffBase, 1, nil, nil))
	if err != nil {
		t.Fatalf("first explicit clear refused: %v", err)
	}
	if acceptance.Kind != factory.AcceptanceCleanReview || acceptance.DecisionID != "" {
		t.Fatalf("first clear is not a zero-call clean acceptance: %+v", acceptance)
	}
	if err := acceptance.Validate(); err != nil {
		t.Fatalf("admitted acceptance fails validation: %v", err)
	}
	if len(first.DeliveryAcceptances) != 1 || first.DeliveryAcceptances[0].ID != acceptance.ID {
		t.Fatal("admitted acceptance was not journaled")
	}
	empty, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	if acceptance.FindingDigest != empty {
		t.Fatal("first-clear acceptance binds no exact empty input")
	}

	carried := []factory.Finding{correctionTestFinding(0, true, factory.FindingFixCode)}
	assessments := []factory.FindingAssessment{{Key: correctionTestKey(0), Status: factory.FindingVerified, Evidence: "required outcome holds on this head"}}
	later, second, err := AdmitClearReview(clearTestInput(t, publicationID, attemptRoot, reviewer, head, base, diffBase, 2, carried, assessments))
	if err != nil {
		t.Fatalf("later clear with verified history refused: %v", err)
	}
	if second.ID == acceptance.ID {
		t.Fatal("later round reused the first acceptance identity")
	}
	if len(later.DeliveryAcceptances) != 1 {
		t.Fatal("later admission journaled the wrong entry count")
	}
}

func TestAdmitClearReviewSpendsZeroDeliveryCalls(t *testing.T) {
	publicationID, attemptRoot, reviewer, head, base, diffBase := clearTestIDs()
	now := time.Now()
	unfunded := factory.AttemptAllowance{
		Limits:         factory.AttemptLimits{ActiveMinutes: 120, CorrectionCycles: 3, ReviewRounds: 6, MaxChildren: 3, DeliveryCalls: 0},
		RootAssignment: strings.Repeat("8", 32), Repository: 7, Issue: 3,
		CheckpointUnix: now.Unix(), Active: true,
	}
	if _, _, err := unfunded.ReserveDeliveryCall(strings.Repeat("9", 32), now.Add(time.Minute)); err == nil {
		t.Fatal("zero-call budget reserved a delivery call")
	}
	before := unfunded
	if _, _, err := AdmitClearReview(clearTestInput(t, publicationID, attemptRoot, reviewer, head, base, diffBase, 1, nil, nil)); err != nil {
		t.Fatalf("clear admission needs no funded call: %v", err)
	}
	if !reflect.DeepEqual(before, unfunded) {
		t.Fatal("clear admission path touched the delivery subledger")
	}
	setDigest := correctionTestDigest(t)
	reconciled, err := ReconcileAcceptedResolutions(
		JudgmentAcceptance{DecisionID: "decision-1", FindingDigest: setDigest},
		CorrectionAdmission{SetDigest: setDigest}, nil, nil,
	)
	if err != nil || len(reconciled.Resolved) != 0 || len(reconciled.Waits) != 0 {
		t.Fatalf("expected resolution readback failed: %+v %v", reconciled, err)
	}
	if !reflect.DeepEqual(before, unfunded) {
		t.Fatal("resolution readback path touched the delivery subledger")
	}
}

func TestAdmitClearReviewRefusesFalseClear(t *testing.T) {
	publicationID, attemptRoot, reviewer, head, base, diffBase := clearTestIDs()
	carried := []factory.Finding{correctionTestFinding(0, true, factory.FindingFixCode)}
	verified := []factory.FindingAssessment{{Key: correctionTestKey(0), Status: factory.FindingVerified, Evidence: "holds"}}
	newHead, newBase := strings.Repeat("9", 40), strings.Repeat("8", 40)

	cases := map[string]func(*ClearAdmissionInput){
		"silence carries no explicit conclusion": func(in *ClearAdmissionInput) {
			in.Reviewer.Result.PhaseResult.Output.Clear.Conclusion = ""
		},
		"native approval is not an explicit clear": func(in *ClearAdmissionInput) {
			in.Reviewer.Result.PhaseResult.Output.Clear.Conclusion = "approve"
		},
		"empty coverage is incomplete": func(in *ClearAdmissionInput) {
			in.Reviewer.Result.PhaseResult.Output.Clear.CoverageDigest = strings.Repeat("a", 64)
		},
		"omitted carried finding never clears": func(in *ClearAdmissionInput) {
			in.Reviewer.Result.PhaseResult.Output.Clear.Assessments = nil
		},
		"stale report tuple never clears": func(in *ClearAdmissionInput) {
			in.Reviewer.Result.PhaseResult.Output.Clear.Head = newHead
		},
		"stale cycle head never clears": func(in *ClearAdmissionInput) {
			in.Cycle.Head = newHead
			in.Publication.ReviewCycles[0].Head = newHead
		},
		"moved base never clears": func(in *ClearAdmissionInput) {
			in.Publication.PRCreate.BaseOID = newBase
		},
		"unfinished process never clears": func(in *ClearAdmissionInput) {
			in.Reviewer.Result.Status = "blocked"
		},
		"finished-on-another-candidate never clears": func(in *ClearAdmissionInput) {
			in.Reviewer.SourceCommit = newHead
		},
		"wrong reviewer assignment never clears": func(in *ClearAdmissionInput) {
			in.Reviewer.ID = strings.Repeat("6", 32)
		},
		"undistinct reviewer session never clears": func(in *ClearAdmissionInput) {
			in.Cycle.CorrectionAssignment = reviewer
			in.Publication.ReviewCycles[0].CorrectionAssignment = reviewer
		},
		"unrecorded cycle never clears": func(in *ClearAdmissionInput) {
			in.Publication.ReviewCycles = nil
		},
	}
	for name, mutate := range cases {
		in := clearTestInput(t, publicationID, attemptRoot, reviewer, head, base, diffBase, 2, carried, verified)
		mutate(&in)
		if _, _, err := AdmitClearReview(in); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
}

// A same-ID substituted cycle refuses: every immutable cycle field
// binds, so swapping the recorded head/base/digest under a known ID
// takes no admission.
func TestAdmitClearReviewRefusesSubstitutedCycle(t *testing.T) {
	reviewer := strings.Repeat("e", 32)
	head := strings.Repeat("5", 40)
	base := strings.Repeat("a", 40)
	in := clearTestInput(t, strings.Repeat("b", 32), strings.Repeat("a", 32), reviewer, head, base, base, 1, nil, nil)
	in.Cycle.Head = strings.Repeat("9", 40)
	if _, _, err := AdmitClearReview(in); err == nil {
		t.Fatal("substituted cycle admitted")
	}
}

// A foreign attempt's reviewer refuses: the reviewer assignment must
// belong to the publication's owner lineage and attempt root.
func TestAdmitClearReviewRefusesForeignReviewer(t *testing.T) {
	reviewer := strings.Repeat("e", 32)
	head := strings.Repeat("5", 40)
	base := strings.Repeat("a", 40)
	in := clearTestInput(t, strings.Repeat("b", 32), strings.Repeat("a", 32), reviewer, head, base, base, 1, nil, nil)
	in.Reviewer.PublicationAssignment = strings.Repeat("f", 32)
	if _, _, err := AdmitClearReview(in); err == nil {
		t.Fatal("foreign publication reviewer admitted")
	}
	in.Reviewer.PublicationAssignment = in.Publication.AssignmentID
	in.Reviewer.AttemptRoot = strings.Repeat("f", 32)
	if _, _, err := AdmitClearReview(in); err == nil {
		t.Fatal("foreign attempt reviewer admitted")
	}
}

func TestAdmitClearReviewDuplicateRefuses(t *testing.T) {
	publicationID, attemptRoot, reviewer, head, base, diffBase := clearTestIDs()
	in := clearTestInput(t, publicationID, attemptRoot, reviewer, head, base, diffBase, 1, nil, nil)
	published, _, err := AdmitClearReview(in)
	if err != nil {
		t.Fatal(err)
	}
	in.Publication = published
	if _, _, err := AdmitClearReview(in); err == nil {
		t.Fatal("replayed admission appended its acceptance twice")
	}
}

func TestAdmitClearReviewInvalidatesRewrittenContent(t *testing.T) {
	publicationID, attemptRoot, reviewer, head, base, diffBase := clearTestIDs()
	carried := []factory.Finding{correctionTestFinding(0, true, factory.FindingFixCode)}
	verified := []factory.FindingAssessment{{Key: correctionTestKey(0), Status: factory.FindingVerified, Evidence: "holds"}}
	in := clearTestInput(t, publicationID, attemptRoot, reviewer, head, base, diffBase, 2, carried, verified)
	rewritten := []factory.Finding{correctionTestFinding(0, true, factory.FindingFixCode)}
	rewritten[0].Defect = "rewritten defect under the same key"
	in.Carried = rewritten
	if _, _, err := AdmitClearReview(in); err == nil {
		t.Fatal("same-key rewritten content cleared under the old coverage")
	}
}

func TestAdmitCorrectionReportCanonical(t *testing.T) {
	publicationID := strings.Repeat("7", 32)
	head, base := strings.Repeat("2", 40), strings.Repeat("1", 40)
	newCandidate := strings.Repeat("9", 40)
	findings := []factory.Finding{correctionTestFinding(0, true, factory.FindingFixCode)}
	cycle := correctionTestCycle(t, publicationID, head, base, 1, correctionTestDigest(t, correctionTestKey(0)))
	report := correctionTestReport(newCandidate, findings, []factory.CorrectionDisposition{{Key: correctionTestKey(0), Disposition: factory.DispositionUnresolved}})
	if _, err := AdmitCorrectionReportCanonical(cycle, strings.Repeat("4", 32), newCandidate, findings, report, nil, nil); err != nil {
		t.Fatalf("canonical correction refused: %v", err)
	}
	rewritten := []factory.Finding{correctionTestFinding(0, true, factory.FindingFixCode)}
	rewritten[0].Defect = "rewritten defect under the same key"
	bad := correctionTestReport(newCandidate, rewritten, []factory.CorrectionDisposition{{Key: correctionTestKey(0), Disposition: factory.DispositionUnresolved}})
	if _, err := AdmitCorrectionReportCanonical(cycle, strings.Repeat("4", 32), newCandidate, findings, bad, nil, nil); err == nil {
		t.Fatal("agent-rewritten canonical finding admitted")
	}
}
