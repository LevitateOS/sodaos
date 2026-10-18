package control

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// attemptLineage carries the FW11 phase/lineage admission for one planned
// attempt: the admitted execution phase, the immediate parent for advanced
// children, and the validated investigation plan an implement child
// executes. The zero value is the pre-phase legacy path. Intent subjects
// enter through the intent-bound admission (intent_bound.go) and completion
// subjects through FW30; this boundary admits issue subjects only and
// refuses every other phased kind.
type attemptLineage struct {
	advancePlan *factory.InvestigationPlan
	phase       string
	parent      string
}

// attemptPlan is everything one dispatch attempt verified, in check order:
// the bound authority and grants, the exact preparation, harness pin and
// source, the verified prompt bytes, and the deadline the reservation
// will hold.
type attemptPlan struct {
	effective            factory.EffectiveAuthority
	policy               factory.RepositoryPolicy
	sponsorship          factory.Sponsorship
	prep                 project.StoredPreparation
	pin                  project.FactoryHarnessPin
	acceptance           factory.Acceptance
	control              factory.IssueControl
	subject              factory.ExecutionSubject
	advancePlan          *factory.InvestigationPlan
	inputs               DispatchInputs
	projectID            string
	attemptRoot          string
	preparationOwner     string
	phase                string
	parent               string
	prompt               []byte
	promptSHA            string
	deadline             time.Time
	requirement          string
	approval             string
	planned              int
	gateRev              int64
	applianceConcurrent  int
	repositoryConcurrent int
	freshAttempt         bool
	attemptLimits        factory.AttemptLimits
}

// dispatchOne dispatches one queued issue or reports exactly why it
// waited. Checks run in fixed order so the same state reports the same reason.
// An explicit Retry can retain bounded private preparation custody while waiting.
type admittedDispatchAttempt struct {
	latest        factory.Assignment
	head          string
	attemptClosed bool
	hasAssignment bool
}

func admitDispatchAttempt(ctx context.Context, deps DispatchDeps, control factory.IssueControl, report *DispatchReport) (admittedDispatchAttempt, bool) {
	var admitted admittedDispatchAttempt
	repository, issue := control.Repository, control.Issue
	latest, err := deps.Store.LatestIssueAssignment(ctx, repository, issue)
	hasAssignment := err == nil
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "assignments unreadable"})
		return admitted, false
	}
	attemptClosed := false
	if hasAssignment {
		attemptClosed, err = deps.Store.CloseExpiredAttempt(ctx, latest.ID)
		if err != nil {
			report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "attempt allowance unreadable"})
			return admitted, false
		}
		if attemptClosed && latest.Stage != factory.AssignmentFinished {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAllowance, Detail: "attempt is closed; maintainer intervention is required"})
			return admitted, false
		}
	}
	if hasAssignment && latest.Stage != factory.AssignmentFinished {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAssigned})
		return admitted, false
	}
	head, err := deps.Store.AcceptanceHead(ctx, repository, issue)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "acceptance head unreadable"})
		return admitted, false
	}
	return admittedDispatchAttempt{latest: latest, head: head, attemptClosed: attemptClosed, hasAssignment: hasAssignment}, true
}

func resolveDispatchRetry(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, control factory.IssueControl, admitted admittedDispatchAttempt, report *DispatchReport) (*factory.RetryDecision, bool) {
	repository, issue := control.Repository, control.Issue
	latest, head, attemptClosed, hasAssignment := admitted.latest, admitted.head, admitted.attemptClosed, admitted.hasAssignment
	var retry *factory.RetryDecision
	if hasAssignment && latest.Acceptance == head {
		pending, retryErr := deps.Store.PendingExplicitRetry(ctx, repository, issue, head)
		if errors.Is(retryErr, store.ErrNotFound) {
			// No explicit Retry waits: a completed investigation
			// advances to its implementation child instead of
			// requiring Retry. A closed root never advances.
			if !attemptClosed && maybeAdvanceCompletedPlanning(ctx, deps, occupancy, control, latest, head, report) {
				return nil, false
			}
			detail := "attempt result is recorded; explicit Retry is required"
			if attemptClosed {
				detail = "attempt is closed; maintainer intervention is required"
			}
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAttemptRecorded, Detail: detail})
			return nil, false
		}
		if retryErr != nil {
			report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "explicit retry unreadable"})
			return nil, false
		}
		retry = &pending
	}
	if attemptClosed && retry == nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAttemptRecorded, Detail: "attempt is closed; maintainer intervention is required"})
		return nil, false
	}
	return retry, true
}

