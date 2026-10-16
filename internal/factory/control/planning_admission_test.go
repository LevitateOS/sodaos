package control

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// admissionCoordinator seeds one repository with the given ready issues and
// wires a planning coordinator over them, mirroring planningCoordinator.
func admissionCoordinator(t *testing.T, issues ...int64) (*Coordinator, *planningChooser, dispatchFixture) {
	t.Helper()
	db, _ := dispatchTestDB(t)
	db.PriorityCallBudget = 20
	fx := dispatchSeed(t, db)
	evidence := map[string]AcceptanceEvidence{}
	for _, issue := range issues {
		decision := fx.accept(t, issue, fmt.Sprintf("d%024d", issue))
		fx.queue(t, issue, decision.ID)
		key := fmt.Sprintf("%d/%d", fx.repo, issue)
		input := fx.reads.inputs[key]
		evidence[key] = AcceptanceEvidence{Revision: input.Revision, Issue: AcceptanceIssueView{
			Index: fmt.Sprintf("%d", issue), TitleOccurrence: input.Issue.TitleOccurrence,
			TitleDigest: input.Issue.TitleDigest, ContentDigest: input.Issue.ContentDigest,
			ContentVer: input.Issue.ContentVer, Visible: true,
		}, Comments: []AcceptanceComment{{ID: "11", Digest: input.Comments[0].Digest, Visible: true}}}
	}
	// Launched runs stay live under inspection so recovery holds them
	// instead of proving them unused; settlement arrives through the
	// reconciled run observation below.
	launched := map[string]bool{}
	fx.host.inspect = func(in project.FactoryInspect) (project.FactoryState, error) {
		if launched[in.ID] {
			return project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryRunning}, nil
		}
		return project.FactoryState{}, host.ErrRunNotFound
	}
	inner := fx.host.launch
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		launched[in.Run.ID] = true
		return inner(in)
	}
	source := &fakeEvidenceSource{evidence: evidence}
	chooser := &planningChooser{}
	c := NewCoordinator(db, fx.host, fx.broker)
	c.AcceptanceReads, c.DispatchReads = source, fx.reads
	c.Planning = &PlanningJudgments{Journal: db, Funding: db, Chooser: chooser}
	return c, chooser, fx
}

// finishReportedInvestigation completes the stored owner with a reported
// executable plan bound to the owner's attempt root, source and
// preparation, then persists the finished assignment.
func finishReportedInvestigation(t *testing.T, ctx context.Context, db *store.Store, owner factory.Assignment, mutate func(*factory.InvestigationPlan)) factory.InvestigationPlan {
	t.Helper()
	plan := phaseTestPlan(t, owner.AttemptRoot, owner.SourceCommit, owner.Preparation)
	if mutate != nil {
		mutate(&plan)
		digest, err := factory.PlanDigest(plan)
		if err != nil {
			t.Fatal(err)
		}
		plan.Digest = digest
		if err := plan.Validate(); err != nil {
			t.Fatalf("mutated plan invalid: %v", err)
		}
	}
	run, err := db.FactoryRun(ctx, owner.Run)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "host says so", true
	if err := db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatalf("investigation run unsettled: %v", err)
	}
	output := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionInvestigate, Status: "completed",
		Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
	})
	if _, ok := AccountSettledRun(ctx, db, run, output, time.Now()); !ok {
		t.Fatal("completed investigation unaccounted")
	}
	return plan
}

// T10 (FW12 scope): two ready issues produce one persisted investigation.
// The unselected issue gets no assignment and no prompt run; a replayed
// pass spends no duplicate investigation.
func TestSelectedInvestigationAdmitsOnce(t *testing.T) {
	c, _, fx := admissionCoordinator(t, 3, 4)
	ctx := context.Background()
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	report := c.Dispatch(ctx)
	if len(report.Launched) != 1 || len(fx.host.launches) != 1 {
		t.Fatalf("two ready issues launched %+v", report)
	}
	selected := report.Launched[0].Issue
	unselected := int64(3)
	if selected == 3 {
		unselected = 4
	}
	if _, err := c.Store.LatestIssueAssignment(ctx, fx.repo, selected); err != nil {
		t.Fatalf("selected investigation unpersisted: %v", err)
	}
	if _, err := c.Store.LatestIssueAssignment(ctx, fx.repo, unselected); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unselected issue incurred investigation: %v", err)
	}
	replay := c.Dispatch(ctx)
	if len(replay.Launched) != 0 || len(fx.host.launches) != 1 {
		t.Fatalf("replay duplicated investigation %+v", replay)
	}
}

