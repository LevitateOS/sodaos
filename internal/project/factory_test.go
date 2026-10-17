package project

import (
	"strings"
	"testing"
	"time"
)

func factoryRunFixture() FactoryRun {
	return FactoryRun{
		Deadline:     time.Now().Add(time.Hour),
		Actor:        1001,
		ID:           strings.Repeat("a", 32),
		Project:      "p" + strings.Repeat("b", 24),
		Role:         RoleCoder,
		Preparation:  "f" + strings.Repeat("c", 24),
		Harness:      FactoryHarnessCodex,
		HarnessVers:  "0.157.1",
		Assignment:   strings.Repeat("d", 64),
		SourceCommit: strings.Repeat("e", 40),
		Connection:   "subscription",
	}
}

func TestFactoryRunValidation(t *testing.T) {
	if err := factoryRunFixture().Validate(); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*FactoryRun){
		"run":         func(r *FactoryRun) { r.ID = "short" },
		"project":     func(r *FactoryRun) { r.Project = "nope" },
		"role":        func(r *FactoryRun) { r.Role = "soda-tester" },
		"preparation": func(r *FactoryRun) { r.Preparation = "bad" },
		"harness":     func(r *FactoryRun) { r.Harness = "other" },
		"version":     func(r *FactoryRun) { r.HarnessVers = "../escape" },
		"assignment":  func(r *FactoryRun) { r.Assignment = "zz" },
		"source":      func(r *FactoryRun) { r.SourceCommit = "zz" },
		"connection":  func(r *FactoryRun) { r.Connection = "" },
		"actor":       func(r *FactoryRun) { r.Actor = 0 },
		"deadline":    func(r *FactoryRun) { r.Deadline = time.Time{} },
	}
	for name, mutate := range mutations {
		run := factoryRunFixture()
		mutate(&run)
		if err := run.Validate(); err == nil {
			t.Fatalf("invalid %s admitted", name)
		}
	}
}

func TestExecutionPhaseVocabulary(t *testing.T) {
	for _, phase := range []string{ExecutionDecompose, ExecutionInvestigate, ExecutionImplement, ExecutionReview, ExecutionCorrect, ExecutionReassess, ExecutionRepair, ExecutionVerify} {
		if !ValidExecutionPhase(phase) {
			t.Fatalf("admitted phase refused: %s", phase)
		}
	}
	if ValidExecutionPhase("publish") {
		t.Fatal("unknown phase admitted")
	}
	if !ExecutionManifestRequired(ExecutionImplement) {
		t.Fatal("implement run skips its manifest")
	}
	if !ExecutionManifestRequired(ExecutionCorrect) {
		t.Fatal("correct run skips its manifest")
	}
	for _, phase := range []string{"", ExecutionDecompose, ExecutionInvestigate, ExecutionReview, ExecutionReassess, ExecutionRepair, ExecutionVerify} {
		if ExecutionManifestRequired(phase) {
			t.Fatalf("single-agent phase stages a manifest: %q", phase)
		}
	}
}

