package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// unionTestAuthority is the fixed authority behind every FW37 union gate
// test: bindings compare it by value, so one shared value keeps the
// identical-gates comparison exact.
func unionTestAuthority() factory.AuthorityRef {
	return factory.AuthorityRef{Policy: 3, Operator: 1, Capacity: 1, Sponsorship: 1, ConnectionUsageBudget: 1, Environment: 1}
}

// unionTestIDs returns the fixed identities behind the pure (storeless)
// union gate tests.
func unionTestIDs() (publicationID, attemptRoot, reviewer, head, base, diffBase string) {
	return strings.Repeat("9", 32), strings.Repeat("3", 32), strings.Repeat("4", 32),
		strings.Repeat("2", 40), strings.Repeat("1", 40), strings.Repeat("1", 40)
}

// unionTestCycle builds one valid review cycle over a finding-set digest.
func unionTestCycle(t *testing.T, publicationID, attemptRoot, head, base string, round int, reviewer string, digest string) factory.ReviewCycle {
	t.Helper()
	cycle := factory.ReviewCycle{
		PublicationID: publicationID, AttemptRoot: attemptRoot,
		ID:                 factory.ReviewCycleID(publicationID, attemptRoot, round, "review"),
		Phase:              "review",
		TriggerDecision:    factory.DeliveryReviewAgain,
		TriggerReason:      "round needs its independent inspection",
		Head:               head,
		BaseOID:            base,
		FindingDigest:      digest,
		ReviewerAssignment: reviewer,
		Round:              round,
	}
	if err := cycle.Validate(); err != nil {
		t.Fatal(err)
	}
	return cycle
}

// unionPublicationID derives one distinct test publication identity from
// a small distinguisher. Deterministic helpers otherwise admit
// byte-identical acceptances, which would make foreign-record cases pass.
func unionPublicationID(n int) string {
	return strings.Repeat("9", 31) + string(rune('0'+n%10))
}

// unionCleanAdmit admits one zero-delivery-call clean acceptance on a
// minimal publication and returns the journaled publication with it. The
// path takes no chooser, funding or decision: spending is impossible by
// construction.
func unionCleanAdmit(t *testing.T, n int) (factory.Publication, factory.DeliveryAcceptance, factory.ReviewCycle) {
	t.Helper()
	publicationID := unionPublicationID(n)
	_, attemptRoot, reviewer, head, base, diffBase := unionTestIDs()
	keys := []string(nil)
	digest, err := factory.FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	cycle := unionTestCycle(t, publicationID, attemptRoot, head, base, 1, reviewer, digest)
	content, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := factory.ClearCoverageDigest(head, base, content)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	published, acceptance, err := AdmitClearReview(ClearAdmissionInput{
		Publication: factory.Publication{
			ID: publicationID, Candidate: head,
			PRCreate:     factory.PublicationOperation{BaseOID: base},
			ReviewCycles: []factory.ReviewCycle{cycle},
		},
		Cycle: cycle,
		Report: factory.ClearReviewReport{
			CycleID: cycle.ID, Conclusion: factory.ClearReviewConclusion,
			Head: head, BaseOID: base, CoverageDigest: coverage,
		},
		Result: factory.Result{
			Status: "completed", Summary: "nothing found", Candidate: head,
			ReviewPassed: true, Findings: []string{},
		},
		ScopeDigest: scope, PolicyRevision: "policy-revision-9", CheckDigest: scope,
		ReviewerRunID: reviewer, DiffBase: diffBase,
	})
	if err != nil {
		t.Fatal(err)
	}
	return published, acceptance, cycle
}

// unionJevAdmit admits one funded acceptance on a minimal publication and
// returns the journaled publication with it, its ready decision and its
// fingerprint. The decision is built, never bought: no provider exists on
// this path.
func unionJevAdmit(t *testing.T, n int, intents []string) (factory.Publication, factory.DeliveryAcceptance, factory.DeliveryDecision, factory.ReviewCycle, string) {
	t.Helper()
	publicationID := unionPublicationID(n)
	_, attemptRoot, reviewer, head, base, diffBase := unionTestIDs()
	finding := correctionTestFinding(1, true, factory.FindingFixVerification)
	digest := correctionTestDigest(t, finding.Key)
	cycle := unionTestCycle(t, publicationID, attemptRoot, head, base, 1, reviewer, digest)
	scope, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	fingerprintInput := factory.DeliveryFingerprintInput{
		Reviews:       []string{reviewer},
		Conversations: []string{strings.Repeat("2", 64)},
		AttemptRoot:   attemptRoot,
		PublicationID: publicationID,
		ScopeDigest:   scope,
		FixDigest:     scope,
		DiffDigest:    scope,
		CheckDigest:   scope,
		TreeDigest:    scope,
		Head:          head,
		Base:          base,
		Model:         "jev-test",
		Rubric:        factory.DeliveryRubric,
		Round:         1,
	}
	fingerprint, err := factory.DeliveryFingerprint(fingerprintInput)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := readyDeliveryAnswer(fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	decision := factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true}
	if err := decision.Validate(); err != nil {
		t.Fatal(err)
	}
	published, acceptance, err := AdmitJevAcceptance(JevAdmissionInput{
		Publication: factory.Publication{
			ID: publicationID, Candidate: head,
			PRCreate:     factory.PublicationOperation{BaseOID: base},
			ReviewCycles: []factory.ReviewCycle{cycle},
		},
		Cycle:             cycle,
		Carried:           []factory.Finding{finding},
		FingerprintInput:  fingerprintInput,
		Fingerprint:       fingerprint,
		Decision:          decision,
		DecisionID:        strings.Repeat("9", 32),
		ScopeDigest:       scope,
		PolicyRevision:    "policy-revision-9",
		CheckDigest:       scope,
		DiffBase:          diffBase,
		ResolutionIntents: intents,
	})
	if err != nil {
		t.Fatal(err)
	}
	return published, acceptance, decision, cycle, fingerprint
}

// unionGeneration builds the generation seeking admission for one
// journaled acceptance: the acceptance capture plus the exact
// head/base/check evidence behind it.
func unionGeneration(t *testing.T, acceptance factory.DeliveryAcceptance, authority factory.AuthorityRef, fingerprint string) factory.MergeGeneration {
	t.Helper()
	generation := factory.MergeGeneration{
		Authority: authority, PublicationID: acceptance.PublicationID,
		DecisionID:         acceptance.DecisionID,
		NativeOperationID:  factory.MergeGenerationOperationID(acceptance.PublicationID, 1),
		Fingerprint:        fingerprint,
		Head:               acceptance.Tuple.Head,
		Base:               acceptance.Tuple.Base,
		DiffBase:           acceptance.Tuple.DiffBase,
		ChecksDigest:       strings.Repeat("c", 64),
		AcceptanceKind:     acceptance.Kind,
		AcceptanceID:       acceptance.ID,
		AttemptRoot:        acceptance.AttemptRoot,
		OriginReceipt:      acceptance.OriginReceipt,
		Generation:         1,
		ReviewRound:        acceptance.Round,
		AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
	}
	if err := generation.Validate(); err != nil {
		t.Fatal(err)
	}
	return generation
}

