package control

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// FW11 pure admission/result tests: selected-issue planning first, common
// phase/lineage admission and strict v2 results. No store or host is
// involved; PG-gated packet tests cover the transaction boundary.

func phaseTestSHA(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func phaseTestCommit() string { return strings.Repeat("c", 40) }

func phaseTestPlan(t *testing.T, root, source, prep string) factory.InvestigationPlan {
	t.Helper()
	plan := factory.InvestigationPlan{
		ID: factory.NewID(), AttemptRoot: root,
		ScopeDigest: phaseTestSHA("scope"), InputDigest: phaseTestSHA("input"),
		SourceCommit: source, IntegrationCommit: source,
		Preparation:        prep,
		Approach:           "inspect the selected source, then plan behind checks",
		AcceptanceCoverage: []string{"accept-one"},
		Tasks: []factory.PlanTask{{
			ID: "task-one", Objective: "inspect the selected source",
			Outputs: []string{"notes"}, AcceptanceRefs: []string{"accept-one"},
			Lane: "lane-one", OwnedPaths: []string{"internal/x.go"},
		}},
		Lanes: []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-one"}}},
	}
	digest, err := factory.PlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Digest = digest
	if err := plan.Validate(); err != nil {
		t.Fatalf("plan fixture invalid: %v", err)
	}
	if err := plan.Executable(); err != nil {
		t.Fatalf("plan fixture not executable: %v", err)
	}
	return plan
}

func phaseTestAssignment(t *testing.T, phase, role string, validate bool) factory.Assignment {
	t.Helper()
	id, run := factory.NewID(), factory.NewID()
	prompt := []byte("prompt bytes for " + phase)
	sum := sha256.Sum256(prompt)
	acceptance := "d" + strings.Repeat("a", 24)
	a := factory.Assignment{
		ID: id, AttemptRoot: id, PublicationAssignment: id,
		Subject: factory.ExecutionSubject{
			Kind:  factory.SubjectIssue,
			Issue: &factory.IssueSubject{Issue: 7, Acceptance: acceptance},
		},
		Phase:     phase,
		ProjectID: "p" + strings.Repeat("1", 24), Role: role,
		Repository: 7, Issue: 7, NativeRev: 1,
		Acceptance: acceptance, Preparation: "f" + strings.Repeat("d", 24),
		Harness: "codex-1.2.3", HarnessVers: "1.2.3",
		Model: "test-model", Connection: "conn",
		SourceCommit: phaseTestCommit(), Prompt: prompt, PromptSHA: hex.EncodeToString(sum[:]),
		Run: run, RunHistory: []string{run}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: 1700000000, ActorID: 7,
	}
	if validate {
		if err := a.Validate(); err != nil {
			t.Fatalf("assignment fixture invalid: %v", err)
		}
	}
	return a
}

func phaseTestRun(a factory.Assignment, outcome factory.Outcome) factory.Run {
	return factory.Run{
		ID: a.Run, ProjectID: a.ProjectID, Role: a.Role, InputSHA: a.SourceCommit,
		Started: time.Unix(1700000000, 0), Deadline: time.Unix(1700003600, 0),
		Image: "img", Harness: a.Harness, Model: a.Model, Outcome: outcome,
	}
}

func fencedResult(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return "work log\n```" + factory.ResultFence + "\n" + string(data) + "\n```\ntrailer\n"
}

func finishPhasedFixture(t *testing.T, a factory.Assignment, run factory.Run, output string) factory.Assignment {
	t.Helper()
	finished, err := finishAssignmentFromRun(a, run, output, time.Unix(1700000100, 0))
	if err != nil {
		t.Fatalf("phased finish refused: %v", err)
	}
	if err := finished.Validate(); err != nil {
		t.Fatalf("finished phased assignment invalid: %v", err)
	}
	return finished
}

// T03 (FW11 scope): a completed investigation records its executable plan,
// finishes without a commit or PR, and can never authorize publication.
func TestDerivePhasedInvestigateCompleted(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionInvestigate, project.RoleCoder, true)
	plan := phaseTestPlan(t, a.AttemptRoot, a.SourceCommit, a.Preparation)
	output := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionInvestigate, Status: "completed",
		Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
	})
	finished := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), output)
	if finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
		t.Fatalf("investigation outcome = %v/%v", finished.Outcome, finished.Reason)
	}
	result := finished.Result
	if result == nil || !result.Reported || result.Phase != project.ExecutionInvestigate {
		t.Fatalf("investigation result not bound: %+v", result)
	}
	if result.Candidate != "" {
		t.Fatal("completed planning names a candidate")
	}
	encoded, _ := json.Marshal(result.PhaseResult)
	if strings.Contains(string(encoded), `"candidate":`) {
		t.Fatal("completed planning payload exposes a candidate")
	}
	if err := result.ValidatePhaseBinding(project.RoleCoder, project.ExecutionInvestigate); err != nil {
		t.Fatalf("completed planning does not bind: %v", err)
	}
}