// FW12: a completed investigation advances exactly once to its
// implementation child while persisting and admitting the reported plan
// head. Restart reuses the admitted phase instead of relaunching.
func TestCompletedInvestigationAdvancesOnce(t *testing.T) {
	c, _, fx := admissionCoordinator(t, 3)
	ctx := context.Background()
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	if report := c.Dispatch(ctx); len(report.Launched) != 1 {
		t.Fatalf("investigation did not launch %+v", report)
	}
	owner, err := c.Store.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatal(err)
	}
	plan := finishReportedInvestigation(t, ctx, c.Store, owner, nil)
	advance := c.Dispatch(ctx)
	if len(advance.Launched) != 1 {
		t.Fatalf("completed investigation did not advance %+v", advance)
	}
	child, err := c.Store.Assignment(ctx, advance.Launched[0].AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if child.Phase != project.ExecutionImplement || child.ParentAssignment != owner.ID {
		t.Fatalf("child is not the implementation advance: %+v", child)
	}
	head, _, err := c.Store.PlanHead(ctx, owner.AttemptRoot)
	if err != nil || head != plan.ID {
		t.Fatalf("plan head = %q, want %q (%v)", head, plan.ID, err)
	}
	restart := c.Dispatch(ctx)
	if len(restart.Launched) != 0 {
		t.Fatalf("restart relaunched the admitted phase %+v", restart)
	}
	if again, _, err := c.Store.PlanHead(ctx, owner.AttemptRoot); err != nil || again != plan.ID {
		t.Fatalf("restart moved the admitted head to %q (%v)", again, err)
	}
}

// FW12: a plan carrying unresolved mandatory facts is not executable and
// can never launch implementation. The issue waits for explicit Retry,
// and no plan head is persisted for it.
func TestUnresolvedPlanCannotLaunch(t *testing.T) {
	c, _, fx := admissionCoordinator(t, 3)
	ctx := context.Background()
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	if report := c.Dispatch(ctx); len(report.Launched) != 1 {
		t.Fatalf("investigation did not launch %+v", report)
	}
	owner, err := c.Store.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatal(err)
	}
	finishReportedInvestigation(t, ctx, c.Store, owner, func(plan *factory.InvestigationPlan) {
		plan.Unresolved = []string{"which endpoint owns the export"}
	})
	advance := c.Dispatch(ctx)
	if len(advance.Launched) != 0 {
		t.Fatalf("unresolved plan launched %+v", advance)
	}
	if reason := waitReason(advance, 3); reason != WaitAttemptRecorded {
		t.Fatalf("unresolved plan waits %q, want explicit retry", reason)
	}
	if _, _, err := c.Store.PlanHead(ctx, owner.AttemptRoot); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unresolved plan persisted a head: %v", err)
	}
}

// FW12: changed source inputs trigger reassessment instead of launching
// from the stale investigation. The moved tip first stages a source
// preparation rebuild; the stale head persists nothing either way.
func TestChangedSourceWithholdsAdvance(t *testing.T) {
	c, _, fx := admissionCoordinator(t, 3)
	ctx := context.Background()
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	if report := c.Dispatch(ctx); len(report.Launched) != 1 {
		t.Fatalf("investigation did not launch %+v", report)
	}
	owner, err := c.Store.LatestIssueAssignment(ctx, fx.repo, 3)
	if err != nil {
		t.Fatal(err)
	}
	finishReportedInvestigation(t, ctx, c.Store, owner, nil)
	moved := fx.reads.inputs[fmt.Sprintf("%d/3", fx.repo)]
	moved.Tip = strings.Repeat("d", 40)
	fx.reads.inputs[fmt.Sprintf("%d/3", fx.repo)] = moved
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		advance := c.Dispatch(ctx)
		if len(advance.Launched) != 0 {
			t.Fatalf("stale investigation launched %+v", advance)
		}
		if reason := waitReason(advance, 3); reason != WaitPreparation {
			t.Fatalf("moved source waits %q, want source preparation rebuild", reason)
		}
		if _, _, err := c.Store.PlanHead(ctx, owner.AttemptRoot); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("stale investigation persisted a head: %v", err)
		}
	}
}

