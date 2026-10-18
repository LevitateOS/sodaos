package control

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func mergeGateTestHeads() (head, base, candidate string) {
	return strings.Repeat("c", 40), strings.Repeat("b", 40), strings.Repeat("d", 40)
}

func mergeGateTestPublication(head, base, candidate string) factory.Publication {
	publicationID := strings.Repeat("9", 32)
	return factory.Publication{
		ID: publicationID, Stage: factory.PublicationPublished,
		Candidate: candidate, BaseSHA: base,
		PRCreate: factory.PublicationOperation{
			Work:    &factory.PublicationIntent{Candidate: head},
			Effect:  factory.OpEffectCommitted,
			BaseOID: base,
		},
		PRNumber: 3, PRID: 5, Repository: 7, Issue: 11,
	}
}

func mergeGateTestSetup(t *testing.T) (MergeJudgmentGateInput, factory.DeliveryFingerprintInput, factory.AuthorityRef) {
	t.Helper()
	_, base, candidate := mergeGateTestHeads()
	head := candidate
	publicationID := strings.Repeat("9", 32)
	finding := correctionTestFinding(1, true, factory.FindingFixVerification)
	digest := correctionTestDigest(t, finding.Key)
	cycle := correctionTestCycle(t, publicationID, head, base, 1, digest)
	report := correctionTestReport(candidate, []factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionFixed}})
	checks := correctionTestChecks(candidate, base)
	admission, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), candidate, report, nil, checks)
	if err != nil {
		t.Fatal(err)
	}
	publication := mergeGateTestPublication(head, base, candidate)
	publication.ID = publicationID
	// The admitted correction answers the recorded cycle on the current
	// head; the publication head follows the committed correction.
	mappings := []factory.ReviewCommentMapping{{Key: finding.Key, CommentID: 11}}
	threadDigest := strings.Repeat("a", 64)
	threadEvidence := map[int64]factory.ThreadEvidence{
		11: {Digest: threadDigest, Repository: 7, PRNumber: 3, RootCommentID: 11},
	}
	binding := factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpThreadResolve}
	planned, err := PlanAcceptedResolutions(publication, admission, mappings, threadEvidence, binding, 1,
		"refs/heads/fix", "refs/heads/main", candidate, base, 7, 3, 5)
	if err != nil {
		t.Fatal(err)
	}
	acceptance := JudgmentAcceptance{DecisionID: strings.Repeat("d", 32), FindingDigest: admission.SetDigest}
	confirmed := map[string]bool{finding.Key: true}
	authority := factory.AuthorityRef{Policy: 3, Operator: 1, Capacity: 1, Sponsorship: 1, ConnectionUsageBudget: 1, Environment: 1}
	fpIn := factory.DeliveryFingerprintInput{
		Reviews: []string{strings.Repeat("1", 32)}, Conversations: []string{threadDigest},
		AttemptRoot: strings.Repeat("3", 32), PublicationID: publicationID,
		ScopeDigest: digest, FixDigest: strings.Repeat("6", 64), DiffDigest: strings.Repeat("7", 64),
		CheckDigest: strings.Repeat("8", 64), TreeDigest: strings.Repeat("9", 64),
		Head: candidate, Base: base, Model: "jev-test",
		Rubric:         "ready_to_merge only with full evidence cover",
		PolicyRevision: 3, Round: 1,
	}
	fingerprint, err := factory.DeliveryFingerprint(fpIn)
	if err != nil {
		t.Fatal(err)
	}
	generation := factory.MergeGeneration{
		Authority: authority, PublicationID: publicationID,
		DecisionID: strings.Repeat("d", 32), NativeOperationID: factory.MergeGenerationOperationID(publicationID, 1),
		Fingerprint: fingerprint, Head: candidate, Base: base, DiffBase: base,
		ChecksDigest:       strings.Repeat("c", 64),
		Generation:         1,
		ReviewRound:        1,
		AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
	}
	in := MergeJudgmentGateInput{
		Publication: publication, Admission: admission, Cycle: cycle,
		Acceptance: acceptance, Planned: planned, Confirmed: confirmed,
		Generation: generation, Native: mergeableGate(candidate, base),
	}
	return in, fpIn, authority
}

func mergeGateTestAdvice(t *testing.T, fpIn factory.DeliveryFingerprintInput, answer func(string) (factory.DeliveryAnswer, error)) (DeliveryAdvice, *deliveryChooser, *deliveryFunding) {
	t.Helper()
	chooser := &deliveryChooser{answer: answer}
	funding := newDeliveryFunding(8, 8)
	advice := DeliveryAdvice{
		Input: fpIn, Repository: 7, Issue: 11,
		AttemptRoot: strings.Repeat("3", 32), RequestID: strings.Repeat("e", 32),
		ReviewsFit: true, ObservedNativeRevision: 12,
		Chooser: chooser, Funding: funding,
	}
	return advice, chooser, funding
}

