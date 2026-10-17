package control

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// progress_phase_fixture_test.go — NATIVE100 canonical wake fixture outputs.
//
// The dedicated wake-test hosts must emit the CURRENT typed v2
// phase/assignment/subject contract production settles (dispatch_result.go
// derivePhasedAttemptResult): a reported investigate plan, an implement
// receipt, and a typed review clear. Legacy pre-phase children (CaseA's
// reviewer under its pre-phase owner) keep the legacy fences via the
// hosts' default branch in progress_test.go. All builders are t-free and
// report errors so hosts can record (never panic) them; the tests assert
// the recorded errors stay empty. No production/store/domain/host/parser/
// gate/docs behavior is touched here.

// wakePhaseFence renders one v2 phase result behind the harness fence the
// supervisor parses. Unknown fields refuse at parse; the payload must be
// exactly the admitted phase shape.
func wakePhaseFence(result factory.PhaseResult) (string, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return "work log\n```" + factory.ResultFence + "\n" + string(payload) + "\n```\ntrailer\n", nil
}

// wakeInvestigatePlanFor builds the real typed investigate plan one wake
// investigator reports. It mirrors the canonical phased seed plan content
// (single task-one lane with its check-one evidence) bound to the live
// root: its attempt root, source commit and preparation, so advancement
// (advanceChildID) and plan-head admission (admitInvestigationPlan) accept
// it. The seed acceptance carries no prerequisites, so the prerequisite
// binding stays empty and stable across the test's readiness revision bump.
func wakeInvestigatePlanFor(root factory.Assignment) (factory.InvestigationPlan, error) {
	var empty factory.InvestigationPlan
	plan := factory.InvestigationPlan{
		AttemptRoot: root.AttemptRoot,
		ScopeDigest: dispatchDigest("wake scope"), InputDigest: dispatchDigest("wake input"),
		SourceCommit: root.SourceCommit, IntegrationCommit: root.SourceCommit,
		Preparation:        root.Preparation,
		Approach:           "inspect the selected source, then implement behind checks",
		AcceptanceCoverage: []string{"accept-one"},
		Tasks: []factory.PlanTask{{
			ID: "task-one", Objective: "objective 31; inspect README.md",
			Outputs: []string{"notes"}, AcceptanceRefs: []string{"accept-one"},
			Lane: "lane-one", OwnedPaths: []string{"README.md"}, CheckIDs: []string{"check-one"},
		}},
		Lanes: []factory.PlanLane{{ID: "lane-one", TaskIDs: []string{"task-one"}}},
		Checks: []factory.CheckSpec{{
			ID: "check-one", Argv: []string{"make", "check"}, Cwd: ".",
			TimeoutSeconds: 60, Scope: factory.CheckScopeTask, Required: true,
		}},
	}
	// The plan identity is fresh per report; the root stays the attempt
	// binding advancement and admission check.
	plan.ID = factory.NewID()
	digest, err := factory.PlanDigest(plan)
	if err != nil {
		return empty, err
	}
	plan.Digest = digest
	if err := plan.Validate(); err != nil {
		return empty, err
	}
	if err := plan.Executable(); err != nil {
		return empty, err
	}
	return plan, nil
}

// wakeInvestigateOutput reports the live investigate run's typed plan. The
// plan is bound from the recorded assignment so it always names the exact
// root, source and preparation production admitted.
func wakeInvestigateOutput(ctx context.Context, db *store.Store, runID string) (string, error) {
	a, err := db.AssignmentByRun(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("wake investigate assignment unreadable: %w", err)
	}
	plan, err := wakeInvestigatePlanFor(a)
	if err != nil {
		return "", fmt.Errorf("wake investigate plan invalid: %w", err)
	}
	return wakePhaseFence(factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionInvestigate, Status: "completed",
		Summary: "plan ready", Output: &factory.PhaseOutput{Plan: &plan},
	})
}