// D13/D15/D16 scope: the reassess/repair 320 KiB prompt exception rides
// the recorded phase; reassess and repair runs require the coder role,
// verify runs the reviewer role; other phases keep the standard bound.
func TestReassessPromptBoundAndRole(t *testing.T) {
	if MaxPromptForPhase(ExecutionReassess) != MaxReassessPrompt {
		t.Fatal("reassess prompt exception missing")
	}
	if MaxPromptForPhase(ExecutionRepair) != MaxReassessPrompt {
		t.Fatal("repair prompt exception missing")
	}
	for _, phase := range []string{"", ExecutionDecompose, ExecutionInvestigate, ExecutionImplement, ExecutionReview, ExecutionCorrect, ExecutionVerify} {
		if MaxPromptForPhase(phase) != MaxFactoryPrompt {
			t.Fatalf("phase %q escapes the standard prompt bound", phase)
		}
	}
	launchFor := func(phase, role string, size int) FactoryLaunch {
		run := factoryRunFixture()
		prompt := []byte(strings.Repeat("p", size))
		run.Assignment = FactoryPromptDigest(prompt)
		run.ExecutionPhase, run.Role = phase, role
		return FactoryLaunch{Run: run, Prompt: prompt, HarnessSHA256: strings.Repeat("f", 64)}
	}
	if err := launchFor(ExecutionReassess, RoleCoder, MaxReassessPrompt).Validate(); err != nil {
		t.Fatalf("bounded reassess launch refused: %v", err)
	}
	if err := launchFor(ExecutionRepair, RoleCoder, MaxReassessPrompt).Validate(); err != nil {
		t.Fatalf("bounded repair launch refused: %v", err)
	}
	if err := launchFor(ExecutionVerify, RoleReviewer, MaxFactoryPrompt).Validate(); err != nil {
		t.Fatalf("bounded verify launch refused: %v", err)
	}
	for name, launch := range map[string]FactoryLaunch{
		"reassess over bound":    launchFor(ExecutionReassess, RoleCoder, MaxReassessPrompt+1),
		"reassess reviewer":      launchFor(ExecutionReassess, RoleReviewer, 64),
		"repair over bound":      launchFor(ExecutionRepair, RoleCoder, MaxReassessPrompt+1),
		"repair reviewer":        launchFor(ExecutionRepair, RoleReviewer, 64),
		"verify coder":           launchFor(ExecutionVerify, RoleCoder, 64),
		"verify at 320 KiB":      launchFor(ExecutionVerify, RoleReviewer, MaxReassessPrompt),
		"implement at 320 KiB":   launchFor(ExecutionImplement, RoleCoder, MaxReassessPrompt),
		"investigate at 320 KiB": launchFor(ExecutionInvestigate, RoleCoder, MaxReassessPrompt),
	} {
		if err := launch.Validate(); err == nil {
			t.Fatalf("launch %s admitted", name)
		}
	}
}

// T16 wire scope: the manifest/phase/digest rule binds every launch.
// Implement and correct runs stage their exact manifest bytes; every other
// phase stages none; digests never drift from bytes.
func TestFactoryLaunchManifestRule(t *testing.T) {
	launchFor := func(phase string, manifest []byte, digest string) FactoryLaunch {
		run := factoryRunFixture()
		prompt := []byte("trivial coding task")
		run.Assignment = FactoryPromptDigest(prompt)
		run.ExecutionPhase, run.ManifestDigest = phase, digest
		return FactoryLaunch{Run: run, Prompt: prompt, HarnessSHA256: strings.Repeat("f", 64), ExecutionManifest: manifest}
	}
	manifest := []byte("staged plan bytes")
	good := launchFor(ExecutionImplement, manifest, ExecutionManifestDigest(manifest))
	if err := good.Validate(); err != nil {
		t.Fatalf("manifested implement launch refused: %v", err)
	}
	if err := good.Run.Validate(); err != nil {
		t.Fatalf("manifested implement run refused: %v", err)
	}
	correctGood := launchFor(ExecutionCorrect, manifest, ExecutionManifestDigest(manifest))
	if err := correctGood.Validate(); err != nil {
		t.Fatalf("manifested correct launch refused: %v", err)
	}
	if err := correctGood.Run.Validate(); err != nil {
		t.Fatalf("manifested correct run refused: %v", err)
	}
	cases := map[string]FactoryLaunch{
		"missing manifest":         launchFor(ExecutionImplement, nil, strings.Repeat("d", 64)),
		"missing correct manifest": launchFor(ExecutionCorrect, nil, strings.Repeat("d", 64)),
		"drifted digest":           launchFor(ExecutionImplement, manifest, strings.Repeat("d", 64)),
		"drifted correct digest":   launchFor(ExecutionCorrect, manifest, strings.Repeat("d", 64)),
		"manifest elsewhere":       launchFor(ExecutionInvestigate, manifest, ExecutionManifestDigest(manifest)),
		"manifest on repair":       launchFor(ExecutionRepair, manifest, ExecutionManifestDigest(manifest)),
		"manifest on verify":       launchFor(ExecutionVerify, manifest, ExecutionManifestDigest(manifest)),
		"digest elsewhere":         launchFor(ExecutionReview, nil, strings.Repeat("d", 64)),
		"unknown phase":            launchFor("publish", nil, ""),
		"oversized manifest":       launchFor(ExecutionImplement, make([]byte, MaxExecutionManifest+1), strings.Repeat("d", 64)),
	}
	for name, launch := range cases {
		if err := launch.Validate(); err == nil {
			t.Fatalf("launch %s admitted", name)
		}
	}
	// Pre-phase records without manifests keep validating for readback.
	if err := launchFor("", nil, "").Validate(); err != nil {
		t.Fatalf("pre-phase launch refused: %v", err)
	}
	if len(manifest)+len([]byte("trivial coding task")) >= 1<<20 {
		t.Fatal("launch envelope exceeds its bound")
	}
}

