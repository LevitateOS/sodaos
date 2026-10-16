package factory

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestST10ReviewerCannotBecomePublication(t *testing.T) {
	p := testPublication()
	if err := p.Validate(); err != nil {
		t.Fatalf("coder fixture: %v", err)
	}
	p.Role = project.RoleReviewer
	if err := p.Validate(); err == nil {
		t.Fatal("reviewer candidate accepted for publication")
	}
}

func TestParseReviewReport(t *testing.T) {
	good := "notes\n```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"LGTM\",\"findings\":[]}\n```\ntail"
	report, ok := ParseReviewReport(good)
	if !ok || report.Verdict != "approve" || report.Body != "LGTM" {
		t.Fatalf("valid report rejected: %+v %v", report, ok)
	}
	changes := "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"line 3 is wrong\",\"findings\":[\"line 3\"]}\n```"
	if report, ok := ParseReviewReport(changes); !ok || report.Verdict != "request-changes" || len(report.Findings) != 1 {
		t.Fatalf("changes report rejected: %+v %v", report, ok)
	}
	nested := "```review-json\n{\"verdict\":\"approve\",\"summary\":\"solid\",\"body\":\"see:\\n```diff\\n-a\\n+b\\n```\",\"findings\":[]}\n```"
	if report, ok := ParseReviewReport(nested); !ok || report.Verdict != "approve" || !strings.Contains(report.Body, "```diff") {
		t.Fatalf("nested-fence report rejected: %+v %v", report, ok)
	}
	for name, output := range map[string]string{
		"missing fence":     "no report here",
		"unterminated":      "```review-json\n{\"verdict\":\"approve\"}",
		"unknown field":     "```review-json\n{\"verdict\":\"approve\",\"summary\":\"s\",\"body\":\"b\",\"findings\":[],\"extra\":1}\n```",
		"bad verdict":       "```review-json\n{\"verdict\":\"maybe\",\"summary\":\"s\",\"body\":\"b\",\"findings\":[]}\n```",
		"changes w/o body":  "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"s\",\"body\":\"  \",\"findings\":[]}\n```",
		"object finding":    "```review-json\n{\"verdict\":\"request-changes\",\"summary\":\"fix\",\"body\":\"README.md:13 still has the pending line\",\"findings\":[{\"location\":\"README.md:13\",\"evidence\":\"pending line remains\",\"required\":\"replace the line\"}]}\n```",
		"first block stale": "```review-json\n{\"verdict\":\"maybe\"}\n```\n" + changes,
	} {
		t.Run(name, func(t *testing.T) {
			report, ok := ParseReviewReport(output)
			if name == "first block stale" {
				if !ok || report.Verdict != "request-changes" {
					t.Fatalf("last block not preferred: %+v %v", report, ok)
				}
				return
			}
			if ok {
				t.Fatalf("invalid report accepted: %+v", report)
			}
		})
	}
	if got := ReviewAuthRevision("a1", "r1"); got == "" || len(got) > 512 {
		t.Fatalf("review auth revision malformed: %q", got)
	}
}

func TestReviewReportValidationBoundsPersistedEvidence(t *testing.T) {
	tooMany := ReviewReport{Verdict: "approve", Findings: make([]string, 65)}
	if err := tooMany.Validate(); err == nil {
		t.Fatal("direct reviewer evidence above the result finding-count bound accepted")
	}

	tooLarge := ReviewReport{Verdict: "approve", Findings: []string{strings.Repeat("x", 4097)}}
	if err := tooLarge.Validate(); err == nil {
		t.Fatal("direct reviewer evidence above the result finding-size bound accepted")
	}

	encoded, err := json.Marshal(tooLarge)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ParseReviewReport("```review-json\n" + string(encoded) + "\n```"); ok {
		t.Fatal("parser accepted reviewer output beyond persisted finding bounds")
	}

	withinBounds := ReviewReport{Verdict: "request-changes", Summary: strings.Repeat("s", 4096), Body: "finding body", Findings: make([]string, 64)}
	for i := range withinBounds.Findings {
		withinBounds.Findings[i] = strings.Repeat("f", 4096)
	}
	if err := withinBounds.Validate(); err != nil {
		t.Fatalf("review evidence at existing result bounds refused: %v", err)
	}
}