// ensureWakeTaskEvidence records the declared trusted task evidence one
// phased implement publication joins (FW15 gatePhasedPublication): every
// planned task terminally completed with its checkpoint commit. The plan
// head is admitted by production before the implement child dispatches,
// so this runs at implement launch. Tasks already completed are left
// alone; anything else is advanced through the exact pending to completed
// chain.
func ensureWakeTaskEvidence(ctx context.Context, db *store.Store, child factory.Assignment) error {
	headID, _, err := db.PlanHead(ctx, child.AttemptRoot)
	if err != nil {
		return fmt.Errorf("wake plan head unreadable: %w", err)
	}
	plan, err := db.InvestigationPlan(ctx, child.AttemptRoot, headID)
	if err != nil {
		return fmt.Errorf("wake plan unreadable: %w", err)
	}
	receipts, err := db.PlanTaskReceipts(ctx, child.AttemptRoot, headID)
	if err != nil {
		return fmt.Errorf("wake task receipts unreadable: %w", err)
	}
	completed := map[string]bool{}
	for _, receipt := range receipts {
		if receipt.State == store.PlanTaskCompleted {
			completed[receipt.TaskID] = true
		}
	}
	for _, task := range plan.Tasks {
		if completed[task.ID] {
			continue
		}
		chain := []store.PlanTaskReceipt{
			{AttemptRoot: child.AttemptRoot, PlanID: plan.ID, TaskID: task.ID, Revision: 0, State: store.PlanTaskPending},
			{AttemptRoot: child.AttemptRoot, PlanID: plan.ID, TaskID: task.ID, Revision: 0, State: store.PlanTaskRunning, Child: "child-task-one"},
			{AttemptRoot: child.AttemptRoot, PlanID: plan.ID, TaskID: task.ID, Revision: 1, State: store.PlanTaskOutputReady, Child: "child-task-one", OutputDigest: dispatchDigest("task output")},
			{AttemptRoot: child.AttemptRoot, PlanID: plan.ID, TaskID: task.ID, Revision: 2, State: store.PlanTaskCompleted, Child: "child-task-one", OutputDigest: dispatchDigest("task output"), Commit: phasedSeedCandidate, Tree: phasedSeedCandidate, Reason: "planned task finished with its checkpoint commit"},
		}
		for _, receipt := range chain {
			if err := db.SavePlanTaskReceipt(ctx, receipt); err != nil {
				return fmt.Errorf("wake task evidence unrecorded: %w", err)
			}
		}
	}
	return nil
}

// wakeImplementOutput reports the live implement run's typed receipt with
// the exact candidate the wake journey publishes, and records the trusted
// task evidence its publication joins.
func wakeImplementOutput(ctx context.Context, db *store.Store, runID string) (string, error) {
	a, err := db.AssignmentByRun(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("wake implement assignment unreadable: %w", err)
	}
	if err := ensureWakeTaskEvidence(ctx, db, a); err != nil {
		return "", err
	}
	return wakePhaseFence(factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionImplement, Status: "completed",
		Summary: "implemented",
		Output:  &factory.PhaseOutput{Implementation: &factory.ImplementationReceipt{Candidate: phasedSeedCandidate}},
	})
}

// wakeReviewClearOutput reports the live reviewer run's typed explicit
// clear. With no carried findings the clear binds the exact reviewed
// candidate and PR base with an empty assessment set and the canonical
// empty-set coverage digest. Cycle and base come from the recorded
// publication so the consumer binds the verdict to its round.
func wakeReviewClearOutput(ctx context.Context, db *store.Store, runID string) (string, error) {
	a, err := db.AssignmentByRun(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("wake reviewer assignment unreadable: %w", err)
	}
	p, err := db.PublicationByAssignment(ctx, a.PublicationAssignment)
	if err != nil {
		return "", fmt.Errorf("wake reviewer publication unreadable: %w", err)
	}
	cycleID := ""
	for _, cycle := range p.ReviewCycles {
		if cycle.ReviewerAssignment == a.ID {
			cycleID = cycle.ID
		}
	}
	if cycleID == "" {
		return "", fmt.Errorf("wake reviewer names no recorded cycle for %s", a.ID)
	}
	contentDigest, err := factory.FindingSetDigest(nil)
	if err != nil {
		return "", fmt.Errorf("wake clear content digest unavailable: %w", err)
	}
	coverage, err := factory.ClearCoverageDigest(p.Candidate, p.PRCreate.BaseOID, contentDigest)
	if err != nil {
		return "", fmt.Errorf("wake clear coverage unavailable: %w", err)
	}
	return wakePhaseFence(factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: project.ExecutionReview, Status: "completed",
		Summary: "verdict recorded",
		Output: &factory.PhaseOutput{Clear: &factory.ClearReviewReport{
			CycleID: cycleID, Conclusion: factory.ClearReviewConclusion,
			Head: p.Candidate, BaseOID: p.PRCreate.BaseOID, CoverageDigest: coverage,
		}},
	})
}