// T03 (FW11 scope): a completed implementation records its exact candidate.
func TestDerivePhasedImplementCompleted(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionImplement, project.RoleCoder, true)
	candidate := strings.Repeat("e", 40)
	output := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionImplement, Status: "completed",
		Summary: "candidate ready",
		Output:  &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: candidate}},
	})
	finished := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), output)
	if finished.Outcome != factory.Succeeded || finished.Result.Candidate != candidate {
		t.Fatalf("implementation outcome = %v/%v", finished.Outcome, finished.Result.Candidate)
	}
}

// T03 (FW11 scope): honest non-completed reports map to their terminal
// states, and a settled non-successful run never reports.
func TestDerivePhasedTerminalMappings(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionImplement, project.RoleCoder, true)
	for _, tc := range []struct {
		status         string
		outcome        factory.Outcome
		reason         string
		legacy         string
		run            factory.Outcome
		output         string
		implementation *factory.ImplementationReceipt
	}{
		{status: "blocked", outcome: factory.NeedsHuman, reason: factory.AssignReasonNoReport, legacy: "blocked", run: factory.Succeeded, implementation: &factory.ImplementationReceipt{DiagnosticRefs: []string{"log tail"}}},
		{status: "failed", outcome: factory.Failed, reason: factory.AssignReasonRunFailed, legacy: "failed", run: factory.Succeeded, implementation: &factory.ImplementationReceipt{DiagnosticRefs: []string{"log tail"}}},
		{status: "cancelled", outcome: factory.Cancelled, reason: factory.AssignReasonCancelled, legacy: "cancelled", run: factory.Succeeded, implementation: nil},
	} {
		output := "unparseable agent chatter"
		if tc.implementation != nil {
			output = fencedResult(t, factory.PhaseResult{
				Version: 2, Phase: project.ExecutionImplement, Status: tc.status,
				Summary: "honest report", Output: &factory.PhaseOutput{Implementation: tc.implementation},
			})
		} else if tc.status == "cancelled" {
			output = fencedResult(t, factory.PhaseResult{
				Version: 2, Phase: project.ExecutionImplement, Status: tc.status,
				Summary: "honest report",
			})
		}
		finished := finishPhasedFixture(t, a, phaseTestRun(a, tc.run), output)
		if finished.Outcome != tc.outcome || finished.Reason != tc.reason || finished.Result.Status != tc.legacy {
			t.Fatalf("status %s mapped to %v/%v/%v", tc.status, finished.Outcome, finished.Reason, finished.Result.Status)
		}
	}
	for _, run := range []factory.Outcome{factory.Failed, factory.Cancelled} {
		finished := finishPhasedFixture(t, a, phaseTestRun(a, run), "garbage output without a report")
		want := factory.Failed
		if run == factory.Cancelled {
			want = factory.Cancelled
		}
		if finished.Outcome != want || finished.Result.Candidate != "" {
			t.Fatalf("run %v mapped to %v", run, finished.Outcome)
		}
	}
}