func testFindingKey(n int) string { return strings.Repeat("0", 31) + string(rune('a'+n)) }

func testFinding(n int, actionable bool) Finding {
	return Finding{
		Key: testFindingKey(n), OriginRun: strings.Repeat("1", 32), Severity: FindingSeverityMajor,
		Defect: "missing bound", Evidence: "line 42 lacks a check", RequiredOutcome: "add the check",
		FixKind: FindingFixCode, Actionable: actionable,
	}
}

func TestFindingValidationBounds(t *testing.T) {
	if err := testFinding(0, true).Validate(); err != nil {
		t.Fatalf("keyed finding refused: %v", err)
	}
	badKey := testFinding(0, true)
	badKey.Key = "short"
	if err := badKey.Validate(); err == nil {
		t.Fatal("short finding key accepted")
	}
	badSeverity := testFinding(0, true)
	badSeverity.Severity = "blocker"
	if err := badSeverity.Validate(); err == nil {
		t.Fatal("unknown severity accepted")
	}
	badKind := testFinding(0, true)
	badKind.FixKind = "docs"
	if err := badKind.Validate(); err == nil {
		t.Fatal("unknown fix kind accepted")
	}
	noEvidence := testFinding(0, true)
	noEvidence.Evidence = "  "
	if err := noEvidence.Validate(); err == nil {
		t.Fatal("evidenceless finding accepted")
	}
	verbose := testFinding(0, true)
	verbose.Defect = strings.Repeat("d", MaxFindingProseBytes+1)
	if err := verbose.Validate(); err == nil {
		t.Fatal("finding prose above its bound accepted")
	}
	located := testFinding(0, true)
	located.Location = &FindingLocation{Path: "internal/factory/review_native.go", Side: "new", Line: 42}
	if err := located.Validate(); err != nil {
		t.Fatalf("located finding refused: %v", err)
	}
	badSide := testFinding(0, true)
	badSide.Location = &FindingLocation{Path: "x.go", Side: "both", Line: 1}
	if err := badSide.Validate(); err == nil {
		t.Fatal("bad anchor side accepted")
	}
	keys := []string{testFindingKey(1), testFindingKey(0)}
	first, err := FindingSetDigest(keys)
	if err != nil || !ValidDigest(first) {
		t.Fatalf("finding-set digest malformed: %v", err)
	}
	reordered, err := FindingSetDigest([]string{testFindingKey(0), testFindingKey(1)})
	if err != nil || reordered != first {
		t.Fatal("key order changed finding-set digest")
	}
	if _, err := FindingSetDigest([]string{testFindingKey(0), testFindingKey(0)}); err == nil {
		t.Fatal("duplicate finding key accepted")
	}
}

func TestReviewReportBoundsPersistMaximum(t *testing.T) {
	maxBody := ReviewReport{Verdict: "request-changes", Summary: "s", Body: strings.Repeat("b", 65536), Findings: []string{"f"}}
	if err := maxBody.Validate(); err != nil {
		t.Fatalf("maximum report refused: %v", err)
	}
	overBody := maxBody
	overBody.Body = strings.Repeat("b", 65537)
	if err := overBody.Validate(); err == nil {
		t.Fatal("report body above 64 KiB accepted")
	}
	overSummary := maxBody
	overSummary.Summary = strings.Repeat("s", 4097)
	if err := overSummary.Validate(); err == nil {
		t.Fatal("report summary above 4 KiB accepted")
	}
}

func testResolveBinding() ActorBindingRef {
	return ActorBindingRef{TokenID: 2, ActorID: 6, Kind: OpThreadResolve}
}