func dispatchOne(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, control factory.IssueControl, report *DispatchReport) {
	admitted, ok := admitDispatchAttempt(ctx, deps, control, report)
	if !ok {
		return
	}
	retry, ok := resolveDispatchRetry(ctx, deps, occupancy, control, admitted, report)
	if !ok {
		return
	}
	launchDispatchPlan(ctx, deps, occupancy, control, admitted.head, retry, report)
}

func launchDispatchPlan(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, control factory.IssueControl, head string, retry *factory.RetryDecision, report *DispatchReport) {
	repository, issue := control.Repository, control.Issue
	if deps.Refresh != nil {
		refreshed, err := deps.Refresh(ctx, control)
		if err != nil {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsUnavailable, Detail: "readiness refresh is unavailable"})
			return
		}
		control = refreshed
	}
	plan, wait, failed := planAttempt(ctx, deps, occupancy, repository, issue, head, control, retry)
	if failed != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: failed.Reason, Detail: failed.Detail})
		return
	}
	if wait != nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: wait.Reason, Detail: wait.Detail})
		return
	}
	if deps.PlanCurrent != nil && !deps.PlanCurrent(ctx, control) {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: "plan_not_current", Detail: "implementation planning must refresh against current outputs"})
		return
	}
	if deps.collectPlan != nil {
		deps.collectPlan(plan)
		return
	}
	if deps.checkPlan != nil && !deps.checkPlan(plan) {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsStale, Detail: "priority input snapshot changed"})
		return
	}
	// D13 completion fence (FW31): planning above keeps its shape so
	// observation and probes continue; only admission waits. Retry and
	// automatic advancement funnel through the same packet gate below,
	// which refuses authoritatively through ErrCompletionFenced.
	if checkDispatchFence(ctx, deps, repository, issue, report) {
		return
	}
	executeFreshAttempt(ctx, deps, occupancy, plan, repository, issue, report, retry)
}

// planWait is one non-launch outcome: either a retryable wait or an
// unexpected failure. Exactly one of the dispatchOne results is set.
type planWait struct {
	Reason string
	Detail string
}

func waitFor(reason, detail string) *planWait { return &planWait{Reason: reason, Detail: detail} }

// admissionWait maps an atomic packet refusal to its wait. A refusal
// proves a concurrent admission consumed the room first; the issue waits
// for a later pass instead of launching.
func admissionWait(err error) *planWait {
	switch {
	case errors.Is(err, store.ErrCapacityFull):
		return waitFor(WaitCapacity, "appliance runs at its limit")
	case errors.Is(err, store.ErrRepositoryFull):
		return waitFor(WaitRepository, "repository runs at its limit")
	case errors.Is(err, store.ErrSponsorshipFull):
		return waitFor(WaitSponsorship, "sponsorship runs at its limit")
	case errors.Is(err, store.ErrAllowanceExhausted):
		return waitFor(WaitAllowance, "sponsorship allowance is exhausted")
	case errors.Is(err, factory.ErrAttemptTimeExhausted):
		return waitFor(WaitAllowance, "attempt active-time allowance is exhausted")
	case errors.Is(err, factory.ErrAttemptClosed):
		return waitFor(WaitAttemptRecorded, "attempt is terminal; explicit Retry is required")
	case errors.Is(err, store.ErrConnectionUsageBudget):
		return waitFor(WaitAllowance, "connection rolling usage budget is exhausted")
	case errors.Is(err, store.ErrAdmissionChanged):
		return waitFor(WaitAuthority, "grants changed during dispatch")
	case errors.Is(err, store.ErrCompletionFenced):
		return waitFor(WaitReassessmentPending, "completion fence holds admissions until the reassessment decision releases")
	default:
		return nil
	}
}

