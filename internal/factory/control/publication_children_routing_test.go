package control

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

// TestRouteDeliveryFeedbackBeforeSpending proves the T44 routing leg: a
// concrete unfixed defect takes the bounded correction; complete proposed
// fixes without a later clear review, optional-only, disputed and
// repetitive feedback take the funded D08 Jev decision before another
// coding or review run, and optional advice never forces a correction.
func TestRouteDeliveryFeedbackBeforeSpending(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   deliveryFeedback
		want deliveryRoute
	}{
		{"first inspection reviews", deliveryFeedback{}, routeReview},
		{
			"owed policy approval reviews with cause",
			deliveryFeedback{PolicyOwed: true},
			routePolicyReview,
		},
		{
			"concrete defect corrects",
			deliveryFeedback{
				HasReview: true, ReviewVerdict: "request-changes", ChecksVerdict: factory.CheckFailed,
			},
			routeCorrection,
		},
		{
			"unresolved defect corrects again",
			deliveryFeedback{
				HasReview: true, ReviewVerdict: "request-changes", ChecksVerdict: factory.CheckPass,
				NewestDispositions: []string{factory.DispositionUnresolved, factory.DispositionDeclinedOptional},
			},
			routeCorrection,
		},
		{
			"failing checks correct despite approval",
			deliveryFeedback{
				HasReview: true, ReviewVerdict: "approve", ChecksVerdict: factory.CheckFailed,
				NewestDispositions: []string{factory.DispositionFixed},
			},
			routeCorrection,
		},
		{
			"dispute invokes the decision, never a second coder run",
			deliveryFeedback{
				HasReview: true, ReviewVerdict: "request-changes", ChecksVerdict: factory.CheckPass,
				NewestDispositions: []string{factory.DispositionFixed, factory.DispositionDisputed},
			},
			routeJev,
		},
		{
			"dispute dominates unresolved",
			deliveryFeedback{
				HasReview: true, ReviewVerdict: "request-changes", ChecksVerdict: factory.CheckFailed,
				NewestDispositions: []string{factory.DispositionUnresolved, factory.DispositionDisputed},
			},
			routeJev,
		},
		{
			"optional-only never forces a coder run",
			deliveryFeedback{
				HasReview: true, ReviewVerdict: "request-changes", ChecksVerdict: factory.CheckPass,
				NewestDispositions: []string{factory.DispositionDeclinedOptional},
			},
			routeJev,
		},
		{
			"all-fixed yet still rejected is repetitive",
			deliveryFeedback{
				HasReview: true, ReviewVerdict: "request-changes", ChecksVerdict: factory.CheckPass,
				NewestDispositions: []string{factory.DispositionFixed, factory.DispositionFixed},
			},
			routeJev,
		},
	} {
		if got := routeDeliveryFeedback(tc.in); got != tc.want {
			t.Fatalf("%s routed %d, want %d", tc.name, got, tc.want)
		}
	}
}

