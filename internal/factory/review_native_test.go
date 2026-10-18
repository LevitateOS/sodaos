package factory

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func journalTestFinding(n int, actionable bool, located bool) Finding {
	finding := testFinding(n, actionable)
	if located {
		finding.Location = &FindingLocation{Path: "internal/factory/review_native.go", Side: "new", Line: int64(40 + n)}
	}
	return finding
}

func journalTestCycle(p Publication, digest string, round int) ReviewCycle {
	attemptRoot := strings.Repeat("b", 32)
	return ReviewCycle{
		PublicationID: p.ID, AttemptRoot: attemptRoot,
		ID:                 ReviewCycleID(p.ID, attemptRoot, round, "review"),
		Phase:              "review",
		TriggerDecision:    DeliveryReviewAgain,
		TriggerReason:      "correction set needs another inspection",
		Head:               p.Candidate,
		BaseOID:            p.PRCreate.BaseOID,
		FindingDigest:      digest,
		ReviewerAssignment: strings.Repeat("4", 32),
		Round:              round,
	}
}

func journalTestJournal(cycle ReviewCycle, runID string, findings []Finding, dispositions []CorrectionDisposition) CorrectionJournal {
	return CorrectionJournal{CycleID: cycle.ID, RunID: runID, Report: CorrectionReport{
		SetDigest:    cycle.FindingDigest,
		NewCandidate: strings.Repeat("d", 40),
		Findings:     findings,
		Dispositions: dispositions,
	}}
}

func journalTestPublication(t *testing.T) (Publication, ReviewCycle, CorrectionJournal) {
	t.Helper()
	p := publishedPublication()
	findings := []Finding{journalTestFinding(0, true, true), journalTestFinding(1, false, true)}
	keys := []string{findings[0].Key, findings[1].Key}
	digest, err := FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	cycle := journalTestCycle(p, digest, 1)
	journal := journalTestJournal(cycle, strings.Repeat("c", 32), findings, []CorrectionDisposition{
		{Key: findings[0].Key, Disposition: DispositionFixed},
		{Key: findings[1].Key, Disposition: DispositionDeclinedOptional},
	})
	return p, cycle, journal
}

func TestFindingInlineCommentMapping(t *testing.T) {
	finding := journalTestFinding(0, true, true)
	comment, err := finding.InlineComment()
	if err != nil {
		t.Fatalf("located finding refused: %v", err)
	}
	want := finding.Key + "\nseverity: major\nfix: code\n\nmissing bound\n\nEvidence:\nline 42 lacks a check\n\nRequired outcome:\nadd the check"
	if comment.Body != want {
		t.Fatalf("unstable comment rendering:\n%q", comment.Body)
	}
	if comment.Key != finding.Key || comment.Path != "internal/factory/review_native.go" || comment.Side != "new" || comment.Line != 40 {
		t.Fatalf("comment identity differs: %+v", comment)
	}
	if err := comment.Validate(); err != nil {
		t.Fatalf("rendered comment refused: %v", err)
	}
	unlocated := journalTestFinding(2, false, false)
	if _, err := unlocated.InlineComment(); err == nil {
		t.Fatal("non-line finding became an inline comment")
	}
	bad := journalTestFinding(3, true, true)
	bad.Location.Side = "both"
	if _, err := bad.InlineComment(); err == nil {
		t.Fatal("bad anchor side became an inline comment")
	}
	verbose := journalTestFinding(4, true, true)
	verbose.Defect = strings.Repeat("d", MaxFindingProseBytes-2)
	verbose.Evidence, verbose.RequiredOutcome = "e", "r"
	large, err := verbose.InlineComment()
	if err != nil {
		t.Fatalf("maximum prose finding refused: %v", err)
	}
	if len(large.Body) > MaxFindingCommentBodyBytes {
		t.Fatal("rendered comment exceeds its bound")
	}
}

func TestFindingCommentsForFindings(t *testing.T) {
	comments, err := FindingCommentsForFindings([]Finding{journalTestFinding(0, true, true), journalTestFinding(1, false, true)})
	if err != nil || len(comments) != 2 || comments[0].Key != testFindingKey(0) || comments[1].Key != testFindingKey(1) {
		t.Fatalf("finding set not mapped: %+v %v", comments, err)
	}
	shared := []Finding{journalTestFinding(0, true, true), journalTestFinding(1, true, true)}
	shared[1].Location.Line = shared[0].Location.Line
	if _, err := FindingCommentsForFindings(shared); err == nil {
		t.Fatal("duplicate finding anchor accepted")
	}
	rekeyed := []Finding{journalTestFinding(0, true, true), journalTestFinding(0, true, true)}
	rekeyed[1].Location.Line++
	if _, err := FindingCommentsForFindings(rekeyed); err == nil {
		t.Fatal("duplicate finding key accepted")
	}
	if _, err := FindingCommentsForFindings([]Finding{journalTestFinding(2, false, false)}); err == nil {
		t.Fatal("unlocated finding mapped to an inline comment")
	}
	many := make([]Finding, 0, MaxFindingsPerReport+1)
	for i := 0; i <= MaxFindingsPerReport; i++ {
		finding := journalTestFinding(0, true, true)
		finding.Key = fmt.Sprintf("%032x", 0xe00000+i)
		finding.Location.Line = int64(1000 + i)
		many = append(many, finding)
	}
	if _, err := FindingCommentsForFindings(many); err == nil {
		t.Fatal("finding set above its report bound accepted")
	}
}