// refreshOccupancy reloads the pass snapshot after a lost admission race
// so later issues plan against the winner's reservation instead of stale
// emptiness. The packet gate stays authoritative: if the reload itself
// fails, the pass continues and the gate refuses any over-admission.
func refreshOccupancy(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, report *DispatchReport) {
	fresh, err := snapshotOccupancy(ctx, deps.Store)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "capacity accounting unavailable"})
		return
	}
	*occupancy = fresh
}

// planAttempt verifies one dispatch end to end. Explicit Retry first records
// custody for its fresh preparation; run admission waits for confirmed readiness.
// Every failed check reports its wait or failure. Fresh selected-issue work
// always opens with investigation: planning first, implementation only as
// the advanced child of completed planning.
func planAttempt(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, repository, issue int64, head string, control factory.IssueControl, retry *factory.RetryDecision) (*attemptPlan, *planWait, *planWait) {
	id := ""
	if retry != nil {
		id = retry.CommandID
	}
	return planAttemptForRole(ctx, deps, occupancy, repository, issue, head, control, retry != nil, project.RoleCoder, "", false, id, attemptLineage{phase: project.ExecutionInvestigate})
}

// planAttemptForRole shares the mutable dispatch checks between initial coder
// work and retries. A recorded child supplies its own role and exact approved
// preparation; callers keep its prompt bytes instead of rebuilding a new plan.
// Phased callers additionally name the admitted phase, the advanced parent
// and, for a rebuilt implementation prompt, the recorded investigation plan.
// Only investigation opens a phased root and only a planned implementation
// advances under its parent; review and correction children of a published
// publication keep their caller-staged prompt under their parent instead.
// Every other phased combination refuses, with intent subjects arriving
// through the intent-bound admission and other phases through their owning
// boundaries.
func admitAttemptLineage(plan *attemptPlan, lineage attemptLineage, role string, recordedPrompt bool, issue int64, head string) (*planWait, *planWait) {
	if lineage.phase == "" {
		return nil, nil
	}
	if lineage.phase != project.ExecutionInvestigate && lineage.phase != project.ExecutionImplement &&
		lineage.phase != project.ExecutionReview && lineage.phase != project.ExecutionCorrect {
		return nil, waitFor("phase_not_admitted", "execution phase enters through its owning boundary")
	}
	if !factory.ValidPhaseRole(lineage.phase, role) {
		return nil, waitFor("phase_not_admitted", "assignment role does not match its execution phase")
	}
	if failed := admitAttemptParent(lineage, recordedPrompt); failed != nil {
		return nil, failed
	}
	plan.subject = factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: issue, Acceptance: head}}
	if !factory.SubjectKindAllowed(plan.subject.Kind, lineage.phase) {
		return nil, waitFor("phase_not_admitted", "execution subject cannot enter its phase")
	}
	return nil, nil
}

func admitAttemptParent(lineage attemptLineage, recordedPrompt bool) *planWait {
	if lineage.parent == "" {
		if lineage.phase != project.ExecutionInvestigate {
			return waitFor("phase_not_admitted", "only investigation opens a phased attempt root")
		}
		return nil
	}
	switch lineage.phase {
	case project.ExecutionImplement:
		if !recordedPrompt && lineage.advancePlan == nil {
			return waitFor("phase_not_admitted", "only a planned implementation advances under its parent")
		}
	case project.ExecutionReview, project.ExecutionCorrect:
		// Publication children never rebuild prompts: the caller
		// stages the admitted publication prompt carrying the
		// exact current review/check evidence, and planning never
		// invents review or correction content. The investigation
		// plan slot stays empty on this path; the correction
		// manifest is staged from the derived singleton plan at
		// launch instead.
		if !recordedPrompt || lineage.advancePlan != nil {
			return waitFor("phase_not_admitted", "publication children keep their staged prompt under their parent")
		}
	default:
		return waitFor("phase_not_admitted", "only a planned implementation advances under its parent")
	}
	return nil
}

