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

func TestJudgmentFingerprintBindsSevenPurposes(t *testing.T) {
	purposes := JudgmentPurposes()
	if len(purposes) != 7 {
		t.Fatalf("purposes=%d, want 7", len(purposes))
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
