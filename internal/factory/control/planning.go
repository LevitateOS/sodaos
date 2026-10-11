package control

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

type ProductJudgment interface {
	Select(context.Context, any, string, map[string]any) (factory.PriorityAnswer, error)
}

type PlanningJournal interface {
	PlanningRecord(context.Context, int64, int64) (factory.PlanningRecord, error)
	SavePlanningRecord(context.Context, factory.PlanningRecord) error
	PlanningControlsAfter(context.Context, int, int64, int64, int64, bool) ([]factory.IssueControl, error)
	ReserveIssuePlanning(context.Context, string, string) (bool, error)
}

type PlanningJudgments struct {
	Journal PlanningJournal
	Funding PriorityJournal
	Chooser ProductJudgment
}

// PlanningPass records bounded implementation approaches over stable accepted
// scope, even while visible prerequisite outcomes are pending. Classification
// does not invent source changes or execute work; blocked portions stay explicit.
func (c *Coordinator) PlanningPass(ctx context.Context) error {
	ctx, unlock := c.progressPass(ctx)
	defer unlock()
	deps := c.Planning
	if deps == nil {
		return nil
	}
	if deps.Journal == nil || deps.Funding == nil || deps.Chooser == nil {
		return ErrPriorityUnavailable
	}
	seen, repo, issue, set := c.planningQueue.start()
	controls, err := deps.Journal.PlanningControlsAfter(ctx, 16, seen, repo, issue, set)
	if err != nil {
		return err
	}
	if set && len(controls) == 0 {
		c.planningQueue.reset()
		controls, err = deps.Journal.PlanningControlsAfter(ctx, 16, 0, 0, 0, false)
		if err != nil {
			return err
		}
	}
	for _, control := range controls {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		c.planningQueue.advance(control, false)
		if err := c.planIssue(ctx, control); err != nil {
			return err
		}
	}
	if len(controls) > 0 {
		c.planningQueue.advance(controls[len(controls)-1], len(controls) < 16)
	}
	return nil
}

func (c *Coordinator) planningInput(ctx context.Context, control factory.IssueControl) (factory.PlanningInput, error) {
	control, err := c.refreshReadiness(ctx, control)
	if err != nil {
		return factory.PlanningInput{}, err
	}
	if !factory.ReadyToPlan(control) {
		return factory.PlanningInput{}, ErrIneffectiveAuthority
	}
	status, evidence, err := c.acceptanceStatus(ctx, control.Repository, strconv.FormatInt(control.Issue, 10))
	if err != nil {
		return factory.PlanningInput{}, err
	}
	if !status.Valid || status.Acceptance == nil || evidence.Issue.Closed {
		return factory.PlanningInput{}, ErrIneffectiveAuthority
	}
	deps := c.dispatchDeps()
	authority, err := deps.Authority(ctx, control.Repository)
	if err != nil {
		return factory.PlanningInput{}, err
	}
	if !authority.Effective {
		return factory.PlanningInput{}, ErrIneffectiveAuthority
	}
	p, err := c.Store.ProjectByRepository(ctx, control.Repository)
	if err != nil {
		return factory.PlanningInput{}, err
	}
	policy, err := c.Store.RepositoryPolicy(ctx, control.Repository)
	if err != nil {
		return factory.PlanningInput{}, err
	}
	sponsorship, wait, failed := selectSponsorship(ctx, deps, control.Repository, authority, project.RoleCoder)
	if wait != nil || failed != nil || sponsorship.ProjectID != p.ID {
		return factory.PlanningInput{}, ErrIneffectiveAuthority
	}
	plan := &attemptPlan{control: control, acceptance: *status.Acceptance, effective: authority, projectID: p.ID, policy: policy, sponsorship: sponsorship}
	if wait, failed := selectPreparationFor(ctx, deps, plan, project.RoleCoder, ""); wait != nil || failed != nil {
		return factory.PlanningInput{}, ErrIneffectiveAuthority
	}
	if wait, failed := readAttemptInputs(ctx, deps, plan, control.Repository, control.Issue); wait != nil || failed != nil {
		return factory.PlanningInput{}, ErrPriorityUnavailable
	}
	if plan.inputs.Revision != evidence.Revision {
		return factory.PlanningInput{}, ErrPriorityUnavailable
	}
	plan.effective.Authority.RequirementsID, plan.effective.Authority.ApprovalID = plan.requirement, plan.approval
	open, gate, _, err := c.Store.DispatchState(ctx, control.Repository)
	if err != nil {
		return factory.PlanningInput{}, err
	}
	if !open {
		return factory.PlanningInput{}, ErrIneffectiveAuthority
	}
	plan.gateRev = gate
	return factory.PlanningInput{Candidate: priorityCandidate(plan), Missing: control.Blockers}, nil
}

const planningRubric = "Choose a bounded implementation approach using only accepted objective, approved preparation and observed prerequisite facts. Pending outputs and source-dependent interfaces remain provisional. Prefer useful work on stable acceptance scope when dependencies do not determine the entire approach; otherwise wait for concrete outputs. For executable scope, prefer direct small changes when evidence supports simplicity; use incremental acceptance work for multiple independent outcomes and prerequisite-first work when existing dependency outputs define interfaces. Do not invent estimates, expand scope, request additional access or authorize execution."

