package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func correctionTestKey(n int) string { return fmt.Sprintf("%032x", 0xa0+n) }

func correctionTestFinding(n int, actionable bool, fixKind string) factory.Finding {
	return factory.Finding{
		Key: correctionTestKey(n), OriginRun: strings.Repeat("1", 32),
		Severity: factory.FindingSeverityMajor,
		Defect:   "missing bound", Evidence: "line 42 lacks a check", RequiredOutcome: "add the check",
		FixKind: fixKind, Actionable: actionable,
	}
}

func correctionTestDigest(t *testing.T, keys ...string) string {
	t.Helper()
	digest, err := factory.FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func correctionTestCycle(t *testing.T, publicationID string, head, base string, round int, digest string) factory.ReviewCycle {
	t.Helper()
	attemptRoot := strings.Repeat("b", 32)
	return factory.ReviewCycle{
		PublicationID: publicationID, AttemptRoot: attemptRoot,
		ID:                 factory.ReviewCycleID(publicationID, attemptRoot, round, "review"),
		Phase:              "review",
		TriggerDecision:    factory.DeliveryReviewAgain,
		TriggerReason:      "correction set needs another inspection",
		Head:               head,
		BaseOID:            base,
		FindingDigest:      digest,
		ReviewerAssignment: strings.Repeat("4", 32),
		Round:              round,
	}
}

func correctionTestReport(candidate string, findings []factory.Finding, dispositions []factory.CorrectionDisposition) factory.CorrectionReport {
	keys := make([]string, 0, len(findings))
	for _, finding := range findings {
		keys = append(keys, finding.Key)
	}
	digest, err := factory.FindingSetDigest(keys)
	if err != nil {
		return factory.CorrectionReport{}
	}
	return factory.CorrectionReport{SetDigest: digest, NewCandidate: candidate, Findings: findings, Dispositions: dispositions}
}

func correctionTestHeads() (head, base, candidate string) {
	return strings.Repeat("c", 40), strings.Repeat("b", 40), strings.Repeat("d", 40)
}

func correctionTestChecks(head, base string) *factory.CheckAssessment {
	checks := []string{"ci"}
	return &factory.CheckAssessment{
		Results:          []factory.CheckResult{{Context: "ci", State: factory.CheckStateSuccess, Passed: true}},
		Checks:           checks,
		Repository:       7,
		PRNumber:         3,
		PRID:             5,
		IssueID:          9,
		PolicyRevision:   1,
		NativeRev:        2,
		Revision:         1,
		AssessedUnix:     time.Now().Unix(),
		ObservedContexts: 1,
		HeadRef:          "refs/heads/fix",
		BaseRef:          "refs/heads/main",
		HeadOID:          head,
		BaseOID:          base,
		ChecksDigest:     factory.ChecksDigest(checks),
		Verdict:          factory.CheckPass,
		Reason:           factory.CheckReasonPass,
	}
}

func TestParseCorrectionReportRoundTrip(t *testing.T) {
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)}
	report := correctionTestReport(strings.Repeat("d", 40), findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionFixed}})
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	output := "fixed the finding\n```" + factory.CorrectionReportFence + "\n" + string(raw) + "\n```\n"
	parsed, ok := ParseCorrectionReport(output)
	if !ok {
		t.Fatal("valid correction block refused")
	}
	if parsed.SetDigest != report.SetDigest || len(parsed.Dispositions) != 1 || parsed.Dispositions[0].Disposition != factory.DispositionFixed {
		t.Fatalf("correction round trip changed the report: %+v", parsed)
	}
	if _, ok := ParseCorrectionReport("no fenced block here"); ok {
		t.Fatal("missing correction block accepted")
	}
	unknown := strings.Replace(string(raw), `"disposition":"fixed"`, `"disposition":"fixed","extra":1`, 1)
	if _, ok := ParseCorrectionReport("```" + factory.CorrectionReportFence + "\n" + unknown + "\n```"); ok {
		t.Fatal("correction block with unknown fields accepted")
	}
	// The last fenced block wins; trailing chatter without a new opener
	// never mis-terminates it.
	second := correctionTestReport(strings.Repeat("d", 40), findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionUnresolved}})
	secondRaw, err := json.Marshal(second)
	if err != nil {
		t.Fatal(err)
	}
	doubled := output + "\nleftover note\n```" + factory.CorrectionReportFence + "\n" + string(secondRaw) + "\n```"
	again, ok := ParseCorrectionReport(doubled)
	if !ok || again.SetDigest != second.SetDigest || again.Dispositions[0].Disposition != factory.DispositionUnresolved {
		t.Fatalf("last correction block did not win: %+v %v", again, ok)
	}
	chatter, ok := ParseCorrectionReport(output + "\nleftover note without fences")
	if !ok || chatter.SetDigest != report.SetDigest {
		t.Fatalf("trailing chatter broke correction parsing: %+v %v", chatter, ok)
	}
}

