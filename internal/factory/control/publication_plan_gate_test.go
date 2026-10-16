package control

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// FW15 pure publication-join tests: the join decides over the admitted plan
// revision, its persisted receipts and the reported candidate. No store or
// host is involved.

func planGatePlan(t *testing.T, root string) factory.InvestigationPlan {
	t.Helper()
	commit := phaseTestCommit()
	prep := "f" + strings.Repeat("d", 24)
	plan := factory.InvestigationPlan{
		ID: factory.NewID(), AttemptRoot: root,
		ScopeDigest: phaseTestSHA("scope"), InputDigest: phaseTestSHA("input"),
		SourceCommit: commit, IntegrationCommit: commit,
		Preparation:        prep,
		Approach:           "two small tasks in one lane",
		AcceptanceCoverage: []string{"accept-one"},
		Tasks: []factory.PlanTask{
			{ID: "task-a", Objective: "first unit of work",
				Outputs: []string{"a.go"}, AcceptanceRefs: []string{"accept-one"},
				Lane: "lane-one", OwnedPaths: []string{"internal/a.go"},
				CheckIDs: []string{"check-one"}},
			{ID: "task-b", Objective: "second unit of work",
				Outputs: []string{"b.go"}, AcceptanceRefs: []string{"accept-one"},
				Lane: "lane-one", OwnedPaths: []string{"internal/b.go"},
				CheckIDs: []string{"check-one"}},
		},
		Lanes: []factory.PlanLane{
			{ID: "lane-one", TaskIDs: []string{"task-a", "task-b"}},
		},
		Checks: []factory.CheckSpec{{
			ID: "check-one", Argv: []string{"go", "test", "./internal/a"},
			Cwd: ".", TimeoutSeconds: 60, Scope: factory.CheckScopeTask, Required: true,
		}},
	}
	digest, err := factory.PlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Digest = digest
	if err := plan.Validate(); err != nil {
		t.Fatal(err)
	}
	return plan
}

func planGateReceipt(plan factory.InvestigationPlan, taskID, state string) store.PlanTaskReceipt {
	return store.PlanTaskReceipt{
		AttemptRoot: plan.AttemptRoot, PlanID: plan.ID, TaskID: taskID,
		State: state, Child: "child-" + taskID,
		OutputDigest: phaseTestSHA("output-" + taskID),
		Events:       []string{"child_reported@1"},
	}
}

func planGateCommitted(plan factory.InvestigationPlan, taskID, seed string, checks ...string) store.PlanTaskReceipt {
	receipt := planGateReceipt(plan, taskID, store.PlanTaskCompleted)
	receipt.Commit = strings.Repeat(seed, 40)[:40]
	receipt.Tree = strings.Repeat("d", 40)
	receipt.Checks = checks
	return receipt
}

func TestPlanJoinReadiesCompletedTasksWithCheckpointCommits(t *testing.T) {
	plan := planGatePlan(t, factory.NewID())
	receipts := []store.PlanTaskReceipt{
		planGateCommitted(plan, "task-a", "a", "check-one"),
		planGateCommitted(plan, "task-b", "b", "check-one"),
	}
	verdict := judgePlanPublication(plan, receipts, strings.Repeat("2", 40))
	if !verdict.Ready || verdict.WaitReason != "" || verdict.RefuseReason != "" {
		t.Fatalf("ready join refused: %+v", verdict)
	}
	if len(verdict.Evidence.Commits) != 2 || verdict.Evidence.TaskCount != 2 || verdict.Evidence.VerificationOnly {
		t.Fatalf("checkpoint evidence incomplete: %+v", verdict.Evidence)
	}
	if len(verdict.Evidence.Checks) != 1 || verdict.Evidence.Checks[0] != "check-one" {
		t.Fatalf("check evidence lost: %+v", verdict.Evidence)
	}
}

func TestPlanJoinWaitsWhileDescendantsUnsettle(t *testing.T) {
	plan := planGatePlan(t, factory.NewID())
	for _, state := range []string{store.PlanTaskPending, store.PlanTaskRunning, store.PlanTaskOutputReady} {
		receipts := []store.PlanTaskReceipt{
			planGateCommitted(plan, "task-a", "a", "check-one"),
			planGateReceipt(plan, "task-b", state),
		}
		verdict := judgePlanPublication(plan, receipts, strings.Repeat("2", 40))
		if verdict.Ready || verdict.WaitReason != PlanWaitTasksIncomplete || verdict.RefuseReason != "" {
			t.Fatalf("state %s did not wait: %+v", state, verdict)
		}
	}
	receipts := []store.PlanTaskReceipt{planGateCommitted(plan, "task-a", "a", "check-one")}
	verdict := judgePlanPublication(plan, receipts, strings.Repeat("2", 40))
	if verdict.Ready || verdict.WaitReason != PlanWaitTasksIncomplete {
		t.Fatalf("missing receipt did not wait: %+v", verdict)
	}
}

