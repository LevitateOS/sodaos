package control

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
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

func sealedDecisionRecord() store.ReassessmentRecord {
	title, body := "sealed title", "sealed body with substance"
	evidence, err := factory.CompletionIssueEvidence{
		Index: 4, Title: title, Body: body, Dependencies: []string{"2"},
		TitleDigest: issueDigest(title), ContentDigest: issueDigest(body),
	}.CanonicalData()
	if err != nil {
		panic(err)
	}
	parsed, err := factory.ParseCompletionIssueEvidence(evidence)
	if err != nil {
		panic(err)
	}
	return store.ReassessmentRecord{
		Cycle: factory.ReassessmentCycle{
			EventID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
			State: factory.ReassessmentDeciding, Repository: 7, Generation: 1,
		},
		Completion: factory.CompletionEvent{
			PublicationID: strings.Repeat("5", 32), MergedCommit: strings.Repeat("c", 40),
			ClosingOccurrence: "501", AcceptanceID: "d" + strings.Repeat("a", 24),
			ID: strings.Repeat("1", 32), Repository: 7, Issue: 3, Generation: 1,
		},
		Local: factory.CompletionLocal{
			SourceCommit: strings.Repeat("c", 40), PreparationDigest: strings.Repeat("d", 64),
			ScopeDigest: strings.Repeat("e", 64), AuthorityFingerprint: "authority",
		},
		Evidence:            []factory.CompletionIssueEvidence{parsed},
		InventoryDigest:     strings.Repeat("a", 64),
		EvidenceFingerprint: strings.Repeat("b", 64),
		Issue:               3, EvidenceIssues: 1, EvidenceRevision: 41,
	}
}

// The decision input carries the sealed substance: the completed
// change, the local bindings and every issue body, within the
// pre-spend bound.
func TestDecisionStateFromRecordCarriesSubstance(t *testing.T) {
	state, err := DecisionStateFromRecord(sealedDecisionRecord())
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if state.CompletionID != strings.Repeat("1", 32) || state.Issues != 1 {
		t.Fatalf("state=%+v, want the sealed cycle identity and count", state)
	}
	if state.Completion.MergedCommit != strings.Repeat("c", 40) || state.Completion.Issue != 3 {
		t.Fatalf("completion=%+v, want the sealed completed change", state.Completion)
	}
	if !state.Local.Sealed() {
		t.Fatalf("local=%+v, want the sealed local bindings", state.Local)
	}
	if len(state.Evidence) != 1 || state.Evidence[0].Title != "sealed title" ||
		state.Evidence[0].Body != "sealed body with substance" || len(state.Evidence[0].Dependencies) != 1 {
		t.Fatalf("evidence=%+v, want the sealed issue substance", state.Evidence)
	}
}

func TestDecisionStateFromRecordRefusesUnsealed(t *testing.T) {
	record := sealedDecisionRecord()
	record.InventoryDigest = ""
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("unsubstantial state built from an unsealed record")
	}
	record = sealedDecisionRecord()
	record.Evidence = nil
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("hash-only state built from a bodiless record")
	}
	record = sealedDecisionRecord()
	record.EvidenceIssues = 2
	if _, err := DecisionStateFromRecord(record); err == nil {
		t.Fatal("state built from bodies outside their sealed count")
	}
}

func TestDecisionFingerprintBindsBodies(t *testing.T) {
	state, err := DecisionStateFromRecord(sealedDecisionRecord())
	if err != nil {
		t.Fatal(err)
	}
	edited := state
	edited.Evidence = append([]factory.CompletionIssueEvidence(nil), state.Evidence...)
	edited.Evidence[0].Body = "edited body after the seal"
	if fingerprintOf(edited) == fingerprintOf(state) {
		t.Fatal("edited bodies share the sealed decision cache entry")
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