func TestAdmitCorrectionAcceptsFixedCodeCorrection(t *testing.T) {
	head, base, candidate := correctionTestHeads()
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)}
	cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, correctionTestDigest(t, findings[0].Key))
	report := correctionTestReport(candidate, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionFixed}})
	evidence := []factory.CorrectionFixEvidence{{Key: findings[0].Key, ChangedPaths: []string{"internal/factory/x.go"}, Commits: []string{candidate}}}
	admission, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, report, evidence, nil)
	if err != nil {
		t.Fatalf("mechanically complete fix refused: %v", err)
	}
	if len(admission.Fixed) != 1 || admission.Fixed[0] != findings[0].Key || admission.NewCandidate != candidate {
		t.Fatalf("admission lost the fixed finding: %+v", admission)
	}
}

func TestAdmitCorrectionRefusesUnchangedCodeFix(t *testing.T) {
	head, base, _ := correctionTestHeads()
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)}
	cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, correctionTestDigest(t, findings[0].Key))
	report := correctionTestReport(head, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionFixed}})
	evidence := []factory.CorrectionFixEvidence{{Key: findings[0].Key, ChangedPaths: []string{"internal/factory/x.go"}, Commits: []string{head}}}
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), head, report, evidence, correctionTestChecks(head, base)); err == nil {
		t.Fatal("unchanged code fix accepted")
	}
}

func TestAdmitCorrectionAllowsSameHeadVerificationFix(t *testing.T) {
	head, base, _ := correctionTestHeads()
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixVerification)}
	cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, correctionTestDigest(t, findings[0].Key))
	report := correctionTestReport(head, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionFixed}})
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), head, report, nil, correctionTestChecks(head, base)); err != nil {
		t.Fatalf("same-head verification fix with passing checks refused: %v", err)
	}
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), head, report, nil, nil); err == nil {
		t.Fatal("same-head fix without passing checks accepted")
	}
}

func TestAdmitCorrectionRefusesStaleDigest(t *testing.T) {
	head, base, candidate := correctionTestHeads()
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)}
	cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, correctionTestDigest(t, correctionTestKey(99)))
	report := correctionTestReport(candidate, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionFixed}})
	evidence := []factory.CorrectionFixEvidence{{Key: findings[0].Key, ChangedPaths: []string{"x.go"}, Commits: []string{candidate}}}
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, report, evidence, nil); err == nil {
		t.Fatal("stale finding-set digest accepted")
	}
}

// A correction whose nomination differs from its recorded candidate
// refuses: the consumer advances exactly the accounted tree, never a
// mismatched wiring.
func TestAdmitCorrectionRefusesNominationMismatch(t *testing.T) {
	head, base, candidate := correctionTestHeads()
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)}
	cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, correctionTestDigest(t, findings[0].Key))
	report := correctionTestReport(head, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionFixed}})
	evidence := []factory.CorrectionFixEvidence{{Key: findings[0].Key, ChangedPaths: []string{"x.go"}, Commits: []string{candidate}}}
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, report, evidence, nil); err == nil {
		t.Fatal("mismatched nomination admitted")
	}
}

