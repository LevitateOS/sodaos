package factory

import "testing"

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