// TestUnionBindingAdmitsBothVariantsIdentically (T45) proves the union
// gate itself: a zero-delivery-call clean acceptance and a funded
// acceptance pass the exact same binding and submit gates over the same
// native mergeable expression.
func TestUnionBindingAdmitsBothVariantsIdentically(t *testing.T) {
	authority := unionTestAuthority()
	cleanPub, clean, _ := unionCleanAdmit(t, 1)
	cleanGen := unionGeneration(t, clean, authority, strings.Repeat("e", 64))
	cleanBinding := factory.MergeUnionBindingInput{
		Publication: cleanPub, Acceptance: clean, Generation: cleanGen,
		ResolutionsConfirmed: true, CurrentAuthority: authority,
	}
	if err := factory.CheckMergeUnionBinding(cleanBinding); err != nil {
		t.Fatalf("clean binding refused: %v", err)
	}
	if err := CheckMergeUnionReady(MergeUnionReadyInput{
		Publication: cleanPub, Acceptance: clean, Generation: cleanGen,
		ResolutionsConfirmed: true, CurrentAuthority: authority,
		Native: mergeableGate(cleanGen.Head, cleanGen.Base),
	}); err != nil {
		t.Fatalf("clean submit gate refused: %v", err)
	}
	finding := correctionTestFinding(1, true, factory.FindingFixVerification)
	mappings := []factory.ReviewCommentMapping{{Key: finding.Key, CommentID: 11}}
	threadDigest := strings.Repeat("a", 64)
	_, _, _, head, base, _ := unionTestIDs()
	admission, planned := unionAdmitAndPlan(t, unionPublicationID(2), head, base, finding, mappings, threadDigest)
	intents := []string{planned[0].OperationID}
	jevPub, jev, decision, _, fingerprint := unionJevAdmit(t, 2, intents)
	jevGen := unionGeneration(t, jev, authority, fingerprint)
	jevBinding := factory.MergeUnionBindingInput{
		Publication: jevPub, Acceptance: jev, Generation: jevGen, Decision: decision,
		ResolutionsConfirmed: true, CurrentAuthority: authority,
	}
	if err := factory.CheckMergeUnionBinding(jevBinding); err != nil {
		t.Fatalf("funded binding refused: %v", err)
	}
	if err := CheckMergeUnionReady(MergeUnionReadyInput{
		Publication: jevPub, Acceptance: jev, Generation: jevGen, Decision: decision,
		ResolutionsConfirmed: true, CurrentAuthority: authority,
		Native: mergeableGate(jevGen.Head, jevGen.Base),
	}); err != nil {
		t.Fatalf("funded submit gate refused: %v", err)
	}
	outcome, err := ReconcileAcceptedResolutionsWithUnion(jev, unionCycleFor(t, jevPub, admission), admission, planned, map[string]bool{finding.Key: true})
	if err != nil || len(outcome.Waits) != 0 {
		t.Fatalf("funded resolutions unreconciled: %+v %v", outcome, err)
	}
}

// unionAdmitAndPlan admits one verification fix and plans its native
// resolution, purely: no store, no provider, no mutation.
func unionAdmitAndPlan(t *testing.T, publicationID, head, base string, finding factory.Finding, mappings []factory.ReviewCommentMapping, threadDigest string) (CorrectionAdmission, []AcceptedResolution) {
	t.Helper()
	attemptRoot := strings.Repeat("3", 32)
	digest := correctionTestDigest(t, finding.Key)
	cycle := factory.ReviewCycle{
		PublicationID: publicationID, AttemptRoot: attemptRoot,
		ID:                 factory.ReviewCycleID(publicationID, attemptRoot, 1, "review"),
		Phase:              "review",
		TriggerDecision:    factory.DeliveryReviewAgain,
		TriggerReason:      "correction set needs another inspection",
		Head:               head,
		BaseOID:            base,
		FindingDigest:      digest,
		ReviewerAssignment: strings.Repeat("4", 32),
		Round:              1,
	}
	report := correctionTestReport([]factory.Finding{finding}, []factory.CorrectionDisposition{{Key: finding.Key, Disposition: factory.DispositionFixed}})
	admission, err := AdmitCorrectionReport(cycle, strings.Repeat("c", 32), head, report, nil, correctionTestChecks(head, base))
	if err != nil {
		t.Fatal(err)
	}
	publication := factory.Publication{ID: publicationID}
	evidence := map[int64]factory.ThreadEvidence{
		11: {Digest: threadDigest, Repository: 7, PRNumber: 3, RootCommentID: 11},
	}
	binding := factory.ActorBindingRef{TokenID: 2, ActorID: 6, Kind: factory.OpThreadResolve}
	planned, err := PlanAcceptedResolutions(publication, admission, mappings, evidence, binding, 1,
		"refs/heads/fix", "refs/heads/main", head, base, 7, 3, 5)
	if err != nil {
		t.Fatal(err)
	}
	return admission, planned
}

// unionCycleFor returns the journaled cycle behind one acceptance: the
// acceptance round answers the publication's recorded cycle.
func unionCycleFor(t *testing.T, p factory.Publication, admission CorrectionAdmission) factory.ReviewCycle {
	t.Helper()
	for _, cycle := range p.ReviewCycles {
		if cycle.ID == admission.CycleID {
			return cycle
		}
	}
	t.Fatal("admission answers an unrecorded cycle")
	return factory.ReviewCycle{}
}

