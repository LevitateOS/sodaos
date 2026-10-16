package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
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
	report := correctionTestReport([]factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionFixed}})
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
