package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

// withReportCycle appends one finding-set-digest correction cycle for a
// corrector child to a stored publication.
func withReportCycle(t *testing.T, ctx context.Context, fx *publishFixture, p factory.Publication, child factory.Assignment, digest string) factory.ReviewCycle {
	t.Helper()
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	round := 1
	for _, cycle := range p.ReviewCycles {
		if cycle.Round >= round {
			round = cycle.Round + 1
		}
	}
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID:                   factory.ReviewCycleID(p.ID, owner.AttemptRoot, round, "review"),
		Phase:                "review",
		TriggerDecision:      "correction-after-review",
		TriggerReason:        "address request-changes on the published head",
		Head:                 p.Candidate,
		BaseOID:              p.PRCreate.BaseOID,
		FindingDigest:        digest,
		ReviewerAssignment:   factory.NewID(),
		CorrectionAssignment: child.ID,
		Round:                round,
	}
	p.ReviewCycles = append(p.ReviewCycles, cycle)
	p.Revision++
	if err := fx.db.UpdatePublication(ctx, p); err != nil {
		t.Fatal(err)
	}
	return cycle
}

// withReportFixed builds one fixed code-finding report with its mechanical
// evidence and a check assessment on the new candidate.
func withReportFixed(t *testing.T, p factory.Publication, newHead string) (factory.CorrectionReport, []CorrectionFixEvidence, *factory.CheckAssessment) {
	t.Helper()
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	report := correctionTestReport([]factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionFixed}})
	evidence := []CorrectionFixEvidence{{Key: finding.Key, ChangedPaths: []string{"service.go"}, Commits: []string{newHead}}}
	checks := correctionTestChecks(newHead, p.PRCreate.BaseOID)
	return report, evidence, checks
}

// TestPublishCorrectionWithReportJournalsFixed (real pg) proves the
// canonical serving correction entry: a fixed report journals through
// the canonical gate and advances the head, exactly once per run.
func TestPublishCorrectionWithReportJournalsFixed(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeed(t, 42)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	run := correctionRun(t, fx, owner, newHead)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report, evidence, checks := withReportFixed(t, p, newHead)
	served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, checks)
	if len(served.Errors) != 0 || len(served.Waits) != 0 || len(served.Corrected) != 1 || served.Corrected[0].HeadOID != newHead {
		t.Fatalf("served correction: %+v", served)
	}
	after, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.CorrectionJournals) != 1 || after.CorrectionJournals[0].RunID != run.ID {
		t.Fatalf("correction journal missing: %+v", after.CorrectionJournals)
	}
	if after.Candidate != newHead {
		t.Fatalf("head did not advance: %q", after.Candidate)
	}
	replay := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, checks)
	if len(replay.Corrected) != 0 {
		t.Fatalf("replay re-advanced: %+v", replay)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.CorrectionJournals) != 1 {
		t.Fatalf("replay duplicated its journal: %+v", held.CorrectionJournals)
	}
}

// TestPublishCorrectionWithReportWithholdsDisputed (real pg) proves the
// disputed-journal shape now arises through serving: a disputed report
// journals, the head does not advance on contested feedback, and the
// next child-production pass withholds the run pending the funded
// decision instead of launching an automatic second coder run.
func TestPublishCorrectionWithReportWithholdsDisputed(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeed(t, 43)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	run := correctionRun(t, fx, owner, p.Candidate)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	finding := correctionTestFinding(1, true, factory.FindingFixCode)
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report := correctionTestReport([]factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionDisputed}})
	served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, nil, nil)
	if len(served.Errors) != 0 || len(served.Corrected) != 0 || len(served.Waits) != 1 || served.Waits[0].Reason != "delivery_judgment_required" {
		t.Fatalf("disputed correction mis-served: %+v", served)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.CorrectionJournals) != 1 {
		t.Fatalf("disputed journal missing: %+v", held.CorrectionJournals)
	}
	if held.Candidate != p.Candidate {
		t.Fatalf("contested feedback advanced the head: %q", held.Candidate)
	}
	if got := newestCurrentHeadDispositions(held); len(got) != 1 || got[0] != factory.DispositionDisputed {
		t.Fatalf("routing cannot read its journal: %+v", got)
	}
}

// TestPublishCorrectionWithReportRefusesRewrite (real pg) proves the
// canonical binding on the serving entry: a replay that rewrites its
// recorded findings refuses instead of replacing the canonical set, and
// records no second journal.
func TestPublishCorrectionWithReportRefusesRewrite(t *testing.T) {
	ctx := context.Background()
	fx, p := checkSeed(t, 44)
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	newHead := strings.Repeat("d", 40)
	run := correctionRun(t, fx, owner, newHead)
	child, err := fx.db.AssignmentByRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	finding := correctionTestFinding(2, true, factory.FindingFixCode)
	withReportCycle(t, ctx, fx, p, child, correctionTestDigest(t, finding.Key))
	report, evidence, checks := withReportFixed(t, p, newHead)
	if served := fx.coord.PublishCorrectionWithReport(ctx, run.ID, report, evidence, checks); len(served.Corrected) != 1 {
		t.Fatalf("fixed correction mis-served: %+v", served)
	}
	rewritten := finding
	rewritten.Defect = "rewritten defect"
	bad := correctionTestReport([]factory.Finding{rewritten}, []factory.CorrectionDisposition{{Key: rewritten.Key, Disposition: factory.DispositionFixed}})
	replay := fx.coord.PublishCorrectionWithReport(ctx, run.ID, bad, evidence, checks)
	if len(replay.Waits) != 1 || replay.Waits[0].Reason != "correction_refused" {
		t.Fatalf("rewritten replay mis-served: %+v", replay)
	}
	held, err := fx.db.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.CorrectionJournals) != 1 {
		t.Fatalf("rewrite recorded a second journal: %+v", held.CorrectionJournals)
	}
	if held.CorrectionJournals[0].Report.Findings[0].Defect != finding.Defect {
		t.Fatalf("canonical set rewritten: %+v", held.CorrectionJournals[0].Report.Findings)
	}
}