// TestUnionBindingRefusesStaleAndAbsent (T45) proves every new mutation
// entry closes without its current applicable union: another
// publication, another acceptance, another round, another candidate, a
// superseded tuple, unconfirmed resolutions, changed authority, an
// unready or missing funded decision, a spending clean binding and a
// generation without its union all refuse.
func TestUnionBindingRefusesStaleAndAbsent(t *testing.T) {
	authority := unionTestAuthority()
	_, clean, _ := unionCleanAdmit(t, 7)
	cleanPub, _, _ := unionCleanAdmit(t, 8)
	base := func() (factory.Publication, factory.DeliveryAcceptance, factory.MergeGeneration) {
		pub, acceptance, _ := unionCleanAdmit(t, 9)
		return pub, acceptance, unionGeneration(t, acceptance, authority, strings.Repeat("e", 64))
	}
	t.Run("another publication", func(t *testing.T) {
		_, acceptance, generation := base()
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: cleanPub, Acceptance: acceptance, Generation: generation,
			ResolutionsConfirmed: true, CurrentAuthority: authority,
		}); err == nil {
			t.Fatal("foreign publication passed its union gate")
		}
	})
	t.Run("another acceptance", func(t *testing.T) {
		pub, _, generation := base()
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: pub, Acceptance: clean, Generation: generation,
			ResolutionsConfirmed: true, CurrentAuthority: authority,
		}); err == nil {
			t.Fatal("foreign acceptance passed its union gate")
		}
	})
	t.Run("unconfirmed resolutions", func(t *testing.T) {
		pub, acceptance, generation := base()
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: pub, Acceptance: acceptance, Generation: generation,
			ResolutionsConfirmed: false, CurrentAuthority: authority,
		}); err == nil {
			t.Fatal("unconfirmed resolutions passed their union gate")
		}
	})
	t.Run("changed authority", func(t *testing.T) {
		pub, acceptance, generation := base()
		moved := authority
		moved.Policy = 9
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: pub, Acceptance: acceptance, Generation: generation,
			ResolutionsConfirmed: true, CurrentAuthority: moved,
		}); err == nil {
			t.Fatal("changed authority passed its union gate")
		}
	})
	t.Run("superseded tuple", func(t *testing.T) {
		pub, acceptance, generation := base()
		pub.Candidate = strings.Repeat("f", 40)
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: pub, Acceptance: acceptance, Generation: generation,
			ResolutionsConfirmed: true, CurrentAuthority: authority,
		}); err == nil {
			t.Fatal("superseded acceptance passed its union gate")
		}
	})
	t.Run("generation without union", func(t *testing.T) {
		pub, acceptance, generation := base()
		generation.AcceptanceKind, generation.AcceptanceID = "", ""
		generation.AttemptRoot, generation.OriginReceipt = "", ""
		generation.DecisionID = strings.Repeat("d", 32)
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: pub, Acceptance: acceptance, Generation: generation,
			ResolutionsConfirmed: true, CurrentAuthority: authority,
		}); err == nil {
			t.Fatal("unionless generation passed its union gate")
		}
	})
	t.Run("clean spending decision", func(t *testing.T) {
		pub, acceptance, generation := base()
		answer, err := readyDeliveryAnswer(generation.Fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: pub, Acceptance: acceptance, Generation: generation,
			Decision:             factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true},
			ResolutionsConfirmed: true, CurrentAuthority: authority,
		}); err == nil {
			t.Fatal("spending clean binding passed its union gate")
		}
	})
	t.Run("funded without decision", func(t *testing.T) {
		_, _, _, head, base, _ := unionTestIDs()
		admission, planned := unionAdmitAndPlan(t, unionPublicationID(3), head, base,
			correctionTestFinding(1, true, factory.FindingFixVerification),
			[]factory.ReviewCommentMapping{{Key: correctionTestKey(1), CommentID: 11}},
			strings.Repeat("a", 64))
		intents := []string{planned[0].OperationID}
		pub, acceptance, _, _, fingerprint := unionJevAdmit(t, 3, intents)
		generation := unionGeneration(t, acceptance, authority, fingerprint)
		_ = admission
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: pub, Acceptance: acceptance, Generation: generation,
			ResolutionsConfirmed: true, CurrentAuthority: authority,
		}); err == nil {
			t.Fatal("decisionless funded binding passed its union gate")
		}
	})
	t.Run("funded review_again decision", func(t *testing.T) {
		_, _, _, head, base, _ := unionTestIDs()
		admission, planned := unionAdmitAndPlan(t, unionPublicationID(3), head, base,
			correctionTestFinding(1, true, factory.FindingFixVerification),
			[]factory.ReviewCommentMapping{{Key: correctionTestKey(1), CommentID: 11}},
			strings.Repeat("a", 64))
		intents := []string{planned[0].OperationID}
		pub, acceptance, _, _, fingerprint := unionJevAdmit(t, 3, intents)
		generation := unionGeneration(t, acceptance, authority, fingerprint)
		_ = admission
		answer, err := reviewDeliveryAnswer(fingerprint)
		if err != nil {
			t.Fatal(err)
		}
		if err := factory.CheckMergeUnionBinding(factory.MergeUnionBindingInput{
			Publication: pub, Acceptance: acceptance, Generation: generation,
			Decision:             factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReview, ReviewsFit: true},
			ResolutionsConfirmed: true, CurrentAuthority: authority,
		}); err == nil {
			t.Fatal("review_again binding passed its union gate")
		}
	})
}

// unionCleanAdmitCarried admits one clean acceptance over one carried
// actionable finding with its verified assessment: the historical fix
// stays visible while the round still clears.
func unionCleanAdmitCarried(t *testing.T, n int) (factory.Publication, factory.DeliveryAcceptance, factory.ReviewCycle, factory.Finding) {
	t.Helper()
	publicationID := unionPublicationID(n)
	_, attemptRoot, reviewer, head, base, diffBase := unionTestIDs()
	finding := correctionTestFinding(1, true, factory.FindingFixVerification)
	carried := []factory.Finding{finding}
	keys := []string{finding.Key}
	digest, err := factory.FindingSetDigest(keys)
	if err != nil {
		t.Fatal(err)
	}
	cycle := unionTestCycle(t, publicationID, attemptRoot, head, base, 1, reviewer, digest)
	content, err := factory.FindingContentDigest(carried)
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := factory.ClearCoverageDigest(head, base, content)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	published, acceptance, err := AdmitClearReview(ClearAdmissionInput{
		Publication: factory.Publication{
			ID: publicationID, Candidate: head,
			PRCreate:     factory.PublicationOperation{BaseOID: base},
			ReviewCycles: []factory.ReviewCycle{cycle},
		},
		Cycle: cycle,
		Report: factory.ClearReviewReport{
			CycleID: cycle.ID, Conclusion: factory.ClearReviewConclusion,
			Head: head, BaseOID: base, CoverageDigest: coverage,
			Assessments: []factory.FindingAssessment{
				{Key: finding.Key, Status: factory.FindingVerified, Evidence: "check verify passes on the head"},
			},
		},
		Carried: carried,
		Result: factory.Result{
			Status: "completed", Summary: "nothing found", Candidate: head,
			ReviewPassed: true, Findings: []string{},
		},
		ScopeDigest: scope, PolicyRevision: "policy-revision-9", CheckDigest: scope,
		ReviewerRunID: reviewer, DiffBase: diffBase,
	})
	if err != nil {
		t.Fatal(err)
	}
	return published, acceptance, cycle, finding
}