func TestAdmitCorrectionRefusesAccountingGaps(t *testing.T) {
	head, base, candidate := correctionTestHeads()
	newAdmission := func(findings []factory.Finding, dispositions []factory.CorrectionDisposition) error {
		digest := correctionTestDigest(t, findings[0].Key, findings[1].Key)
		cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, digest)
		report := factory.CorrectionReport{Findings: findings, Dispositions: dispositions, SetDigest: digest, NewCandidate: candidate}
		evidence := []factory.CorrectionFixEvidence{
			{Key: findings[0].Key, ChangedPaths: []string{"x.go"}, Commits: []string{candidate}},
			{Key: findings[1].Key, ChangedPaths: []string{"y.go"}, Commits: []string{candidate}},
		}
		_, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, report, evidence, nil)
		return err
	}
	actionable := correctionTestFinding(1, true, factory.FindingFixCode)
	optional := correctionTestFinding(2, false, factory.FindingFixCode)
	fixed := func(key string) factory.CorrectionDisposition {
		return factory.CorrectionDisposition{Key: key, Disposition: factory.DispositionFixed}
	}
	if err := newAdmission([]factory.Finding{actionable, optional}, []factory.CorrectionDisposition{fixed(actionable.Key)}); err == nil {
		t.Fatal("missing disposition accepted")
	}
	if err := newAdmission([]factory.Finding{actionable, optional}, []factory.CorrectionDisposition{fixed(actionable.Key), fixed(actionable.Key)}); err == nil {
		t.Fatal("duplicate disposition accepted")
	}
	unknown := factory.CorrectionDisposition{Key: correctionTestKey(77), Disposition: factory.DispositionFixed}
	if err := newAdmission([]factory.Finding{actionable, optional}, []factory.CorrectionDisposition{fixed(actionable.Key), unknown}); err == nil {
		t.Fatal("unknown disposition accepted")
	}
}

func TestAdmitCorrectionBlocksDisputedActionable(t *testing.T) {
	head, base, candidate := correctionTestHeads()
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)}
	cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, correctionTestDigest(t, findings[0].Key))
	report := correctionTestReport(candidate, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionDisputed}})
	_, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, report, nil, nil)
	if err == nil {
		t.Fatal("disputed actionable finding passed")
	}
	var block *CorrectionBlock
	if !errors.As(err, &block) {
		t.Fatalf("dispute did not block with intervention: %v", err)
	}
}

func TestAdmitCorrectionRefusesActionableDeclinedOptional(t *testing.T) {
	head, base, candidate := correctionTestHeads()
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)}
	cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, correctionTestDigest(t, findings[0].Key))
	report := correctionTestReport(candidate, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionDeclinedOptional}})
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, report, nil, nil); err == nil {
		t.Fatal("actionable finding declined as optional accepted")
	}
}

func TestAdmitCorrectionAcceptsOptionalOnlyWithoutForcingFailure(t *testing.T) {
	head, base, candidate := correctionTestHeads()
	optional := correctionTestFinding(1, false, factory.FindingFixCode)
	cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, correctionTestDigest(t, optional.Key))
	report := correctionTestReport(candidate, []factory.Finding{optional}, []factory.CorrectionDisposition{{Key: optional.Key, Disposition: factory.DispositionDeclinedOptional}})
	admission, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, report, nil, nil)
	if err != nil {
		t.Fatalf("optional-only report refused: %v", err)
	}
	if len(admission.DeclinedOptional) != 1 {
		t.Fatalf("optional disposition lost: %+v", admission)
	}
	verdict := factory.ReviewCycleReport{CycleID: cycle.ID}
	if verdict.RequiresCorrectionRun() {
		t.Fatal("optional-only verdict forces a correction run")
	}
}

func TestAdmitCorrectionRefusesEvidenceMismatch(t *testing.T) {
	head, base, candidate := correctionTestHeads()
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)}
	cycle := correctionTestCycle(t, strings.Repeat("9", 32), head, base, 1, correctionTestDigest(t, findings[0].Key))
	fixed := correctionTestReport(candidate, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionFixed}})
	good := []factory.CorrectionFixEvidence{{Key: findings[0].Key, ChangedPaths: []string{"x.go"}, Commits: []string{candidate}}}
	unknown := []factory.CorrectionFixEvidence{{Key: correctionTestKey(77), ChangedPaths: []string{"x.go"}, Commits: []string{candidate}}}
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, fixed, unknown, nil); err == nil {
		t.Fatal("fix evidence for an unknown finding accepted")
	}
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, fixed, nil, nil); err == nil {
		t.Fatal("code fix without changed paths and commits accepted")
	}
	unresolved := correctionTestReport(candidate, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionUnresolved}})
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, unresolved, good, nil); err == nil {
		t.Fatal("fix evidence on an unresolved finding accepted")
	}
	if _, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, unresolved, nil, nil); err != nil {
		t.Fatalf("honest unresolved report refused: %v", err)
	}
}

