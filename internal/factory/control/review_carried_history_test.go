package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// F8: a candidate change invalidates acceptances but NEVER erases
// unresolved canonical finding identity/content/evidence/disposition.
// The carried set rebuilds in monotonic cycle order from recorded
// rounds only; journals for unrecorded cycles, rewritten content and
// missing history all hold instead of succeeding. Keys retire only on
// bound attributable verification.

func carriedHead(prefix byte) string { return strings.Repeat(string([]byte{prefix}), 40) }

func carriedFinding(key, evidence string) factory.Finding {
	return factory.Finding{
		Key: key, OriginRun: strings.Repeat("1", 32),
		Severity: factory.FindingSeverityMajor, Defect: "missing bound",
		Evidence: evidence, RequiredOutcome: "add the check",
		FixKind: factory.FindingFixCode, Actionable: true,
	}
}

func carriedJournal(t *testing.T, cycleID string, findings []factory.Finding, dispositions map[string]string) factory.CorrectionJournal {
	t.Helper()
	keys := make([]string, 0, len(findings))
	for _, finding := range findings {
		keys = append(keys, finding.Key)
	}
	digest, err := factory.FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	ordered := make([]factory.CorrectionDisposition, 0, len(findings))
	for _, finding := range findings {
		disposition, ok := dispositions[finding.Key]
		if !ok {
			disposition = factory.DispositionUnresolved
		}
		ordered = append(ordered, factory.CorrectionDisposition{Key: finding.Key, Disposition: disposition})
	}
	return factory.CorrectionJournal{
		CycleID: cycleID, RunID: strings.Repeat("2", 32),
		Report: factory.CorrectionReport{SetDigest: digest, Findings: findings, Dispositions: ordered},
	}
}

func carriedCycle(pubID, root, head, base, reviewer string, round int) factory.ReviewCycle {
	return factory.ReviewCycle{
		ID:            factory.ReviewCycleID(pubID, root, round, "review"),
		PublicationID: pubID, AttemptRoot: root,
		Round: round, Phase: "review",
		TriggerDecision: "re-review", TriggerReason: "candidate changed",
		Head: head, BaseOID: base,
		FindingDigest:      reviewCycleInputDigest(head, base, nil, nil),
		ReviewerAssignment: reviewer,
	}
}

func carriedPublication(pubID, head, base string) factory.Publication {
	return factory.Publication{
		ID: pubID, AssignmentID: strings.Repeat("9", 32),
		Candidate: head, PRCreate: factory.PublicationOperation{BaseOID: base},
	}
}

func carriedKeys(findings []factory.Finding) map[string]bool {
	keys := map[string]bool{}
	for _, finding := range findings {
		keys[finding.Key] = true
	}
	return keys
}

// The current cycle's correction journal passes the same canonical
// pipeline as every other set: its findings enter the carried set
// because nothing later retires them, not through a shortcut.
func TestCarriedHistorySameCycleJournalAnswersCurrent(t *testing.T) {
	head, base := carriedHead('a'), carriedHead('f')
	key := strings.Repeat("3", 32)
	pubID, root := strings.Repeat("6", 32), strings.Repeat("7", 32)
	finding := carriedFinding(key, "line 7 lacks a check")
	p := carriedPublication(pubID, head, base)
	cycle := carriedCycle(pubID, root, head, base, strings.Repeat("8", 32), 1)
	p.CorrectionJournals = []factory.CorrectionJournal{
		carriedJournal(t, cycle.ID, []factory.Finding{finding}, map[string]string{key: factory.DispositionFixed}),
	}
	p.ReviewCycles = []factory.ReviewCycle{cycle}
	carried, err := (&Coordinator{}).reviewCarriedFindings(context.Background(), p, cycle)
	if err != nil {
		t.Fatalf("same-cycle journal refused: %v", err)
	}
	if len(carried) != 1 || carried[0].Key != key {
		t.Fatalf("same-cycle journal findings not carried: %+v", carried)
	}
}

