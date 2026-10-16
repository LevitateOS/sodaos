package control

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func decidingCycle() factory.ReassessmentCycle {
	return factory.ReassessmentCycle{
		EventID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
		State: factory.ReassessmentDeciding, Repository: 7, Generation: 1,
	}
}

func decisionAnswer(selected string, confidence float64) factory.PriorityAnswer {
	return factory.PriorityAnswer{Selected: selected, Confidence: confidence}
}

// Every completion gets its own funded decision, including an empty
// inventory: valid confidence records, anything else holds.
func TestApplyReassessmentAnswerRecordsValidContinue(t *testing.T) {
	updated, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer(factory.ReassessmentOptionContinue, 0.9))
	if err != nil || updated.Decision != factory.ReassessmentOptionContinue {
		t.Fatalf("updated=%+v err=%v, want a recorded continue", updated, err)
	}
}

func TestApplyReassessmentAnswerRecordsBoundaryConfidence(t *testing.T) {
	updated, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer(factory.ReassessmentOptionReassess, 0.50))
	if err != nil || updated.Decision != factory.ReassessmentOptionReassess {
		t.Fatalf("updated=%+v err=%v, want a recorded reassess at the bound", updated, err)
	}
}

func TestApplyReassessmentAnswerRefusesLowConfidence(t *testing.T) {
	if _, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer(factory.ReassessmentOptionContinue, 0.49)); err == nil {
		t.Fatal("low-confidence continue recorded instead of holding")
	}
	if _, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer(factory.ReassessmentOptionReassess, 0.0)); err == nil {
		t.Fatal("zero-confidence reassess recorded instead of holding")
	}
}

func TestApplyReassessmentAnswerRefusesUnknownOption(t *testing.T) {
	if _, err := applyReassessmentAnswer(decidingCycle(), decisionAnswer("maybe", 0.95)); err == nil {
		t.Fatal("unknown option recorded instead of holding")
	}
}

func TestApplyReassessmentAnswerRefusesOutsideDeciding(t *testing.T) {
	cycle := decidingCycle()
	cycle.State = factory.ReassessmentCollecting
	if _, err := applyReassessmentAnswer(cycle, decisionAnswer(factory.ReassessmentOptionContinue, 0.9)); err == nil {
		t.Fatal("decision recorded outside its deciding state")
	}
}

func TestReassessmentDecisionOptionsAreExactlyTwo(t *testing.T) {
	options := reassessmentDecisionOptions()
	if len(options) != 2 {
		t.Fatalf("options=%v, want exactly reassess and continue", options)
	}
	if _, ok := options[factory.ReassessmentOptionReassess]; !ok {
		t.Fatal("reassess option missing")
	}
	if _, ok := options[factory.ReassessmentOptionContinue]; !ok {
		t.Fatal("continue option missing")
	}
}

func TestDecisionFingerprintIsStableAndPerCompletion(t *testing.T) {
	state := ReassessmentDecisionState{
		CompletionID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
		InventoryDigest: strings.Repeat("a", 64), EvidenceFingerprint: strings.Repeat("b", 64),
		Repository: 7, Generation: 1, Issues: 3,
	}
	first, second := fingerprintOf(state), fingerprintOf(state)
	if first == "" || first != second || !factory.ValidDigest(first) {
		t.Fatal("decision fingerprint unstable or malformed")
	}
	other := state
	other.CompletionID = strings.Repeat("3", 32)
	other.CycleID = strings.Repeat("4", 32)
	if fingerprintOf(other) == first {
		t.Fatal("two completions share one decision cache entry")
	}
}
