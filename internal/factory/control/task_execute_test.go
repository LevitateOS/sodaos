package control

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// FW14 pure task-execution tests: ready release, event folding, failure
// containment, run stop and restart idempotency. No store or host is
// involved; the PG-gated receipt tests cover the CAS boundary.

func taskExecPlan(t *testing.T, root string) factory.InvestigationPlan {
	t.Helper()
	sha := func(seed string) string { return phaseTestSHA(seed) }
	commit := phaseTestCommit()
	prep := "f" + strings.Repeat("d", 24)
	plan := factory.InvestigationPlan{
		ID: factory.NewID(), AttemptRoot: root,
		ScopeDigest: sha("scope"), InputDigest: sha("input"),
		SourceCommit: commit, IntegrationCommit: commit,
		Preparation:        prep,
		Approach:           "split the issue into three small tasks",
		AcceptanceCoverage: []string{"accept-one"},
		Tasks: []factory.PlanTask{
			{ID: "task-a", Objective: "first unit of work",
				Outputs: []string{"a.go"}, AcceptanceRefs: []string{"accept-one"},
				Lane: "lane-one", OwnedPaths: []string{"internal/a.go"},
				CheckIDs: []string{"check-one"}},
			{ID: "task-a2", Objective: "second unit in the same lane",
				Outputs: []string{"a2.go"}, AcceptanceRefs: []string{"accept-one"},
				Lane: "lane-one", OwnedPaths: []string{"internal/a2.go"},
				CheckIDs: []string{"check-one"}},
			{ID: "task-b", Objective: "independent lane work",
				Outputs: []string{"b.go"}, AcceptanceRefs: []string{"accept-one"},
				Lane: "lane-two", OwnedPaths: []string{"internal/b.go"}},
			{ID: "task-c", Objective: "downstream of the first task",
				Outputs: []string{"c.go"}, AcceptanceRefs: []string{"accept-one"},
				DependsOn: []string{"task-a"}, Lane: "lane-one",
				OwnedPaths: []string{"internal/c.go"}},
		},
		Lanes: []factory.PlanLane{
			{ID: "lane-one", TaskIDs: []string{"task-a", "task-a2", "task-c"}},
			{ID: "lane-two", TaskIDs: []string{"task-b"}},
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
		t.Fatalf("plan fixture invalid: %v", err)
	}
	if err := plan.Executable(); err != nil {
		t.Fatalf("plan fixture not executable: %v", err)
	}
	return plan
}

func taskExecImport(root, planID, invocation, planDigest string, maxChildren int) TaskEventImport {
	return TaskEventImport{
		AttemptRoot: root, PlanID: planID,
		Invocation: invocation, PlanDigest: planDigest,
		MaxChildren: maxChildren,
	}
}

func taskExecEvent(seq int64, prev, eventType, task, child, invocation, planDigest, detail string) project.TaskEvent {
	return project.TaskEvent{
		Sequence: seq, PrevDigest: prev, Type: eventType,
		TaskKey: task, ChildID: child,
		Invocation: invocation, PlanDigest: planDigest, Detail: detail,
	}
}

func taskExecChain(seq int64) string {
	if seq <= 1 {
		return ""
	}
	return strings.Repeat("e", 64)
}

func taskExecChild(seed string) string {
	return phaseTestSHA("child-" + seed)[:32]
}

// latestReceipts keeps the newest receipt per task across stacked pages,
// as the store holds exactly one row per task.
func latestReceipts(pages ...[]store.PlanTaskReceipt) []store.PlanTaskReceipt {
	byTask := make(map[string]store.PlanTaskReceipt)
	var order []string
	for _, page := range pages {
		for _, receipt := range page {
			if _, ok := byTask[receipt.TaskID]; !ok {
				order = append(order, receipt.TaskID)
			}
			byTask[receipt.TaskID] = receipt
		}
	}
	var out []store.PlanTaskReceipt
	for _, id := range order {
		out = append(out, byTask[id])
	}
	return out
}

func TestImportSeedsPendingAndReleasesReady(t *testing.T) {
	root := factory.NewID()
	plan := taskExecPlan(t, root)
	invocation := factory.NewID()
	imp := taskExecImport(root, plan.ID, invocation, plan.Digest, 3)
	result, err := ImportTaskEvents(plan, nil, imp)
	if err != nil {
		t.Fatalf("import refused: %v", err)
	}
	if len(result.Writes) != 4 {
		t.Fatalf("writes = %d, want 4 pending seeds", len(result.Writes))
	}
	for _, write := range result.Writes {
		if write.State != store.PlanTaskPending || write.Revision != 0 {
			t.Fatalf("seed %+v is not a pending revision-zero receipt", write)
		}
	}
	// Plan order release: task-a holds lane-one, task-b holds lane-two;
	// task-a2 waits on its lane and task-c on its prerequisite.
	want := []string{"task-a", "task-b"}
	if strings.Join(result.Decisions.NewlyReady, ",") != strings.Join(want, ",") {
		t.Fatalf("ready = %v, want %v", result.Decisions.NewlyReady, want)
	}
	if result.Decisions.StopRun || result.Contained || result.Stopped {
		t.Fatal("fresh import stops nothing")
	}
}

func TestImportStartReportCommitReleasesDownstream(t *testing.T) {
	root := factory.NewID()
	plan := taskExecPlan(t, root)
	invocation := factory.NewID()
	childA := taskExecChild("a")
	output := phaseTestSHA("output-a")
	commit := strings.Repeat("f", 40)
	tree := strings.Repeat("b", 40)
	events := []project.TaskEvent{
		taskExecEvent(1, "", project.TaskEventChildStartIntent, "task-a", "", invocation, plan.Digest, ""),
		taskExecEvent(2, taskExecChain(2), project.TaskEventChildStarted, "task-a", childA, invocation, plan.Digest, ""),
		taskExecEvent(3, taskExecChain(3), project.TaskEventChildReported, "task-a", childA, invocation, plan.Digest, output),
		taskExecEvent(4, taskExecChain(4), project.TaskEventCheckpointReady, "task-a", childA, invocation, plan.Digest, "check-one"),
		taskExecEvent(5, taskExecChain(5), project.TaskEventTaskCommitted, "task-a", childA, invocation, plan.Digest, "tree="+tree+" commit="+commit),
	}
	imp := taskExecImport(root, plan.ID, invocation, plan.Digest, 3)
	imp.Events = events
	// Fold over the seeded receipts, as a second poll would.
	seeded, err := ImportTaskEvents(plan, nil, taskExecImport(root, plan.ID, invocation, plan.Digest, 3))
	if err != nil {
		t.Fatal(err)
	}
	var receipts []store.PlanTaskReceipt
	for _, write := range seeded.Writes {
		receipts = append(receipts, write)
	}
	result, err := ImportTaskEvents(plan, receipts, imp)
	if err != nil {
		t.Fatalf("import refused: %v", err)
	}
	byTask := make(map[string]store.PlanTaskReceipt)
	for _, receipt := range receipts {
		byTask[receipt.TaskID] = receipt
	}
	for _, write := range result.Writes {
		byTask[write.TaskID] = write
	}
	done := byTask["task-a"]
	if done.State != store.PlanTaskCompleted || done.OutputDigest != output ||
		done.Tree != tree || done.Commit != commit || done.Child != childA {
		t.Fatalf("task-a receipt %+v lacks its executor-observed evidence", done)
	}
	if len(done.Checks) != 1 || done.Checks[0] != "check-one" {
		t.Fatalf("task-a checks = %v, want [check-one]", done.Checks)
	}
	// Completion prompts downstream release: task-a2 takes lane-one in
	// plan order ahead of task-c, task-b takes lane-two, and task-c waits
	// on its lane despite its completed prerequisite.
	want := []string{"task-a2", "task-b"}
	if strings.Join(result.Decisions.NewlyReady, ",") != strings.Join(want, ",") {
		t.Fatalf("ready = %v, want %v", result.Decisions.NewlyReady, want)
	}
	// An exact replay of the same page emits no writes.
	replay, err := ImportTaskEvents(plan, latestReceipts(receipts, result.Writes), imp)
	if err != nil {
		t.Fatalf("replay refused: %v", err)
	}
	if len(replay.Writes) != 0 {
		t.Fatalf("replay writes = %d, want 0", len(replay.Writes))
	}
	if replay.NextSequence != 5 {
		t.Fatalf("replay cursor = %d, want 5", replay.NextSequence)
	}
}

func TestImportFailureBlocksDescendantsAndStopsRun(t *testing.T) {
	root := factory.NewID()
	plan := taskExecPlan(t, root)
	invocation := factory.NewID()
	childA, childB := taskExecChild("a"), taskExecChild("b")
	seeded, err := ImportTaskEvents(plan, nil, taskExecImport(root, plan.ID, invocation, plan.Digest, 3))
	if err != nil {
		t.Fatal(err)
	}
	imp := taskExecImport(root, plan.ID, invocation, plan.Digest, 3)
	imp.Events = []project.TaskEvent{
		taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", childA, invocation, plan.Digest, ""),
		taskExecEvent(2, taskExecChain(2), project.TaskEventChildStarted, "task-b", childB, invocation, plan.Digest, ""),
		taskExecEvent(3, taskExecChain(3), project.TaskEventTaskFailed, "task-a", childA, invocation, plan.Digest, "check-one refused the output"),
	}
	result, err := ImportTaskEvents(plan, seeded.Writes, imp)
	if err != nil {
		t.Fatalf("import refused: %v", err)
	}
	byTask := make(map[string]store.PlanTaskReceipt)
	for _, write := range result.Writes {
		byTask[write.TaskID] = write
	}
	if byTask["task-a"].State != store.PlanTaskFailed {
		t.Fatalf("task-a = %s, want failed", byTask["task-a"].State)
	}
	// Descendants of the failed prerequisite never release: durable
	// cancellation names the failed prerequisite.
	blocked, ok := byTask["task-c"]
	if !ok || blocked.State != store.PlanTaskCancelled || !strings.Contains(blocked.Reason, "task-a") {
		t.Fatalf("task-c = %+v, want cancellation naming task-a", blocked)
	}
	if _, ok := byTask["task-a2"]; ok {
		t.Fatal("unrelated pending task-a2 was rewritten by the failure")
	}
	// The unrelated running child is retired with the run, not failed.
	if !result.Decisions.StopRun || !result.Contained {
		t.Fatal("failure does not stop the run")
	}
	if len(result.Decisions.RetireChildren) != 1 || result.Decisions.RetireChildren[0] != childB {
		t.Fatalf("retire = %v, want [%s]", result.Decisions.RetireChildren, childB)
	}
	if len(result.Decisions.NewlyReady) != 0 {
		t.Fatalf("ready after failure = %v, want none", result.Decisions.NewlyReady)
	}
}

func TestImportRunStopCancelsRemainder(t *testing.T) {
	root := factory.NewID()
	plan := taskExecPlan(t, root)
	invocation := factory.NewID()
	childA := taskExecChild("a")
	seeded, err := ImportTaskEvents(plan, nil, taskExecImport(root, plan.ID, invocation, plan.Digest, 3))
	if err != nil {
		t.Fatal(err)
	}
	imp := taskExecImport(root, plan.ID, invocation, plan.Digest, 3)
	imp.Events = []project.TaskEvent{
		taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", childA, invocation, plan.Digest, ""),
		taskExecEvent(2, taskExecChain(2), project.TaskEventRunStopped, "task-a", "", invocation, plan.Digest, "parent stop confirmed"),
	}
	result, err := ImportTaskEvents(plan, seeded.Writes, imp)
	if err != nil {
		t.Fatalf("import refused: %v", err)
	}
	if !result.Stopped || !result.Decisions.StopRun {
		t.Fatal("run stop does not stop the run")
	}
	final := make(map[string]store.PlanTaskReceipt)
	for _, receipt := range latestReceipts(seeded.Writes, result.Writes) {
		final[receipt.TaskID] = receipt
	}
	if len(final) != 4 {
		t.Fatalf("final receipts = %d, want 4", len(final))
	}
	for id, receipt := range final {
		if !terminalState(receipt.State) {
			t.Fatalf("%s = %s after stop, want terminal", id, receipt.State)
		}
	}
	if final["task-a"].Child != childA {
		t.Fatalf("stop drops the child identity: %+v", final["task-a"])
	}
	if len(result.Decisions.NewlyReady) != 0 {
		t.Fatalf("ready after stop = %v, want none", result.Decisions.NewlyReady)
	}
	// Replaying the terminal page emits nothing.
	after, err := ImportTaskEvents(plan, latestReceipts(seeded.Writes, result.Writes), imp)
	if err != nil {
		t.Fatalf("terminal replay refused: %v", err)
	}
	if len(after.Writes) != 0 {
		t.Fatalf("terminal replay writes = %d, want 0", len(after.Writes))
	}
}

func terminalState(state string) bool {
	return state == store.PlanTaskCompleted || state == store.PlanTaskFailed || state == store.PlanTaskCancelled
}

func TestImportRefusals(t *testing.T) {
	root := factory.NewID()
	plan := taskExecPlan(t, root)
	invocation := factory.NewID()
	childA := taskExecChild("a")
	seeded, err := ImportTaskEvents(plan, nil, taskExecImport(root, plan.ID, invocation, plan.Digest, 3))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		events []project.TaskEvent
		want   error
	}{
		"sequence gap": {
			events: []project.TaskEvent{
				taskExecEvent(2, taskExecChain(2), project.TaskEventChildStarted, "task-a", childA, invocation, plan.Digest, ""),
			},
			want: ErrTaskEventGap,
		},
		"foreign invocation": {
			events: []project.TaskEvent{
				taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", childA, factory.NewID(), plan.Digest, ""),
			},
			want: ErrTaskEventBinding,
		},
		"foreign plan digest": {
			events: []project.TaskEvent{
				taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", childA, invocation, phaseTestSHA("other"), ""),
			},
			want: ErrTaskEventBinding,
		},
		"unknown task": {
			events: []project.TaskEvent{
				taskExecEvent(1, "", project.TaskEventChildStarted, "task-zzz", childA, invocation, plan.Digest, ""),
			},
			want: ErrTaskEventUnknown,
		},
		"start without child identity": {
			events: []project.TaskEvent{
				taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", "", invocation, plan.Digest, ""),
			},
			want: ErrTaskEventTransition,
		},
		"report without output digest": {
			events: []project.TaskEvent{
				taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", childA, invocation, plan.Digest, ""),
				taskExecEvent(2, taskExecChain(2), project.TaskEventChildReported, "task-a", childA, invocation, plan.Digest, "prose, not a digest"),
			},
			want: ErrTaskEventTransition,
		},
		"commit without report": {
			events: []project.TaskEvent{
				taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", childA, invocation, plan.Digest, ""),
				taskExecEvent(2, taskExecChain(2), project.TaskEventTaskCommitted, "task-a", childA, invocation, plan.Digest, ""),
			},
			want: ErrTaskEventTransition,
		},
		"prose commit evidence": {
			events: []project.TaskEvent{
				taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", childA, invocation, plan.Digest, ""),
				taskExecEvent(2, taskExecChain(2), project.TaskEventChildReported, "task-a", childA, invocation, plan.Digest, phaseTestSHA("out")),
				taskExecEvent(3, taskExecChain(3), project.TaskEventTaskCommitted, "task-a", childA, invocation, plan.Digest, "looks good, ship it"),
			},
			want: ErrTaskEventTransition,
		},
		"second child for one task": {
			events: []project.TaskEvent{
				taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", childA, invocation, plan.Digest, ""),
				taskExecEvent(2, taskExecChain(2), project.TaskEventChildStarted, "task-a", taskExecChild("other"), invocation, plan.Digest, ""),
			},
			want: ErrTaskEventTransition,
		},
	}
	for name, tc := range cases {
		imp := taskExecImport(root, plan.ID, invocation, plan.Digest, 3)
		imp.Events = tc.events
		if _, err := ImportTaskEvents(plan, seeded.Writes, imp); !errors.Is(err, tc.want) {
			t.Fatalf("%s: err = %v, want %v", name, err, tc.want)
		}
	}
}