// TestUnionResolutionReadsBackBothVariants (T45) proves expected
// resolutions read back under the union: confirmed fixes resolve,
// unconfirmed wait, repeats are identical and spend nothing, and stale
// unions or mismatched intents refuse on either variant.
func TestUnionResolutionReadsBackBothVariants(t *testing.T) {
	_, _, _, head, base, _ := unionTestIDs()
	t.Run("funded resolves and waits", func(t *testing.T) {
		admission, planned := unionAdmitAndPlan(t, unionPublicationID(2), head, base,
			correctionTestFinding(1, true, factory.FindingFixVerification),
			[]factory.ReviewCommentMapping{{Key: correctionTestKey(1), CommentID: 11}},
			strings.Repeat("a", 64))
		intents := []string{planned[0].OperationID}
		pub, acceptance, _, _, _ := unionJevAdmit(t, 2, intents)
		cycle := unionCycleFor(t, pub, admission)
		confirmed := map[string]bool{correctionTestKey(1): true}
		first, err := ReconcileAcceptedResolutionsWithUnion(acceptance, cycle, admission, planned, confirmed)
		if err != nil || len(first.Waits) != 0 || len(first.Resolved) != 1 {
			t.Fatalf("confirmed resolutions unreconciled: %+v %v", first, err)
		}
		second, err := ReconcileAcceptedResolutionsWithUnion(acceptance, cycle, admission, planned, confirmed)
		if err != nil || len(second.Resolved) != 1 || second.Resolved[0] != first.Resolved[0] {
			t.Fatalf("readback not identical: %+v %v", second, err)
		}
		pending, err := ReconcileAcceptedResolutionsWithUnion(acceptance, cycle, admission, planned, map[string]bool{})
		if err != nil || len(pending.Waits) != 1 {
			t.Fatalf("unconfirmed fix did not wait: %+v %v", pending, err)
		}
	})
	t.Run("clean resolves vacuously", func(t *testing.T) {
		pub, acceptance, _, finding := unionCleanAdmitCarried(t, 4)
		admission, planned := unionAdmitAndPlan(t, unionPublicationID(4), head, base, finding, nil, strings.Repeat("a", 64))
		if len(planned) != 1 || !planned[0].NoNativeThread {
			t.Fatalf("carried fix without a thread planned natively: %+v", planned)
		}
		outcome, err := ReconcileAcceptedResolutionsWithUnion(acceptance, unionCycleFor(t, pub, admission), admission, planned, map[string]bool{})
		if err != nil || len(outcome.Waits) != 0 || len(outcome.Resolved) != 1 {
			t.Fatalf("vacuous resolutions unreconciled: %+v %v", outcome, err)
		}
	})
	t.Run("stale round refuses", func(t *testing.T) {
		admission, planned := unionAdmitAndPlan(t, unionPublicationID(2), head, base,
			correctionTestFinding(1, true, factory.FindingFixVerification),
			[]factory.ReviewCommentMapping{{Key: correctionTestKey(1), CommentID: 11}},
			strings.Repeat("a", 64))
		intents := []string{planned[0].OperationID}
		pub, acceptance, _, _, _ := unionJevAdmit(t, 2, intents)
		cycle := unionCycleFor(t, pub, admission)
		cycle.Round = 2
		if _, err := ReconcileAcceptedResolutionsWithUnion(acceptance, cycle, admission, planned, map[string]bool{correctionTestKey(1): true}); err == nil {
			t.Fatal("stale round reconciled")
		}
	})
	t.Run("mismatched intent refuses", func(t *testing.T) {
		admission, planned := unionAdmitAndPlan(t, unionPublicationID(2), head, base,
			correctionTestFinding(1, true, factory.FindingFixVerification),
			[]factory.ReviewCommentMapping{{Key: correctionTestKey(1), CommentID: 11}},
			strings.Repeat("a", 64))
		pub, acceptance, _, _, _ := unionJevAdmit(t, 2, []string{strings.Repeat("0", 32)})
		cycle := unionCycleFor(t, pub, admission)
		if _, err := ReconcileAcceptedResolutionsWithUnion(acceptance, cycle, admission, planned, map[string]bool{correctionTestKey(1): true}); err == nil {
			t.Fatal("foreign intent reconciled")
		}
	})
	t.Run("native thread under clean refuses", func(t *testing.T) {
		pub, acceptance, _, finding := unionCleanAdmitCarried(t, 5)
		admission, planned := unionAdmitAndPlan(t, unionPublicationID(5), head, base, finding,
			[]factory.ReviewCommentMapping{{Key: finding.Key, CommentID: 11}}, strings.Repeat("a", 64))
		cycle := unionCycleFor(t, pub, admission)
		if _, err := ReconcileAcceptedResolutionsWithUnion(acceptance, cycle, admission, planned, map[string]bool{finding.Key: true}); err == nil {
			t.Fatal("clean acceptance resolved a native thread")
		}
	})
}

// unionReadyCase is one retained native-policy fixture case: the native
// mergeable + reviewable expression (the deterministic model of Fountain
// policy behind every merge gate since FW22) with the exact submit-gate
// verdict it must produce on either acceptance variant.
type unionReadyCase struct {
	name    string
	native  func(head, base string) MergeNativeGate
	refused string
	wait    string
}

// TestUnionReadyHoldsNativePolicyIdentically (T46) proves a clear review
// never waives native policy: required approvals, checks, blocking human
// review and protected states hold identically for the zero-call clean
// variant and the funded variant through the retained native-gate
// fixture. An unrelated actor's missing approval waits; it never merges
// and never refuses as the wrong reason.
func TestUnionReadyHoldsNativePolicyIdentically(t *testing.T) {
	authority := unionTestAuthority()
	cases := []unionReadyCase{
		{name: "missing approval waits", native: func(head, base string) MergeNativeGate {
			gate := mergeableGate(head, base)
			gate.Approved = false
			return gate
		}, wait: "approval_missing"},
		{name: "blocking review waits", native: func(head, base string) MergeNativeGate {
			gate := mergeableGate(head, base)
			gate.ChangesRequested = true
			return gate
		}, wait: "changes_requested"},
		{name: "checking waits", native: func(head, base string) MergeNativeGate { return blockedGate(head, base, "checking", true) }, wait: "checking"},
		{name: "status checks wait", native: func(head, base string) MergeNativeGate { return blockedGate(head, base, "status_checks", true) }, wait: "status_checks"},
		{name: "approvals wait", native: func(head, base string) MergeNativeGate { return blockedGate(head, base, "approvals", true) }, wait: "approvals"},
		{name: "review requests wait", native: func(head, base string) MergeNativeGate {
			return blockedGate(head, base, "review_requests", true)
		}, wait: "review_requests"},
		{name: "protected files refuse", native: func(head, base string) MergeNativeGate {
			return blockedGate(head, base, "protected_files", true)
		}, refused: "protected_files"},
		{name: "permission refuses", native: func(head, base string) MergeNativeGate { return blockedGate(head, base, "permission", true) }, refused: "permission"},
		{name: "hidden evidence waits", native: func(head, base string) MergeNativeGate {
			gate := mergeableGate(head, base)
			gate.Visible = false
			return gate
		}, wait: "merge_evidence_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cleanPub, clean, _ := unionCleanAdmit(t, 1)
			cleanGen := unionGeneration(t, clean, authority, strings.Repeat("e", 64))
			cleanErr := CheckMergeUnionReady(MergeUnionReadyInput{
				Publication: cleanPub, Acceptance: clean, Generation: cleanGen,
				ResolutionsConfirmed: true, CurrentAuthority: authority,
				Native: tc.native(cleanGen.Head, cleanGen.Base),
			})
			_, _, _, head, base, _ := unionTestIDs()
			admission, planned := unionAdmitAndPlan(t, unionPublicationID(2), head, base,
				correctionTestFinding(1, true, factory.FindingFixVerification),
				[]factory.ReviewCommentMapping{{Key: correctionTestKey(1), CommentID: 11}},
				strings.Repeat("a", 64))
			intents := []string{planned[0].OperationID}
			jevPub, jev, decision, _, fingerprint := unionJevAdmit(t, 2, intents)
			_ = admission
			jevGen := unionGeneration(t, jev, authority, fingerprint)
			jevErr := CheckMergeUnionReady(MergeUnionReadyInput{
				Publication: jevPub, Acceptance: jev, Generation: jevGen, Decision: decision,
				ResolutionsConfirmed: true, CurrentAuthority: authority,
				Native: tc.native(jevGen.Head, jevGen.Base),
			})
			assertUnionVerdict(t, "clean", tc, cleanErr)
			assertUnionVerdict(t, "funded", tc, jevErr)
		})
	}
}