// Same-key changed content holds conflict instead of silently replacing
// original provenance: neither the old nor the rewritten content wins.
func TestCarriedHistoryChangedContentHoldsConflict(t *testing.T) {
	head, base := carriedHead('a'), carriedHead('f')
	key := strings.Repeat("3", 32)
	pubID, root := strings.Repeat("6", 32), strings.Repeat("7", 32)
	p := carriedPublication(pubID, head, base)
	cycle := carriedCycle(pubID, root, head, base, strings.Repeat("8", 32), 1)
	p.CorrectionJournals = []factory.CorrectionJournal{
		carriedJournal(t, cycle.ID, []factory.Finding{carriedFinding(key, "original evidence")}, nil),
		carriedJournal(t, cycle.ID, []factory.Finding{carriedFinding(key, "rewritten evidence")}, nil),
	}
	p.ReviewCycles = []factory.ReviewCycle{cycle}
	carried, err := (&Coordinator{}).reviewCarriedFindings(context.Background(), p, cycle)
	if err == nil {
		t.Fatalf("changed content succeeded with %d findings", len(carried))
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("changed content refused for the wrong reason: %v", err)
	}
}

// A fixed disposition without bound verification evidence is an agent
// claim, not disappearance: the key stays carried.
func TestCarriedHistoryFixedWithoutVerificationStays(t *testing.T) {
	head, base := carriedHead('a'), carriedHead('f')
	key := strings.Repeat("3", 32)
	pubID, root := strings.Repeat("6", 32), strings.Repeat("7", 32)
	p := carriedPublication(pubID, head, base)
	cycle := carriedCycle(pubID, root, head, base, strings.Repeat("8", 32), 1)
	p.CorrectionJournals = []factory.CorrectionJournal{
		carriedJournal(t, cycle.ID,
			[]factory.Finding{carriedFinding(key, "line 7 lacks a check")},
			map[string]string{key: factory.DispositionFixed}),
	}
	p.ReviewCycles = []factory.ReviewCycle{cycle}
	carried, err := (&Coordinator{}).reviewCarriedFindings(context.Background(), p, cycle)
	if err != nil {
		t.Fatalf("fixed-without-verification refused: %v", err)
	}
	if !carriedKeys(carried)[key] {
		t.Fatal("unverified fixed claim erased its finding")
	}
}

// A journal for a cycle the publication never recorded holds
// incomplete: unrecorded history cannot seed the canonical set.
func TestCarriedHistoryUnrecordedJournalHoldsIncomplete(t *testing.T) {
	head, base := carriedHead('a'), carriedHead('f')
	pubID, root := strings.Repeat("6", 32), strings.Repeat("7", 32)
	p := carriedPublication(pubID, head, base)
	cycle := carriedCycle(pubID, root, head, base, strings.Repeat("8", 32), 1)
	p.CorrectionJournals = []factory.CorrectionJournal{
		carriedJournal(t, strings.Repeat("0", 32),
			[]factory.Finding{carriedFinding(strings.Repeat("3", 32), "line 7 lacks a check")}, nil),
	}
	p.ReviewCycles = []factory.ReviewCycle{cycle}
	carried, err := (&Coordinator{}).reviewCarriedFindings(context.Background(), p, cycle)
	if err == nil {
		t.Fatalf("unrecorded journal succeeded with %d findings", len(carried))
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("unrecorded journal refused for the wrong reason: %v", err)
	}
}

// A complete current assessment succeeds: every carried actionable
// finding has its verified assessment with evidence.
func TestCarriedHistoryCompleteAssessmentSucceeds(t *testing.T) {
	head, base := carriedHead('b'), carriedHead('f')
	key := strings.Repeat("3", 32)
	pubID, root := strings.Repeat("6", 32), strings.Repeat("7", 32)
	finding := carriedFinding(key, "line 7 lacks a check")
	p := carriedPublication(pubID, head, base)
	cycle := carriedCycle(pubID, root, head, base, strings.Repeat("8", 32), 1)
	p.CorrectionJournals = []factory.CorrectionJournal{
		carriedJournal(t, cycle.ID, []factory.Finding{finding}, nil),
	}
	p.ReviewCycles = []factory.ReviewCycle{cycle}
	carried, err := (&Coordinator{}).reviewCarriedFindings(context.Background(), p, cycle)
	if err != nil {
		t.Fatalf("carried history refused: %v", err)
	}
	clear := &factory.ClearReviewReport{
		CycleID: cycle.ID, Conclusion: factory.ClearReviewConclusion,
		Head: head, BaseOID: base, CoverageDigest: strings.Repeat("c", 64),
		Assessments: []factory.FindingAssessment{
			{Key: key, Status: factory.FindingVerified, Evidence: "check added at line 7, suite green"},
		},
	}
	if err := clear.Validate(carried); err != nil {
		t.Fatalf("complete assessment refused: %v", err)
	}
}