func testResolveRequest(digest string) ResolveRequest {
	return ResolveRequest{
		HeadRef: "refs/heads/fix", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		Digest: digest, Repository: 7, PRNumber: 3, PRID: 5,
		RootCommentID: 11, ActorID: 6, Resolved: true,
	}
}

func testThreadEvidence(digest string) ThreadEvidence {
	return ThreadEvidence{Digest: digest, Repository: 7, PRNumber: 3, RootCommentID: 11}
}

func TestResolveAuthorityRefusals(t *testing.T) {
	lines := []string{
		ResolveConversationLine(ResolveConversationMember{ID: 11, Body: "needs a bounds check", ContentVer: 1, Location: "review_native.go|new|42", ResolverState: "open"}),
		ResolveConversationLine(ResolveConversationMember{ID: 12, Body: "added in the next push", ContentVer: 2, Location: "review_native.go|new|42", ResolverState: "open"}),
	}
	digest, err := ConversationDigest(lines)
	if err != nil || !ValidDigest(digest) {
		t.Fatalf("conversation digest malformed: %v", err)
	}
	// Shared cross-repo vector: the X SDK twin must reproduce these bytes
	// exactly; any divergence refuses as changed content.
	if digest != "ac7ab3e7cf25a41981ae75395c5ee3d7a2106d9fe515bcf68de9ffa1d23824c6" {
		t.Fatalf("conversation digest moved: %s", digest)
	}
	if err := testResolveRequest(digest).ValidateForEvidence(testResolveBinding(), 1, testThreadEvidence(digest)); err != nil {
		t.Fatalf("exact resolve replay refused: %v", err)
	}
	changed, err := ConversationDigest(append(lines, ResolveConversationLine(ResolveConversationMember{ID: 13, Body: "concurrent reply", ContentVer: 1, Location: "review_native.go|new|42", ResolverState: "open"})))
	if err != nil {
		t.Fatal(err)
	}
	if changed == digest {
		t.Fatal("concurrent reply kept conversation digest")
	}
	cases := map[string]func() error{
		"missing binding": func() error {
			return testResolveRequest(digest).ValidateForEvidence(ActorBindingRef{}, 1, testThreadEvidence(digest))
		},
		"review binding reused": func() error {
			binding := testResolveBinding()
			binding.Kind = OpReviewSubmit
			return testResolveRequest(digest).ValidateForEvidence(binding, 1, testThreadEvidence(digest))
		},
		"wrong actor": func() error {
			binding := testResolveBinding()
			binding.ActorID = 9
			return testResolveRequest(digest).ValidateForEvidence(binding, 1, testThreadEvidence(digest))
		},
		"author resolves own thread": func() error {
			return testResolveRequest(digest).ValidateForEvidence(testResolveBinding(), 6, testThreadEvidence(digest))
		},
		"reply passed as root": func() error {
			evidence := testThreadEvidence(digest)
			evidence.RootCommentID = 12
			return testResolveRequest(digest).ValidateForEvidence(testResolveBinding(), 1, evidence)
		},
		"deleted root": func() error {
			evidence := testThreadEvidence(digest)
			evidence.Deleted = true
			return testResolveRequest(digest).ValidateForEvidence(testResolveBinding(), 1, evidence)
		},
		"changed content": func() error {
			return testResolveRequest(changed).ValidateForEvidence(testResolveBinding(), 1, testThreadEvidence(digest))
		},
		"unresolve refused": func() error {
			request := testResolveRequest(digest)
			request.Resolved = false
			return request.ValidateForEvidence(testResolveBinding(), 1, testThreadEvidence(digest))
		},
	}
	for name, resolve := range cases {
		if err := resolve(); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestReviewCycleAndCorrectionContracts(t *testing.T) {
	publicationID := strings.Repeat("a", 32)
	attemptRoot := strings.Repeat("b", 32)
	cycle := ReviewCycle{
		PublicationID: publicationID, AttemptRoot: attemptRoot,
		ID:                 ReviewCycleID(publicationID, attemptRoot, 1, "review"),
		Phase:              "review",
		TriggerDecision:    DeliveryReviewAgain,
		TriggerReason:      "correction set needs another inspection",
		Head:               strings.Repeat("2", 40),
		BaseOID:            strings.Repeat("1", 40),
		FindingDigest:      strings.Repeat("3", 64),
		ReviewerAssignment: strings.Repeat("4", 32),
		Round:              1,
	}
	if err := cycle.Validate(); err != nil {
		t.Fatalf("review cycle refused: %v", err)
	}
	sameHead := cycle
	sameHead.Round = 2
	sameHead.ID = ReviewCycleID(publicationID, attemptRoot, 2, "review")
	if err := sameHead.Validate(); err != nil {
		t.Fatalf("second round on the same head refused: %v", err)
	}
	if sameHead.ID == cycle.ID {
		t.Fatal("distinct rounds share an identity")
	}
	moved := cycle
	moved.Round = 2
	if err := moved.Validate(); err == nil {
		t.Fatal("round advance kept its old identity")
	}
	carried := []Finding{testFinding(0, true), testFinding(1, false)}
	report := ReviewCycleReport{
		CycleID: strings.Repeat("5", 32),
		Assessments: []FindingAssessment{
			{Key: testFindingKey(0), Status: FindingVerified, Evidence: "check run 7 passes the required outcome"},
		},
	}
	if err := report.Validate(carried); err != nil {
		t.Fatalf("verified round refused: %v", err)
	}
	if report.RequiresCorrectionRun() {
		t.Fatal("verified round demands a correction run")
	}
	optionalOnly := ReviewCycleReport{
		CycleID: strings.Repeat("5", 32),
		NewFindings: []Finding{func() Finding {
			optional := testFinding(2, false)
			optional.RequiredOutcome = "no action required"
			return optional
		}()},
	}
	if err := optionalOnly.Validate(nil); err != nil {
		t.Fatalf("optional-only round refused: %v", err)
	}
	if optionalOnly.RequiresCorrectionRun() {
		t.Fatal("optional-only round forces a coder run")
	}
	stillOpen := ReviewCycleReport{
		CycleID: strings.Repeat("5", 32),
		Assessments: []FindingAssessment{
			{Key: testFindingKey(0), Status: FindingStillOpen, Evidence: "defect still present at line 42"},
		},
	}
	if err := stillOpen.Validate(carried); err != nil {
		t.Fatalf("still-open round refused: %v", err)
	}
	if !stillOpen.RequiresCorrectionRun() {
		t.Fatal("still-open round skips its correction run")
	}
	omitted := ReviewCycleReport{CycleID: strings.Repeat("5", 32)}
	if err := omitted.Validate(carried); err == nil {
		t.Fatal("carried actionable finding silently omitted")
	}
	keys := []string{testFindingKey(0), testFindingKey(1)}
	digest, err := FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	correction := CorrectionReport{
		SetDigest: digest,
		Findings:  carried,
		Dispositions: []CorrectionDisposition{
			{Key: testFindingKey(0), Disposition: DispositionFixed},
			{Key: testFindingKey(1), Disposition: DispositionDeclinedOptional},
		},
	}
	if err := correction.Validate(); err != nil {
		t.Fatalf("correction dispositions refused: %v", err)
	}
	declined := correction
	declined.Dispositions = []CorrectionDisposition{
		{Key: testFindingKey(0), Disposition: DispositionDeclinedOptional},
		{Key: testFindingKey(1), Disposition: DispositionDeclinedOptional},
	}
	if err := declined.Validate(); err == nil {
		t.Fatal("actionable finding declined")
	}
	partial := correction
	partial.Dispositions = partial.Dispositions[:1]
	if err := partial.Validate(); err == nil {
		t.Fatal("partial dispositions accepted")
	}
	stale := correction
	stale.SetDigest = strings.Repeat("0", 64)
	if err := stale.Validate(); err == nil {
		t.Fatal("stale finding-set digest accepted")
	}
}
