package project

import (
	"strings"
	"testing"
)

func promptEntryFixture(intent string, revision uint64, state string) FactoryPromptEntry {
	prompt := []byte("investigate the checkout and write a plan")
	return FactoryPromptEntry{
		Intent:      intent,
		Revision:    revision,
		IntentState: state,
		Project:     "p" + strings.Repeat("b", 24),
		Actor:       7,
		Connection:  "conn-1",
		Role:        RoleCoder,
		Preparation: "f" + strings.Repeat("a", 24),
		Harness:     FactoryHarnessMuse,
		HarnessVers: "1.4.9",
		Phase:       ExecutionInvestigate,
		Prompt:      prompt,
		Assignment:  FactoryPromptDigest(prompt),
	}
}

func TestPromptEntryAdmitsFirstInvestigation(t *testing.T) {
	in := promptEntryFixture("d"+strings.Repeat("1", 24), 1, PromptIntentAccepted)
	if err := in.Validate(); err != nil {
		t.Fatalf("valid entry refused: %v", err)
	}
	record, replay, err := AdmitPromptEntry(nil, in)
	if err != nil || replay {
		t.Fatalf("first entry must admit without replay: %+v %v", record, err)
	}
	if record.Intent != in.Intent || record.Revision != 1 || record.Assignment != in.Assignment {
		t.Fatalf("record does not bind the entry: %+v", record)
	}
}

func TestPromptEntryRefusesMisuse(t *testing.T) {
	intent := "d" + strings.Repeat("2", 24)
	cases := map[string]func(*FactoryPromptEntry){
		"bad intent":      func(e *FactoryPromptEntry) { e.Intent = "nope" },
		"zero revision":   func(e *FactoryPromptEntry) { e.Revision = 0 },
		"draft intent":    func(e *FactoryPromptEntry) { e.IntentState = PromptIntentDraft },
		"declined":        func(e *FactoryPromptEntry) { e.IntentState = PromptIntentDeclined },
		"rejected":        func(e *FactoryPromptEntry) { e.IntentState = PromptIntentRejected },
		"abandoned":       func(e *FactoryPromptEntry) { e.IntentState = PromptIntentAbandoned },
		"empty project":   func(e *FactoryPromptEntry) { e.Project = "" },
		"zero actor":      func(e *FactoryPromptEntry) { e.Actor = 0 },
		"empty conn":      func(e *FactoryPromptEntry) { e.Connection = "" },
		"wrong phase":     func(e *FactoryPromptEntry) { e.Phase = ExecutionImplement },
		"bad role":        func(e *FactoryPromptEntry) { e.Role = "soda-admin" },
		"bad prep":        func(e *FactoryPromptEntry) { e.Preparation = "checkout" },
		"bad harness":     func(e *FactoryPromptEntry) { e.Harness = "shell" },
		"bad version":     func(e *FactoryPromptEntry) { e.HarnessVers = "" },
		"empty prompt":    func(e *FactoryPromptEntry) { e.Prompt = nil; e.Assignment = FactoryPromptDigest(nil) },
		"oversize":        func(e *FactoryPromptEntry) { e.Prompt = make([]byte, MaxFactoryPrompt+1) },
		"digest mismatch": func(e *FactoryPromptEntry) { e.Assignment = strings.Repeat("0", 64) },
	}
	for name, mutate := range cases {
		in := promptEntryFixture(intent, 1, PromptIntentAccepted)
		mutate(&in)
		if err := in.Validate(); err == nil {
			t.Fatalf("%s: misuse admitted", name)
		}
		if _, _, err := AdmitPromptEntry(nil, in); err == nil {
			t.Fatalf("%s: admission admitted", name)
		}
	}
}

func TestPromptEntryExactlyOncePerIntent(t *testing.T) {
	intent := "d" + strings.Repeat("3", 24)
	first := promptEntryFixture(intent, 1, PromptIntentAccepted)
	record, _, err := AdmitPromptEntry(nil, first)
	if err != nil {
		t.Fatalf("first entry refused: %v", err)
	}
	// Same revision with identical bytes replays: no duplicate investigation.
	dup, replay, err := AdmitPromptEntry(&record, first)
	if err != nil || !replay || dup != record {
		t.Fatalf("replay must return the recorded investigation: %+v %v %v", dup, replay, err)
	}
	// Same revision with new bytes is an attempt-revision conflict.
	changed := first
	changed.Prompt = []byte("a different prompt")
	changed.Assignment = FactoryPromptDigest(changed.Prompt)
	if _, _, err := AdmitPromptEntry(&record, changed); err == nil {
		t.Fatal("same-revision new bytes admitted")
	}
	// A higher revision supersedes with its own record.
	next := first
	next.Revision = 2
	next.Prompt = []byte("retry with revised prompt")
	next.Assignment = FactoryPromptDigest(next.Prompt)
	second, replay, err := AdmitPromptEntry(&record, next)
	if err != nil || replay || second.Revision != 2 || second.Assignment != next.Assignment {
		t.Fatalf("higher revision must supersede: %+v %v %v", second, replay, err)
	}
	// The old revision is now stale.
	if _, _, err := AdmitPromptEntry(&second, first); err == nil {
		t.Fatal("stale revision admitted")
	}
	// Another intent never reuses this record.
	other := promptEntryFixture("d"+strings.Repeat("4", 24), 1, PromptIntentAccepted)
	if _, _, err := AdmitPromptEntry(&record, other); err == nil {
		t.Fatal("cross-intent entry admitted")
	}
	// A cancelled record admits nothing further.
	cancelled := record
	cancelled.Cancelled = true
	if _, _, err := AdmitPromptEntry(&cancelled, next); err == nil {
		t.Fatal("cancelled intent admitted")
	}
}