// Persisted-path proofs below need real PostgreSQL (SODA_PG_*; they skip
// without it): cross-head reviewer-assignment reads, bound verification
// retirement and incomplete-history refusal.

func carriedStoreAssignment(t *testing.T, s *store.Store, dsn string, a factory.Assignment) {
	t.Helper()
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	// Routing columns mirror the JSON: finished rows carry the
	// assignment's own run, matching production FinishAssignment.
	if _, err := db.Exec(`INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, 7, 9, a.Run, "finished", 1, string(raw)); err != nil {
		t.Fatal(err)
	}
}

func carriedReviewer(t *testing.T, id, root, head, cycleID string, report *factory.ReviewCycleReport) factory.Assignment {
	t.Helper()
	runID := strings.Repeat("d", 32)
	return factory.Assignment{
		ID: id, Role: project.RoleReviewer, SourceCommit: head,
		Phase: project.ExecutionReview, AttemptRoot: root,
		Run: runID,
		Result: &factory.AssignmentResult{Status: "completed", Reported: true, Summary: "review recorded",
			AssignmentID: id, RunID: runID, RecordedUnix: 1700000000,
			PhaseResult: &factory.PhaseResult{Version: 2, Phase: project.ExecutionReview,
				Status: "completed", Output: &factory.PhaseOutput{Review: report}}},
	}
}

func carriedEphemeral(t *testing.T) (*store.Store, string) {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s, dsn, cleanup, err := store.OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return s, dsn
}

// The F8 repro at bind level on the persisted path: a valid unresolved
// correction journal on the old head must still be answered after the
// head changes. An empty clear must refuse — never APPROVED with no
// assessment — with zero native review effect.
func TestCarriedHistoryKeepsUnresolvedAcrossHeadChange(t *testing.T) {
	s, dsn := carriedEphemeral(t)
	ctx := context.Background()
	oldHead, newHead, base := carriedHead('a'), carriedHead('b'), carriedHead('f')
	key := strings.Repeat("3", 32)
	pubID, root := strings.Repeat("6", 32), strings.Repeat("7", 32)
	oldReviewer, newReviewer := strings.Repeat("7", 32), strings.Repeat("8", 32)
	finding := carriedFinding(key, "line 7 lacks a check")
	prior := carriedCycle(pubID, root, oldHead, base, oldReviewer, 1)
	current := carriedCycle(pubID, root, newHead, base, newReviewer, 2)
	carriedStoreAssignment(t, s, dsn, carriedReviewer(t, oldReviewer, root, oldHead, prior.ID,
		&factory.ReviewCycleReport{CycleID: prior.ID, NewFindings: []factory.Finding{finding},
			Assessments: []factory.FindingAssessment{
				{Key: key, Status: factory.FindingStillOpen, Evidence: "fix not yet attempted"},
			}}))
	p := carriedPublication(pubID, newHead, base)
	p.CorrectionJournals = []factory.CorrectionJournal{
		carriedJournal(t, prior.ID, []factory.Finding{finding}, nil),
	}
	p.ReviewCycles = []factory.ReviewCycle{prior, current}
	coord := &Coordinator{Store: s}
	carried, err := coord.reviewCarriedFindings(ctx, p, current)
	if err != nil {
		t.Fatalf("carried history refused: %v", err)
	}
	if !carriedKeys(carried)[key] {
		t.Fatalf("old-head unresolved finding erased after head change: %v", carriedKeys(carried))
	}
	clear := &factory.ClearReviewReport{
		CycleID: current.ID, Conclusion: factory.ClearReviewConclusion,
		Head: newHead, BaseOID: base, CoverageDigest: strings.Repeat("c", 64),
	}
	if err := clear.Validate(carried); err == nil {
		t.Fatal("empty clear approved an unassessed carried finding")
	}
	assignment := factory.Assignment{
		ID: current.ReviewerAssignment, Role: project.RoleReviewer, SourceCommit: newHead,
		Phase: project.ExecutionReview,
		Result: &factory.AssignmentResult{Status: "completed", Reported: true, Summary: "clear recorded",
			PhaseResult: &factory.PhaseResult{Version: 2, Phase: project.ExecutionReview,
				Status: "completed", Output: &factory.PhaseOutput{Clear: clear}}},
	}
	resolved, err := resolveAssignmentVerdict(assignment)
	if err != nil {
		t.Fatalf("clear verdict not resolved: %v", err)
	}
	if event, _, err := coord.bindPhasedReviewVerdict(ctx, p, assignment, newHead, resolved); err == nil || event == "APPROVED" {
		t.Fatalf("bind approved without assessment: event=%q err=%v", event, err)
	}
}

// A cross-head prior round's phased findings enter the carried set;
// bound recorded verification retires the key; an earlier verification
// cannot erase a later rediscovery of the identical finding.
func TestCarriedHistoryCrossHeadReadsAndRetires(t *testing.T) {
	s, dsn := carriedEphemeral(t)
	ctx := context.Background()
	oldHead, newHead, base := carriedHead('a'), carriedHead('b'), carriedHead('f')
	key := strings.Repeat("3", 32)
	pubID, root := strings.Repeat("6", 32), strings.Repeat("9", 32)
	oldReviewer, newReviewer := strings.Repeat("7", 32), strings.Repeat("8", 32)
	prior := carriedCycle(pubID, root, oldHead, base, oldReviewer, 1)
	current := carriedCycle(pubID, root, newHead, base, newReviewer, 2)
	report := func(assessments []factory.FindingAssessment) *factory.ReviewCycleReport {
		return &factory.ReviewCycleReport{CycleID: prior.ID,
			NewFindings: []factory.Finding{carriedFinding(key, "line 7 lacks a check")},
			Assessments: assessments}
	}
	coord := &Coordinator{Store: s}
	p := carriedPublication(pubID, newHead, base)
	p.ReviewCycles = []factory.ReviewCycle{prior, current}
	carriedStoreAssignment(t, s, dsn, carriedReviewer(t, oldReviewer, root, oldHead, prior.ID,
		report([]factory.FindingAssessment{{Key: key, Status: factory.FindingStillOpen, Evidence: "not yet fixed"}})))
	carried, err := coord.reviewCarriedFindings(ctx, p, current)
	if err != nil {
		t.Fatalf("cross-head read refused: %v", err)
	}
	if !carriedKeys(carried)[key] {
		t.Fatal("still-open cross-head finding erased")
	}
	oldReviewer2 := strings.Repeat("a", 32)
	carriedStoreAssignment(t, s, dsn, carriedReviewer(t, oldReviewer2, root, oldHead, prior.ID,
		report([]factory.FindingAssessment{{Key: key, Status: factory.FindingVerified, Evidence: "check added, suite green"}})))
	p2 := carriedPublication(pubID, newHead, base)
	prior2 := carriedCycle(pubID, root, oldHead, base, oldReviewer2, 1)
	p2.ReviewCycles = []factory.ReviewCycle{prior2, current}
	carried2, err := coord.reviewCarriedFindings(ctx, p2, current)
	if err != nil {
		t.Fatalf("verified read refused: %v", err)
	}
	if carriedKeys(carried2)[key] {
		t.Fatal("verified finding stayed carried")
	}
	rediscoverReviewer := strings.Repeat("b", 32)
	mid := carriedCycle(pubID, root, newHead, base, rediscoverReviewer, 2)
	carriedStoreAssignment(t, s, dsn, carriedReviewer(t, rediscoverReviewer, root, newHead, mid.ID,
		&factory.ReviewCycleReport{CycleID: mid.ID,
			NewFindings: []factory.Finding{carriedFinding(key, "line 7 lacks a check")},
			Assessments: []factory.FindingAssessment{
				{Key: key, Status: factory.FindingStillOpen, Evidence: "regressed on the new head"},
			}}))
	p3 := carriedPublication(pubID, newHead, base)
	current3 := carriedCycle(pubID, root, newHead, base, newReviewer, 3)
	p3.ReviewCycles = []factory.ReviewCycle{prior2, mid, current3}
	carried3, err := coord.reviewCarriedFindings(ctx, p3, current3)
	if err != nil {
		t.Fatalf("rediscovery read refused: %v", err)
	}
	if !carriedKeys(carried3)[key] {
		t.Fatal("older verification erased a newer rediscovery")
	}
}

// A prior report that verifies one carried key while omitting another
// required key refuses at collection time: round 3 can never clear on
// the partial retirement, and no downstream approval follows.
func TestCarriedHistoryOmittedCoverageHoldsIncomplete(t *testing.T) {
	s, dsn := carriedEphemeral(t)
	ctx := context.Background()
	oldHead, newHead, base := carriedHead('a'), carriedHead('b'), carriedHead('f')
	keyA, keyB, keyC := strings.Repeat("3", 32), strings.Repeat("4", 32), strings.Repeat("5", 32)
	pubID, root := strings.Repeat("6", 32), strings.Repeat("9", 32)
	oldReviewer, newReviewer := strings.Repeat("7", 32), strings.Repeat("8", 32)
	prior := carriedCycle(pubID, root, oldHead, base, oldReviewer, 1)
	round2Reviewer := strings.Repeat("a", 32)
	round2 := carriedCycle(pubID, root, oldHead, base, round2Reviewer, 2)
	current := carriedCycle(pubID, root, newHead, base, newReviewer, 3)
	both := []factory.Finding{carriedFinding(keyA, "first defect"), carriedFinding(keyB, "second defect")}
	carriedStoreAssignment(t, s, dsn, carriedReviewer(t, oldReviewer, root, oldHead, prior.ID,
		&factory.ReviewCycleReport{CycleID: prior.ID, NewFindings: both,
			Assessments: []factory.FindingAssessment{
				{Key: keyA, Status: factory.FindingStillOpen, Evidence: "still open"},
				{Key: keyB, Status: factory.FindingStillOpen, Evidence: "still open"},
			}}))
	// Round 2 verifies A but omits required B while introducing C: the
	// canonical report validator must refuse before any merge or
	// retirement, so no round-3 clear can build on a partial set.
	carriedStoreAssignment(t, s, dsn, carriedReviewer(t, round2Reviewer, root, oldHead, round2.ID,
		&factory.ReviewCycleReport{CycleID: round2.ID,
			NewFindings: []factory.Finding{carriedFinding(keyC, "third defect")},
			Assessments: []factory.FindingAssessment{
				{Key: keyA, Status: factory.FindingVerified, Evidence: "fixed and checked"},
				{Key: keyC, Status: factory.FindingStillOpen, Evidence: "just found"},
			}}))
	p := carriedPublication(pubID, newHead, base)
	p.ReviewCycles = []factory.ReviewCycle{prior, round2, current}
	coord := &Coordinator{Store: s}
	carried, err := coord.reviewCarriedFindings(ctx, p, current)
	if err == nil {
		t.Fatalf("omitted coverage succeeded with %d findings", len(carried))
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("omitted coverage refused for the wrong reason: %v", err)
	}
}

// A cross-head prior round whose reviewer assignment is missing holds
// the cycle incomplete instead of succeeding with an empty set. A
// recorded verified assessment for a key the canonical set never
// recorded is likewise malformed evidence, not retirement.
func TestCarriedHistoryMissingCrossHeadHoldsIncomplete(t *testing.T) {
	s, dsn := carriedEphemeral(t)
	ctx := context.Background()
	oldHead, newHead, base := carriedHead('a'), carriedHead('b'), carriedHead('f')
	pubID, root := strings.Repeat("6", 32), strings.Repeat("9", 32)
	p := carriedPublication(pubID, newHead, base)
	prior := carriedCycle(pubID, root, oldHead, base, strings.Repeat("0", 32), 1)
	current := carriedCycle(pubID, root, newHead, base, strings.Repeat("8", 32), 2)
	p.ReviewCycles = []factory.ReviewCycle{prior, current}
	coord := &Coordinator{Store: s}
	carried, err := coord.reviewCarriedFindings(ctx, p, current)
	if err == nil {
		t.Fatalf("missing cross-head history succeeded with %d findings", len(carried))
	}
	if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("missing history refused for the wrong reason: %v", err)
	}
	stranger := strings.Repeat("e", 32)
	carriedStoreAssignment(t, s, dsn, carriedReviewer(t, stranger, root, oldHead, prior.ID,
		&factory.ReviewCycleReport{CycleID: prior.ID,
			Assessments: []factory.FindingAssessment{
				{Key: strings.Repeat("f", 32), Status: factory.FindingVerified, Evidence: "checked nowhere"},
			}}))
	p2 := carriedPublication(pubID, newHead, base)
	prior2 := carriedCycle(pubID, root, oldHead, base, stranger, 1)
	p2.ReviewCycles = []factory.ReviewCycle{prior2, current}
	if carried, err := coord.reviewCarriedFindings(ctx, p2, current); err == nil {
		t.Fatalf("unrecorded-key verification succeeded with %d findings", len(carried))
	} else if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("unrecorded-key verification refused for the wrong reason: %v", err)
	}
}