func TestMergeJudgmentGateAdmitsEligibleMergeable(t *testing.T) {
	in, fpIn, authority := mergeGateTestSetup(t)
	if err := CheckMergeJudgmentReady(in); err != nil {
		t.Fatalf("eligible mergeable refused: %v", err)
	}
	coord := &Coordinator{}
	advice, chooser, _ := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
	decision, reused, err := coord.AdviseMergeJudgment(context.Background(), in, advice, authority)
	if err != nil {
		t.Fatalf("funded merge judgment refused: %v", err)
	}
	if reused || chooser.calls != 1 {
		t.Fatalf("funded judgment did not spend exactly once: reused=%v calls=%d", reused, chooser.calls)
	}
	if err := factory.CheckMergeDecisionGate(factory.MergeDecisionInput{
		Generation: in.Generation, Decision: decision,
		CurrentAuthority: authority, ResolutionsConfirmed: true,
	}); err != nil {
		t.Fatalf("common merge gate refused funded ready: %v", err)
	}
}

func TestMergeJudgmentGateRefusesSupersededBeforeSpend(t *testing.T) {
	in, fpIn, authority := mergeGateTestSetup(t)
	in.Native = mergeableGate(strings.Repeat("e", 40), in.Generation.Base)
	if err := CheckMergeJudgmentReady(in); err == nil {
		t.Fatal("superseded native head passed its gate")
	}
	coord := &Coordinator{}
	advice, chooser, funding := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), in, advice, authority); err == nil {
		t.Fatal("superseded merge spent its provider call")
	}
	if chooser.calls != 0 || funding.reserveCalls != 0 {
		t.Fatalf("superseded merge spent: calls=%d reserves=%d", chooser.calls, funding.reserveCalls)
	}
	stale := in
	head, _, _ := mergeGateTestHeads()
	stale.Generation.Head = head
	stale.Native = mergeableGate(head, in.Generation.Base)
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), stale, advice, authority); err == nil {
		t.Fatal("stale generation spent its provider call")
	}
	if chooser.calls != 0 || funding.reserveCalls != 0 {
		t.Fatalf("stale generation spent: calls=%d reserves=%d", chooser.calls, funding.reserveCalls)
	}
}

func TestMergeJudgmentGateRefusesProtectedBeforeSpend(t *testing.T) {
	for _, blocker := range []string{"protected_files", "branch_protection", "permission", "merged", "closed"} {
		in, fpIn, authority := mergeGateTestSetup(t)
		in.Native = blockedGate(in.Generation.Head, in.Generation.Base, blocker, true)
		err := CheckMergeJudgmentReady(in)
		var refused *factory.PublicationRefusal
		if !errors.As(err, &refused) || refused.Reason != blocker {
			t.Fatalf("blocker %q gave %v, want refusal %q", blocker, err, blocker)
		}
		coord := &Coordinator{}
		advice, chooser, funding := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
		if _, _, err := coord.AdviseMergeJudgment(context.Background(), in, advice, authority); err == nil {
			t.Fatalf("protected blocker %q spent its provider call", blocker)
		}
		if chooser.calls != 0 || funding.reserveCalls != 0 {
			t.Fatalf("protected blocker %q spent: calls=%d reserves=%d", blocker, chooser.calls, funding.reserveCalls)
		}
	}
}

func TestMergeJudgmentGateWaitsTransientBlocksBeforeSpend(t *testing.T) {
	for _, blocker := range []string{"checking", "status_checks", "approvals", "review_requests", "dependency", "outdated_branch", "not_mergeable"} {
		in, fpIn, authority := mergeGateTestSetup(t)
		in.Native = blockedGate(in.Generation.Head, in.Generation.Base, blocker, true)
		err := CheckMergeJudgmentReady(in)
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) || wait.Reason != blocker {
			t.Fatalf("blocker %q gave %v, want wait %q", blocker, err, blocker)
		}
		coord := &Coordinator{}
		advice, chooser, funding := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
		if _, _, err := coord.AdviseMergeJudgment(context.Background(), in, advice, authority); err == nil {
			t.Fatalf("transient blocker %q spent its provider call", blocker)
		}
		if chooser.calls != 0 || funding.reserveCalls != 0 {
			t.Fatalf("transient blocker %q spent: calls=%d reserves=%d", blocker, chooser.calls, funding.reserveCalls)
		}
	}
}

