package factory

import (
	"strings"
	"testing"
)

func deliveryTestAcceptance() DeliveryAcceptance {
	publicationID := strings.Repeat("7", 32)
	attemptRoot := strings.Repeat("b", 32)
	head, base, diffBase := strings.Repeat("2", 40), strings.Repeat("1", 40), strings.Repeat("3", 40)
	return DeliveryAcceptance{
		Tuple:         CandidateTuple{Head: head, Base: base, DiffBase: diffBase},
		ID:            DeliveryAcceptanceID(publicationID, attemptRoot, 1, AcceptanceCleanReview),
		PublicationID: publicationID, AttemptRoot: attemptRoot,
		ScopeDigest: deliveryTestSeedDigest(1), FindingDigest: deliveryTestSeedDigest(2),
		CoverageDigest: deliveryTestSeedDigest(3), CheckDigest: deliveryTestSeedDigest(4),
		PolicyRevision: "policy-revision-9",
		OriginReceipt:  strings.Repeat("4", 32),
		Kind:           AcceptanceCleanReview, Round: 1,
	}
}

func deliveryTestSeedDigest(n int) string {
	return strings.Repeat("abcdef0123456789"[n:n+1], 64)
}

func TestFindingContentDigestStableAndOrderFree(t *testing.T) {
	empty, err := FindingContentDigest(nil)
	if err != nil || !ValidDigest(empty) {
		t.Fatalf("empty finding set needs its content digest: %v", err)
	}
	first := []Finding{testFinding(0, true), testFinding(1, false)}
	second := []Finding{testFinding(1, false), testFinding(0, true)}
	a, err := FindingContentDigest(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := FindingContentDigest(second)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("finding content digest depends on key order")
	}
	if a == empty {
		t.Fatal("nonempty finding set aliases the empty digest")
	}
}

func TestFindingContentDigestCoversEveryField(t *testing.T) {
	base := []Finding{testFinding(0, true)}
	want, err := FindingContentDigest(base)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*Finding){
		"defect":        func(f *Finding) { f.Defect = "another defect" },
		"evidence":      func(f *Finding) { f.Evidence = "another evidence" },
		"outcome":       func(f *Finding) { f.RequiredOutcome = "another outcome" },
		"severity":      func(f *Finding) { f.Severity = FindingSeverityMinor },
		"actionability": func(f *Finding) { f.Actionable = false },
		"fix kind":      func(f *Finding) { f.FixKind = FindingFixVerification },
		"origin run":    func(f *Finding) { f.OriginRun = strings.Repeat("9", 32) },
		"origin review": func(f *Finding) { f.OriginReview = 41 },
	}
	for name, mutate := range mutations {
		next := testFinding(0, true)
		mutate(&next)
		got, err := FindingContentDigest([]Finding{next})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got == want {
			t.Fatalf("%s change left the content digest unchanged", name)
		}
	}
	located := testFinding(0, true)
	located.Location = &FindingLocation{Path: "x.go", Side: "new", Line: 3}
	locatedWant, err := FindingContentDigest([]Finding{located})
	if err != nil {
		t.Fatal(err)
	}
	if locatedWant == want {
		t.Fatal("added location left the content digest unchanged")
	}
	for name, anchor := range map[string]FindingLocation{
		"location line": {Path: "x.go", Side: "new", Line: 4},
		"location side": {Path: "x.go", Side: "old", Line: 3},
		"location path": {Path: "y.go", Side: "new", Line: 3},
	} {
		other := located
		other.Location = &FindingLocation{Path: anchor.Path, Side: anchor.Side, Line: anchor.Line}
		got, err := FindingContentDigest([]Finding{other})
		if err != nil {
			t.Fatal(err)
		}
		if got == locatedWant {
			t.Fatalf("%s left the content digest unchanged", name)
		}
	}
	if _, err := FindingContentDigest([]Finding{testFinding(0, true), testFinding(0, false)}); err == nil {
		t.Fatal("duplicate finding key accepted")
	}
	bad := testFinding(0, true)
	bad.Severity = "blocker"
	if _, err := FindingContentDigest([]Finding{bad}); err == nil {
		t.Fatal("invalid finding accepted into the content digest")
	}
}