func TestReviewWorkCommentsValidation(t *testing.T) {
	work := ReviewWork{
		OperationID: "review-" + strings.Repeat("1", 32), AuthRevision: "soda-assignment:a:review-run:r",
		Repository: 7, ActorID: 6, PRNumber: 9, PRID: 8, IssueID: 10, PRAuthorID: 5,
		HeadRef: "refs/heads/candidate", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		NativeRev: 1, NotAfter: 1300, Event: "REQUEST_CHANGES", Body: "two defects remain",
	}
	comments, err := FindingCommentsForFindings([]Finding{journalTestFinding(0, true, true), journalTestFinding(1, false, true)})
	if err != nil {
		t.Fatal(err)
	}
	work.Comments = comments
	if err := work.Validate(); err != nil {
		t.Fatalf("commented work refused: %v", err)
	}
	duplicated := work
	duplicated.Comments = append(append([]FindingComment(nil), comments...), comments[0])
	if err := duplicated.Validate(); err == nil {
		t.Fatal("duplicate finding key admitted in persisted work")
	}
	unbounded := work
	unbounded.Comments = append(append([]FindingComment(nil), comments...), FindingComment{Key: testFindingKey(2), Path: "x.go", Side: "both", Line: 1, Body: "b"})
	if err := unbounded.Validate(); err == nil {
		t.Fatal("bad comment anchor admitted in persisted work")
	}
}