func TestMergeJudgmentGateWaitsReviewAndCorrectionPaths(t *testing.T) {
	in, fpIn, authority := mergeGateTestSetup(t)
	coord := &Coordinator{}
	missing := in
	missing.Native.Approved = false
	var wait *factory.PublicationWait
	if err := CheckMergeJudgmentReady(missing); !errors.As(err, &wait) || wait.Reason != "approval_missing" {
		t.Fatalf("missing approval gave %v, want approval_missing wait", err)
	}
	advice, chooser, funding := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), missing, advice, authority); err == nil {
		t.Fatal("missing approval spent its provider call")
	}
	if chooser.calls != 0 || funding.reserveCalls != 0 {
		t.Fatalf("missing approval spent: calls=%d reserves=%d", chooser.calls, funding.reserveCalls)
	}
	requested := in
	requested.Native.ChangesRequested = true
	if err := CheckMergeJudgmentReady(requested); !errors.As(err, &wait) || wait.Reason != "changes_requested" {
		t.Fatalf("requested changes gave %v, want changes_requested wait", err)
	}
	unresolved := in
	unresolved.Admission.Unresolved = []string{unresolved.Admission.Fixed[0]}
	if err := CheckMergeJudgmentReady(unresolved); !errors.As(err, &wait) || wait.Reason != "correction_pending" {
		t.Fatalf("unresolved fix gave %v, want correction_pending wait", err)
	}
	pending := in
	pending.Confirmed = map[string]bool{}
	if err := CheckMergeJudgmentReady(pending); !errors.As(err, &wait) || wait.Reason != "resolution_pending" {
		t.Fatalf("unconfirmed resolution gave %v, want resolution_pending wait", err)
	}
	staleHead := in
	staleHead.Admission.NewCandidate = strings.Repeat("f", 40)
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), staleHead, advice, authority); err == nil {
		t.Fatal("superseded correction spent its provider call")
	}
	if chooser.calls != 0 || funding.reserveCalls != 0 {
		t.Fatalf("superseded correction spent: calls=%d reserves=%d", chooser.calls, funding.reserveCalls)
	}
}

func TestMergeJudgmentGateRefusesStaleDigestAndFingerprint(t *testing.T) {
	in, fpIn, authority := mergeGateTestSetup(t)
	coord := &Coordinator{}
	advice, chooser, funding := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
	stale := in
	stale.Acceptance = JudgmentAcceptance{DecisionID: strings.Repeat("d", 32), FindingDigest: strings.Repeat("0", 64)}
	if err := CheckMergeJudgmentReady(stale); err == nil {
		t.Fatal("stale acceptance passed its gate")
	}
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), stale, advice, authority); err == nil {
		t.Fatal("stale acceptance spent its provider call")
	}
	if chooser.calls != 0 || funding.reserveCalls != 0 {
		t.Fatalf("stale acceptance spent: calls=%d reserves=%d", chooser.calls, funding.reserveCalls)
	}
	mismatchedIn := fpIn
	mismatchedIn.Head = strings.Repeat("e", 40)
	mismatchedAdvice, chooser2, funding2 := mergeGateTestAdvice(t, mismatchedIn, readyDeliveryAnswer)
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), in, mismatchedAdvice, authority); err == nil {
		t.Fatal("fingerprint mismatch spent its provider call")
	}
	if chooser2.calls != 0 || funding2.reserveCalls != 0 {
		t.Fatalf("fingerprint mismatch spent: calls=%d reserves=%d", chooser2.calls, funding2.reserveCalls)
	}
}

