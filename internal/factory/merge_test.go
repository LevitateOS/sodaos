package factory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func testMerge() Merge {
	return Merge{
		Operation:     MergeOperation{Kind: OpMerge},
		ID:            "abcdef0123456789abcdef0123456789",
		PublicationID: "1234567890abcdef1234567890abcdef",
		AssignmentID:  "0123456789abcdef0123456789abcdef",
		ProjectID:     "p" + strings.Repeat("d", 24),
		Role:          "soda-coder",
		Acceptance:    "d" + strings.Repeat("a", 24),
		HeadRef:       "refs/heads/soda/factory/0123456789abcdef0123456789abcdef",
		BaseRef:       "refs/heads/main",
		HeadOID:       strings.Repeat("2", 40),
		BaseOID:       strings.Repeat("1", 40),
		Stage:         MergeOpen,
		Repository:    7,
		Issue:         3,
		PRNumber:      9,
		PRID:          11,
		IssueID:       13,
		PRAuthorID:    5,
		ReviewerID:    6,
		CreatedUnix:   1100,
	}
}

func TestMergeValidatesOpen(t *testing.T) {
	if err := testMerge().Validate(); err != nil {
		t.Fatalf("open merge refused: %v", err)
	}
}

func TestMergeRejectsMalformed(t *testing.T) {
	cases := map[string]func(*Merge){
		"identity":    func(m *Merge) { m.ID = "short" },
		"publication": func(m *Merge) { m.PublicationID = "" },
		"assignment":  func(m *Merge) { m.AssignmentID = "short" },
		"project":     func(m *Merge) { m.ProjectID = "nope" },
		"role":        func(m *Merge) { m.Role = "coder" },
		"scope":       func(m *Merge) { m.Issue = 0 },
		"pr":          func(m *Merge) { m.PRNumber = 0 },
		"author":      func(m *Merge) { m.PRAuthorID = 0 },
		"reviewer":    func(m *Merge) { m.ReviewerID = m.PRAuthorID },
		"acceptance":  func(m *Merge) { m.Acceptance = "nope" },
		"headref":     func(m *Merge) { m.HeadRef = "main" },
		"sameref":     func(m *Merge) { m.HeadRef = m.BaseRef },
		"head":        func(m *Merge) { m.HeadOID = "xyz" },
		"same":        func(m *Merge) { m.HeadOID = m.BaseOID },
		"realized":    func(m *Merge) { m.MergedCommit = strings.Repeat("3", 40) },
		"observation": func(m *Merge) { m.NativeRev = 4 },
		"created":     func(m *Merge) { m.CreatedUnix = 0 },
		"kind":        func(m *Merge) { m.Operation.Kind = OpPRCreate },
		"stage":       func(m *Merge) { m.Stage = "merging" },
		"open":        func(m *Merge) { m.Outcome = Succeeded },
		"openfinish":  func(m *Merge) { m.FinishedUnix = 1200 },
	}
	for name, mutate := range cases {
		m := testMerge()
		mutate(&m)
		if err := m.Validate(); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}

func testMergeIntent(operationID string) *MergeIntent {
	return &MergeIntent{
		OperationID: operationID, AuthRevision: "accepted-revision",
		HeadRef: "refs/heads/soda/factory/0123456789abcdef0123456789abcdef", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		Repository: 7, ActorID: 8, PRNumber: 9, PRID: 11, IssueID: 13,
		PRAuthorID: 5, ReviewerID: 6,
		NativeRev: 12, NotAfter: 1900, AssessmentRevision: 2, ReviewID: 21,
	}
}

func testMergeWork() MergeWork {
	return MergeWork{
		MergeID: "abcdef0123456789abcdef0123456789", PublicationID: "1234567890abcdef1234567890abcdef",
		OperationID: "soda-1234567890abcdef1234567890abcdef-merge-1", AuthRevision: "accepted-revision",
		HeadRef: "refs/heads/soda/factory/0123456789abcdef0123456789abcdef", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		Repository: 7, Issue: 3, PRNumber: 9, PRID: 11, IssueID: 13,
		PRAuthorID: 5, ReviewerID: 6, ActorID: 8,
		NativeRev: 12, NotAfter: 1900, AssessmentRevision: 2, ReviewID: 21,
	}
}

func TestMergeIntentRoundTrip(t *testing.T) {
	w := testMergeWork()
	restored := w.Intent().Apply(MergeWork{MergeID: w.MergeID, PublicationID: w.PublicationID, Issue: w.Issue})
	if restored != w {
		t.Fatalf("intent round trip differs: %+v", restored)
	}
	empty := MergeWork{MergeID: w.MergeID, PublicationID: w.PublicationID, Issue: w.Issue}
	if err := empty.Intent().validate(false); err == nil {
		t.Fatal("empty intent accepted for observation")
	}
	w.NativeRev, w.NotAfter, w.AssessmentRevision, w.ReviewID = 0, 0, 0, 0
	if err := w.ValidateObservation(); err != nil {
		t.Fatalf("unobserved work refused: %v", err)
	}
	if err := w.Validate(); err == nil {
		t.Fatal("unobserved work accepted for submission")
	}
}

func TestMergeOperationRejectsMalformed(t *testing.T) {
	m := testMerge()
	m.Operation = MergeOperation{Kind: OpMerge, Attempts: 1, OperationID: "soda-x-merge-1", UpdatedUnix: 1200, Work: testMergeIntent("soda-x-merge-1")}
	if err := m.Validate(); err != nil {
		t.Fatalf("in-flight merge refused: %v", err)
	}
	cases := map[string]func(*Merge){
		"identity": func(m *Merge) { m.Operation.OperationID = "bad id" },
		"intent":   func(m *Merge) { m.Operation.Work.OperationID = "soda-y-merge-1" },
		"effect":   func(m *Merge) { m.Operation.Effect = "merged" },
		"time":     func(m *Merge) { m.Operation.UpdatedUnix = 0 },
		"cancel":   func(m *Merge) { m.Operation.Cancellation = "canceled" },
		"complete": func(m *Merge) { m.Operation.Completion = "done" },
		"reason":   func(m *Merge) { m.Operation.Reason = strings.Repeat("r", 65) },
		"links":    func(m *Merge) { m.Operation.PRNumber = 9 },
		"head":     func(m *Merge) { m.Operation.Work.HeadOID = strings.Repeat("c", 40) },
	}
	for name, mutate := range cases {
		next := testMerge()
		next.Operation = MergeOperation{Kind: OpMerge, Attempts: 1, OperationID: "soda-x-merge-1", UpdatedUnix: 1200, Work: testMergeIntent("soda-x-merge-1")}
		mutate(&next)
		if err := next.Validate(); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}

func testMergedMerge() Merge {
	m := testMerge()
	now := int64(1200)
	m.Operation = MergeOperation{
		Work: testMergeIntent("soda-1234567890abcdef1234567890abcdef-merge-1"),
		Kind: OpMerge, Attempts: 1, OperationID: "soda-1234567890abcdef1234567890abcdef-merge-1",
		InstallationID: "8e81334d-94c3-4b96-8f34-19f2160ff4f8", ActorID: 8, RepositoryID: 7,
		Effect: OpEffectCommitted, Cancellation: OpCancelNone, Completion: OpCompletionComplete,
		Receipt: `{"pr_number":9}`, UpdatedUnix: now,
		HeadRef: m.HeadRef, BaseRef: m.BaseRef, HeadOID: m.HeadOID, BaseOID: m.BaseOID, MergedCommit: m.HeadOID,
		PRNumber: 9, PRID: 11, IssueID: 13,
	}
	m.NativeRev, m.ObservedUnix = 12, now
	m.Stage, m.Outcome, m.Reason = MergeMerged, Succeeded, MergeReasonMerged
	m.MergedCommit, m.MergedUnix, m.ClosedUnix, m.FinishedUnix = m.HeadOID, now, now, now
	m.OriginalClosingOccurrence, m.OriginalClosedUnix = "901", now
	return m
}

func TestMergeMergedRequiresConfirmedCompletion(t *testing.T) {
	if err := testMergedMerge().Validate(); err != nil {
		t.Fatalf("merged merge refused: %v", err)
	}
	// A committed ref without finished bookkeeping never finishes.
	cases := map[string]func(*Merge){
		"completion":              func(m *Merge) { m.Operation.Completion = OpCompletionPending },
		"stamps":                  func(m *Merge) { m.MergedUnix, m.ClosedUnix = 0, 0 },
		"commit":                  func(m *Merge) { m.MergedCommit = "" },
		"links":                   func(m *Merge) { m.Operation.PRID = 12 },
		"outcome":                 func(m *Merge) { m.Outcome = Failed },
		"reason":                  func(m *Merge) { m.Reason = MergeReasonRefused },
		"original-occurrence":     func(m *Merge) { m.OriginalClosingOccurrence = "" },
		"original-revision-count": func(m *Merge) { m.OriginalClosingOccurrence = "original" },
		"original-closure-time":   func(m *Merge) { m.OriginalClosedUnix = 0 },
	}
	for name, mutate := range cases {
		m := testMergedMerge()
		mutate(&m)
		if err := m.Validate(); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}

func TestMergeTerminalStages(t *testing.T) {
	m := testMerge()
	m.Stage, m.Outcome, m.Reason, m.FinishedUnix = MergeFailed, Failed, MergeReasonRefused, 1200
	if err := m.Validate(); err != nil {
		t.Fatalf("failed merge refused: %v", err)
	}
	m.Stage, m.Outcome, m.Reason = MergeWithdrawn, Cancelled, MergeReasonWithdrawn
	if err := m.Validate(); err != nil {
		t.Fatalf("withdrawn merge refused: %v", err)
	}
	m.Operation = MergeOperation{Kind: OpMerge, Attempts: 1, OperationID: "soda-x-merge-1", UpdatedUnix: 1200, Work: testMergeIntent("soda-x-merge-1"), Effect: OpEffectPending}
	if err := m.Validate(); err == nil {
		t.Fatal("withdrawn merge with an unresolved effect accepted")
	}
	m.Operation.Effect = OpEffectNotCommitted
	if err := m.Validate(); err != nil {
		t.Fatalf("withdrawn merge with a resolved refusal refused: %v", err)
	}
	m = testMerge()
	m.Stage, m.Outcome, m.Reason, m.FinishedUnix = MergeFenced, NeedsHuman, MergeReasonFenced, 1200
	if err := m.Validate(); err != nil {
		t.Fatalf("fenced merge refused: %v", err)
	}
	m.Reason = MergeReasonMerged
	if err := m.Validate(); err == nil {
		t.Fatal("unfinished merge claims completed linkage")
	}
}

func testMergePolicy() RepositoryPolicy {
	return RepositoryPolicy{
		Repository: 7, GrantedBy: 5, Enabled: true, TargetBranch: "refs/heads/main",
		Roles:  map[string]RoleSelection{"soda-coder": {Harness: project.FactoryHarnessCodex, HarnessVers: "1.0.0", Model: "m"}, "soda-reviewer": {Harness: project.FactoryHarnessCodex, HarnessVers: "1.0.0", Model: "m"}},
		Checks: []string{"verify"}, MergeMethod: MergeFastForward,
		Publish:       ActorBindingRef{TokenID: 1, ActorID: 5, Kind: OpRefPublish},
		Create:        ActorBindingRef{TokenID: 1, ActorID: 5, Kind: OpPRCreate},
		Review:        ActorBindingRef{TokenID: 2, ActorID: 6, Kind: OpReviewSubmit},
		Merge:         ActorBindingRef{TokenID: 3, ActorID: 8, Kind: OpMerge},
		AttemptLimits: DefaultAttemptLimits(),
		MaxConcurrent: 2,
	}
}

func testMergeAssessment() CheckAssessment {
	return CheckAssessment{
		Results: []CheckResult{{Context: "verify", State: CheckStateSuccess, Passed: true}},
		Checks:  []string{"verify"}, Repository: 7, PRNumber: 9, PRID: 11, IssueID: 13,
		PolicyRevision: 0, NativeRev: 12, Revision: 2, AssessedUnix: 1150, ObservedContexts: 1,
		HeadRef: "refs/heads/soda/factory/0123456789abcdef0123456789abcdef", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		ChecksDigest: ChecksDigest([]string{"verify"}), Verdict: CheckPass, Reason: CheckReasonPass,
	}
}

func TestVerifyMergeCheckEvidence(t *testing.T) {
	w, policy, assessment := testMergeWork(), testMergePolicy(), testMergeAssessment()
	if err := VerifyMergeCheckEvidence(w, assessment, policy); err != nil {
		t.Fatalf("exact evidence refused: %v", err)
	}
	w.AssessmentRevision = 0
	if err := VerifyMergeCheckEvidence(w, assessment, policy); err != nil {
		t.Fatalf("unbound evidence refused: %v", err)
	}
	cases := map[string]func(*MergeWork, *CheckAssessment, *RepositoryPolicy){
		"verdict": func(_ *MergeWork, a *CheckAssessment, _ *RepositoryPolicy) {
			a.Verdict, a.Reason = CheckPending, CheckReasonPending
		},
		"head":     func(_ *MergeWork, a *CheckAssessment, _ *RepositoryPolicy) { a.HeadOID = strings.Repeat("c", 40) },
		"base":     func(_ *MergeWork, a *CheckAssessment, _ *RepositoryPolicy) { a.BaseOID = strings.Repeat("c", 40) },
		"pr":       func(_ *MergeWork, a *CheckAssessment, _ *RepositoryPolicy) { a.PRNumber = 10 },
		"policy":   func(_ *MergeWork, _ *CheckAssessment, p *RepositoryPolicy) { p.Revision = 1 },
		"checks":   func(_ *MergeWork, _ *CheckAssessment, p *RepositoryPolicy) { p.Checks = []string{"verify", "extra"} },
		"revision": func(w *MergeWork, _ *CheckAssessment, _ *RepositoryPolicy) { w.AssessmentRevision = 3 },
		"target":   func(w *MergeWork, _ *CheckAssessment, _ *RepositoryPolicy) { w.ReviewerID = w.PRAuthorID },
	}
	for name, mutate := range cases {
		w, policy, assessment := testMergeWork(), testMergePolicy(), testMergeAssessment()
		mutate(&w, &assessment, &policy)
		if err := VerifyMergeCheckEvidence(w, assessment, policy); err == nil {
			t.Errorf("case %s accepted", name)
		}
	}
}

func TestMergeTargetChanged(t *testing.T) {
	m := testMerge()
	p := testPublication()
	p.PRCreate.HeadRef, p.PRCreate.BaseRef, p.PRCreate.HeadOID, p.PRCreate.BaseOID = m.HeadRef, m.BaseRef, m.HeadOID, m.BaseOID
	p.Repository, p.Issue, p.PRNumber, p.PRID = m.Repository, m.Issue, m.PRNumber, m.PRID
	if MergeTargetChanged(m, p) {
		t.Fatal("identical target reported changed")
	}
	p.Candidate = strings.Repeat("c", 40)
	if !MergeTargetChanged(m, p) {
		t.Fatal("moved candidate reported unchanged")
	}
}

func testMergeGeneration(publicationID string, n int, head string) MergeGeneration {
	return MergeGeneration{
		Authority:          AuthorityRef{Policy: 3, Operator: 1, Capacity: 1, Sponsorship: 1, ConnectionUsageBudget: 1, Environment: 1},
		PublicationID:      publicationID,
		DecisionID:         strings.Repeat("d", 32),
		NativeOperationID:  MergeGenerationOperationID(publicationID, n),
		Fingerprint:        strings.Repeat("e", 64),
		Head:               head,
		Base:               strings.Repeat("1", 40),
		DiffBase:           strings.Repeat("1", 40),
		ChecksDigest:       strings.Repeat("c", 64),
		Generation:         n,
		ReviewRound:        1,
		AssessmentRevision: 2,
		ObservedNativeRev:  12,
		NotAfter:           1300,
	}
}

func testReadyDecision(fingerprint string) DeliveryDecision {
	answer := testDeliveryAnswer(fingerprint)
	return DeliveryDecision{Answer: answer, Effective: DeliveryEffectiveReady, ReviewsFit: true}
}

func TestMergeLedgerReplacementAuthorization(t *testing.T) {
	publicationID := strings.Repeat("f", 32)
	ledger, err := NewMergeLedger(publicationID, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	head1 := strings.Repeat("2", 40)
	gen1 := testMergeGeneration(publicationID, 1, head1)
	if err := gen1.Validate(); err != nil {
		t.Fatalf("generation refused: %v", err)
	}
	if got := MergeGenerationOperationID(publicationID, 1); gen1.NativeOperationID != got {
		t.Fatalf("operation identity %q differs from derivation", gen1.NativeOperationID)
	}
	ledger, err = ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerRegister, Generation: gen1})
	if err != nil {
		t.Fatalf("first registration refused: %v", err)
	}
	// Persist intent, invalidate, then replace only on proven effect.
	gen2 := testMergeGeneration(publicationID, 2, strings.Repeat("3", 40))
	if _, err := ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerRegister, Generation: gen2}); err == nil {
		t.Fatal("second registration occupied an authorized ledger")
	}
	uncertain := MergePredecessorProof{Generation: 1, Effect: OpEffectPending}
	if _, err := ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerSupersede, Proof: uncertain}); err == nil {
		t.Fatal("uncertain predecessor authorized replacement")
	}
	mismatched := MergePredecessorProof{Generation: 2, Effect: OpEffectNotCommitted, Cancellation: OpCancelCancelled}
	if _, err := ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerSupersede, Proof: mismatched}); err == nil {
		t.Fatal("supersede proof for another generation accepted")
	}
	cancelled := MergePredecessorProof{Generation: 1, Effect: OpEffectNotCommitted, Cancellation: OpCancelCancelled}
	ledger, err = ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerSupersede, Proof: cancelled})
	if err != nil {
		t.Fatalf("confirmed cancel refused replacement: %v", err)
	}
	ledger, err = ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerRegister, Generation: gen2})
	if err != nil {
		t.Fatalf("new review and judgment generation refused: %v", err)
	}
	// The common decision gate admits the fresh generation on current
	// ready_to_merge over its exact fingerprint with confirmed resolutions.
	authority := AuthorityRef{Policy: 3, Operator: 1, Capacity: 1, Sponsorship: 1, ConnectionUsageBudget: 1, Environment: 1}
	gate := MergeDecisionInput{Generation: gen2, Decision: testReadyDecision(gen2.Fingerprint), CurrentAuthority: authority, ResolutionsConfirmed: true}
	if err := CheckMergeDecisionGate(gate); err != nil {
		t.Fatalf("fresh generation gate refused: %v", err)
	}
	unconfirmed := gate
	unconfirmed.ResolutionsConfirmed = false
	if err := CheckMergeDecisionGate(unconfirmed); err == nil {
		t.Fatal("unconfirmed resolutions passed the gate")
	}
	moved := gate
	moved.CurrentAuthority = AuthorityRef{Policy: 4, Operator: 1, Capacity: 1, Sponsorship: 1, ConnectionUsageBudget: 1, Environment: 1}
	if err := CheckMergeDecisionGate(moved); err == nil {
		t.Fatal("changed authority passed the gate")
	}
	staleEvidence := gate
	staleEvidence.Decision = testReadyDecision(strings.Repeat("0", 64))
	if err := CheckMergeDecisionGate(staleEvidence); err == nil {
		t.Fatal("stale-evidence decision passed the gate")
	}
	reviewAgain := gate
	again := testDeliveryAnswer(gen2.Fingerprint)
	again.Selected = DeliveryReviewAgain
	again.Probabilities = map[string]float64{DeliveryReviewAgain: 0.9, DeliveryReadyToMerge: 0.1}
	reviewAgain.Decision = DeliveryDecision{Answer: again, Effective: DeliveryEffectiveReview, ReviewsFit: true}
	if err := CheckMergeDecisionGate(reviewAgain); err == nil {
		t.Fatal("review_again decision passed the merge gate")
	}
	// Base advance invalidates the pending acceptance.
	if !gen2.SupersededByBase(strings.Repeat("4", 40)) {
		t.Fatal("advanced base kept its acceptance")
	}
	if gen2.SupersededByBase(gen2.Base) {
		t.Fatal("unchanged base invalidated its acceptance")
	}
	// A committed predecessor is adopted, never replaced.
	ledger, err = ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerAdopt, AdoptedCommit: gen2.Head})
	if err != nil {
		t.Fatalf("committed merge not adopted: %v", err)
	}
	replayed, err := ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerAdopt, AdoptedCommit: gen2.Head})
	if err != nil {
		t.Fatalf("identical adoption replay refused: %v", err)
	}
	if !replayed.Adopted || replayed.AdoptedCommit != gen2.Head {
		t.Fatal("adoption replay changed its ledger")
	}
	gen3 := testMergeGeneration(publicationID, 3, strings.Repeat("5", 40))
	if _, err := ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerRegister, Generation: gen3}); err == nil {
		t.Fatal("committed predecessor replaced")
	}
}

