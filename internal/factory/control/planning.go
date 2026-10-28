package control

import (
	"context"
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

var errNonCodePlanning = errors.New("accepted non-code work has no implementation plan")

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
	control, status, evidence, err := c.admitPlanningReadiness(ctx, control)
	if err != nil {
		return factory.PlanningInput{}, err
	}
	deps, authority, p, policy, err := c.admitPlanningAuthority(ctx, control, status)
	if err != nil {
		return factory.PlanningInput{}, err
	}
	plan := &attemptPlan{control: control, acceptance: *status.Acceptance, effective: authority, projectID: p.ID, policy: policy}
	if wait, failed := readAttemptInputs(ctx, deps, plan, control.Repository, control.Issue); wait != nil || failed != nil {
		return factory.PlanningInput{}, ErrPriorityUnavailable
	}
	block, err := factory.ParseSodaIssueBlock(plan.inputs.Issue.Body)
	if err != nil {
		return factory.PlanningInput{}, err
	}
	if block.Role == factory.SodaRoleResearch || block.Role == factory.SodaRoleLibraryAdmin {
		return factory.PlanningInput{}, errNonCodePlanning
	}
	sponsorship, wait, failed := selectSponsorship(ctx, deps, control.Repository, authority, project.RoleCoder)
	if wait != nil || failed != nil || sponsorship.ProjectID != p.ID {
		return factory.PlanningInput{}, ErrIneffectiveAuthority
	}
	plan.sponsorship = sponsorship
	if wait, failed := selectPreparationFor(ctx, deps, plan, project.RoleCoder, ""); wait != nil || failed != nil {
		return factory.PlanningInput{}, ErrIneffectiveAuthority
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

// admitPlanningReadiness refreshes readiness behind one planning input
// with its accepted evidence.
func (c *Coordinator) admitPlanningReadiness(ctx context.Context, control factory.IssueControl) (factory.IssueControl, AcceptanceValidity, AcceptanceEvidence, error) {
	var emptyStatus AcceptanceValidity
	var emptyEvidence AcceptanceEvidence
	control, err := c.refreshReadiness(ctx, control)
	if err != nil {
		return control, emptyStatus, emptyEvidence, err
	}
	if !factory.ReadyToPlan(control) {
		return control, emptyStatus, emptyEvidence, ErrIneffectiveAuthority
	}
	status, evidence, err := c.acceptanceStatus(ctx, control.Repository, strconv.FormatInt(control.Issue, 10))
	if err != nil {
		return control, emptyStatus, emptyEvidence, err
	}
	if !status.Valid || status.Acceptance == nil || evidence.Issue.Closed {
		return control, emptyStatus, emptyEvidence, ErrIneffectiveAuthority
	}
	return control, status, evidence, nil
}

// admitPlanningAuthority resolves dispatch authority, project and
// policy behind one planning input.
func (c *Coordinator) admitPlanningAuthority(ctx context.Context, control factory.IssueControl, status AcceptanceValidity) (DispatchDeps, factory.EffectiveAuthority, store.Project, factory.RepositoryPolicy, error) {
	var noProject store.Project
	var noPolicy factory.RepositoryPolicy
	deps := c.dispatchDeps()
	authority, err := deps.Authority(ctx, control.Repository)
	if err != nil {
		return deps, factory.EffectiveAuthority{}, noProject, noPolicy, err
	}
	if !authority.Effective {
		return deps, factory.EffectiveAuthority{}, noProject, noPolicy, ErrIneffectiveAuthority
	}
	p, err := c.Store.ProjectByRepository(ctx, control.Repository)
	if err != nil {
		return deps, factory.EffectiveAuthority{}, noProject, noPolicy, err
	}
	policy, err := c.Store.RepositoryPolicy(ctx, control.Repository)
	if err != nil {
		return deps, factory.EffectiveAuthority{}, noProject, noPolicy, err
	}
	return deps, authority, p, policy, nil
}

const planningRubric = "Choose a bounded implementation approach using only accepted objective, approved preparation and observed prerequisite facts. Pending outputs and source-dependent interfaces remain provisional. Prefer useful work on stable acceptance scope when dependencies do not determine the entire approach; otherwise wait for concrete outputs. For executable scope, prefer direct small changes when evidence supports simplicity; use incremental acceptance work for multiple independent outcomes and prerequisite-first work when existing dependency outputs define interfaces. Do not invent estimates, expand scope, request additional access or authorize execution."

func planningChoices(provisional bool) map[string]any {
	if provisional {
		return map[string]any{"plan_stable_scope": "Record an acceptance-driven provisional outline; defer interfaces and source choices requiring missing outputs.", "wait_for_outputs": "Missing prerequisite results determine the implementation approach; retain a wait instead of speculating."}
	}
	return map[string]any{"direct_small_change": "Implement the accepted change directly using approved project setup, then verify acceptance outcomes.", "incremental_acceptance": "Implement and verify accepted outcomes in small increments before independent review.", "prerequisite_first": "Inspect verified prerequisite outputs and their interfaces, then implement the dependent acceptance outcomes."}
}

func (c *Coordinator) planIssue(ctx context.Context, control factory.IssueControl) error {
	covered, err := c.investigationCovered(ctx, control)
	if err != nil {
		return err
	}
	if covered {
		return nil
	}
	input, err := c.planningInput(ctx, control)
	if errors.Is(err, ErrIneffectiveAuthority) || errors.Is(err, errNonCodePlanning) {
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
	if err == nil && planningRecordCurrent(prior, input) {
		return nil
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	provisional := len(input.Missing) > 0
	choices := planningChoices(provisional)
	answer, _, err := c.AdvisePurpose(ctx, PurposeAdvice{
		Purpose:    factory.JudgmentPlanningFreshness,
		State:      input,
		Question:   planningRubric,
		Options:    choices,
		Outcome:    "planning",
		Candidates: []factory.PriorityCandidate{input.Candidate},
		Reserve: func(ctx context.Context, fingerprint string) (bool, error) {
			return c.Planning.Journal.ReserveIssuePlanning(ctx, control.Acceptance, fingerprint)
		},
	})
	record := factory.PlanningRecord{Snapshot: snapshot, Input: input, Provisional: provisional, CreatedUnix: time.Now().Unix()}
	if err != nil {
		if errors.Is(err, ErrPriorityUnavailable) {
			// Nothing was recorded; the next pass asks again.
			return nil
		}
		return err
	}
	record.Judgment = answer
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
	}
	// Persist a stale outcome for attribution, but no stale plan can pass the
	// execution check. A later pass rebuilds against the new relevant inputs.
	return c.Planning.Journal.SavePlanningRecord(ctx, record)
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
	record, err := c.Planning.Journal.PlanningRecord(ctx, control.Repository, control.Issue)
	return err == nil && planningRecordCurrent(record, input) && record.Outcome == "planned" && !record.Provisional
}

// A recorded implementation approach is not an authority grant. Pause/Resume
// and credential returns change authorization revisions without changing the
// work it judged. Keep the original receipt and compare its material inputs;
// planningInput and atomic dispatch still require current effective authority.
func planningRecordCurrent(record factory.PlanningRecord, input factory.PlanningInput) bool {
	if record.Validate() != nil {
		return false
	}
	for _, candidate := range []*factory.PriorityCandidate{&record.Input.Candidate, &input.Candidate} {
		candidate.Control.Revision = 0
		candidate.Control.Fingerprint = ""
		candidate.Control.Authority = ""
		candidate.Authority = factory.AuthorityRef{
			RequirementsID:        candidate.Authority.RequirementsID,
			ApprovalID:            candidate.Authority.ApprovalID,
			SponsorshipConnection: candidate.Authority.SponsorshipConnection,
		}
		candidate.GateRevision = 0
	}
	prior, err := factory.PlanningSnapshot(record.Input)
	if err != nil {
		return false
	}
	current, err := factory.PlanningSnapshot(input)
	return err == nil && prior == current
}