// gateTestMergeRow seeds one open merge row directly through the store,
// bypassing the publication flow: the FW22 authorization tests need a
// recorded row, not a full publish journey.
func gateTestMergeRow(t *testing.T, db *store.Store) factory.Merge {
	t.Helper()
	m := factory.Merge{
		Authority: factory.AuthorityRef{Policy: 2, Operator: 1, Capacity: 1, Sponsorship: 1, ConnectionUsageBudget: 1, Environment: 1},
		Operation: factory.MergeOperation{Kind: factory.OpMerge},
		ID:        factory.NewID(), PublicationID: factory.NewID(), AssignmentID: factory.NewID(),
		ProjectID: "p" + strings.Repeat("d", 24),
		Role:      project.RoleCoder, Acceptance: "d" + strings.Repeat("a", 24),
		HeadRef: "refs/heads/soda/factory/candidate", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		Stage:      factory.MergeOpen,
		Repository: 7, Issue: 3, PRNumber: 9, PRID: 11, IssueID: 13,
		PRAuthorID: 5, ReviewerID: 6, CreatedUnix: 1100,
	}
	if err := db.RecordMerge(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	return m
}

// gateTestBinding builds a ready first-generation test authorization for
// the row's exact current head and base. It stands in for the future Jev
// producer through the same decision gate production enforces.
func gateTestBinding(t *testing.T, m factory.Merge) factory.MergeAuthorization {
	t.Helper()
	fingerprint := strings.Repeat("e", 64)
	answer, err := readyDeliveryAnswer(fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	return factory.MergeAuthorization{
		Generation: factory.MergeGeneration{
			Authority: m.Authority, PublicationID: m.PublicationID,
			DecisionID:        strings.Repeat("d", 32),
			NativeOperationID: factory.MergeGenerationOperationID(m.PublicationID, 1),
			Fingerprint:       fingerprint, Head: m.HeadOID, Base: m.BaseOID, DiffBase: m.BaseOID,
			ChecksDigest:       strings.Repeat("c", 64),
			Generation:         1,
			ReviewRound:        1,
			AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
		},
		Decision:             factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true},
		ResolutionsConfirmed: true,
	}
}

// TestMergeJudgmentGateReusesUnchangedDecisionWithoutNewSpend (T25):
// timestamps and native-revision-only changes never invalidate the
// acceptance, and the exact expected-resolution readback buys no paid
// call: the identical second judgment reuses without spending.
func TestMergeJudgmentGateReusesUnchangedDecisionWithoutNewSpend(t *testing.T) {
	in, fpIn, authority := mergeGateTestSetup(t)
	coord := &Coordinator{}
	advice, chooser, funding := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
	if _, reused, err := coord.AdviseMergeJudgment(context.Background(), in, advice, authority); err != nil || reused {
		t.Fatalf("first judgment refused or reused: %v %v", err, reused)
	}
	if chooser.calls != 1 || funding.reserveCalls != 1 {
		t.Fatalf("first judgment spend wrong: calls=%d reserves=%d", chooser.calls, funding.reserveCalls)
	}
	// A native-revision-only change is not semantic evidence: the
	// fingerprint excludes it, so the judgment reuses without spending.
	replayed, chooser2, _ := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
	replayed.Funding, replayed.ObservedNativeRevision = funding, 99
	if _, reused, err := coord.AdviseMergeJudgment(context.Background(), in, replayed, authority); err != nil || !reused {
		t.Fatalf("native-revision-only change refused or respent: %v %v", err, reused)
	}
	if chooser2.calls != 0 {
		t.Fatalf("native-revision-only change consulted again: calls=%d", chooser2.calls)
	}
	// The exact expected-resolution readback advances the receipt without
	// buying another judgment.
	receipt, err := funding.DeliveryJudgment(context.Background(), in.Generation.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	advanced, err := receipt.ConfirmExpectedResolution(100)
	if err != nil {
		t.Fatalf("exact resolution readback refused: %v", err)
	}
	if !advanced.CurrentFor(in.Generation.Fingerprint) {
		t.Fatal("exact resolution readback lost its acceptance")
	}
	if chooser.calls != 1 || chooser2.calls != 0 {
		t.Fatalf("readback spent a funded call: calls=%d replays=%d", chooser.calls, chooser2.calls)
	}
}

// TestMergeJudgmentGateRelevantChangeInvalidatesWithoutReuse (T25): a
// relevant head change invalidates the acceptance: the stale generation
// refuses before any spend, and the old decision is never reused for the
// new evidence. Funding the moved head needs its fresh generation, which
// only a new review and judgment can register.
func TestMergeJudgmentGateRelevantChangeInvalidatesWithoutReuse(t *testing.T) {
	in, fpIn, authority := mergeGateTestSetup(t)
	coord := &Coordinator{}
	advice, _, _ := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), in, advice, authority); err != nil {
		t.Fatal(err)
	}
	movedIn := fpIn
	movedIn.Head = strings.Repeat("f", 40)
	moved := in
	moved.Native = mergeableGate(movedIn.Head, in.Generation.Base)
	movedAdvice, chooser2, funding2 := mergeGateTestAdvice(t, movedIn, readyDeliveryAnswer)
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), moved, movedAdvice, authority); err == nil {
		t.Fatal("relevant head change passed its stale gate")
	}
	if chooser2.calls != 0 || funding2.reserveCalls != 0 {
		t.Fatalf("relevant head change spent: calls=%d reserves=%d", chooser2.calls, funding2.reserveCalls)
	}
}

// TestMergeJudgmentGateResolutionReadbackAddsNoSpend (T25): reconciling
// accepted resolutions against their exact readback is pure bookkeeping;
// it consumes no funded call however often it runs.
func TestMergeJudgmentGateResolutionReadbackAddsNoSpend(t *testing.T) {
	in, fpIn, authority := mergeGateTestSetup(t)
	coord := &Coordinator{}
	advice, chooser, _ := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), in, advice, authority); err != nil {
		t.Fatal(err)
	}
	before := chooser.calls
	for range 3 {
		if err := CheckMergeJudgmentReady(in); err != nil {
			t.Fatalf("funded acceptance refused its readback: %v", err)
		}
		outcome, err := ReconcileAcceptedResolutions(in.Acceptance, in.Admission, in.Planned, in.Confirmed)
		if err != nil || len(outcome.Waits) != 0 {
			t.Fatalf("confirmed resolutions unreconciled: %+v %v", outcome, err)
		}
	}
	if chooser.calls != before {
		t.Fatalf("resolution readback spent funded calls: %d -> %d", before, chooser.calls)
	}
}