func TestImportRespectsLaneAndAggregateLimits(t *testing.T) {
	root := factory.NewID()
	plan := taskExecPlan(t, root)
	invocation := factory.NewID()
	// One aggregate slot: only the first plan-order task releases.
	imp := taskExecImport(root, plan.ID, invocation, plan.Digest, 1)
	result, err := ImportTaskEvents(plan, nil, imp)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Decisions.NewlyReady) != 1 || result.Decisions.NewlyReady[0] != "task-a" {
		t.Fatalf("ready = %v, want [task-a]", result.Decisions.NewlyReady)
	}
	// A native journal that ran two tasks in one lane is recorded
	// truthfully and flagged, never silently accepted.
	childA, childA2 := taskExecChild("a"), taskExecChild("a2")
	over := taskExecImport(root, plan.ID, invocation, plan.Digest, 3)
	over.Events = []project.TaskEvent{
		taskExecEvent(1, "", project.TaskEventChildStarted, "task-a", childA, invocation, plan.Digest, ""),
		taskExecEvent(2, taskExecChain(2), project.TaskEventChildStarted, "task-a2", childA2, invocation, plan.Digest, ""),
	}
	violated, err := ImportTaskEvents(plan, result.Writes, over)
	if err != nil {
		t.Fatalf("violation import refused: %v", err)
	}
	if !violated.Decisions.LimitViolation {
		t.Fatal("same-lane overlap not flagged")
	}
	if len(violated.Decisions.NewlyReady) != 0 {
		t.Fatalf("ready under violation = %v, want none", violated.Decisions.NewlyReady)
	}
}