func TestPlanJoinRefusesFailedTasks(t *testing.T) {
	plan := planGatePlan(t, factory.NewID())
	for _, state := range []string{store.PlanTaskFailed, store.PlanTaskCancelled} {
		receipts := []store.PlanTaskReceipt{
			planGateCommitted(plan, "task-a", "a", "check-one"),
			planGateReceipt(plan, "task-b", state),
		}
		verdict := judgePlanPublication(plan, receipts, strings.Repeat("2", 40))
		if verdict.Ready || verdict.RefuseReason != PlanRefuseTasksFailed {
			t.Fatalf("state %s published: %+v", state, verdict)
		}
	}
}

func TestPlanJoinRefusesMalformedEvidence(t *testing.T) {
	plan := planGatePlan(t, factory.NewID())
	good := []store.PlanTaskReceipt{
		planGateCommitted(plan, "task-a", "a", "check-one"),
		planGateCommitted(plan, "task-b", "b", "check-one"),
	}
	foreign := planGateCommitted(plan, "task-a", "a", "check-one")
	foreign.PlanID = factory.NewID()
	unknown := planGateCommitted(plan, "task-a", "a", "check-one")
	unknown.TaskID = "task-ghost"
	badCheck := planGateCommitted(plan, "task-b", "b", "check-ghost")
	badCommit := planGateCommitted(plan, "task-b", "b", "check-one")
	badCommit.Commit = "not-a-commit"
	cases := []struct {
		name     string
		receipts []store.PlanTaskReceipt
	}{
		{"foreign revision", []store.PlanTaskReceipt{foreign, good[1]}},
		{"unknown task", []store.PlanTaskReceipt{unknown, good[1]}},
		{"unknown check", []store.PlanTaskReceipt{good[0], badCheck}},
		{"malformed commit", []store.PlanTaskReceipt{good[0], badCommit}},
	}
	for _, tc := range cases {
		if verdict := judgePlanPublication(plan, tc.receipts, strings.Repeat("2", 40)); verdict.Ready || verdict.RefuseReason != PlanRefuseEvidence {
			t.Fatalf("%s published: %+v", tc.name, verdict)
		}
	}
	broken := plan
	broken.Approach = ""
	if verdict := judgePlanPublication(broken, good, strings.Repeat("2", 40)); verdict.Ready || verdict.RefuseReason != PlanRefuseEvidence {
		t.Fatalf("invalid plan published: %+v", verdict)
	}
	if verdict := judgePlanPublication(plan, good, "not-a-commit"); verdict.Ready || verdict.RefuseReason != PlanRefuseCandidate {
		t.Fatalf("invalid candidate published: %+v", verdict)
	}
}

func TestPlanJoinRefusesUnchangedIssueWithoutFakePR(t *testing.T) {
	plan := planGatePlan(t, factory.NewID())
	verified := planGateReceipt(plan, "task-a", store.PlanTaskCompleted)
	verified.Checks = []string{"check-one"}
	other := planGateReceipt(plan, "task-b", store.PlanTaskCompleted)
	other.Checks = []string{"check-one"}
	verdict := judgePlanPublication(plan, []store.PlanTaskReceipt{verified, other}, plan.SourceCommit)
	if verdict.Ready || verdict.RefuseReason != PlanRefuseUnchanged {
		t.Fatalf("unchanged issue published: %+v", verdict)
	}
}

func TestPlanJoinReadiesMixedVerificationWithoutEmptyCommit(t *testing.T) {
	plan := planGatePlan(t, factory.NewID())
	verified := planGateReceipt(plan, "task-b", store.PlanTaskCompleted)
	verified.Checks = []string{"check-one"}
	receipts := []store.PlanTaskReceipt{
		planGateCommitted(plan, "task-a", "a", "check-one"),
		verified,
	}
	verdict := judgePlanPublication(plan, receipts, strings.Repeat("2", 40))
	if !verdict.Ready {
		t.Fatalf("mixed verification refused: %+v", verdict)
	}
	if len(verdict.Evidence.Commits) != 1 || verdict.Evidence.VerificationOnly {
		t.Fatalf("empty commit manufactured or miscounted: %+v", verdict.Evidence)
	}
}

func TestPlanJoinRefusesCandidateNoTaskAccountsFor(t *testing.T) {
	plan := planGatePlan(t, factory.NewID())
	verified := planGateReceipt(plan, "task-a", store.PlanTaskCompleted)
	verified.Checks = []string{"check-one"}
	other := planGateReceipt(plan, "task-b", store.PlanTaskCompleted)
	other.Checks = []string{"check-one"}
	verdict := judgePlanPublication(plan, []store.PlanTaskReceipt{verified, other}, strings.Repeat("3", 40))
	if verdict.Ready || verdict.RefuseReason != PlanRefuseEvidence {
		t.Fatalf("unattributed candidate published: %+v", verdict)
	}
	committed := []store.PlanTaskReceipt{
		planGateCommitted(plan, "task-a", "a", "check-one"),
		planGateCommitted(plan, "task-b", "b", "check-one"),
	}
	verdict = judgePlanPublication(plan, committed, plan.SourceCommit)
	if verdict.Ready || verdict.RefuseReason != PlanRefuseEvidence {
		t.Fatalf("base candidate over task commits published: %+v", verdict)
	}
}