// assertUnionVerdict checks one variant produced exactly the fixture's
// verdict: the same refusal or wait reason on either acceptance kind.
func assertUnionVerdict(t *testing.T, variant string, tc unionReadyCase, err error) {
	t.Helper()
	if tc.refused != "" {
		var refused *factory.PublicationRefusal
		if !errors.As(err, &refused) || refused.Reason != tc.refused {
			t.Fatalf("%s gave %v, want refusal %q", variant, err, tc.refused)
		}
		return
	}
	var wait *factory.PublicationWait
	if !errors.As(err, &wait) || wait.Reason != tc.wait {
		t.Fatalf("%s gave %v, want wait %q", variant, err, tc.wait)
	}
}

// unionPgAdmitClean journals one zero-call clean acceptance on a stored
// serving publication and returns the stored publication with it. The
// cycle is the publication's first review round over the exact current
// head and base.
func unionPgAdmitClean(t *testing.T, ctx context.Context, db *store.Store, p factory.Publication, reviewer string) (factory.Publication, factory.DeliveryAcceptance) {
	t.Helper()
	owner, err := db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := factory.FindingSetDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID:                 factory.ReviewCycleID(p.ID, owner.AttemptRoot, 1, "review"),
		Phase:              "review",
		TriggerDecision:    "initial-review",
		TriggerReason:      "first review of the published head",
		Head:               p.Candidate,
		BaseOID:            p.PRCreate.BaseOID,
		FindingDigest:      digest,
		ReviewerAssignment: reviewer,
		Round:              1,
	}
	p.ReviewCycles = append(p.ReviewCycles, cycle)
	content, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	coverage, err := factory.ClearCoverageDigest(p.Candidate, p.PRCreate.BaseOID, content)
	if err != nil {
		t.Fatal(err)
	}
	published, acceptance, err := AdmitClearReview(ClearAdmissionInput{
		Publication: p,
		Cycle:       cycle,
		Report: factory.ClearReviewReport{
			CycleID: cycle.ID, Conclusion: factory.ClearReviewConclusion,
			Head: p.Candidate, BaseOID: p.PRCreate.BaseOID, CoverageDigest: coverage,
		},
		Result: factory.Result{
			Status: "completed", Summary: "nothing found", Candidate: p.Candidate,
			ReviewPassed: true, Findings: []string{},
		},
		ScopeDigest: content, PolicyRevision: "policy-revision-9", CheckDigest: content,
		ReviewerRunID: reviewer, DiffBase: p.PRCreate.BaseOID,
	})
	if err != nil {
		t.Fatal(err)
	}
	published.Revision = p.Revision + 1
	if err := db.UpdatePublication(ctx, published); err != nil {
		t.Fatal(err)
	}
	return published, acceptance
}

// unionPgAdmitJev journals one funded acceptance on a stored serving
// publication and returns the stored publication with it, its ready
// decision and its fingerprint. The decision is built, never bought.
func unionPgAdmitJev(t *testing.T, ctx context.Context, db *store.Store, p factory.Publication, reviewer string, intents []string) (factory.Publication, factory.DeliveryAcceptance, factory.DeliveryDecision, string) {
	t.Helper()
	owner, err := db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	finding := correctionTestFinding(1, true, factory.FindingFixVerification)
	digest := correctionTestDigest(t, finding.Key)
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID:                 factory.ReviewCycleID(p.ID, owner.AttemptRoot, 1, "review"),
		Phase:              "review",
		TriggerDecision:    "initial-review",
		TriggerReason:      "first review of the published head",
		Head:               p.Candidate,
		BaseOID:            p.PRCreate.BaseOID,
		FindingDigest:      digest,
		ReviewerAssignment: reviewer,
		Round:              1,
	}
	p.ReviewCycles = append(p.ReviewCycles, cycle)
	scope, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	fingerprintInput := factory.DeliveryFingerprintInput{
		Reviews:       []string{reviewer},
		Conversations: []string{strings.Repeat("2", 64)},
		AttemptRoot:   owner.AttemptRoot,
		PublicationID: p.ID,
		ScopeDigest:   scope,
		FixDigest:     scope,
		DiffDigest:    scope,
		CheckDigest:   scope,
		TreeDigest:    scope,
		Head:          p.Candidate,
		Base:          p.PRCreate.BaseOID,
		Model:         "jev-test",
		Rubric:        factory.DeliveryRubric,
		Round:         1,
	}
	fingerprint, err := factory.DeliveryFingerprint(fingerprintInput)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := readyDeliveryAnswer(fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	decision := factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true}
	if err := decision.Validate(); err != nil {
		t.Fatal(err)
	}
	published, acceptance, err := AdmitJevAcceptance(JevAdmissionInput{
		Publication:       p,
		Cycle:             cycle,
		Carried:           []factory.Finding{finding},
		FingerprintInput:  fingerprintInput,
		Fingerprint:       fingerprint,
		Decision:          decision,
		DecisionID:        strings.Repeat("9", 32),
		ScopeDigest:       scope,
		PolicyRevision:    "policy-revision-9",
		CheckDigest:       scope,
		DiffBase:          p.PRCreate.BaseOID,
		ResolutionIntents: intents,
	})
	if err != nil {
		t.Fatal(err)
	}
	published.Revision = p.Revision + 1
	if err := db.UpdatePublication(ctx, published); err != nil {
		t.Fatal(err)
	}
	return published, acceptance, decision, fingerprint
}