// TestMergeJudgmentGateClassifiesEveryNativeBlockerOnce (T26): the read
// agrees with the mutation side exactly once per blocker. Revoked or
// protected merges refuse before any funded call; transient blockers
// wait without spending. No blocker is both, and no known blocker is
// unknown.
func TestMergeJudgmentGateClassifiesEveryNativeBlockerOnce(t *testing.T) {
	protected := []string{"merged", "closed", "permission", "protected_files", "branch_protection", "signature"}
	transient := []string{"work_in_progress", "not_mergeable", "checking", "status_checks", "approvals", "requested_changes", "review_requests", "outdated_branch", "dependency"}
	seen := map[string]bool{}
	for _, blocker := range protected {
		in, fpIn, authority := mergeGateTestSetup(t)
		in.Native = blockedGate(in.Generation.Head, in.Generation.Base, blocker, true)
		err := CheckMergeJudgmentReady(in)
		var refused *factory.PublicationRefusal
		if !errors.As(err, &refused) || refused.Reason != blocker {
			t.Fatalf("blocker %q gave %v, want refusal %q", blocker, err, blocker)
		}
		coord := &Coordinator{}
		advice, chooser, funding := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
		if _, _, err := coord.AdviseMergeJudgment(context.Background(), in, advice, authority); err == nil {
			t.Fatalf("protected blocker %q spent its provider call", blocker)
		}
		if chooser.calls != 0 || funding.reserveCalls != 0 {
			t.Fatalf("protected blocker %q spent: calls=%d reserves=%d", blocker, chooser.calls, funding.reserveCalls)
		}
		seen[blocker] = true
	}
	for _, blocker := range transient {
		in, _, _ := mergeGateTestSetup(t)
		in.Native = blockedGate(in.Generation.Head, in.Generation.Base, blocker, true)
		err := CheckMergeJudgmentReady(in)
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) || wait.Reason != blocker {
			t.Fatalf("blocker %q gave %v, want wait %q", blocker, err, blocker)
		}
		seen[blocker] = true
	}
	if len(seen) != len(protected)+len(transient) {
		t.Fatal("blocker classification overlaps or drops a blocker")
	}
	in, _, _ := mergeGateTestSetup(t)
	in.Native = blockedGate(in.Generation.Head, in.Generation.Base, "made_up_blocker", true)
	err := CheckMergeJudgmentReady(in)
	var wait *factory.PublicationWait
	var refused *factory.PublicationRefusal
	if errors.As(err, &wait) || errors.As(err, &refused) {
		t.Fatalf("unknown blocker classified as gate verdict: %v", err)
	}
	if err == nil {
		t.Fatal("unknown blocker passed its gate")
	}
}

func TestMergeJudgmentGateFailureAndRevokePaths(t *testing.T) {
	in, fpIn, authority := mergeGateTestSetup(t)
	coord := &Coordinator{}
	failing := func(string) (factory.DeliveryAnswer, error) {
		return factory.DeliveryAnswer{}, errors.New("provider-down")
	}
	advice, chooser, _ := mergeGateTestAdvice(t, fpIn, failing)
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), in, advice, authority); !errors.Is(err, ErrDeliveryUnavailable) {
		t.Fatalf("provider failure gave %v, want delivery unavailable", err)
	}
	if chooser.calls != 1 {
		t.Fatalf("provider failure did not spend exactly once: calls=%d", chooser.calls)
	}
	// A reserved fingerprint without a completed answer stays charged:
	// the identical retry reuses no prior decision and spends nothing new.
	advice2, chooser2, _ := mergeGateTestAdvice(t, fpIn, readyDeliveryAnswer)
	advice2.Funding = advice.Funding
	if _, _, err := coord.AdviseMergeJudgment(context.Background(), in, advice2, authority); !errors.Is(err, ErrDeliveryUnavailable) {
		t.Fatalf("charged failure retry gave %v, want delivery unavailable", err)
	}
	if chooser2.calls != 0 {
		t.Fatalf("charged failure retry consulted again: calls=%d", chooser2.calls)
	}
	revoked, _, _ := mergeGateTestSetup(t)
	revoked.Publication.Stage = factory.PublicationWithdrawn
	if err := CheckMergeJudgmentReady(revoked); err == nil {
		t.Fatal("withdrawn publication passed its gate")
	} else {
		var refused *factory.PublicationRefusal
		if !errors.As(err, &refused) {
			t.Fatalf("withdrawn publication gave %v, want refusal", err)
		}
	}
}

// TestMergeAuthorizationGateClosesUnboundSubmits (T27): missing Jev
// acceptance closes every new submit path. An unbound row waits for its
// authorization instead of submitting; the wait names the missing
// judgment honestly.
func TestMergeAuthorizationGateClosesUnboundSubmits(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	m := gateTestMergeRow(t, db)
	if err := mergeAuthorizationGate(m, factory.Publication{}); err == nil {
		t.Fatal("unbound row passed its authorization gate")
	} else {
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) || wait.Reason != "merge_judgment_unavailable" {
			t.Fatalf("unbound row gave %v, want merge_judgment_unavailable wait", err)
		}
	}
}