func TestKeyOnlyDigestMissesContentChange(t *testing.T) {
	before := []Finding{testFinding(0, true), testFinding(1, false)}
	after := []Finding{testFinding(0, true), testFinding(1, false)}
	after[0].Defect = "rewritten defect with the same key"
	after[1].Actionable = true
	after[1].FixKind = FindingFixVerification
	keys := func(set []Finding) []string {
		out := make([]string, 0, len(set))
		for _, finding := range set {
			out = append(out, finding.Key)
		}
		return out
	}
	keyBefore, err := FindingSetDigest(keys(before))
	if err != nil {
		t.Fatal(err)
	}
	keyAfter, err := FindingSetDigest(keys(after))
	if err != nil {
		t.Fatal(err)
	}
	if keyBefore != keyAfter {
		t.Fatal("key-only digest should be blind to content rewrites")
	}
	contentBefore, err := FindingContentDigest(before)
	if err != nil {
		t.Fatal(err)
	}
	contentAfter, err := FindingContentDigest(after)
	if err != nil {
		t.Fatal(err)
	}
	if contentBefore == contentAfter {
		t.Fatal("content digest missed a same-key rewrite: key-only authorization would pass")
	}
}

func TestCheckSubmittedFindings(t *testing.T) {
	canonical := []Finding{testFinding(0, true), testFinding(1, false)}
	reordered := []Finding{testFinding(1, false), testFinding(0, true)}
	if err := CheckSubmittedFindings(canonical, reordered); err != nil {
		t.Fatalf("identical content in another order refused: %v", err)
	}
	rewritten := []Finding{testFinding(0, true), testFinding(1, false)}
	rewritten[0].Defect = "rewritten defect"
	if err := CheckSubmittedFindings(canonical, rewritten); err == nil {
		t.Fatal("rewritten canonical defect accepted")
	}
	swapped := []Finding{testFinding(0, true), testFinding(2, false)}
	if err := CheckSubmittedFindings(canonical, swapped); err == nil {
		t.Fatal("unknown finding key accepted")
	}
	short := []Finding{testFinding(0, true)}
	if err := CheckSubmittedFindings(canonical, short); err == nil {
		t.Fatal("dropped finding accepted")
	}
	invalid := []Finding{testFinding(0, true), testFinding(1, false)}
	invalid[1].Severity = "blocker"
	if err := CheckSubmittedFindings(canonical, invalid); err == nil {
		t.Fatal("invalid submitted finding accepted")
	}
	dup := []Finding{testFinding(0, true), testFinding(0, true)}
	if err := CheckSubmittedFindings(dup, dup); err == nil {
		t.Fatal("duplicate canonical key accepted")
	}
}

func clearTestReport(head, base, coverage, cycleID string, assessments []FindingAssessment) ClearReviewReport {
	return ClearReviewReport{
		Assessments: assessments, CycleID: cycleID, Conclusion: ClearReviewConclusion,
		Head: head, BaseOID: base, CoverageDigest: coverage,
	}
}

