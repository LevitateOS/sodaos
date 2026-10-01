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