func loadAttemptPlanBasis(ctx context.Context, deps DispatchDeps, plan *attemptPlan, repository int64, role, head string) (factory.RepositoryPolicy, *planWait, *planWait) {
	var empty factory.RepositoryPolicy
	effective, err := deps.Authority(ctx, repository)
	if err != nil {
		return empty, nil, waitFor(DispatchErrStore, "authority unreadable")
	}
	plan.effective = effective
	if !effective.Effective {
		if onlyDispatchClosed(effective) {
			return empty, waitFor(WaitDispatchClosed, "dispatch gate is closed"), nil
		}
		return empty, waitFor(WaitAuthority, "grants do not authorize dispatch"), nil
	}
	p, err := deps.Store.ProjectByRepository(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return empty, waitFor(WaitPreparation, "repository has no project"), nil
		}
		return empty, nil, waitFor(DispatchErrStore, "project unreadable")
	}
	plan.projectID = p.ID
	policy, err := deps.Store.RepositoryPolicy(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return empty, waitFor(WaitAuthority, "policy changed during dispatch"), nil
		}
		return empty, nil, waitFor(DispatchErrStore, "policy unreadable")
	}
	plan.policy = policy
	sponsorship, wait, failed := selectSponsorship(ctx, deps, repository, effective, role)
	if failed != nil || wait != nil {
		return empty, wait, failed
	}
	plan.sponsorship = sponsorship
	if sponsorship.ProjectID != plan.projectID {
		return empty, waitFor(WaitAuthority, "sponsorship grant belongs to a different Project"), nil
	}
	decision, err := deps.Store.AcceptanceDecision(ctx, head)
	if err != nil {
		return empty, nil, waitFor(DispatchErrStore, "acceptance decision unreadable")
	}
	plan.acceptance = decision
	return policy, nil, nil
}

func prepareAttemptInputs(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, plan *attemptPlan, role, preparation string, repository, issue int64, control factory.IssueControl) (*planWait, *planWait) {
	if wait, failed := checkLimits(ctx, deps, occupancy, plan); failed != nil || wait != nil {
		return wait, failed
	}
	if wait, failed := selectPreparationFor(ctx, deps, plan, role, preparation); failed != nil || wait != nil {
		return wait, failed
	}
	plan.effective.Authority.RequirementsID = plan.requirement
	plan.effective.Authority.ApprovalID = plan.approval
	if wait := checkHarnessFor(ctx, deps, plan, role); wait != nil {
		return wait, nil
	}
	if wait, failed := readAttemptInputs(ctx, deps, plan, repository, issue); failed != nil || wait != nil {
		return wait, failed
	}
	if plan.inputs.Revision != control.NativeRev {
		return waitFor(WaitInputsChanged, "readiness observation changed before dispatch"), nil
	}
	return nil, nil
}

func confirmAttemptPreparation(ctx context.Context, deps DispatchDeps, plan *attemptPlan, role string, recordedPrompt bool, retryID string) *planWait {
	if retryID != "" {
		if deps.PrepareRetry == nil {
			return waitFor(WaitPreparation, "fresh retry preparation is unavailable")
		}
		ready, err := deps.PrepareRetry(ctx, plan, retryID)
		if err != nil {
			return waitFor(WaitPreparation, "fresh retry preparation is not confirmed")
		}
		if !ready {
			return waitFor(WaitPreparation, "fresh retry preparation is running")
		}
	} else if role == project.RoleCoder && !recordedPrompt && plan.prep.Preparation.SourceCommit != plan.inputs.Tip {
		if deps.PrepareSource == nil {
			return waitFor(WaitPreparation, "current source preparation is unavailable")
		}
		ready, err := deps.PrepareSource(ctx, plan)
		if err != nil || !ready {
			return waitFor(WaitPreparation, "current source preparation is not confirmed")
		}
	}
	return nil
}