func planningChoices(provisional bool) map[string]any {
	if provisional {
		return map[string]any{"plan_stable_scope": "Record an acceptance-driven provisional outline; defer interfaces and source choices requiring missing outputs.", "wait_for_outputs": "Missing prerequisite results determine the implementation approach; retain a wait instead of speculating."}
	}
	return map[string]any{"direct_small_change": "Implement the accepted change directly using approved project setup, then verify acceptance outcomes.", "incremental_acceptance": "Implement and verify accepted outcomes in small increments before independent review.", "prerequisite_first": "Inspect verified prerequisite outputs and their interfaces, then implement the dependent acceptance outcomes."}
}

func (c *Coordinator) planIssue(ctx context.Context, control factory.IssueControl) error {
	input, err := c.planningInput(ctx, control)
	if errors.Is(err, ErrIneffectiveAuthority) {
		return nil
	}
	if err != nil {
		return err
	}
	snapshot, err := factory.PlanningSnapshot(input)
	if err != nil {
		return err
	}
	prior, err := c.Planning.Journal.PlanningRecord(ctx, control.Repository, control.Issue)
	if err == nil && prior.Snapshot == snapshot {
		return nil
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	reserved, err := c.Planning.Journal.ReserveIssuePlanning(ctx, control.Acceptance, snapshot)
	if err != nil {
		return err
	}
	if !reserved {
		return nil
	}
	record := factory.PlanningRecord{Snapshot: snapshot, Input: input, Provisional: len(input.Missing) > 0, CreatedUnix: time.Now().Unix()}
	choices := planningChoices(record.Provisional)
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	answer, callErr := c.Planning.Chooser.Select(bounded, input, planningRubric, choices)
	cancel()
	record.Judgment = answer
	request := factory.PriorityRequest{Snapshot: snapshot, Rubric: planningRubric}
	request.State, _ = json.Marshal(input)
	request.Criteria = choices
	for id := range choices {
		request.Candidates = append(request.Candidates, factory.PriorityCandidate{ID: id})
	}
	if callErr == nil {
		callErr = answer.Validate(request)
	}
	call := factory.PriorityCall{Request: request, Answer: answer}
	if callErr != nil {
		record.Outcome = "unavailable"
		call.Failure = "consultation_failed"
	} else {
		record.Strategy = answer.Selected
		if answer.Confidence < 0.5 {
			if record.Provisional {
				record.Strategy = "wait_for_outputs"
			} else {
				record.Strategy = "incremental_acceptance"
			}
		}
		record.Outcome = "planned"
		if record.Strategy == "wait_for_outputs" {
			record.Outcome = "waiting"
		} else {
			record.Steps = planningSteps(record.Strategy)
		}
	}
	receipt := factory.PriorityReceipt{Snapshot: snapshot, Candidates: []factory.PriorityCandidate{input.Candidate}, Calls: []factory.PriorityCall{call}, Selected: record.Strategy, Outcome: record.Outcome, CreatedUnix: record.CreatedUnix}
	if err := c.Planning.Funding.RecordPriorityJudgment(ctx, receipt); err != nil {
		return err
	}
	// Persist a stale outcome for attribution, but no stale plan can pass the
	// execution check. A later pass rebuilds against the new relevant inputs.
	return c.Planning.Journal.SavePlanningRecord(ctx, record)
}

func planningSteps(strategy string) []string {
	first := "Read the exact accepted objective and approved setup before changing source."
	if strategy == "prerequisite_first" {
		first = "Inspect the recorded verified prerequisite outputs and source interfaces."
	}
	if strategy == "plan_stable_scope" {
		return []string{"Map accepted outcomes onto the approved project setup.", "Keep interfaces and source choices depending on unfinished prerequisites provisional.", "Refresh this plan against actual prerequisite outputs before executing."}
	}
	return []string{first, "Implement only accepted outcomes using the approved source and setup.", "Run the recorded acceptance checks and submit to independent review."}
}

func (c *Coordinator) planningCurrent(ctx context.Context, control factory.IssueControl) bool {
	if c.Planning == nil {
		current, generated, err := c.Store.GeneratedIntentCurrent(ctx, control.Acceptance)
		return err == nil && current && !generated
	}
	input, err := c.planningInput(ctx, control)
	if err != nil {
		return false
	}
	snapshot, err := factory.PlanningSnapshot(input)
	if err != nil {
		return false
	}
	record, err := c.Planning.Journal.PlanningRecord(ctx, control.Repository, control.Issue)
	return err == nil && record.Snapshot == snapshot && record.Outcome == "planned" && !record.Provisional
}

func (c *Coordinator) planningRecord(ctx context.Context, repository, issue int64) (factory.PlanningRecord, error) {
	if c.Planning == nil {
		return factory.PlanningRecord{}, nil
	}
	if c.Planning.Journal == nil {
		return factory.PlanningRecord{}, ErrPriorityUnavailable
	}
	return c.Planning.Journal.PlanningRecord(ctx, repository, issue)
}