// unionPgBind opens a merge row for a stored publication and binds one
// union authorization for its journaled acceptance. It returns the bound
// row and the stored publication carrying the journal.
func unionPgBind(t *testing.T, ctx context.Context, fx *mergeFixture, p factory.Publication, acceptance factory.DeliveryAcceptance, decision factory.DeliveryDecision, fingerprint string) (factory.Merge, factory.Publication) {
	t.Helper()
	setup := MergeReport{Merged: []MergeLink{}}
	fx.publish.coord.mergeOne(ctx, p, &setup)
	if len(setup.Errors) != 0 {
		t.Fatalf("merge row open errored: %+v", setup)
	}
	m, err := fx.publish.db.MergeByPublication(ctx, p.ID)
	if err != nil {
		t.Fatalf("merge row unopened: %v", err)
	}
	generation := factory.MergeGeneration{
		Authority: m.Authority, PublicationID: p.ID,
		DecisionID:         acceptance.DecisionID,
		NativeOperationID:  factory.MergeGenerationOperationID(p.ID, 1),
		Fingerprint:        fingerprint,
		Head:               m.HeadOID,
		Base:               m.BaseOID,
		DiffBase:           acceptance.Tuple.DiffBase,
		ChecksDigest:       strings.Repeat("c", 64),
		AcceptanceKind:     acceptance.Kind,
		AcceptanceID:       acceptance.ID,
		AttemptRoot:        acceptance.AttemptRoot,
		OriginReceipt:      acceptance.OriginReceipt,
		Generation:         1,
		ReviewRound:        acceptance.Round,
		AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
	}
	auth := factory.MergeAuthorization{
		Generation: generation, Decision: decision, ResolutionsConfirmed: true,
	}
	if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, auth); err != nil {
		t.Fatalf("union registration refused: %v", err)
	}
	stored, err := fx.publish.db.MergeByPublication(ctx, p.ID)
	if err != nil || !stored.Authorization.Bound() {
		t.Fatalf("binding not persisted: %+v %v", stored, err)
	}
	fresh, err := fx.publish.db.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := mergeAuthorizationGate(stored, fresh); err != nil {
		t.Fatalf("bound row refused its submit gate: %v", err)
	}
	return stored, fresh
}

// unionSeedForBind seeds one serving publication with a journaled funded
// acceptance and one open unbound merge row for registration tests. The
// row stays unbound so each subtest binds its own mutated authorization.
func unionSeedForBind(t *testing.T, ctx context.Context, issue int64) (*mergeFixture, factory.Publication, factory.Merge, factory.DeliveryAcceptance, factory.DeliveryDecision, string) {
	t.Helper()
	fx, p := mergeSeed(t, issue)
	db := fx.publish.db
	storedPub, acceptance, decision, fingerprint := unionPgAdmitJev(t, ctx, db, p, strings.Repeat("4", 32), nil)
	setup := MergeReport{Merged: []MergeLink{}}
	fx.publish.coord.mergeOne(ctx, storedPub, &setup)
	if len(setup.Errors) != 0 {
		t.Fatalf("merge row open errored: %+v", setup)
	}
	m, err := db.MergeByPublication(ctx, p.ID)
	if err != nil {
		t.Fatalf("merge row unopened: %v", err)
	}
	return fx, storedPub, m, acceptance, decision, fingerprint
}

// unionAuthForBind builds one union authorization for an open row behind
// a journaled acceptance.
func unionAuthForBind(t *testing.T, m factory.Merge, acceptance factory.DeliveryAcceptance, decision factory.DeliveryDecision, fingerprint string, confirmed bool) factory.MergeAuthorization {
	t.Helper()
	auth := factory.MergeAuthorization{
		Generation: factory.MergeGeneration{
			Authority: m.Authority, PublicationID: m.PublicationID,
			DecisionID:         acceptance.DecisionID,
			NativeOperationID:  factory.MergeGenerationOperationID(m.PublicationID, 1),
			Fingerprint:        fingerprint,
			Head:               m.HeadOID,
			Base:               m.BaseOID,
			DiffBase:           acceptance.Tuple.DiffBase,
			ChecksDigest:       strings.Repeat("c", 64),
			AcceptanceKind:     acceptance.Kind,
			AcceptanceID:       acceptance.ID,
			AttemptRoot:        acceptance.AttemptRoot,
			OriginReceipt:      acceptance.OriginReceipt,
			Generation:         1,
			ReviewRound:        acceptance.Round,
			AssessmentRevision: 2, ObservedNativeRev: 12, NotAfter: 1300,
		},
		Decision: decision, ResolutionsConfirmed: confirmed,
	}
	return auth
}

// TestUnionPgAuthorizeMergeBindsBothVariants (T45, real pg) proves merge
// registration under the union on the serving store: a clean acceptance
// binds with no funded decision at all, a funded acceptance binds with
// its ready decision, and a second bind, a legacy binding or another
// candidate all refuse.
func TestUnionPgAuthorizeMergeBindsBothVariants(t *testing.T) {
	ctx := context.Background()
	t.Run("clean binds without any funded decision", func(t *testing.T) {
		fx, p := mergeSeed(t, 31)
		db := fx.publish.db
		storedPub, acceptance := unionPgAdmitClean(t, ctx, db, p, strings.Repeat("4", 32))
		stored, _ := unionPgBind(t, ctx, fx, storedPub, acceptance, factory.DeliveryDecision{}, strings.Repeat("e", 64))
		if stored.Authorization.Decision.Effective != "" {
			t.Fatalf("clean binding carries a funded decision: %+v", stored.Authorization.Decision)
		}
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, stored.Authorization); err == nil {
			t.Fatal("second binding replaced its predecessor")
		}
	})
	t.Run("funded binds with its ready decision", func(t *testing.T) {
		fx, p := mergeSeed(t, 32)
		db := fx.publish.db
		storedPub, acceptance, decision, fingerprint := unionPgAdmitJev(t, ctx, db, p, strings.Repeat("4", 32), nil)
		stored, _ := unionPgBind(t, ctx, fx, storedPub, acceptance, decision, fingerprint)
		if stored.Authorization.Generation.DecisionID != acceptance.DecisionID {
			t.Fatalf("funded binding lost its decision: %+v", stored.Authorization)
		}
	})
	t.Run("legacy binding refuses", func(t *testing.T) {
		fx, p := mergeSeed(t, 33)
		setup := MergeReport{Merged: []MergeLink{}}
		fx.publish.coord.mergeOne(ctx, p, &setup)
		m, err := fx.publish.db.MergeByPublication(ctx, p.ID)
		if err != nil {
			t.Fatalf("merge row unopened: %v", err)
		}
		legacy := gateTestBinding(t, m)
		if err := fx.publish.coord.AuthorizeMerge(ctx, p.ID, legacy); err == nil {
			t.Fatal("unionless binding registered")
		}
	})
}