func TestStageExecutionManifestBindsItsRun(t *testing.T) {
	root := factory.NewID()
	plan := taskExecPlan(t, root)
	// Keep the staged plan small enough for the envelope bound.
	plan.Tasks = plan.Tasks[:1]
	plan.Lanes = []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-a"}}}
	digest, err := factory.PlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Digest = digest
	runID := factory.NewID()
	deadline := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	data, err := stageExecutionManifest(&plan, runID, deadline,
		factory.ExecutionProfile{Effort: factory.EffortXHigh, Contributor: true}, 3)
	if err != nil {
		t.Fatalf("stage refused: %v", err)
	}
	manifest, err := project.ParseExecutionManifest(data)
	if err != nil {
		t.Fatalf("staged envelope does not parse: %v", err)
	}
	if manifest.Run != runID || manifest.Invocation != runID || manifest.MaxChildren != 3 ||
		manifest.Effort != project.TaskEffortXHigh || !manifest.Contributor {
		t.Fatalf("staged envelope %+v drops its dispatch bindings", manifest)
	}
	run := project.FactoryRun{
		Deadline: deadline, Actor: 1001,
		ID: runID, Project: "p" + strings.Repeat("b", 24),
		Role: project.RoleCoder, Preparation: "f" + strings.Repeat("c", 24),
		Harness: project.FactoryHarnessMuse, HarnessVers: "1.4.9",
		Assignment: phaseTestSHA("prompt"), SourceCommit: phaseTestCommit(),
		Connection: "subscription", ExecutionPhase: project.ExecutionImplement,
		ManifestDigest: project.ExecutionManifestDigest(data),
	}
	if err := verifyStagedManifest(project.FactoryLaunch{
		Run: run, Prompt: []byte("prompt"), HarnessSHA256: strings.Repeat("0", 64),
		ExecutionManifest: data,
	}); err != nil {
		t.Fatalf("staged envelope does not bind its run: %v", err)
	}
	// An omitted profile resolves to the product defaults.
	defaulted, err := stageExecutionManifest(&plan, runID, deadline, factory.ExecutionProfile{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := project.ParseExecutionManifest(defaulted)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Effort != factory.DefaultEffort || parsed.Contributor != factory.DefaultContributor {
		t.Fatalf("defaulted profile = %s/%v", parsed.Effort, parsed.Contributor)
	}
	// Limits outside the admitted range refuse before any host call.
	if _, err := stageExecutionManifest(&plan, runID, deadline, factory.ExecutionProfile{}, 5); err == nil {
		t.Fatal("over-limit envelope staged")
	}
	if _, err := stageExecutionManifest(nil, runID, deadline, factory.ExecutionProfile{}, 3); err == nil {
		t.Fatal("missing plan staged")
	}
}