// TestMergeAuthorizationGateRefusesStaleTuple (T27): a binding never
// follows its candidate. A moved head or base waits for
// re-authorization behind fresh review and judgment instead of merging
// another candidate.
func TestMergeAuthorizationGateRefusesStaleTuple(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	m := gateTestMergeRow(t, db)
	m.Authorization = gateTestBinding(t, m)
	if err := mergeAuthorizationGate(m, factory.Publication{}); err != nil {
		t.Fatalf("fresh binding refused: %v", err)
	}
	moved := m
	moved.HeadOID = strings.Repeat("4", 40)
	if err := mergeAuthorizationGate(moved, factory.Publication{}); err == nil {
		t.Fatal("moved head passed its stale binding")
	} else {
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) || wait.Reason != "superseded_head" {
			t.Fatalf("moved head gave %v, want superseded_head wait", err)
		}
	}
	rebased := m
	rebased.BaseOID = strings.Repeat("5", 40)
	if err := mergeAuthorizationGate(rebased, factory.Publication{}); err == nil {
		t.Fatal("moved base passed its stale binding")
	} else {
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) || wait.Reason != "superseded_base" {
			t.Fatalf("moved base gave %v, want superseded_base wait", err)
		}
	}
}

// TestMergeAuthorizationGateEnforcesDecisionFreshness (T27): a currently
// invalid acceptance never admits a submit. A review_again decision or
// unconfirmed resolutions hold the row instead of merging.
func TestMergeAuthorizationGateEnforcesDecisionFreshness(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	m := gateTestMergeRow(t, db)
	m.Authorization = gateTestBinding(t, m)
	answer, err := reviewDeliveryAnswer(m.Authorization.Generation.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	review := m
	review.Authorization.Decision = factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReview, ReviewsFit: true}
	if err := mergeAuthorizationGate(review, factory.Publication{}); err == nil {
		t.Fatal("review_again decision passed the merge gate")
	} else {
		var wait *factory.PublicationWait
		if !errors.As(err, &wait) || wait.Reason != "merge_authorization_stale" {
			t.Fatalf("review_again decision gave %v, want merge_authorization_stale wait", err)
		}
	}
	pending := m
	pending.Authorization.ResolutionsConfirmed = false
	if err := mergeAuthorizationGate(pending, factory.Publication{}); err == nil {
		t.Fatal("unconfirmed resolutions passed the merge gate")
	}
}

// TestAuthorizeMergeBindsFirstGeneration (T27/T28 under the FW37 union):
// the registration seam admits exactly one ready union-bound
// first-generation authorization per row. A second bind, a second
// generation, another candidate, a changed authority, an unready
// decision, unconfirmed resolutions and a binding without its union all
// refuse; nothing replaces a bound predecessor on its row.
func TestAuthorizeMergeBindsFirstGeneration(t *testing.T) {
	ctx := context.Background()
	t.Run("binds ready first generation", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 51)
		auth := unionAuthForBind(t, m, acceptance, decision, fingerprint, true)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err != nil {
			t.Fatalf("ready first generation refused: %v", err)
		}
		stored, err := fx.publish.db.MergeByPublication(ctx, p.ID)
		if err != nil || !stored.Authorization.Bound() || stored.Authorization.Generation.Generation != 1 {
			t.Fatalf("binding not persisted: %+v %v", stored, err)
		}
		fresh, err := fx.publish.db.PublicationByAssignment(ctx, p.AssignmentID)
		if err != nil {
			t.Fatal(err)
		}
		if err := mergeAuthorizationGate(stored, fresh); err != nil {
			t.Fatalf("bound row refused its submit gate: %v", err)
		}
	})
	t.Run("never rebinds", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 52)
		auth := unionAuthForBind(t, m, acceptance, decision, fingerprint, true)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err != nil {
			t.Fatal(err)
		}
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err == nil {
			t.Fatal("second binding replaced its predecessor")
		}
	})
	t.Run("refuses second generation without ledger history", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 53)
		auth := unionAuthForBind(t, m, acceptance, decision, fingerprint, true)
		auth.Generation.Generation = 2
		auth.Generation.NativeOperationID = factory.MergeGenerationOperationID(p.ID, 2)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err == nil {
			t.Fatal("second generation bound without its predecessor history")
		}
	})
	t.Run("refuses another candidate", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 54)
		auth := unionAuthForBind(t, m, acceptance, decision, fingerprint, true)
		auth.Generation.Head = strings.Repeat("4", 40)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err == nil {
			t.Fatal("binding for another candidate accepted")
		}
	})
	t.Run("refuses changed authority", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 55)
		auth := unionAuthForBind(t, m, acceptance, decision, fingerprint, true)
		auth.Generation.Authority.Policy = 9
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err == nil {
			t.Fatal("binding under changed authority accepted")
		}
	})
	t.Run("refuses unready decision and unconfirmed resolutions", func(t *testing.T) {
		fx, p, m, acceptance, decision, fingerprint := unionSeedForBind(t, ctx, 56)
		_ = decision
		auth := unionAuthForBind(t, m, acceptance, factory.DeliveryDecision{}, fingerprint, true)
		answer, err := reviewDeliveryAnswer(fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		auth.Decision = factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReview, ReviewsFit: true}
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err == nil {
			t.Fatal("review_again decision bound")
		}
		auth = unionAuthForBind(t, m, acceptance, factory.DeliveryDecision{}, fingerprint, true)
		answer, err = readyDeliveryAnswer(fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		auth.Decision = factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true}
		auth.ResolutionsConfirmed = false
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err == nil {
			t.Fatal("unconfirmed resolutions bound")
		}
	})
	t.Run("refuses a binding without its union", func(t *testing.T) {
		fx, p, m, _, _, _ := unionSeedForBind(t, ctx, 57)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, gateTestBinding(t, m)); err == nil {
			t.Fatal("unionless binding registered")
		}
	})
}