// TestUnionPgMergePassCoversBothVariants (T45, real pg) proves every
// serving progress entry under the union: bound rows submit on MergePass
// for both variants, an unbound row waits for its authorization instead
// of submitting, and a superseded acceptance waits instead of merging
// another candidate.
func TestUnionPgMergePassCoversBothVariants(t *testing.T) {
	ctx := context.Background()
	t.Run("clean submits", func(t *testing.T) {
		fx, p := mergeSeed(t, 34)
		db := fx.publish.db
		storedPub, acceptance := unionPgAdmitClean(t, ctx, db, p, strings.Repeat("4", 32))
		_, _ = unionPgBind(t, ctx, fx, storedPub, acceptance, factory.DeliveryDecision{}, strings.Repeat("e", 64))
		exec := happyMerger()
		fx.wire(exec)
		report := fx.pass(t)
		m := fx.merge(t, p)
		if len(report.Merged) != 1 || m.Stage != factory.MergeMerged || len(exec.submits) != 1 {
			t.Fatalf("clean merge unsettled: %+v %+v submits=%d", report, m, len(exec.submits))
		}
	})
	t.Run("funded submits", func(t *testing.T) {
		fx, p := mergeSeed(t, 35)
		db := fx.publish.db
		storedPub, acceptance, decision, fingerprint := unionPgAdmitJev(t, ctx, db, p, strings.Repeat("4", 32), nil)
		_, _ = unionPgBind(t, ctx, fx, storedPub, acceptance, decision, fingerprint)
		exec := happyMerger()
		fx.wire(exec)
		report := fx.pass(t)
		m := fx.merge(t, p)
		if len(report.Merged) != 1 || m.Stage != factory.MergeMerged || len(exec.submits) != 1 {
			t.Fatalf("funded merge unsettled: %+v %+v submits=%d", report, m, len(exec.submits))
		}
	})
	t.Run("unbound waits for its authorization", func(t *testing.T) {
		fx, p := mergeSeed(t, 36)
		exec := happyMerger()
		fx.wire(exec)
		report := fx.pass(t)
		m := fx.merge(t, p)
		if m.Stage != factory.MergeOpen || len(exec.submits) != 0 {
			t.Fatalf("unbound row submitted: %+v submits=%d", m, len(exec.submits))
		}
		waited := false
		for _, wait := range report.Waits {
			waited = waited || wait.Reason == "merge_judgment_unavailable"
		}
		if !waited {
			t.Fatalf("unbound row did not wait for its authorization: %+v", report)
		}
	})
	t.Run("superseded acceptance waits", func(t *testing.T) {
		fx, p := mergeSeed(t, 37)
		db := fx.publish.db
		storedPub, acceptance := unionPgAdmitClean(t, ctx, db, p, strings.Repeat("4", 32))
		stored, _ := unionPgBind(t, ctx, fx, storedPub, acceptance, factory.DeliveryDecision{}, strings.Repeat("e", 64))
		moved, err := db.PublicationByAssignment(ctx, p.AssignmentID)
		if err != nil {
			t.Fatal(err)
		}
		moved.Candidate = strings.Repeat("f", 40)
		if err := mergeAuthorizationGate(stored, moved); err == nil {
			t.Fatal("superseded acceptance passed its submit gate")
		} else {
			var wait *factory.PublicationWait
			if !errors.As(err, &wait) || wait.Reason != "merge_authorization_stale" {
				t.Fatalf("superseded acceptance gave %v, want merge_authorization_stale wait", err)
			}
		}
	})
}

// TestUnionPgProgressLeavesBoundRowsUndisturbed (T45, real pg) proves
// the progress entry is union-safe for both variants: advancing progress
// after publish neither disturbs a bound row nor opens a second one,
// and the submit gate still passes afterwards.
func TestUnionPgProgressLeavesBoundRowsUndisturbed(t *testing.T) {
	ctx := context.Background()
	t.Run("clean", func(t *testing.T) {
		fx, p := mergeSeed(t, 45)
		db := fx.publish.db
		storedPub, acceptance := unionPgAdmitClean(t, ctx, db, p, strings.Repeat("4", 32))
		_, _ = unionPgBind(t, ctx, fx, storedPub, acceptance, factory.DeliveryDecision{}, strings.Repeat("e", 64))
		fx.publish.coord.progressAfterPublish(ctx, p.AssignmentID)
		stored, err := db.MergeByPublication(ctx, p.ID)
		if err != nil || !stored.Authorization.Bound() || stored.Authorization.Generation.AcceptanceID != acceptance.ID {
			t.Fatalf("progress disturbed its clean binding: %+v %v", stored, err)
		}
	})
	t.Run("funded", func(t *testing.T) {
		fx, p := mergeSeed(t, 46)
		db := fx.publish.db
		storedPub, acceptance, decision, fingerprint := unionPgAdmitJev(t, ctx, db, p, strings.Repeat("4", 32), nil)
		_, _ = unionPgBind(t, ctx, fx, storedPub, acceptance, decision, fingerprint)
		fx.publish.coord.progressAfterPublish(ctx, p.AssignmentID)
		stored, err := db.MergeByPublication(ctx, p.ID)
		if err != nil || !stored.Authorization.Bound() || stored.Authorization.Generation.AcceptanceID != acceptance.ID {
			t.Fatalf("progress disturbed its funded binding: %+v %v", stored, err)
		}
	})
}

// TestUnionPgAdoptsLostCommittedWithoutNewDecision (T45, real pg) proves
// recovery never buys a new paid decision: a clean-bound row whose submit
// reply is lost reconciles its recorded operation by lookup, adopts the
// committed effect and completes without resubmitting and without any
// funded artifact appearing anywhere on the path.
func TestUnionPgAdoptsLostCommittedWithoutNewDecision(t *testing.T) {
	ctx := context.Background()
	fx, p := mergeSeed(t, 38)
	db := fx.publish.db
	storedPub, acceptance := unionPgAdmitClean(t, ctx, db, p, strings.Repeat("4", 32))
	_, _ = unionPgBind(t, ctx, fx, storedPub, acceptance, factory.DeliveryDecision{}, strings.Repeat("e", 64))
	lost := true
	exec := happyMerger()
	exec.submit = func(w factory.MergeWork) (factory.OperationOutcome, error) {
		if lost {
			lost = false
			return factory.OperationOutcome{NotObserved: true}, nil
		}
		return committedOutcome(`{"pr_number":9}`), nil
	}
	exec.lookup = func(operationID string) (factory.OperationOutcome, error) {
		m, err := db.MergeByPublication(ctx, p.ID)
		if err != nil {
			t.Fatalf("lookup lost its row: %v", err)
		}
		if m.Operation.Work == nil {
			t.Fatal("lookup ran before its intent was recorded")
		}
		return factory.OperationOutcome{
			OperationID: m.Operation.OperationID, InstallationID: "test-installation", Kind: factory.OpMerge,
			ActorID: m.Operation.Work.ActorID, RepositoryID: m.Operation.Work.Repository,
			Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone,
			Completion: factory.OpCompletionComplete, Reason: "merged",
			Receipt: []byte(`{"pr_number":9}`),
		}, nil
	}
	fx.wire(exec)
	first := fx.pass(t)
	open := fx.merge(t, p)
	if open.Stage != factory.MergeOpen || open.Operation.Work == nil || open.Operation.Effect != "" {
		t.Fatalf("lost reply did not leave its recorded intent: %+v %+v", first, open)
	}
	second := fx.pass(t)
	done := fx.merge(t, p)
	if len(second.Merged) != 1 || done.Stage != factory.MergeMerged || len(exec.submits) != 1 {
		t.Fatalf("lost committed merge unadopted: %+v %+v submits=%d", second, done, len(exec.submits))
	}
	if done.Authorization.Generation.AcceptanceID != acceptance.ID || done.Authorization.Decision.Effective != "" {
		t.Fatalf("recovery bought a new decision: %+v", done.Authorization)
	}
}