// T02/T03 (FW11 scope): cross-phase payloads, extra JSON, legacy shapes,
// run mismatches and phase/role mismatches all refuse to a blocked result
// that still validates.
func TestDerivePhasedRefusals(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionInvestigate, project.RoleCoder, true)
	plan := phaseTestPlan(t, a.AttemptRoot, a.SourceCommit, a.Preparation)
	good := factory.PhaseResult{
		Version: 2, Phase: project.ExecutionInvestigate, Status: "completed",
		Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
	}
	legacy := `{"status":"completed","summary":"done","candidate":"` + strings.Repeat("d", 40) + `","review_passed":false,"findings":[]}`
	implementPayload, _ := json.Marshal(factory.PhaseResult{
		Version: 2, Phase: project.ExecutionImplement, Status: "completed",
		Summary: "x", Output: &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: strings.Repeat("e", 40)}},
	})
	extraFields, _ := json.Marshal(good)
	extraFields = append(extraFields[:len(extraFields)-1], []byte(`,"extra":1}`)...)
	trailing, _ := json.Marshal(good)
	run := phaseTestRun(a, factory.Succeeded)
	outputs := map[string]string{
		"legacy shape":  "log\n```" + factory.ResultFence + "\n" + legacy + "\n```\n",
		"cross-phase":   fencedResult(t, json.RawMessage(implementPayload)),
		"extra JSON":    fencedResult(t, json.RawMessage(extraFields)),
		"trailing JSON": "log\n```" + factory.ResultFence + "\n" + string(trailing) + `{"after":1}` + "\n```\n",
		"unparseable":   "agent chatter without any report block",
		"empty candidate": fencedResult(t, factory.PhaseResult{
			Version: 2, Phase: project.ExecutionImplement, Status: "completed",
			Summary: "x", Output: &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{}},
		}),
	}
	for name, output := range outputs {
		result, outcome, reason := deriveAttemptResult(a, run, output, "", 1700000100)
		if outcome != factory.NeedsHuman || reason != factory.AssignReasonNoReport {
			t.Fatalf("%s mapped to %v/%v", name, outcome, reason)
		}
		if err := result.Validate(); err != nil {
			t.Fatalf("%s refusal invalid: %v", name, err)
		}
	}
	wrongRun := run
	wrongRun.ID = factory.NewID()
	if _, outcome, _ := deriveAttemptResult(a, wrongRun, fencedResult(t, good), "", 1700000100); outcome != factory.NeedsHuman {
		t.Fatal("run mismatch admitted")
	}
	mismatched := a
	mismatched.Role = project.RoleReviewer
	if _, outcome, _ := deriveAttemptResult(mismatched, phaseTestRun(mismatched, factory.Succeeded), fencedResult(t, good), "", 1700000100); outcome != factory.NeedsHuman {
		t.Fatal("phase/role mismatch admitted")
	}
}

// Reviewer reports stay on the legacy review fence until FW06C admits
// review payloads through this boundary.
func TestDerivePhasedReviewerKeepsLegacyReport(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	output := "notes\n```" + factory.ReviewReportFence + "\n" +
		`{"verdict":"approve","summary":"clean","body":"no blockers","findings":[]}` + "\n```\n"
	finished := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), output)
	if finished.Outcome != factory.Succeeded || finished.Result.Candidate != a.SourceCommit {
		t.Fatalf("reviewer outcome = %v/%v", finished.Outcome, finished.Result.Candidate)
	}
}

// FW11 admission: only a reported completed investigation with an
// executable plan on the current head advances, and the same parent always
// replays to the same child identity.
func TestAdvanceChildID(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionInvestigate, project.RoleCoder, true)
	plan := phaseTestPlan(t, a.AttemptRoot, a.SourceCommit, a.Preparation)
	output := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionInvestigate, Status: "completed",
		Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
	})
	owner := finishPhasedFixture(t, a, phaseTestRun(a, factory.Succeeded), output)
	childID, childPlan, ok := advanceChildID(owner, owner.Acceptance)
	if !ok || childPlan == nil {
		t.Fatal("completed investigation refused advancement")
	}
	want, err := factory.AdvancementID(owner.ID, factory.ResultDigest(*owner.Result), project.ExecutionImplement)
	if err != nil || want != childID {
		t.Fatalf("advancement identity mismatch: %v %v", want, err)
	}
	again, _, ok := advanceChildID(owner, owner.Acceptance)
	if !ok || again != childID {
		t.Fatal("advancement is not replay-safe")
	}
	cases := map[string]func(factory.Assignment) factory.Assignment{
		"pre-phase owner": func(o factory.Assignment) factory.Assignment {
			o.Subject = factory.ExecutionSubject{}
			o.Phase = ""
			return o
		},
		"wrong head": func(o factory.Assignment) factory.Assignment { return o },
		"implement phase": func(o factory.Assignment) factory.Assignment {
			o.Phase = project.ExecutionImplement
			return o
		},
		"non-reported": func(o factory.Assignment) factory.Assignment {
			o.Outcome, o.Reason = factory.Failed, factory.AssignReasonRunFailed
			return o
		},
	}
	for name, mutate := range cases {
		head := owner.Acceptance
		if name == "wrong head" {
			head = "d" + strings.Repeat("b", 24)
		}
		if _, _, ok := advanceChildID(mutate(owner), head); ok {
			t.Fatalf("%s advanced", name)
		}
	}
	blocked := owner
	blockedResult := *owner.Result
	blockedPhased := *owner.Result.PhaseResult
	blockedPhased.Status = "blocked"
	blockedResult.PhaseResult = &blockedPhased
	blocked.Result = &blockedResult
	if _, _, ok := advanceChildID(blocked, owner.Acceptance); ok {
		t.Fatal("blocked planning advanced")
	}
	if digest := factory.ResultDigest(*owner.Result); len(digest) != 64 {
		t.Fatalf("result digest malformed: %q", digest)
	}
}