// TestWithdrawUnboundRowCancelsWithoutGate (T27/T28): cancellation is
// the escape hatch, never an admission. Withdrawing an unbound,
// undriven row finishes it without consulting any judgment; the
// confirmed cancel is what a future replacement must prove.
func TestWithdrawUnboundRowCancelsWithoutGate(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	m := gateTestMergeRow(t, db)
	coord := &Coordinator{Store: db}
	report := &MergeReport{Merged: []MergeLink{}}
	coord.withdrawMerge(ctx, &m, report)
	if len(report.Errors) != 0 {
		t.Fatalf("withdrawal errored: %+v", report)
	}
	if len(report.Waits) != 0 {
		t.Fatalf("withdrawal waited on judgment: %+v", report)
	}
	stored, err := db.MergeByPublication(ctx, m.PublicationID)
	if err != nil || stored.Stage != factory.MergeWithdrawn || stored.Outcome != factory.Cancelled {
		t.Fatalf("unbound row not withdrawn: %+v %v", stored, err)
	}
}

// TestMergeLedgerNoEffectRefusalReplacesSamePR (T28): a deterministic
// no-effect refusal proves replaceability like a confirmed cancel does.
// The new review and judgment generation merges the same PR under the
// common gate, while an uncertain or committed predecessor forbids any
// replacement.
func TestMergeLedgerNoEffectRefusalReplacesSamePR(t *testing.T) {
	publicationID := strings.Repeat("a", 32)
	ledger, err := factory.NewMergeLedger(publicationID, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	head1 := strings.Repeat("2", 40)
	gen1 := gateTestLedgerGeneration(publicationID, 1, head1)
	ledger, err = ledger.Apply(factory.MergeLedgerUpdate{Kind: factory.MergeLedgerRegister, Generation: gen1})
	if err != nil {
		t.Fatalf("first registration refused: %v", err)
	}
	indeterminate := factory.MergePredecessorProof{Generation: 1, Effect: factory.OpEffectIndeterminate}
	if _, err := ledger.Apply(factory.MergeLedgerUpdate{Kind: factory.MergeLedgerSupersede, Proof: indeterminate}); err == nil {
		t.Fatal("indeterminate predecessor authorized replacement")
	}
	committed := factory.MergePredecessorProof{Generation: 1, Effect: factory.OpEffectCommitted}
	if _, err := ledger.Apply(factory.MergeLedgerUpdate{Kind: factory.MergeLedgerSupersede, Proof: committed}); err == nil {
		t.Fatal("committed predecessor authorized replacement")
	}
	refused := factory.MergePredecessorProof{Generation: 1, Effect: factory.OpEffectNotCommitted, NoEffectRefusal: "native_refused"}
	ledger, err = ledger.Apply(factory.MergeLedgerUpdate{Kind: factory.MergeLedgerSupersede, Proof: refused})
	if err != nil {
		t.Fatalf("deterministic refusal forbade replacement: %v", err)
	}
	head2 := strings.Repeat("3", 40)
	gen2 := gateTestLedgerGeneration(publicationID, 2, head2)
	ledger, err = ledger.Apply(factory.MergeLedgerUpdate{Kind: factory.MergeLedgerRegister, Generation: gen2})
	if err != nil {
		t.Fatalf("refused predecessor forbade its same-PR successor: %v", err)
	}
	authority := factory.AuthorityRef{Policy: 2, Operator: 1, Capacity: 1, Sponsorship: 1, ConnectionUsageBudget: 1, Environment: 1}
	answer, err := readyDeliveryAnswer(gen2.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	gate := factory.MergeDecisionInput{
		Generation:       gen2,
		Decision:         factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true},
		CurrentAuthority: authority, ResolutionsConfirmed: true,
	}
	if err := factory.CheckMergeDecisionGate(gate); err != nil {
		t.Fatalf("successor generation refused its gate: %v", err)
	}
	if ledger.Current.Generation != 2 || ledger.Current.Head != head2 {
		t.Fatalf("successor generation not current: %+v", ledger.Current)
	}
}

// gateTestLedgerGeneration builds a valid ledger generation for the
// pure replacement tests.
func gateTestLedgerGeneration(publicationID string, n int, head string) factory.MergeGeneration {
	return factory.MergeGeneration{
		Authority:          factory.AuthorityRef{Policy: 2, Operator: 1, Capacity: 1, Sponsorship: 1, ConnectionUsageBudget: 1, Environment: 1},
		PublicationID:      publicationID,
		DecisionID:         strings.Repeat("d", 32),
		NativeOperationID:  factory.MergeGenerationOperationID(publicationID, n),
		Fingerprint:        strings.Repeat("e", 64),
		Head:               head,
		Base:               strings.Repeat("1", 40),
		DiffBase:           strings.Repeat("1", 40),
		ChecksDigest:       strings.Repeat("c", 64),
		Generation:         n,
		ReviewRound:        1,
		AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
	}
}

// TestAdoptCommittedMergeAfterLostReplyWithoutRewrite (T29): a recorded
// committed merge is adopted on its own identity after a lost reply. No
// observation rewrites a committed effect, flips it, or attributes it
// to another installation, actor or operation.
func TestAdoptCommittedMergeAfterLostReplyWithoutRewrite(t *testing.T) {
	work := &factory.MergeIntent{
		OperationID: factory.MergeOperationID(strings.Repeat("9", 32), 1), AuthRevision: "accepted-revision",
		HeadRef: "refs/heads/soda/factory/candidate", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("2", 40), BaseOID: strings.Repeat("1", 40),
		Repository: 7, ActorID: 8, PRNumber: 9, PRID: 11, IssueID: 13,
		PRAuthorID: 5, ReviewerID: 6,
		NativeRev: 12, NotAfter: 1300, AssessmentRevision: 2, ReviewID: 21,
	}
	op := &factory.MergeOperation{Kind: factory.OpMerge, OperationID: work.OperationID, Work: work}
	outcome := factory.OperationOutcome{
		OperationID: work.OperationID, InstallationID: "test-installation", Kind: factory.OpMerge,
		ActorID: 8, RepositoryID: 7, Effect: factory.OpEffectCommitted,
		Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionComplete, Reason: "merged",
		Receipt: []byte(`{"pr_number":9}`),
	}
	coord := &Coordinator{}
	if !coord.adoptMergeObserved(op, outcome, time.Now()) {
		t.Fatal("committed merge not adopted after its lost reply")
	}
	if op.Effect != factory.OpEffectCommitted || op.InstallationID != "test-installation" {
		t.Fatalf("adopted effect rewritten: %+v", op)
	}
	// A later observation from another installation never rewrites the
	// recorded one; the first observation owns the attribution.
	foreign := outcome
	foreign.InstallationID = "other-installation"
	if coord.adoptMergeObserved(op, foreign, time.Now()) {
		t.Fatal("foreign installation rewrote its merge")
	}
	flipped := outcome
	flipped.Effect = factory.OpEffectNotCommitted
	if coord.adoptMergeObserved(op, flipped, time.Now()) {
		t.Fatal("committed effect flipped to not_committed")
	}
	unseen := outcome
	unseen.NotObserved = true
	if coord.adoptMergeObserved(op, unseen, time.Now()) {
		t.Fatal("unobserved lookup adopted")
	}
	if op.Effect != factory.OpEffectCommitted {
		t.Fatalf("refused observations rewrote the effect: %q", op.Effect)
	}
}

// TestCompleteMergeWaitsForClosureVerification (T29): a committed merge
// still verifies its closure before finishing. While native completion
// stays pending the merge waits with its bookkeeping intact; recovery
// never enters the authorization gate.
func TestCompleteMergeWaitsForClosureVerification(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	m := gateTestMergeRow(t, db)
	operationID := factory.MergeOperationID(m.PublicationID, 1)
	m.Operation = factory.MergeOperation{
		Kind: factory.OpMerge, OperationID: operationID, Effect: factory.OpEffectCommitted,
		Cancellation: factory.OpCancelNone, Completion: factory.OpCompletionPending,
	}
	coord := &Coordinator{Store: db, Merges: happyMerger()}
	report := &MergeReport{Merged: []MergeLink{}}
	coord.completeMerge(ctx, &m, report)
	if len(report.Errors) != 0 {
		t.Fatalf("completion check errored: %+v", report)
	}
	found := false
	for _, wait := range report.Waits {
		found = found || (wait.ID == m.ID && wait.Reason == "native_completion_pending")
	}
	if !found {
		t.Fatalf("pending completion misadvanced: %+v", report)
	}
	if len(report.Merged) != 0 {
		t.Fatalf("unverified merge linked: %+v", report)
	}
}