func TestMergeLedgerHistoryCeiling(t *testing.T) {
	publicationID := strings.Repeat("f", 32)
	ledger, err := NewMergeLedger(publicationID, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	cancelled := func(n int) MergePredecessorProof {
		return MergePredecessorProof{Generation: n, Effect: OpEffectNotCommitted, Cancellation: OpCancelCancelled}
	}
	for n := 1; n <= MaxMergeGenerations; n++ {
		head := fmt.Sprintf("%040x", 1000+n)
		if ledger, err = ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerRegister, Generation: testMergeGeneration(publicationID, n, head)}); err != nil {
			t.Fatalf("generation %d refused: %v", n, err)
		}
		if ledger, err = ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerSupersede, Proof: cancelled(n)}); err != nil {
			t.Fatalf("supersede %d refused: %v", n, err)
		}
	}
	if len(ledger.History) != MaxMergeGenerations {
		t.Fatalf("history=%d, want ceiling", len(ledger.History))
	}
	overflow := testMergeGeneration(publicationID, MaxMergeGenerations+1, strings.Repeat("8", 40))
	if _, err := ledger.Apply(MergeLedgerUpdate{Kind: MergeLedgerRegister, Generation: overflow}); err == nil {
		t.Fatal("generation above the history ceiling accepted")
	}
}
