package factory

import (
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
	return m
}

func TestMergeMergedRequiresConfirmedCompletion(t *testing.T) {
	if err := testMergedMerge().Validate(); err != nil {
		t.Fatalf("merged merge refused: %v", err)
	}
	// A committed ref without finished bookkeeping never finishes.
	cases := map[string]func(*Merge){
		"completion": func(m *Merge) { m.Operation.Completion = OpCompletionPending },
		"stamps":     func(m *Merge) { m.MergedUnix, m.ClosedUnix = 0, 0 },
		"commit":     func(m *Merge) { m.MergedCommit = "" },
		"links":      func(m *Merge) { m.Operation.PRID = 12 },
		"outcome":    func(m *Merge) { m.Outcome = Failed },
		"reason":     func(m *Merge) { m.Reason = MergeReasonRefused },
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