func renderAttemptPrompt(ctx context.Context, deps DispatchDeps, plan *attemptPlan, policy factory.RepositoryPolicy, role string, repository, issue int64, head string, recordedPrompt bool) (*planWait, *planWait) {
	selection := policy.Roles[role]
	if recordedPrompt {
		plan.prompt = nil
		plan.promptSHA = ""
		return nil, nil
	}
	repositoryContext, contextErr := readRepositoryContext(ctx, deps.Host, plan, plan.prep.Preparation, role,
		plan.prep.Preparation.SourceCommit, plan.prep.Preparation.SourceCommit, plan.inputs.Tip, true)
	if contextErr != nil {
		return nil, waitFor(WaitSource, "native prepared repository context is unavailable or changed")
	}
	var implementationPlan *factory.PlanningRecord
	// FW12 joint: the generic planning record stages only on the
	// legacy pre-phase path. Phased prompts carry the admitted
	// dispatcher phase instead; investigation renders its own
	// inspection directive and implementation stages the recorded
	// investigation plan below.
	if plan.phase == "" && plan.advancePlan == nil && deps.PlanRecord != nil {
		record, recordErr := deps.PlanRecord(ctx, repository, issue)
		if recordErr != nil {
			return nil, waitFor("plan_not_current", "implementation approach is unavailable")
		}
		if record.Snapshot != "" {
			if record.Provisional || record.Outcome != "planned" {
				return nil, waitFor("plan_not_current", "implementation approach is provisional or waiting")
			}
			implementationPlan = &record
		}
	}
	// FW12 joint: the admitted dispatcher phase travels into the
	// prompt so the template owns phased rendering. Investigation
	// prompts additionally carry the canonical prerequisite
	// bindings the reported plan must echo for later admission.
	var planPrerequisites []string
	if plan.phase == project.ExecutionInvestigate {
		bindings, bindingsErr := livePlanBindings(plan.acceptance, plan.control, plan.inputs.Tip, plan.prep.Preparation.ID)
		if bindingsErr != nil {
			return nil, waitFor(WaitInputsChanged, "investigation prerequisite bindings are incomplete")
		}
		planPrerequisites = bindings.Prerequisites
	}
	prompt, err := factory.BuildDispatchPrompt(factory.PromptInputs{
		Project: plan.projectID, Repository: repository, Issue: issue, NativeRev: plan.inputs.Revision,
		Phase: plan.phase, PlanPrerequisites: planPrerequisites,
		AcceptanceID: head, TargetBranch: policy.TargetBranch, SourceCommit: plan.inputs.Tip,
		ApprovedBase:   plan.prep.Preparation.SourceCommit,
		Preparation:    plan.prep.Preparation.ID,
		RequirementsID: plan.requirement, ApprovalID: plan.approval,
		Harness: plan.pin.Harness, Model: selection.Model, Role: role,
		ProviderConnection:    plan.sponsorship.Connection,
		RequiredChecks:        policy.Checks,
		ApplianceConcurrent:   plan.applianceConcurrent,
		RepositoryConcurrent:  plan.repositoryConcurrent,
		SponsorshipConcurrent: plan.sponsorship.MaxConcurrent,
		AttemptLimits:         plan.attemptLimits,
		Title:                 plan.inputs.Issue.Title, Body: plan.inputs.Issue.Body,
		Sources:       promptSections(plan.acceptance.Sources, plan.inputs.Comments),
		Resolutions:   promptSections(plan.acceptance.Resolutions, plan.inputs.Comments),
		Prerequisites: plan.acceptance.Prerequisites, Control: plan.control,
		RepositoryContext:  repositoryContext,
		ImplementationPlan: implementationPlan,
	})
	if err != nil {
		if errors.Is(err, factory.ErrPromptPrerequisiteEvidence) {
			return nil, waitFor(WaitInputsChanged, "readiness does not bind accepted prerequisite outcomes")
		}
		return nil, waitFor(WaitPrompt, "accepted inputs exceed the prompt bound")
	}
	// Phased attempts report through the strict v2 phase contract,
	// not the legacy harness shape: replace the trailing legacy
	// report section and, for an advanced implementation, stage the
	// recorded investigation plan the run must execute. The shared
	// prompt template keeps the legacy text for pre-phase runs and
	// FW12 absorbs this rendering when it takes prompt ownership.
	if plan.phase != "" {
		prompt, err = renderPhasedDispatchPrompt(prompt, plan.phase, plan.advancePlan)
		if err != nil {
			return nil, waitFor(WaitPrompt, "phased dispatch prompt exceeds its bound")
		}
	}
	plan.prompt = prompt
	plan.promptSHA = project.FactoryPromptDigest(prompt)
	return nil, nil
}

