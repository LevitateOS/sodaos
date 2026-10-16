package control

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func jevTestInput(t *testing.T, carried []factory.Finding) (JevAdmissionInput, string) {
	t.Helper()
	publicationID, attemptRoot, reviewer, head, base, diffBase := clearTestIDs()
	cycle := clearTestCycle(t, publicationID, attemptRoot, reviewer, head, base, 1, carried)
	scope, err := factory.FindingContentDigest(nil)
	if err != nil {
		t.Fatal(err)
	}
	check, err := factory.FindingContentDigest([]factory.Finding{correctionTestFinding(9, false, factory.FindingFixVerification)})
	if err != nil {
		t.Fatal(err)
	}
	fingerprintInput := factory.DeliveryFingerprintInput{
		Reviews:       []string{reviewer},
		Conversations: []string{strings.Repeat("2", 64)},
		Constraints:   []string{"protected-files:docs/public"},
		AttemptRoot:   attemptRoot,
		PublicationID: publicationID,
		ScopeDigest:   scope,
		FixDigest:     scope,
		DiffDigest:    scope,
		CheckDigest:   check,
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
	answer := factory.DeliveryAnswer{
		Probabilities: map[string]float64{factory.DeliveryReviewAgain: 0.1, factory.DeliveryReadyToMerge: 0.9},
		Fingerprint:   fingerprint,
		Selected:      factory.DeliveryReadyToMerge,
		Model:         "jev-test",
		Probability:   0.9,
		Confidence:    0.9,
	}
	decision := factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReady, ReviewsFit: true}
	if err := decision.Validate(); err != nil {
		t.Fatal(err)
	}
	return JevAdmissionInput{
		Publication: factory.Publication{
			ID: publicationID, Candidate: head,
			PRCreate:     factory.PublicationOperation{BaseOID: base},
			ReviewCycles: []factory.ReviewCycle{cycle},
		},
		Cycle:             cycle,
		Carried:           carried,
		FingerprintInput:  fingerprintInput,
		Fingerprint:       fingerprint,
		Decision:          decision,
		DecisionID:        strings.Repeat("9", 32),
		ScopeDigest:       scope,
		PolicyRevision:    "policy-revision-9",
		CheckDigest:       check,
		DiffBase:          diffBase,
		ResolutionIntents: []string{correctionTestKey(7)},
	}, fingerprint
}

// TestAdmitJevAcceptanceFundsOnlyReady proves the D08 positive-evidence
// leg of the union constructor: exactly one funded ready_to_merge
// decision over the cycle's exact evidence creates jev_acceptance with
// its funded DecisionID and resolution intents; anything else refuses
// without journaling.
func TestAdmitJevAcceptanceFundsOnlyReady(t *testing.T) {
	carried := []factory.Finding{correctionTestFinding(0, true, factory.FindingFixCode)}
	in, _ := jevTestInput(t, carried)
	published, acceptance, err := AdmitJevAcceptance(in)
	if err != nil {
		t.Fatalf("funded ready decision refused: %v", err)
	}
	if acceptance.Kind != factory.AcceptanceJevAcceptance {
		t.Fatalf("admitted acceptance is not funded: %+v", acceptance)
	}
	if acceptance.DecisionID != in.DecisionID || len(acceptance.ResolutionIntents) != 1 {
		t.Fatalf("funded acceptance lost its decision identity or intents: %+v", acceptance)
	}
	if acceptance.OriginReceipt != in.Cycle.ReviewerAssignment {
		t.Fatalf("funded acceptance lost its independent reviewer run: %+v", acceptance)
	}
	content, err := factory.FindingContentDigest(carried)
	if err != nil {
		t.Fatal(err)
	}
	if acceptance.FindingDigest != content {
		t.Fatal("funded acceptance binds no exact complete content digest")
	}
	if err := acceptance.Validate(); err != nil {
		t.Fatalf("admitted acceptance fails validation: %v", err)
	}
	if len(published.DeliveryAcceptances) != 1 || published.DeliveryAcceptances[0].ID != acceptance.ID {
		t.Fatal("admitted acceptance was not journaled")
	}
	// Replay of the same admission against the journaled publication
	// rebuilds the same entry instead of appending twice.
	replay := in
	replay.Publication = published
	if _, _, err := AdmitJevAcceptance(replay); err == nil {
		t.Fatal("duplicate funded admission appended twice")
	}

	// Only positive evidence creates acceptance: review_again,
	// judgment_attention and unavailable answers all refuse.
	for name, mutate := range map[string]func(*JevAdmissionInput){
		"review_again": func(next *JevAdmissionInput) {
			answer := next.Decision.Answer
			answer.Selected, answer.Probabilities, answer.Probability = factory.DeliveryReviewAgain,
				map[string]float64{factory.DeliveryReviewAgain: 0.9, factory.DeliveryReadyToMerge: 0.1},
				0.9
			next.Decision = factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveReview, ReviewsFit: true}
		},
		"attention": func(next *JevAdmissionInput) {
			answer := next.Decision.Answer
			answer.Probabilities, answer.Probability, answer.Confidence = map[string]float64{factory.DeliveryReviewAgain: 0.5, factory.DeliveryReadyToMerge: 0.5},
				0.9, 0.9
			next.Decision = factory.DeliveryDecision{Answer: answer, Effective: factory.DeliveryEffectiveAttention, ReviewsFit: true}
		},
		"stale fingerprint": func(next *JevAdmissionInput) {
			next.Fingerprint = strings.Repeat("0", 64)
		},
		"another round": func(next *JevAdmissionInput) {
			next.FingerprintInput.Round = 2
			fingerprint, err := factory.DeliveryFingerprint(next.FingerprintInput)
			if err != nil {
				t.Fatal(err)
			}
			next.Fingerprint = fingerprint
			next.Decision.Answer.Fingerprint = fingerprint
		},
		"missing funding": func(next *JevAdmissionInput) {
			next.DecisionID = ""
		},
		"unrecorded cycle": func(next *JevAdmissionInput) {
			next.Publication.ReviewCycles = nil
		},
	} {
		next, _ := jevTestInput(t, carried)
		mutate(&next)
		if _, _, err := AdmitJevAcceptance(next); err == nil {
			t.Fatalf("%s created acceptance", name)
		}
		if len(next.Publication.DeliveryAcceptances) != 0 {
			t.Fatalf("%s journaled its refusal", name)
		}
	}
}