// TestPolicyReviewDedupAndCharge proves the T44 policy leg: the bounded
// policy-required pass is owed exactly while the configured reviewer has
// not supplied a committed approval, it is deduplicated by exact head,
// actor and policy blocker, and the recorded cause stays separate from a
// discretionary review.
func TestPolicyReviewDedupAndCharge(t *testing.T) {
	head, base := strings.Repeat("e", 40), strings.Repeat("b", 40)
	publicated := func() factory.Publication {
		return factory.Publication{
			ID: strings.Repeat("1", 32), Candidate: head,
			PRCreate: factory.PublicationOperation{BaseOID: base},
		}
	}
	// No operation yet: the approval is missing and no pass is recorded,
	// so the cause is owed.
	p := publicated()
	if !missingPolicyApproval(p, 6) {
		t.Fatal("missing approval not owed")
	}
	if !policyCauseOwed(p, factory.CheckPass, 6) {
		t.Fatal("policy cause not owed with passing checks")
	}
	// Any other blocker state waits instead of spending a review that
	// cannot satisfy policy.
	if policyCauseOwed(p, factory.CheckFailed, 6) {
		t.Fatal("policy cause owed while checks fail")
	}
	if policyCauseOwed(p, "", 6) {
		t.Fatal("policy cause owed without check evidence")
	}
	if policyCauseOwed(p, factory.CheckPass, 0) {
		t.Fatal("policy cause owed without a configured reviewer")
	}
	// A committed APPROVED operation by the configured reviewer on the
	// exact head and base satisfies it; another actor's approval, a
	// blocking request, or a stale head does not.
	approved := func(actor int64, event, opHead, opBase string) factory.ReviewOperation {
		return factory.ReviewOperation{
			Work: factory.ReviewWork{
				ActorID: actor, Event: event, HeadOID: opHead, BaseOID: opBase,
			},
			Outcome: factory.OperationOutcome{Effect: factory.OpEffectCommitted, Completion: factory.OpCompletionComplete},
		}
	}
	p.ReviewOperations = []factory.ReviewOperation{approved(6, "APPROVED", head, base)}
	if missingPolicyApproval(p, 6) {
		t.Fatal("supplied approval still owed")
	}
	for _, other := range []factory.ReviewOperation{
		approved(7, "APPROVED", head, base),
		approved(6, "REQUEST_CHANGES", head, base),
		approved(6, "APPROVED", strings.Repeat("f", 40), base),
	} {
		p.ReviewOperations = []factory.ReviewOperation{other}
		if !missingPolicyApproval(p, 6) {
			t.Fatalf("non-supplying operation satisfied the cause: %+v", other.Work)
		}
	}
	// Recording the bounded pass deduplicates it: the same head, actor
	// and blocker never owes twice, while a new head owes again.
	p.ReviewOperations = nil
	p.ReviewCycles = []factory.ReviewCycle{{
		PublicationID: p.ID, AttemptRoot: strings.Repeat("2", 32),
		ID: factory.ReviewCycleID(p.ID, strings.Repeat("2", 32), 1, "review"), Phase: "review",
		TriggerDecision: policyReviewDecision, TriggerReason: policyReviewReason(6, head),
		Head: head, BaseOID: base, FindingDigest: strings.Repeat("3", 64),
		ReviewerAssignment: strings.Repeat("4", 32), Round: 1,
	}}
	if !policyReviewRecorded(p, head, 6) {
		t.Fatal("recorded policy pass not deduplicated")
	}
	if policyCauseOwed(p, factory.CheckPass, 6) {
		t.Fatal("recorded policy pass still owed")
	}
	if policyReviewRecorded(p, strings.Repeat("f", 40), 6) {
		t.Fatal("policy pass deduplicated across heads")
	}
	if policyReviewRecorded(p, head, 7) {
		t.Fatal("policy pass deduplicated across actors")
	}
	if got := policyReviewReason(6, head); !strings.HasPrefix(got, policyReviewBlockerApprovals+" ") || !strings.Contains(got, "6") {
		t.Fatalf("policy cause does not name its blocker and actor: %q", got)
	}
}

// TestNewestCurrentHeadDispositionsBindsCurrentHead proves journals on
// superseded heads never route the current one: only the newest journal
// whose cycle inspected the exact current head supplies dispositions.
func TestNewestCurrentHeadDispositionsBindsCurrentHead(t *testing.T) {
	head, base := strings.Repeat("e", 40), strings.Repeat("b", 40)
	old := strings.Repeat("d", 40)
	root := strings.Repeat("2", 32)
	cycle := func(round int, cycleHead string) factory.ReviewCycle {
		return factory.ReviewCycle{
			PublicationID: strings.Repeat("1", 32), AttemptRoot: root,
			ID: factory.ReviewCycleID(strings.Repeat("1", 32), root, round, "review"), Phase: "review",
			TriggerDecision: "correction-after-review", TriggerReason: "address findings",
			Head: cycleHead, BaseOID: base, FindingDigest: strings.Repeat("3", 64),
			ReviewerAssignment:   strings.Repeat("4", 32),
			CorrectionAssignment: strings.Repeat("5", 32), Round: round,
		}
	}
	journal := func(cycleID, disposition string) factory.CorrectionJournal {
		return factory.CorrectionJournal{
			CycleID: cycleID, RunID: strings.Repeat("6", 32),
			Report: factory.CorrectionReport{
				SetDigest: strings.Repeat("3", 64), NewCandidate: strings.Repeat("d", 40),
				Findings: []factory.Finding{correctionTestFinding(1, true, factory.FindingFixCode)},
				Dispositions: []factory.CorrectionDisposition{
					{Key: correctionTestKey(1), Disposition: disposition},
				},
			},
		}
	}
	cycles := []factory.ReviewCycle{cycle(1, old), cycle(2, head)}
	p := factory.Publication{
		ID: strings.Repeat("1", 32), Candidate: head,
		PRCreate:     factory.PublicationOperation{BaseOID: base},
		ReviewCycles: cycles,
		CorrectionJournals: []factory.CorrectionJournal{
			journal(cycles[0].ID, factory.DispositionDisputed),
			journal(cycles[1].ID, factory.DispositionUnresolved),
		},
	}
	got := newestCurrentHeadDispositions(p)
	if len(got) != 1 || got[0] != factory.DispositionUnresolved {
		t.Fatalf("superseded-head journal routed the current head: %+v", got)
	}
	stale := p
	stale.CorrectionJournals = stale.CorrectionJournals[:1]
	if newestCurrentHeadDispositions(stale) != nil {
		t.Fatal("superseded-head journal alone supplied dispositions")
	}
}