func TestAdmitAndJournalCorrectionPersistsJournal(t *testing.T) {
	fx, p := checkSeed(t, 30)
	ctx := context.Background()
	owner, err := fx.db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	finding := correctionTestFinding(1, true, factory.FindingFixCode)
	digest := correctionTestDigest(t, finding.Key)
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID:                 factory.ReviewCycleID(p.ID, owner.AttemptRoot, 1, "review"),
		Phase:              "review",
		TriggerDecision:    factory.DeliveryReviewAgain,
		TriggerReason:      "correction set needs another inspection",
		Head:               p.Candidate,
		BaseOID:            p.PRCreate.BaseOID,
		FindingDigest:      digest,
		ReviewerAssignment: factory.NewID(),
		Round:              1,
	}
	p.ReviewCycles = append(p.ReviewCycles, cycle)
	p.Revision++
	if err := fx.db.UpdatePublication(ctx, p); err != nil {
		t.Fatalf("record review cycle: %v", err)
	}
	candidate := strings.Repeat("e", 40)
	if candidate == p.Candidate {
		candidate = strings.Repeat("f", 40)
	}
	report := correctionTestReport(candidate, []factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionFixed}})
	evidence := []factory.CorrectionFixEvidence{{Key: finding.Key, ChangedPaths: []string{"internal/factory/x.go"}, Commits: []string{candidate}}}
	runID := factory.NewID()
	admission, err := fx.coord.AdmitAndJournalCorrection(ctx, p.AssignmentID, cycle.ID, runID, candidate, report, evidence, nil)
	if err != nil {
		t.Fatalf("admitted journal unrecorded: %v", err)
	}
	if len(admission.Fixed) != 1 || admission.NewCandidate != candidate {
		t.Fatalf("admission lost the fix: %+v", admission)
	}
	stored, err := fx.db.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.CorrectionJournals) != 1 || stored.CorrectionJournals[0].RunID != runID ||
		stored.CorrectionJournals[0].Report.SetDigest != digest {
		t.Fatalf("journal not persisted: %+v", stored.CorrectionJournals)
	}
	if err := stored.Validate(); err != nil {
		t.Fatalf("journaled publication invalid: %v", err)
	}
	if _, err := fx.coord.AdmitAndJournalCorrection(ctx, p.AssignmentID, cycle.ID, runID, candidate, report, evidence, nil); err == nil {
		t.Fatal("correction run reported twice accepted")
	}
	disputed := correctionTestReport(candidate, []factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionDisputed}})
	if _, err := fx.coord.AdmitAndJournalCorrection(ctx, p.AssignmentID, cycle.ID, factory.NewID(), candidate, disputed, nil, nil); err == nil {
		t.Fatal("disputed report journaled")
	}
	restored, err := fx.db.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.CorrectionJournals) != 1 {
		t.Fatalf("refused reports polluted the journal: %+v", restored.CorrectionJournals)
	}
}

func TestAppendCorrectionJournalBindsCycleAndRun(t *testing.T) {
	head, base, _ := correctionTestHeads()
	publicationID := strings.Repeat("9", 32)
	findings := []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)}
	cycle := correctionTestCycle(t, publicationID, head, base, 1, correctionTestDigest(t, findings[0].Key))
	report := correctionTestReport(head, findings, []factory.CorrectionDisposition{{Key: findings[0].Key, Disposition: factory.DispositionFixed}})
	p := &factory.Publication{ID: publicationID, ReviewCycles: []factory.ReviewCycle{cycle}}
	runID := strings.Repeat("c", 32)
	if err := appendCorrectionJournal(p, cycle, runID, report); err != nil {
		t.Fatalf("admitted journal refused: %v", err)
	}
	if len(p.CorrectionJournals) != 1 {
		t.Fatalf("journal not appended: %+v", p.CorrectionJournals)
	}
	if err := appendCorrectionJournal(p, cycle, runID, report); err == nil {
		t.Fatal("correction run reported twice accepted")
	}
	other := correctionTestCycle(t, strings.Repeat("8", 32), head, base, 1, correctionTestDigest(t, findings[0].Key))
	if err := appendCorrectionJournal(p, other, strings.Repeat("d", 32), report); err == nil {
		t.Fatal("another publication's cycle accepted")
	}
}
