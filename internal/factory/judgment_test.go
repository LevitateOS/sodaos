package factory

import (
	"math"
	"strings"
	"testing"
)

func testDeliveryFingerprintInput() DeliveryFingerprintInput {
	return DeliveryFingerprintInput{
		Reviews:        []string{strings.Repeat("1", 32)},
		Conversations:  []string{strings.Repeat("2", 64)},
		Constraints:    []string{"protected-files:docs/public"},
		AttemptRoot:    strings.Repeat("3", 32),
		PublicationID:  strings.Repeat("4", 32),
		ScopeDigest:    strings.Repeat("5", 64),
		FixDigest:      strings.Repeat("6", 64),
		DiffDigest:     strings.Repeat("7", 64),
		CheckDigest:    strings.Repeat("8", 64),
		TreeDigest:     strings.Repeat("9", 64),
		Head:           strings.Repeat("a", 40),
		Base:           strings.Repeat("b", 40),
		Model:          "jev-test",
		Rubric:         "ready_to_merge only with full evidence cover",
		PolicyRevision: 3,
		Round:          1,
	}
}

func testDeliveryAnswer(fingerprint string) DeliveryAnswer {
	return DeliveryAnswer{
		Probabilities: map[string]float64{DeliveryReviewAgain: 0.1, DeliveryReadyToMerge: 0.9},
		Fingerprint:   fingerprint,
		Selected:      DeliveryReadyToMerge,
		Model:         "jev-test",
		Probability:   0.9,
		Confidence:    0.85,
	}
}

func TestDeliveryPolicyBoundaries(t *testing.T) {
	fingerprint, err := DeliveryFingerprint(testDeliveryFingerprintInput())
	if err != nil {
		t.Fatal(err)
	}
	ready := testDeliveryAnswer(fingerprint)
	if got := ready.EffectiveAction(true); got != DeliveryEffectiveReady {
		t.Fatalf("ready answer routed %q", got)
	}
	if err := (DeliveryDecision{Answer: ready, Effective: DeliveryEffectiveReady, ReviewsFit: true}).Validate(); err != nil {
		t.Fatalf("ready decision refused: %v", err)
	}
	below := ready
	below.Probability = 0.79
	if got := below.EffectiveAction(true); got != DeliveryEffectiveReview {
		t.Fatalf("below-threshold answer routed %q", got)
	}
	tied := ready
	tied.Probabilities = map[string]float64{DeliveryReviewAgain: 0.5, DeliveryReadyToMerge: 0.5}
	tied.Probability, tied.Confidence = 0.9, 0.9
	if got := tied.EffectiveAction(true); got != DeliveryEffectiveReview {
		t.Fatalf("tied answer routed %q", got)
	}
	// An accepted answer stands on its funded call however little allowance
	// remains; exhaustion routes only the otherwise-branch.
	if got := ready.EffectiveAction(false); got != DeliveryEffectiveReady {
		t.Fatalf("funded ready answer routed %q", got)
	}
	exhausted := ready
	exhausted.Selected = DeliveryReviewAgain
	exhausted.Probabilities = map[string]float64{DeliveryReviewAgain: 0.9, DeliveryReadyToMerge: 0.1}
	if got := exhausted.EffectiveAction(false); got != DeliveryEffectiveExhausted {
		t.Fatalf("exhausted allowance routed %q", got)
	}
	for name, broken := range map[string]func(*DeliveryAnswer){
		"failure":        func(a *DeliveryAnswer) { a.Failure = "provider-down" },
		"bad option":     func(a *DeliveryAnswer) { a.Selected = "maybe" },
		"nan confidence": func(a *DeliveryAnswer) { a.Confidence = math.NaN() },
		"low confidence": func(a *DeliveryAnswer) { a.Confidence = 0.2 },
	} {
		answer := ready
		broken(&answer)
		if name == "low confidence" {
			if got := answer.EffectiveAction(true); got != DeliveryEffectiveReview {
				t.Fatalf("low-confidence answer routed %q", got)
			}
			continue
		}
		if got := answer.EffectiveAction(true); got != DeliveryUnavailable {
			t.Fatalf("%s routed %q, want unavailable", name, got)
		}
		if err := (DeliveryDecision{Answer: answer, Effective: DeliveryEffectiveReview, ReviewsFit: true}).Validate(); err == nil {
			t.Fatalf("%s decision accepted", name)
		}
	}
	// A clean first review still consults: empty finding scope fingerprints
	// and answers normally instead of bypassing judgment.
	clean := testDeliveryFingerprintInput()
	clean.Conversations = nil
	if _, err := DeliveryFingerprint(clean); err != nil {
		t.Fatalf("clean-review evidence refused: %v", err)
	}
}