func TestFactoryLaunchBindsPromptDigest(t *testing.T) {
	run := factoryRunFixture()
	prompt := []byte("trivial coding task")
	run.Assignment = FactoryPromptDigest(prompt)
	launch := FactoryLaunch{Run: run, Prompt: prompt, HarnessSHA256: strings.Repeat("f", 64)}
	if err := launch.Validate(); err != nil {
		t.Fatal(err)
	}
	launch.Prompt = []byte("changed task")
	if err := launch.Validate(); err == nil {
		t.Fatal("changed prompt reused assignment digest")
	}
	launch.Prompt = nil
	if err := launch.Validate(); err == nil {
		t.Fatal("empty prompt admitted")
	}
}

func TestFactoryRunPathsAreFixed(t *testing.T) {
	checkout, runDir, home, codex := FactoryRunPaths(RoleCoder, "f"+strings.Repeat("c", 24), strings.Repeat("a", 32))
	if checkout != "/home/soda-coder/checkouts/f"+strings.Repeat("c", 24) {
		t.Fatal("checkout path changed", checkout)
	}
	if !strings.HasPrefix(runDir, checkout+"/.soda-home/runs/") || !strings.HasPrefix(home, runDir+"/home") || codex != home+"/.codex" {
		t.Fatal("run paths changed", runDir, home, codex)
	}
	if c, _, _, _ := FactoryRunPaths("soda-tester", "f"+strings.Repeat("c", 24), strings.Repeat("a", 32)); c != "" {
		t.Fatal("foreign role resolved paths")
	}
	if guest := FactoryCodexGuest("0.157.1"); guest != "/usr/local/bin/codex-factory-0.157.1" {
		t.Fatal("guest harness path changed", guest)
	}
	if guest := FactoryCodexGuest("../x"); guest != "" {
		t.Fatal("unsafe version resolved a guest path")
	}
	if unit := FactoryUnitName(strings.Repeat("a", 32)); unit != "soda-factory-"+strings.Repeat("a", 32)+".service" {
		t.Fatal("unit name changed", unit)
	}
	if unit := FactoryUnitName("short"); unit != "" {
		t.Fatal("invalid run resolved a unit")
	}
}

func TestFactoryPhasesAndAddresses(t *testing.T) {
	for _, phase := range []string{FactoryApproved, FactoryRunning, FactoryCompleted, FactoryFailed, FactoryStopped, FactoryUncertain} {
		if !ValidFactoryPhase(phase) {
			t.Fatalf("phase %s rejected", phase)
		}
	}
	if ValidFactoryPhase("launching") {
		t.Fatal("unknown phase admitted")
	}
	if err := (FactoryInspect{Project: "p" + strings.Repeat("b", 24), ID: strings.Repeat("a", 32)}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (FactoryStop{Project: "bad", ID: strings.Repeat("a", 32)}).Validate(); err == nil {
		t.Fatal("invalid stop address admitted")
	}
}