func planAttemptForRole(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, repository, issue int64, head string, control factory.IssueControl, freshAttempt bool, role, preparation string, recordedPrompt bool, retryID string, lineage attemptLineage) (*attemptPlan, *planWait, *planWait) {
	if control.Validate() != nil || control.Readiness != factory.ReadinessQueued || control.Repository != repository || control.Issue != issue || control.Acceptance != head {
		return nil, waitFor(WaitInputsChanged, "queued readiness changed during dispatch"), nil
	}
	plan := &attemptPlan{control: control, freshAttempt: freshAttempt, phase: lineage.phase, parent: lineage.parent, advancePlan: lineage.advancePlan}
	if wait, failed := admitAttemptLineage(plan, lineage, role, recordedPrompt, issue, head); failed != nil || wait != nil {
		return nil, wait, failed
	}
	policy, wait, failed := loadAttemptPlanBasis(ctx, deps, plan, repository, role, head)
	if failed != nil || wait != nil {
		return nil, wait, failed
	}
	if wait, failed := prepareAttemptInputs(ctx, deps, occupancy, plan, role, preparation, repository, issue, control); failed != nil || wait != nil {
		return nil, wait, failed
	}
	if wait := confirmAttemptPreparation(ctx, deps, plan, role, recordedPrompt, retryID); wait != nil {
		return nil, wait, nil
	}
	if wait, failed := renderAttemptPrompt(ctx, deps, plan, policy, role, repository, issue, head, recordedPrompt); failed != nil || wait != nil {
		return nil, wait, failed
	}
	open, gateRev, _, err := deps.Store.DispatchState(ctx, repository)
	if err != nil {
		return nil, nil, waitFor(DispatchErrStore, "dispatch gate unreadable")
	}
	if !open {
		return nil, waitFor(WaitDispatchClosed, "dispatch gate closed during dispatch"), nil
	}
	plan.gateRev = gateRev
	return plan, nil, nil
}

// advanceChildID derives the deterministic implementation-child identity
// for one finished investigation owner and verifies the recorded result
// still justifies advancement: a reported completed phase result carrying
// an executable plan, on the current acceptance, with the owner's lineage
// intact. Pre-phase owners, non-completed planning and anything without an
// executable plan refuse with no child: the issue stays recorded instead
// of advancing.
func advanceChildOwner(owner factory.Assignment, head string) bool {
	return owner.Subject.Kind == factory.SubjectIssue && owner.Subject.Issue != nil &&
		owner.Subject.Issue.Acceptance == head && owner.Phase == project.ExecutionInvestigate &&
		owner.ParentAssignment == "" && owner.Outcome == factory.Succeeded &&
		owner.Reason == factory.AssignReasonReported && owner.Result != nil && owner.Result.PhaseResult != nil
}

func advanceChildPlan(owner factory.Assignment) *factory.InvestigationPlan {
	phased := owner.Result.PhaseResult
	if phased.Phase != project.ExecutionInvestigate || phased.Status != "completed" ||
		phased.Output == nil || phased.Output.Plan == nil {
		return nil
	}
	plan := phased.Output.Plan
	if plan.Validate() != nil || plan.Executable() != nil {
		return nil
	}
	if plan.AttemptRoot != owner.AttemptRoot || plan.SourceCommit != owner.SourceCommit ||
		plan.Preparation != owner.Preparation {
		return nil
	}
	return plan
}

func advanceChildID(owner factory.Assignment, head string) (string, *factory.InvestigationPlan, bool) {
	if !advanceChildOwner(owner, head) {
		return "", nil, false
	}
	plan := advanceChildPlan(owner)
	if plan == nil {
		return "", nil, false
	}
	digest := factory.ResultDigest(*owner.Result)
	if digest == "" {
		return "", nil, false
	}
	id, err := factory.AdvancementID(owner.ID, digest, project.ExecutionImplement)
	if err != nil {
		return "", nil, false
	}
	return id, plan, true
}