func TestParseReviewReportMaxBounds(t *testing.T) {
	maximal := ReviewReport{Verdict: "request-changes", Summary: strings.Repeat("s", 4096), Body: strings.Repeat("b", 65536), Findings: make([]string, 64)}
	for i := range maximal.Findings {
		maximal.Findings[i] = strings.Repeat("f", 4096)
	}
	encoded, err := json.Marshal(maximal)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > MaxReviewReportBytes {
		t.Fatalf("maximum valid report exceeds the parser bound: %d", len(encoded))
	}
	if err := maximal.Validate(); err != nil {
		t.Fatalf("maximum report refused: %v", err)
	}
	if report, ok := ParseReviewReport("notes\n```review-json\n" + string(encoded) + "\n```\ntail"); !ok || len(report.Findings) != 64 {
		t.Fatal("maximum report did not parse")
	}
	tooMany := maximal
	tooMany.Findings = append(tooMany.Findings, "one more")
	if _, ok := ParseReviewReport("```review-json\n" + string(mustMarshal(t, tooMany)) + "\n```"); ok {
		t.Fatal("report above its finding bound parsed")
	}
	overBody := maximal
	overBody.Body = strings.Repeat("b", 65537)
	if _, ok := ParseReviewReport("```review-json\n" + string(mustMarshal(t, overBody)) + "\n```"); ok {
		t.Fatal("report above its body bound parsed")
	}
	if _, ok := ParseReviewReport("```review-json\n" + strings.Repeat("x", MaxReviewReportBytes+1) + "\n```"); ok {
		t.Fatal("report block above the parser bound parsed")
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestReviewWorkEqual(t *testing.T) {
	comments, err := FindingCommentsForFindings([]Finding{journalTestFinding(0, true, true)})
	if err != nil {
		t.Fatal(err)
	}
	work := ReviewWork{
		OperationID: "review-" + strings.Repeat("1", 32), AuthRevision: "rev",
		Repository: 7, ActorID: 6, PRNumber: 9, PRID: 8, IssueID: 10, PRAuthorID: 5,
		HeadRef: "refs/heads/candidate", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		NativeRev: 1, NotAfter: 1300, Event: "APPROVED", Body: "LGTM",
	}
	same := work
	same.Comments = append([]FindingComment(nil), comments...)
	work.Comments = append([]FindingComment(nil), comments...)
	if !ReviewWorkEqual(work, same) {
		t.Fatal("identical work compares unequal")
	}
	moved := same
	moved.Comments = append([]FindingComment(nil), comments...)
	moved.Comments[0].Body += " "
	if ReviewWorkEqual(work, moved) {
		t.Fatal("changed comment compares equal")
	}
	dropped := same
	dropped.Comments = nil
	if ReviewWorkEqual(work, dropped) {
		t.Fatal("dropped comments compare equal")
	}
	retargeted := same
	retargeted.HeadOID = strings.Repeat("3", 40)
	if ReviewWorkEqual(work, retargeted) {
		t.Fatal("changed head compares equal")
	}
	if !ReviewWorkEqual(ReviewWork{}, ReviewWork{}) {
		t.Fatal("empty work compares unequal")
	}
}

func TestReviewCommentMappingValidation(t *testing.T) {
	if err := (ReviewCommentMapping{Key: testFindingKey(0), CommentID: 41}).Validate(); err != nil {
		t.Fatalf("comment mapping refused: %v", err)
	}
	for name, mapping := range map[string]ReviewCommentMapping{
		"short key":   {Key: "short", CommentID: 41},
		"zero record": {Key: testFindingKey(0), CommentID: 0},
	} {
		if err := mapping.Validate(); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestCorrectionJournalValidation(t *testing.T) {
	if err := (CorrectionJournal{}).Validate(); err == nil {
		t.Fatal("empty journal accepted")
	}
	_, _, journal := journalTestPublication(t)
	if err := journal.Validate(); err != nil {
		t.Fatalf("correction journal refused: %v", err)
	}
	// Entry validation checks shape only; cycle linkage belongs to the
	// publication, which sees the whole journal.
	renamed := journal
	renamed.CycleID = strings.Repeat("9", 32)
	if err := renamed.Validate(); err != nil {
		t.Fatalf("well-formed journal refused at entry validation: %v", err)
	}
	badRun := journal
	badRun.RunID = "short"
	if err := badRun.Validate(); err == nil {
		t.Fatal("journal with a bad run accepted")
	}
}

func TestPublicationReviewCycleJournal(t *testing.T) {
	p, cycle, journal := journalTestPublication(t)
	published := publishedPublication()
	published.ReviewCycles = []ReviewCycle{cycle}
	published.CorrectionJournals = []CorrectionJournal{journal}
	if err := published.Validate(); err != nil {
		t.Fatalf("cycle journal refused: %v", err)
	}
	// Cycles alone persist before any correction run reports.
	cyclesOnly := publishedPublication()
	cyclesOnly.ReviewCycles = []ReviewCycle{cycle}
	if err := cyclesOnly.Validate(); err != nil {
		t.Fatalf("cycle without correction refused: %v", err)
	}
	open := testPublication()
	open.ReviewCycles = []ReviewCycle{cycle}
	if err := open.Validate(); err == nil {
		t.Fatal("cycles on an open publication accepted")
	}
	foreign := publishedPublication()
	foreignCycle := cycle
	foreignCycle.PublicationID = strings.Repeat("9", 32)
	foreignCycle.ID = ReviewCycleID(foreignCycle.PublicationID, foreignCycle.AttemptRoot, foreignCycle.Round, foreignCycle.Phase)
	foreign.ReviewCycles = []ReviewCycle{foreignCycle}
	if err := foreign.Validate(); err == nil {
		t.Fatal("foreign cycle accepted")
	}
	duplicated := publishedPublication()
	duplicated.ReviewCycles = []ReviewCycle{cycle, cycle}
	if err := duplicated.Validate(); err == nil {
		t.Fatal("duplicate cycle accepted")
	}
	unrecorded := publishedPublication()
	unrecorded.ReviewCycles = []ReviewCycle{cycle}
	stray := journal
	stray.CycleID = ReviewCycleID(p.ID, strings.Repeat("b", 32), 2, "review")
	unrecorded.CorrectionJournals = []CorrectionJournal{stray}
	if err := unrecorded.Validate(); err == nil {
		t.Fatal("correction consuming an unrecorded cycle accepted")
	}
	stale := publishedPublication()
	stale.ReviewCycles = []ReviewCycle{cycle}
	moved := journal
	moved.Report.SetDigest = strings.Repeat("0", 64)
	stale.CorrectionJournals = []CorrectionJournal{moved}
	if err := stale.Validate(); err == nil {
		t.Fatal("correction answering a changed finding set accepted")
	}
	reported := publishedPublication()
	reported.ReviewCycles = []ReviewCycle{cycle}
	second := journal
	second.RunID = strings.Repeat("d", 32)
	reported.CorrectionJournals = []CorrectionJournal{journal, second}
	if err := reported.Validate(); err != nil {
		t.Fatalf("second correction run on its cycle refused: %v", err)
	}
	clash := publishedPublication()
	clash.ReviewCycles = []ReviewCycle{cycle}
	clash.CorrectionJournals = []CorrectionJournal{journal, journal}
	if err := clash.Validate(); err == nil {
		t.Fatal("correction run reported twice accepted")
	}
}