// FW11 admission: the phased prompt replaces the legacy report contract
// and stages the recorded plan; anything over the phase bound waits.
func TestRenderPhasedDispatchPrompt(t *testing.T) {
	legacy := "## Objective\n\nwork\n\n## Report\n\nClose your work with exactly one fenced block:\n\n```" +
		factory.ResultFence + "\n" +
		`{"status":"completed|blocked|failed|cancelled","summary":"...","candidate":"<40-hex commit or empty>","review_passed":false,"findings":[]}` + "\n```\n"
	rendered, err := renderPhasedDispatchPrompt([]byte(legacy), project.ExecutionInvestigate, nil)
	if err != nil {
		t.Fatalf("investigate prompt refused: %v", err)
	}
	if strings.Contains(string(rendered), "review_passed") {
		t.Fatal("legacy report contract survived")
	}
	if !strings.Contains(string(rendered), `"phase":"investigate"`) {
		t.Fatal("investigate contract missing")
	}
	compact, err := renderPhasedDispatchPrompt([]byte("x\n## Report\n\ntail\n"), project.ExecutionInvestigate, nil)
	if err != nil {
		t.Fatalf("compact prompt refused: %v", err)
	}
	if !strings.Contains(string(compact), "x\n\n## Report\n\n") {
		t.Fatal("report section not separated by a blank line")
	}
	a := phaseTestAssignment(t, project.ExecutionInvestigate, project.RoleCoder, true)
	plan := phaseTestPlan(t, a.AttemptRoot, a.SourceCommit, a.Preparation)
	rendered, err = renderPhasedDispatchPrompt([]byte(legacy), project.ExecutionImplement, &plan)
	if err != nil {
		t.Fatalf("implement prompt refused: %v", err)
	}
	for _, want := range []string{"## Implementation approach", plan.ID, "task-one", `"phase":"implement"`} {
		if !strings.Contains(string(rendered), want) {
			t.Fatalf("implement prompt lacks %q", want)
		}
	}
	guarded := plan
	guarded.Approach = "see ```" + factory.ResultFence + " now"
	guarded.Digest = ""
	digest, err := factory.PlanDigest(guarded)
	if err != nil {
		t.Fatal(err)
	}
	guarded.Digest = digest
	rendered, err = renderPhasedDispatchPrompt([]byte(legacy), project.ExecutionImplement, &guarded)
	if err != nil {
		t.Fatalf("guarded plan refused: %v", err)
	}
	if strings.Contains(string(rendered), "see ```"+factory.ResultFence+" now") {
		t.Fatal("recorded plan text can forge a report block")
	}
	if _, err := renderPhasedDispatchPrompt([]byte("no marker"), project.ExecutionInvestigate, nil); err == nil {
		t.Fatal("markerless prompt admitted")
	}
	if _, err := renderPhasedDispatchPrompt([]byte(legacy), project.ExecutionReview, nil); err == nil {
		t.Fatal("non-dispatcher phase admitted")
	}
	if _, err := renderPhasedDispatchPrompt([]byte(legacy), project.ExecutionInvestigate, &plan); err == nil {
		t.Fatal("plan staged on investigation")
	}
	big := "## Objective\n\n" + strings.Repeat("x", project.MaxFactoryPrompt) + "\n## Report\n\ntail\n"
	if _, err := renderPhasedDispatchPrompt([]byte(big), project.ExecutionInvestigate, nil); err == nil {
		t.Fatal("oversized phased prompt admitted")
	}
}