// TestUnionPgCleanCannotWaivePolicy (T46, real pg) proves the serving
// merge path still enforces native policy under a clean acceptance:
// regressed checks and a missing approval wait instead of submitting,
// and the bounded policy-required pass still dispatches when the
// configured reviewer owes the approval.
func TestUnionPgCleanCannotWaivePolicy(t *testing.T) {
	ctx := context.Background()
	t.Run("regressed checks wait", func(t *testing.T) {
		fx, p := mergeSeed(t, 39)
		db := fx.publish.db
		storedPub, acceptance := unionPgAdmitClean(t, ctx, db, p, strings.Repeat("4", 32))
		_, _ = unionPgBind(t, ctx, fx, storedPub, acceptance, factory.DeliveryDecision{}, strings.Repeat("e", 64))
		stored, err := db.CheckAssessment(ctx, p.Repository, p.PRNumber)
		if err != nil {
			t.Fatal(err)
		}
		stored.HeadOID = strings.Repeat("c", 40)
		stored.Revision = 0
		if _, err := db.RecordCheckAssessment(ctx, stored); err != nil {
			t.Fatal(err)
		}
		exec := happyMerger()
		fx.wire(exec)
		report := fx.pass(t)
		m := fx.merge(t, p)
		if m.Stage != factory.MergeOpen || len(exec.submits) != 0 {
			t.Fatalf("clean acceptance waived its checks: %+v submits=%d", m, len(exec.submits))
		}
		waited := false
		for _, wait := range report.Waits {
			waited = waited || wait.Reason == "check_evidence_stale"
		}
		if !waited {
			t.Fatalf("regressed checks did not wait: %+v", report)
		}
	})
	t.Run("missing approval waits", func(t *testing.T) {
		fx, p := mergeSeed(t, 40)
		db := fx.publish.db
		storedPub, acceptance := unionPgAdmitClean(t, ctx, db, p, strings.Repeat("4", 32))
		_, _ = unionPgBind(t, ctx, fx, storedPub, acceptance, factory.DeliveryDecision{}, strings.Repeat("e", 64))
		exec := happyMerger()
		exec.observe = func(factory.MergeWork) (factory.MergeObservation, error) {
			return factory.MergeObservation{}, &factory.PublicationWait{Reason: "approval_missing"}
		}
		fx.wire(exec)
		report := fx.pass(t)
		m := fx.merge(t, p)
		if m.Stage != factory.MergeOpen || len(exec.submits) != 0 {
			t.Fatalf("clean acceptance waived its approval: %+v submits=%d", m, len(exec.submits))
		}
		waited := false
		for _, wait := range report.Waits {
			waited = waited || wait.Reason == "approval_missing"
		}
		if !waited {
			t.Fatalf("missing approval did not wait: %+v", report)
		}
	})
	t.Run("missing approval still earns its bounded pass", func(t *testing.T) {
		fx, host, owner := policyReviewSeed(t, 41)
		p, err := fx.db.PublicationByAssignment(ctx, owner.ID)
		if err != nil {
			t.Fatal(err)
		}
		// The clean acceptance lands first: the serving pass must still
		// dispatch exactly one bounded policy-required review for the
		// missing approval instead of treating the head as accepted, and
		// it must open no merge row without its approval.
		if _, err := dbAdmitCleanOnStored(t, ctx, fx.db, p, strings.Repeat("4", 32)); err != nil {
			t.Fatalf("clean admit: %v", err)
		}
		fx.coord.Reviews = &fakeReviewer{}
		fx.coord.Checks = happyCheckObserver()
		if report := fx.coord.CheckPass(ctx); len(report.Errors) != 0 {
			t.Fatalf("check pass: %+v", report)
		}
		if len(host.launches) != 1 {
			t.Fatalf("clean acceptance waived its bounded policy pass: launches=%d", len(host.launches))
		}
		after, err := fx.db.PublicationByAssignment(ctx, owner.ID)
		if err != nil {
			t.Fatal(err)
		}
		cycle := after.ReviewCycles[len(after.ReviewCycles)-1]
		if cycle.TriggerDecision != policyReviewDecision {
			t.Fatalf("bounded pass lost its separately recorded cause: %q", cycle.TriggerDecision)
		}
		if _, err := fx.db.MergeByPublication(ctx, after.ID); err == nil {
			t.Fatal("merge row opened without its required approval")
		}
	})
}

// dbAdmitCleanOnStored journals one clean acceptance on a stored
// publication without disturbing its recorded cycles.
func dbAdmitCleanOnStored(t *testing.T, ctx context.Context, db *store.Store, p factory.Publication, reviewer string) (factory.DeliveryAcceptance, error) {
	t.Helper()
	owner, err := db.Assignment(ctx, p.AssignmentID)
	if err != nil {
		return factory.DeliveryAcceptance{}, err
	}
	digest, err := factory.FindingSetDigest(nil)
	if err != nil {
		return factory.DeliveryAcceptance{}, err
	}
	cycle := factory.ReviewCycle{
		PublicationID: p.ID, AttemptRoot: owner.AttemptRoot,
		ID:                 factory.ReviewCycleID(p.ID, owner.AttemptRoot, 9, "review"),
		Phase:              "review",
		TriggerDecision:    "initial-review",
		TriggerReason:      "first review of the published head",
		Head:               p.Candidate,
		BaseOID:            p.PRCreate.BaseOID,
		FindingDigest:      digest,
		ReviewerAssignment: reviewer,
		Round:              9,
	}
	p.ReviewCycles = append(p.ReviewCycles, cycle)
	content, err := factory.FindingContentDigest(nil)
	if err != nil {
		return factory.DeliveryAcceptance{}, err
	}
	coverage, err := factory.ClearCoverageDigest(p.Candidate, p.PRCreate.BaseOID, content)
	if err != nil {
		return factory.DeliveryAcceptance{}, err
	}
	published, acceptance, err := AdmitClearReview(ClearAdmissionInput{
		Publication: p,
		Cycle:       cycle,
		Report: factory.ClearReviewReport{
			CycleID: cycle.ID, Conclusion: factory.ClearReviewConclusion,
			Head: p.Candidate, BaseOID: p.PRCreate.BaseOID, CoverageDigest: coverage,
		},
		Result: factory.Result{
			Status: "completed", Summary: "nothing found", Candidate: p.Candidate,
			ReviewPassed: true, Findings: []string{},
		},
		ScopeDigest: content, PolicyRevision: "policy-revision-9", CheckDigest: content,
		ReviewerRunID: reviewer, DiffBase: p.PRCreate.BaseOID,
	})
	if err != nil {
		return factory.DeliveryAcceptance{}, err
	}
	published.Revision = p.Revision + 1
	if err := db.UpdatePublication(ctx, published); err != nil {
		return factory.DeliveryAcceptance{}, err
	}
	return acceptance, nil
}
