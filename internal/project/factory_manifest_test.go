package project

import (
	"strings"
	"testing"
	"time"
)

// T12/T16 wire scope: the immutable execution manifest binds plan bytes,
// invocation, phase/run/deadline/profile and child limit for implement
// runs. Every binding refuses independently.
func TestExecutionManifestRoundTripAndRunBinding(t *testing.T) {
	deadline := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	plan := []byte(`{"tasks":[],"lanes":[]}`)
	manifest := ExecutionManifest{
		Plan: plan, PlanDigest: PlanDigest(plan),
		Invocation: strings.Repeat("1", 32), Phase: ExecutionImplement,
		Run: strings.Repeat("a", 32), Deadline: deadline,
		Effort: TaskEffortXHigh, Contributor: true, MaxChildren: 3,
	}
	data, err := MarshalExecutionManifest(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	parsed, err := ParseExecutionManifest(data)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	prompt := []byte("task")
	run := FactoryRun{
		Deadline: deadline, Actor: 1001,
		ID: strings.Repeat("a", 32), Project: "p" + strings.Repeat("b", 24),
		Role: RoleCoder, Preparation: "f" + strings.Repeat("c", 24),
		Harness: FactoryHarnessMuse, HarnessVers: "1.4.9",
		Assignment: FactoryPromptDigest(prompt), SourceCommit: strings.Repeat("f", 40),
		Connection: "subscription", ExecutionPhase: ExecutionImplement,
		ManifestDigest: ExecutionManifestDigest(data),
	}
	if err := parsed.ValidateManifestForRun(run); err != nil {
		t.Fatalf("manifest does not bind its run: %v", err)
	}
	launch := FactoryLaunch{Run: run, Prompt: prompt, HarnessSHA256: strings.Repeat("0", 64), ExecutionManifest: data}
	if err := launch.Validate(); err != nil {
		t.Fatalf("launch with bound manifest refuses: %v", err)
	}
}

func TestExecutionManifestRefusals(t *testing.T) {
	deadline := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	good := func() ExecutionManifest {
		plan := []byte(`{"tasks":[]}`)
		return ExecutionManifest{
			Plan: plan, PlanDigest: PlanDigest(plan),
			Invocation: strings.Repeat("1", 32), Phase: ExecutionImplement,
			Run: strings.Repeat("a", 32), Deadline: deadline,
			Effort: TaskEffortLow, MaxChildren: 3,
		}
	}
	cases := map[string]func(*ExecutionManifest){
		"empty plan":         func(m *ExecutionManifest) { m.Plan, m.PlanDigest = nil, "" },
		"plan digest drift":  func(m *ExecutionManifest) { m.PlanDigest = strings.Repeat("0", 64) },
		"non-implement":      func(m *ExecutionManifest) { m.Phase = ExecutionInvestigate },
		"bad run":            func(m *ExecutionManifest) { m.Run = "short" },
		"bad invocation":     func(m *ExecutionManifest) { m.Invocation = "" },
		"zero deadline":      func(m *ExecutionManifest) { m.Deadline = time.Time{} },
		"bad effort":         func(m *ExecutionManifest) { m.Effort = "turbo" },
		"zero children":      func(m *ExecutionManifest) { m.MaxChildren = 0 },
		"children above cap": func(m *ExecutionManifest) { m.MaxChildren = 5 },
	}
	for name, mutate := range cases {
		m := good()
		mutate(&m)
		data, err := MarshalExecutionManifest(m)
		if err != nil {
			continue
		}
		if _, err := ParseExecutionManifest(data); err == nil {
			t.Fatalf("%s parsed", name)
		}
	}
	// Unknown fields and trailing JSON refuse.
	plan := []byte(`{}`)
	m := good()
	m.Plan, m.PlanDigest = plan, PlanDigest(plan)
	data, err := MarshalExecutionManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	withUnknown := append(append([]byte{}, data[:len(data)-1]...), []byte(`,"future":1}`)...)
	if _, err := ParseExecutionManifest(withUnknown); err == nil {
		t.Fatal("unknown manifest field parsed")
	}
	if _, err := ParseExecutionManifest(append(data, ' ')); err != nil {
		// Single trailing space is JSON whitespace: allowed.
		t.Fatalf("whitespace suffix refuses: %v", err)
	}
	if _, err := ParseExecutionManifest(append(data, []byte(` {}`)...)); err == nil {
		t.Fatal("trailing JSON parsed")
	}
	// Contributor mode binds the Muse harness at run binding.
	m = good()
	m.Contributor = true
	run := FactoryRun{
		Deadline: deadline, Actor: 1001,
		ID: strings.Repeat("a", 32), Project: "p" + strings.Repeat("b", 24),
		Role: RoleCoder, Preparation: "f" + strings.Repeat("c", 24),
		Harness: FactoryHarnessCodex, HarnessVers: "0.161.0",
		Assignment: strings.Repeat("e", 64), SourceCommit: strings.Repeat("f", 40),
		Connection: "subscription", ExecutionPhase: ExecutionImplement,
	}
	if err := m.ValidateManifestForRun(run); err == nil {
		t.Fatal("contributor manifest bound a codex run")
	}
}