func TestClearReviewReportValidate(t *testing.T) {
	head, base := strings.Repeat("2", 40), strings.Repeat("1", 40)
	cycleID := strings.Repeat("5", 32)
	content, err := FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := ClearCoverageDigest(head, base, content)
	if err != nil {
		t.Fatal(err)
	}
	if err := clearTestReport(head, base, coverage, cycleID, nil).Validate(nil); err != nil {
		t.Fatalf("first-review clear refused: %v", err)
	}
	carried := []Finding{testFinding(0, true), testFinding(1, false)}
	content, err = FindingContentDigest(carried)
	if err != nil {
		t.Fatal(err)
	}
	coverage, err = ClearCoverageDigest(head, base, content)
	if err != nil {
		t.Fatal(err)
	}
	verified := []FindingAssessment{{Key: testFindingKey(0), Status: FindingVerified, Evidence: "outcome holds on this head"}}
	if err := clearTestReport(head, base, coverage, cycleID, verified).Validate(carried); err != nil {
		t.Fatalf("verified later clear refused: %v", err)
	}
	stillOpen := []FindingAssessment{{Key: testFindingKey(0), Status: FindingStillOpen, Evidence: "not yet"}}
	if err := clearTestReport(head, base, coverage, cycleID, stillOpen).Validate(carried); err == nil {
		t.Fatal("clear with a still-open finding accepted")
	}
	if err := clearTestReport(head, base, coverage, cycleID, nil).Validate(carried); err == nil {
		t.Fatal("clear with an omitted carried assessment accepted")
	}
	unknown := []FindingAssessment{{Key: testFindingKey(9), Status: FindingVerified, Evidence: "x"}}
	if err := clearTestReport(head, base, coverage, cycleID, append(verified, unknown...)).Validate(carried); err == nil {
		t.Fatal("clear assessment for an unknown finding accepted")
	}
	dup := append(verified, verified...)
	if err := clearTestReport(head, base, coverage, cycleID, dup).Validate(carried); err == nil {
		t.Fatal("duplicate clear assessment accepted")
	}
	unclear := clearTestReport(head, base, coverage, cycleID, verified)
	unclear.Conclusion = "approve"
	if err := unclear.Validate(carried); err == nil {
		t.Fatal("non-explicit conclusion accepted as clear")
	}
	sameHead := clearTestReport(head, head, coverage, cycleID, verified)
	if err := sameHead.Validate(carried); err == nil {
		t.Fatal("clear without an exact candidate accepted")
	}
	noCoverage := clearTestReport(head, base, "x", cycleID, verified)
	if err := noCoverage.Validate(carried); err == nil {
		t.Fatal("clear without a coverage digest accepted")
	}
}

func TestParseClearReviewReport(t *testing.T) {
	head, base := strings.Repeat("2", 40), strings.Repeat("1", 40)
	cycleID := strings.Repeat("5", 32)
	content, err := FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := ClearCoverageDigest(head, base, content)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"cycle_id":"` + cycleID + `","conclusion":"clear","head":"` + head + `","base_oid":"` + base + `","coverage_digest":"` + coverage + `"}`
	output := "nothing found\n```" + ClearReviewReportFence + "\n" + body + "\n```\ntail"
	report, ok := ParseClearReviewReport(output)
	if !ok || report.Head != head || report.CoverageDigest != coverage {
		t.Fatalf("explicit clear block refused: %+v %v", report, ok)
	}
	for name, text := range map[string]string{
		"silence":         "nothing found, no block at all",
		"empty output":    "",
		"native approval": "looks good\n```" + ReviewReportFence + "\n{\"verdict\":\"approve\",\"summary\":\"fine\",\"body\":\"x\",\"findings\":[]}\n```",
		"wrong fence":     "```review-json\n" + body + "\n```",
		"wrong verdict":   "```" + ClearReviewReportFence + "\n{\"cycle_id\":\"" + cycleID + "\",\"conclusion\":\"approve\",\"head\":\"" + head + "\",\"base_oid\":\"" + base + "\",\"coverage_digest\":\"" + coverage + "\"}\n```",
		"unknown field":   "```" + ClearReviewReportFence + "\n{\"cycle_id\":\"" + cycleID + "\",\"conclusion\":\"clear\",\"head\":\"" + head + "\",\"base_oid\":\"" + base + "\",\"coverage_digest\":\"" + coverage + "\",\"extra\":1}\n```",
	} {
		if _, ok := ParseClearReviewReport(text); ok {
			t.Fatalf("%s parsed as an explicit clear", name)
		}
	}
}

func TestClearCoverageDigest(t *testing.T) {
	head, base := strings.Repeat("2", 40), strings.Repeat("1", 40)
	content, err := FindingContentDigest([]Finding{testFinding(0, true)})
	if err != nil {
		t.Fatal(err)
	}
	first, err := ClearCoverageDigest(head, base, content)
	if err != nil || !ValidDigest(first) {
		t.Fatalf("coverage digest refused: %v", err)
	}
	again, err := ClearCoverageDigest(head, base, content)
	if err != nil || again != first {
		t.Fatal("coverage digest is not deterministic")
	}
	moved, err := ClearCoverageDigest(strings.Repeat("9", 40), base, content)
	if err != nil || moved == first {
		t.Fatal("moved head left the coverage digest unchanged")
	}
	other, err := FindingContentDigest([]Finding{testFinding(1, true)})
	if err != nil {
		t.Fatal(err)
	}
	rewritten, err := ClearCoverageDigest(head, base, other)
	if err != nil || rewritten == first {
		t.Fatal("rewritten findings left the coverage digest unchanged")
	}
	if _, err := ClearCoverageDigest(head, head, content); err == nil {
		t.Fatal("coverage without an exact candidate accepted")
	}
	if _, err := ClearCoverageDigest(head, base, "nope"); err == nil {
		t.Fatal("coverage without a content digest accepted")
	}
}