func TestDeliveryFingerprintBindsEvidence(t *testing.T) {
	base, err := DeliveryFingerprint(testDeliveryFingerprintInput())
	if err != nil || !ValidDigest(base) {
		t.Fatalf("delivery fingerprint malformed: %q %v", base, err)
	}
	again, err := DeliveryFingerprint(testDeliveryFingerprintInput())
	if err != nil || again != base {
		t.Fatal("delivery fingerprint unstable")
	}
	mutants := []func(*DeliveryFingerprintInput){
		func(in *DeliveryFingerprintInput) { in.Head = strings.Repeat("c", 40) },
		func(in *DeliveryFingerprintInput) { in.ScopeDigest = strings.Repeat("0", 64) },
		func(in *DeliveryFingerprintInput) { in.CheckDigest = strings.Repeat("0", 64) },
		func(in *DeliveryFingerprintInput) { in.Round = 2 },
		func(in *DeliveryFingerprintInput) { in.Constraints = []string{"other:gate"} },
	}
	for i, mutate := range mutants {
		in := testDeliveryFingerprintInput()
		mutate(&in)
		other, err := DeliveryFingerprint(in)
		if err != nil || other == base {
			t.Fatalf("mutant %d kept fingerprint", i)
		}
	}
	// Set order never affects the fingerprint.
	reordered := testDeliveryFingerprintInput()
	reordered.Constraints = []string{"b:2", "a:1"}
	ordered := testDeliveryFingerprintInput()
	ordered.Constraints = []string{"a:1", "b:2"}
	first, err := DeliveryFingerprint(reordered)
	if err != nil {
		t.Fatal(err)
	}
	second, err := DeliveryFingerprint(ordered)
	if err != nil || first != second {
		t.Fatal("constraint order changed fingerprint")
	}
	empty := testDeliveryFingerprintInput()
	empty.Round = 0
	if _, err := DeliveryFingerprint(empty); err == nil {
		t.Fatal("round-zero evidence accepted")
	}
}

func TestPriorityAnswerRejectsOmittedAndFabricatedChoices(t *testing.T) {
	r := PriorityRequest{Candidates: []PriorityCandidate{{ID: "a"}, {ID: "b"}}}
	a := PriorityAnswer{Selected: "a", Confidence: 1, Model: "synthetic-model", Probabilities: map[string]float64{"a": 1, "b": 0}}
	if err := a.Validate(r); err != nil {
		t.Fatal(err)
	}
	a.Probabilities = map[string]float64{"a": 1, "c": 0}
	if err := a.Validate(r); err == nil {
		t.Fatal("omitted candidate accepted")
	}
	a.Probabilities = map[string]float64{"a": 0.1, "b": 0.9}
	if err := a.Validate(r); err == nil {
		t.Fatal("nonmaximal answer accepted")
	}
	r.Candidates[1].ID = "a"
	if _, err := PrioritySnapshot(r.Candidates); err == nil {
		t.Fatal("duplicate identities accepted")
	}
}

func TestJudgmentFingerprintBindsTenPurposes(t *testing.T) {
	purposes := JudgmentPurposes()
	if len(purposes) != 10 {
		t.Fatalf("purposes=%d, want 10", len(purposes))
	}
	seen := map[JudgmentPurpose]bool{}
	options := map[string]any{"a": "First option", "b": "Second option"}
	for _, purpose := range purposes {
		if seen[purpose] {
			t.Fatal("duplicate purpose")
		}
		seen[purpose] = true
		first, err := JudgmentFingerprint(purpose, map[string]string{"fact": "evidence"}, "Which option fits?", options)
		if err != nil {
			t.Fatal(err)
		}
		second, err := JudgmentFingerprint(purpose, map[string]string{"fact": "evidence"}, "Which option fits?", options)
		if err != nil || first != second || !ValidDigest(first) {
			t.Fatal("fingerprint unstable or malformed")
		}
	}
	changed := func(purpose JudgmentPurpose, state any, question string, opts map[string]any) string {
		fingerprint, err := JudgmentFingerprint(purpose, state, question, opts)
		if err != nil {
			t.Fatal(err)
		}
		return fingerprint
	}
	base := changed(JudgmentConsolidation, map[string]string{"fact": "evidence"}, "Which option fits?", options)
	for _, other := range []string{
		changed(JudgmentConsistency, map[string]string{"fact": "evidence"}, "Which option fits?", options),
		changed(JudgmentConsolidation, map[string]string{"fact": "changed"}, "Which option fits?", options),
		changed(JudgmentConsolidation, map[string]string{"fact": "evidence"}, "Which option fits best?", options),
		changed(JudgmentConsolidation, map[string]string{"fact": "evidence"}, "Which option fits?", map[string]any{"a": "First option", "b": "Changed option"}),
	} {
		if other == base {
			t.Fatal("material input change kept fingerprint")
		}
	}
	for _, invalid := range []struct {
		purpose  JudgmentPurpose
		question string
		options  map[string]any
	}{
		{"invented", "Which option fits?", options},
		{JudgmentConsolidation, "", options},
		{JudgmentConsolidation, "Which option fits?", map[string]any{"a": "Only option"}},
		{JudgmentConsolidation, "Which option fits?", map[string]any{"": "Empty identity", "b": "Second"}},
	} {
		if _, err := JudgmentFingerprint(invalid.purpose, "state", invalid.question, invalid.options); err == nil {
			t.Fatalf("invalid judgment accepted: %+v", invalid)
		}
	}
}