// maybeAdvanceCompletedPlanning admits the implementation child of one
// finished investigation owner through a separate recoverable packet
// transaction instead of requiring Retry. The child reuses its root's
// attempt history, preparation and remaining allowances: no allowance
// resets, no publication, no priority competition. It reports true when
// the report carries the outcome and the caller must stop.
func maybeAdvanceCompletedPlanning(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, control factory.IssueControl, latest factory.Assignment, head string, report *DispatchReport) bool {
	repository, issue := control.Repository, control.Issue
	childID, plan, ok := advanceChildID(latest, head)
	if !ok {
		return false
	}
	existing, err := deps.Store.Assignment(ctx, childID)
	if err == nil {
		// Replay-safe: the same completed parent replays to its recorded
		// child instead of launching twice. A finished child leaves the
		// issue recorded; later phases own its continuation.
		if existing.Stage != factory.AssignmentFinished {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAssigned, Detail: "implementation child is admitted; waiting its result"})
			return true
		}
		return false
	}
	if !errors.Is(err, store.ErrNotFound) {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "advancement identity unreadable"})
		return true
	}
	if deps.Refresh != nil {
		refreshed, err := deps.Refresh(ctx, control)
		if err != nil {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsUnavailable, Detail: "readiness refresh is unavailable"})
			return true
		}
		control = refreshed
	}
	lineage := attemptLineage{phase: project.ExecutionImplement, parent: latest.ID, advancePlan: plan}
	attempt, wait, failed := planAttemptForRole(ctx, deps, occupancy, repository, issue, head, control, false, project.RoleCoder, latest.Preparation, false, "", lineage)
	if failed != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: failed.Reason, Detail: failed.Detail})
		return true
	}
	if wait != nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: wait.Reason, Detail: wait.Detail})
		return true
	}
	if attempt.inputs.Tip != latest.SourceCommit {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsChanged, Detail: "source moved after investigation; implementation waits for fresh planning"})
		return true
	}
	// FW12 joint: persist the reported investigation plan and admit the
	// current head against live bindings before coding. Stale,
	// provisional or rebound plans wait for fresh investigation instead
	// of launching; a forged plan identity fails the pass.
	live, bindingsErr := livePlanBindings(attempt.acceptance, attempt.control, attempt.inputs.Tip, attempt.prep.Preparation.ID)
	if bindingsErr != nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsChanged, Detail: "investigation prerequisite bindings changed; implementation waits for fresh planning"})
		return true
	}
	admitted, admitErr := admitInvestigationPlan(ctx, deps.Store, latest, live)
	if admitErr != nil {
		if errors.Is(admitErr, store.ErrStalePlanBindings) || errors.Is(admitErr, store.ErrStaleRevision) {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsChanged, Detail: "investigation plan is stale; implementation waits for fresh planning"})
			return true
		}
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "investigation plan unadmitted"})
		return true
	}
	attempt.advancePlan = admitted
	admitImplementChild(ctx, deps, occupancy, attempt, childID, latest, report)
	return true
}

// phasedReportMarker ends the shared prompt template: the legacy report
// section is always last, so phased rendering replaces from this marker.
const phasedReportMarker = "\n## Report\n\n"

// promptFenceGuard keeps recorded plan text from forging the harness
// report block, mirroring the template's own collision guard.
var promptFenceGuard = strings.NewReplacer("```"+factory.ResultFence, "``` "+factory.ResultFence, "```"+factory.ReviewReportFence, "``` "+factory.ReviewReportFence)

// renderPhasedDispatchPrompt replaces the legacy trailing report section
// with the strict v2 phase contract and, for an advanced implementation,
// stages the recorded investigation plan the run must execute. Prompts
// stay within the admitted phase bound; anything larger waits instead of
// truncating requirements.
func renderPhasedDispatchPrompt(prompt []byte, phase string, plan *factory.InvestigationPlan) ([]byte, error) {
	if phase != project.ExecutionInvestigate && phase != project.ExecutionImplement {
		return nil, errors.New("phased prompt needs an admitted dispatcher phase")
	}
	if plan != nil && phase != project.ExecutionImplement {
		return nil, errors.New("recorded plan stages only on its implementation")
	}
	idx := bytes.LastIndex(prompt, []byte(phasedReportMarker))
	if idx < 0 {
		return nil, errors.New("dispatch prompt lacks its report section")
	}
	out := append([]byte(nil), prompt[:idx]...)
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	if plan != nil {
		rendered, err := renderRecordedPlan(plan)
		if err != nil {
			return nil, err
		}
		out = append(out, rendered...)
	}
	out = append(out, phasedReportContract(phase)...)
	if len(out) > project.MaxPromptForPhase(phase) {
		return nil, errors.New("phased dispatch prompt exceeds its bound")
	}
	if !utf8.Valid(out) {
		return nil, errors.New("phased dispatch prompt is not valid text")
	}
	return out, nil
}