// FW12: planning spends no judgment on an issue an admitted attempt
// already covers, even when its inputs moved. Selection eligibility owns
// the spend; the admitted phase owns the path.
func TestPlanningSkipsCoveredIssue(t *testing.T) {
	c, chooser, _ := admissionCoordinator(t, 3)
	ctx := context.Background()
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 {
		t.Fatalf("fresh planning spent %d calls", chooser.calls)
	}
	if report := c.Dispatch(ctx); len(report.Launched) != 1 {
		t.Fatalf("investigation did not launch %+v", report)
	}
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 {
		t.Fatalf("covered issue respent planning: %d calls", chooser.calls)
	}
}

// FW12: admission persists the reported revision once, admits the current
// head against live bindings, replays without duplicate writes, and
// refuses stale bindings, forged identities and unexecutable plans.
func TestAdmitInvestigationPlanBindings(t *testing.T) {
	db, _ := dispatchTestDB(t)
	ctx := context.Background()
	owner := phaseTestAssignment(t, project.ExecutionInvestigate, project.RoleCoder, true)
	plan := phaseTestPlan(t, owner.AttemptRoot, owner.SourceCommit, owner.Preparation)
	output := fencedResult(t, factory.PhaseResult{
		Version: 2, Phase: project.ExecutionInvestigate, Status: "completed",
		Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
	})
	finished, err := finishAssignmentFromRun(owner, phaseTestRun(owner, factory.Succeeded), output, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	live, err := livePlanBindings(factory.Acceptance{}, factory.IssueControl{}, owner.SourceCommit, owner.Preparation)
	if err != nil {
		t.Fatal(err)
	}
	admitted, err := admitInvestigationPlan(ctx, db, finished, live)
	if err != nil || admitted.Digest != plan.Digest {
		t.Fatalf("fresh plan unadmitted: %+v %v", admitted, err)
	}
	replayed, err := admitInvestigationPlan(ctx, db, finished, live)
	if err != nil || replayed.Digest != plan.Digest {
		t.Fatalf("replay did not reuse the admitted phase: %+v %v", replayed, err)
	}
	stale := live
	stale.SourceCommit = strings.Repeat("d", 40)
	if _, err := admitInvestigationPlan(ctx, db, finished, stale); !errors.Is(err, store.ErrStalePlanBindings) {
		t.Fatalf("stale source admitted: %v", err)
	}
	forged := plan
	forged.Approach = "a different approach under the same identity"
	forged.Digest = ""
	redigest, err := factory.PlanDigest(forged)
	if err != nil {
		t.Fatal(err)
	}
	forged.Digest = redigest
	if _, err := admitInvestigationPlan(ctx, db, withPlan(finished, forged), live); !errors.Is(err, store.ErrPlanConflict) {
		t.Fatalf("forged plan identity admitted: %v", err)
	}
}

func withPlan(owner factory.Assignment, plan factory.InvestigationPlan) factory.Assignment {
	phased := *owner.Result.PhaseResult
	out := *owner.Result.PhaseResult.Output
	out.Plan = &plan
	phased.Output = &out
	result := *owner.Result
	result.PhaseResult = &phased
	owner.Result = &result
	return owner
}

// FW12: prerequisite bindings are canonical over outcome references:
// order-free, sensitive to reference changes, and refused when a
// reference is missing.
func TestLivePlanBindingsCanonical(t *testing.T) {
	code := factory.AcceptedPrerequisite{
		Occurrence: "21", DependsOn: "9", EndpointRepo: 7, EndpointIssue: 9,
		Outcome: factory.PrereqCode, PrereqAcceptance: "d" + strings.Repeat("f", 24),
	}
	decision := factory.Acceptance{Prerequisites: []factory.AcceptedPrerequisite{code}}
	control := factory.IssueControl{EndpointHeads: map[string]string{"21": "d" + strings.Repeat("f", 24)}}
	tip, prep := strings.Repeat("c", 40), "f"+strings.Repeat("b", 24)
	first, err := livePlanBindings(decision, control, tip, prep)
	if err != nil || len(first.Prerequisites) != 1 {
		t.Fatalf("bindings unbuilt: %+v %v", first, err)
	}
	moved := factory.IssueControl{EndpointHeads: map[string]string{"21": "d" + strings.Repeat("0", 24)}}
	second, err := livePlanBindings(decision, moved, tip, prep)
	if err != nil {
		t.Fatal(err)
	}
	if first.Prerequisites[0] == second.Prerequisites[0] {
		t.Fatal("changed prerequisite reference kept its binding")
	}
	empty := factory.IssueControl{}
	if _, err := livePlanBindings(decision, empty, tip, prep); !errors.Is(err, errPlanInputsIncomplete) {
		t.Fatalf("missing reference built bindings: %v", err)
	}
}