func TestDeliveryAcceptanceValidate(t *testing.T) {
	if err := deliveryTestAcceptance().Validate(); err != nil {
		t.Fatalf("clean acceptance refused: %v", err)
	}
	jev := deliveryTestAcceptance()
	jev.Kind = AcceptanceJevAcceptance
	jev.DecisionID = "decision-1"
	jev.ID = DeliveryAcceptanceID(jev.PublicationID, jev.AttemptRoot, jev.Round, jev.Kind)
	jev.ResolutionIntents = []string{strings.Repeat("6", 32)}
	if err := jev.Validate(); err != nil {
		t.Fatalf("funded acceptance refused: %v", err)
	}
	spender := deliveryTestAcceptance()
	spender.DecisionID = "decision-1"
	if err := spender.Validate(); err == nil {
		t.Fatal("clean acceptance spending a delivery judgment accepted")
	}
	resolver := deliveryTestAcceptance()
	resolver.ResolutionIntents = []string{strings.Repeat("6", 32)}
	if err := resolver.Validate(); err == nil {
		t.Fatal("clean acceptance resolving a native thread accepted")
	}
	unfunded := jev
	unfunded.DecisionID = ""
	if err := unfunded.Validate(); err == nil {
		t.Fatal("funded acceptance without its decision accepted")
	}
	for name, mutate := range map[string]func(*DeliveryAcceptance){
		"kind":     func(a *DeliveryAcceptance) { a.Kind = "approved" },
		"identity": func(a *DeliveryAcceptance) { a.ID = strings.Repeat("0", 32) },
		"scope":    func(a *DeliveryAcceptance) { a.AttemptRoot = "short" },
		"round":    func(a *DeliveryAcceptance) { a.Round = 0 },
		"tuple":    func(a *DeliveryAcceptance) { a.Tuple.Head = a.Tuple.Base },
		"digest":   func(a *DeliveryAcceptance) { a.FindingDigest = "short" },
		"policy":   func(a *DeliveryAcceptance) { a.PolicyRevision = "" },
		"receipt":  func(a *DeliveryAcceptance) { a.OriginReceipt = "short" },
		"intents": func(a *DeliveryAcceptance) {
			a.Kind = AcceptanceJevAcceptance
			a.DecisionID = "decision-1"
			a.ID = DeliveryAcceptanceID(a.PublicationID, a.AttemptRoot, a.Round, a.Kind)
			a.ResolutionIntents = []string{strings.Repeat("6", 32), strings.Repeat("6", 32)}
		},
	} {
		next := deliveryTestAcceptance()
		mutate(&next)
		if err := next.Validate(); err == nil {
			t.Fatalf("%s defect accepted", name)
		}
	}
}

func TestResultCompletedCandidate(t *testing.T) {
	head := strings.Repeat("2", 40)
	done := Result{Status: "completed", Summary: "reviewed", Candidate: head, ReviewPassed: true, Findings: []string{}}
	if got, err := done.CompletedCandidate(); err != nil || got != head {
		t.Fatalf("completed result backs no candidate: %v", err)
	}
	for _, status := range []string{"blocked", "failed", "cancelled"} {
		held := Result{Status: status, Summary: "not done", Candidate: head, Findings: []string{}}
		if _, err := held.CompletedCandidate(); err == nil {
			t.Fatalf("%s process backed a clear review", status)
		}
	}
	empty := Result{Status: "completed", Summary: "reviewed", Findings: []string{}}
	if _, err := empty.CompletedCandidate(); err == nil {
		t.Fatal("completed result without a candidate accepted")
	}
}