// renderRecordedPlan stages one validated executable investigation plan
// for its implementation run: identity and digest bindings, the approach,
// and the ordered task/check graph. Agent-visible text carries the fence
// guard so recorded content can never forge a report block.
func renderRecordedPlan(plan *factory.InvestigationPlan) ([]byte, error) {
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	if err := plan.Executable(); err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("\n## Implementation approach\n\n")
	b.WriteString("Recorded plan " + plan.ID + " generation " + strconv.Itoa(plan.Generation) + " digest " + plan.Digest + ". Execute exactly this plan.\n")
	b.WriteString(promptFenceGuard.Replace(plan.Approach) + "\n")
	for _, task := range plan.Tasks {
		b.WriteString("- " + task.ID + " [" + task.Lane + "]: " + promptFenceGuard.Replace(task.Objective) + "\n")
		b.WriteString("  accept: " + strings.Join(task.AcceptanceRefs, ", ") + "; needs: " + strings.Join(task.DependsOn, ", ") + "; checks: " + strings.Join(task.CheckIDs, ", ") + "\n")
		if len(task.Outputs) != 0 {
			b.WriteString("  outputs: " + promptFenceGuard.Replace(strings.Join(task.Outputs, "; ")) + "\n")
		}
		if paths := append(append([]string(nil), task.OwnedPaths...), task.InterfaceWrite...); len(paths) != 0 {
			b.WriteString("  writes: " + strings.Join(paths, ", ") + "\n")
		}
	}
	for _, check := range plan.Checks {
		b.WriteString("  check " + check.ID + " [" + check.Scope + "]: " + strings.Join(check.Argv, " ") + "\n")
	}
	return []byte(b.String()), nil
}

// phasedReportContract renders the strict v2 result contract for one
// admitted dispatcher phase. Completed planning carries its plan and never
// a publishable candidate; completed implementation names its exact
// candidate commit. Planning finishes without a commit or PR and can never
// publish.
func phasedReportContract(phase string) string {
	var b strings.Builder
	b.WriteString(phasedReportMarker)
	b.WriteString("Close your work with exactly one fenced block:\n\n")
	b.WriteString("```" + factory.ResultFence + "\n")
	if phase == project.ExecutionInvestigate {
		b.WriteString(`{"version":2,"phase":"investigate","status":"completed|blocked|failed|cancelled","summary":"...","output":{"plan":{...}}}` + "\n")
		b.WriteString("```\n\n")
		b.WriteString("Report \"completed\" only with the executable investigation plan in \"output.plan\". Planning carries no candidate: never name a commit, open a PR or publish. \"blocked\" keeps the facts that stop the work; agent failure without findings is \"failed\". ")
	} else if phase == project.ExecutionDecompose {
		b.WriteString(`{"version":2,"phase":"decompose","status":"completed|blocked|failed|cancelled","summary":"...","output":{"decomposition":{"intent":"<decision-id>","revision":"<decision-id>","operation":"<this run's operation>","candidates":[{"key":"<candidate key>","rationale":"...","tasks":[...]}]}}}` + "\n")
		b.WriteString("```\n\n")
		b.WriteString("Report \"completed\" only with one to three candidate task graphs in \"output.decomposition\" answering this run's exact intent, revision and operation; every candidate covers the authorized outcomes the admitted input records. Planning carries no candidate: never name a commit, open a PR or publish. \"blocked\" carries attention with no candidates when the input cannot plan; agent failure without findings is \"failed\". ")
	} else {
		b.WriteString(`{"version":2,"phase":"implement","status":"completed|blocked|failed|cancelled","summary":"...","output":{"implementation":{"candidate":"<40-hex commit or empty>"}}}` + "\n")
		b.WriteString("```\n\n")
		b.WriteString("Report \"completed\" only with the exact candidate commit the executed plan produced and no diagnostic references. Non-completed runs name no candidate. ")
	}
	b.WriteString("Any other shape, unknown field or trailing JSON is refused. Never print credentials, tokens or secret files; the report carries IDs and digests only.\n")
	return b.String()
}
