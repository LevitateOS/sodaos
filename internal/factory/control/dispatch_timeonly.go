package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// DispatchTimeOnlyRoot admits and launches one Issue-zero time-only
// fresh root — an intent decompose root or a completion reassess
// root — through the shared serving route. It is the owning boundary
// for both subjects: the issue-enumerating pass (dispatchOne,
// planAttemptForRole) admits issue subjects only, and intent planning
// prompts belong to its R12 consumer, so the prompt arrives
// caller-staged exactly like review and correction children keep
// theirs. Checks run in the same fixed order as the issue path —
// authority, project, policy, sponsorship, capacity, preparation,
// harness, dispatch gate — and the packet transaction stays
// authoritative for subject admission and funding: persistence and
// funding commit before the host call, which always runs outside the
// transaction. Refusals report through the shared packet-refusal
// mapping, so replay and concurrent admission surface as waits, never
// as a second run or charge.
//
// No factory_issue_controls row exists for issue zero, so the packet's
// NativeRev self-check travels on a zero control value carrying only
// the repository, the zero issue and the root's native revision.
// DispatchIntentPlan connects one reviewed repository-bound intent head
// to its decompose run through the shared admission and launch: it loads
// the owner's current head, admits the requested planning operation for
// the exact input, stages the bounded prompt and launches through
// DispatchTimeOnlyRoot, whose packet authoritatively owns funding,
// cancellation and replay. Unassociated drafts launch nothing, and a
// wrong owner, phase, grant or staged input refuses before any effect.
// Provider calls and native activation stay outside: the enrolled host
// account answers the staged prompt, and settlement persists the typed
// result through the existing phased path.
func (c *Coordinator) DispatchIntentPlan(ctx context.Context, intentID string, owner int64, operation string, in factory.SemanticPlannerInput, source string) DispatchReport {
	var report DispatchReport
	head, err := c.Store.IntentRevision(ctx, intentID, owner)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "intent head unreadable"})
		return report
	}
	subject := factory.ExecutionSubject{
		Kind:   factory.SubjectIntent,
		Intent: &factory.IntentSubject{IntentID: intentID, RevisionID: head.ID, ScopeDigest: head.Digest(), OwnerID: owner},
	}
	if err := subject.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "intent subject invalid"})
		return report
	}
	// The packet authoritatively funds or refuses; the pre-check only
	// binds the operation to the exact head and input.
	if err := AdmitIntentBoundLaunch(subject, project.ExecutionDecompose, project.RoleCoder, operation, head, true, in); err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: err.Error()})
		return report
	}
	prompt, err := StageIntentPlanningPrompt(head, in)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: err.Error()})
		return report
	}
	occupancy, err := snapshotOccupancy(ctx, c.Store)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "occupancy unreadable"})
		return report
	}
	DispatchTimeOnlyRoot(ctx, c.dispatchDeps(), &occupancy, head.Repository, subject, source, prompt, &report)
	return report
}

// Native revisions name the sealed native bracket for reassess roots
// and the lowest valid value for intent roots, which have no native
// observation: the sealed record and the head revision are their
// authoritative input bindings.
func DispatchTimeOnlyRoot(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, repository int64, subject factory.ExecutionSubject, source string, prompt []byte, report *DispatchReport) {
	// Truncated to the second like every packet caller: the funding
	// check compares the deadline against a Unix-second horizon, so a
	// sub-second skew on exact-horizon funding would refuse.
	now := time.Now().UTC().Truncate(time.Second)
	phase := ""
	switch subject.Kind {
	case factory.SubjectIntent:
		phase = project.ExecutionDecompose
		if subject.Intent == nil {
			report.Errors = append(report.Errors, DispatchError{Repository: repository, Reason: DispatchErrStore, Detail: "intent root lacks its intent subject"})
			return
		}
	case factory.SubjectCompletion:
		phase = project.ExecutionReassess
		if subject.Completion == nil {
			report.Errors = append(report.Errors, DispatchError{Repository: repository, Reason: DispatchErrStore, Detail: "reassessment root lacks its completion subject"})
			return
		}
	default:
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Reason: DispatchErrStore, Detail: "time-only dispatch needs its intent or completion subject"})
		return
	}
	if !factory.ValidPhaseRole(phase, project.RoleCoder) || !factory.SubjectKindAllowed(subject.Kind, phase) {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Reason: DispatchErrStore, Detail: "time-only root does not match its execution phase"})
		return
	}
	if len(prompt) == 0 || len(prompt) > project.MaxFactoryPrompt {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Reason: WaitPrompt, Detail: "time-only prompt is empty or over its bound"})
		return
	}
	if !factory.ValidCommit(source) {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Reason: DispatchErrStore, Detail: "time-only root lacks its source commit"})
		return
	}
	plan := &attemptPlan{subject: subject, phase: phase, freshAttempt: true}
	effective, err := deps.Authority(ctx, repository)
	if err != nil || !effective.Effective {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Reason: WaitAuthority, Detail: "authority ineffective for time-only dispatch"})
		return
	}
	if !effective.DispatchOpen {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Reason: WaitDispatchClosed, Detail: "dispatch closed for time-only dispatch"})
		return
	}
	plan.effective = effective
	proj, err := deps.Store.ProjectByRepository(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Reason: WaitAuthority, Detail: "project changed during time-only dispatch"})
		} else {
			report.Errors = append(report.Errors, DispatchError{Repository: repository, Reason: DispatchErrStore, Detail: "project unreadable"})
		}
		return
	}
	projectID := proj.ID
	plan.projectID = projectID
	policy, err := deps.Store.RepositoryPolicy(ctx, repository)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Reason: WaitAuthority, Detail: "policy changed during time-only dispatch"})
		} else {
			report.Errors = append(report.Errors, DispatchError{Repository: repository, Reason: DispatchErrStore, Detail: "policy unreadable"})
		}
		return
	}
	plan.policy = policy
	selection, ok := policy.Roles[project.RoleCoder]
	if !ok {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Reason: WaitSponsorshipRole, Detail: "policy has no coder role selection"})
		return
	}
	sponsorship, wait, failure := selectSponsorship(ctx, deps, repository, effective, project.RoleCoder)
	if failure != nil {
		report.Errors = append(report.Errors, timeOnlyFailure(repository, failure))
		return
	}
	if wait != nil {
		report.Waits = append(report.Waits, timeOnlyWait(repository, wait))
		return
	}
	if sponsorship.ProjectID != projectID {
		report.Waits = append(report.Waits, timeOnlyWait(repository, waitFor(WaitAuthority, "sponsorship changed during time-only dispatch")))
		return
	}
	plan.sponsorship = sponsorship
	if first, second := checkLimits(ctx, deps, occupancy, plan); second != nil {
		report.Errors = append(report.Errors, timeOnlyFailure(repository, second))
		return
	} else if first != nil {
		report.Waits = append(report.Waits, timeOnlyWait(repository, first))
		return
	}
	if first, second := selectPreparationFor(ctx, deps, plan, project.RoleCoder, ""); second != nil {
		report.Errors = append(report.Errors, timeOnlyFailure(repository, second))
		return
	} else if first != nil {
		report.Waits = append(report.Waits, timeOnlyWait(repository, first))
		return
	}
	if wait := checkHarnessFor(ctx, deps, plan, project.RoleCoder); wait != nil {
		report.Waits = append(report.Waits, timeOnlyWait(repository, wait))
		return
	}
	open, gateRev, _, err := deps.Store.DispatchState(ctx, repository)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Reason: DispatchErrStore, Detail: "dispatch state unreadable"})
		return
	}
	if !open {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Reason: WaitDispatchClosed, Detail: "dispatch closed for time-only dispatch"})
		return
	}
	plan.gateRev = gateRev
	nativeRev, minutes, wait, failure := timeOnlyScope(ctx, deps, repository, subject)
	if failure != nil {
		report.Errors = append(report.Errors, timeOnlyFailure(repository, failure))
		return
	}
	if wait != nil {
		report.Waits = append(report.Waits, timeOnlyWait(repository, wait))
		return
	}
	deadline := plan.deadline
	if horizon := now.Add(time.Duration(minutes) * time.Minute); horizon.Before(deadline) {
		deadline = horizon
	}
	promptSum := sha256.Sum256(prompt)
	assignmentID, runID := factory.NewID(), factory.NewID()
	assignment := factory.Assignment{
		Authority: effective.Authority,
		ID:        assignmentID, AttemptRoot: assignmentID, PublicationAssignment: "", ParentAssignment: "",
		Subject: subject, Phase: phase,
		ProjectID: projectID, ActorID: sponsorship.ActorID, Role: project.RoleCoder,
		Repository: repository, Issue: 0, NativeRev: nativeRev, Acceptance: "", Preparation: plan.prep.Preparation.ID,
		Harness: plan.pin.Harness + "-" + plan.pin.Version, HarnessVers: plan.pin.Version, Model: selection.Model,
		Connection: sponsorship.Connection, SourceCommit: source,
		Prompt: prompt, PromptSHA: hex.EncodeToString(promptSum[:]),
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: now.Unix(),
	}
	assignment.Authority.RequirementsID, assignment.Authority.ApprovalID = plan.requirement, plan.approval
	if err := assignment.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Reason: DispatchErrStore, Detail: "time-only root invalid: " + err.Error()})
		return
	}
	registration := factory.DispatchRegistration{ID: assignment.ID, Repository: repository, Authority: assignment.Authority}
	reservation := factory.Reservation{AssignmentID: assignment.ID, Repository: repository, Connection: sponsorship.Connection, State: factory.ReservationHeld, PlannedMinutes: plan.planned}
	run := factory.Run{
		ID: runID, ProjectID: projectID, Role: project.RoleCoder, InputSHA: source,
		Started: now, Deadline: deadline,
		Image: "sha256:" + plan.pin.SHA256, Harness: selection.Harness + "-" + selection.HarnessVers, Model: selection.Model,
	}
	view := factory.RunView{RunID: runID, Repository: repository, Issue: 0, Attempt: assignment.ID}
	plan.control = factory.IssueControl{Repository: repository, Issue: 0, NativeRev: nativeRev}
	plan.prompt, plan.promptSHA = prompt, assignment.PromptSHA
	plan.preparationOwner = plan.prep.Preparation.ID
	if err := deps.Store.RecordDispatchPacket(ctx, registration, plan.control, assignment, reservation, run, view); err != nil {
		reportPacketRefusal(ctx, deps, occupancy, repository, 0, err, report)
		return
	}
	occupancy.hold(reservation)
	launch, failure := stageTimeOnlyLaunch(plan, selection.Model, assignment, run, deadline)
	if failure != nil {
		report.Errors = append(report.Errors, timeOnlyFailure(repository, failure))
		return
	}
	launchTail(ctx, deps, assignment, run, launch, report)
}

// timeOnlyWait and timeOnlyFailure carry shared planning checks into
// the serving report for issue-zero roots, which report under the zero
// issue they dispatch.
func timeOnlyWait(repository int64, wait *planWait) DispatchWait {
	return DispatchWait{Repository: repository, Issue: 0, Reason: wait.Reason, Detail: wait.Detail}
}

func timeOnlyFailure(repository int64, failure *planWait) DispatchError {
	return DispatchError{Repository: repository, Issue: 0, Reason: failure.Reason, Detail: failure.Detail}
}

// timeOnlyScope resolves the root's native revision and funded minutes
// from recorded state: the sealed evidence bracket and the
// policy-admitted reassessment horizon for completion roots, the lowest
// valid revision and the policy-admitted intent horizon for intent
// roots. An unfunded intent posture waits for explicit funding; it
// never launches on omission.
func timeOnlyScope(ctx context.Context, deps DispatchDeps, repository int64, subject factory.ExecutionSubject) (int64, int, *planWait, *planWait) {
	fail := func(detail string) (int64, int, *planWait, *planWait) {
		return 0, 0, nil, waitFor(DispatchErrStore, detail)
	}
	if subject.Kind == factory.SubjectIntent {
		policy, err := deps.Store.RepositoryPolicy(ctx, repository)
		if err != nil {
			return fail("policy unreadable")
		}
		if policy.IntentMinutes < 1 {
			return 0, 0, waitFor(WaitAllowance, "intent root is unfunded; explicit funding required"), nil
		}
		return 1, policy.IntentMinutes, nil, nil
	}
	record, err := deps.Store.ReassessmentRecord(ctx, subject.Completion.CycleID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fail("reassessment answers an unstaged cycle")
		}
		return fail("reassessment cycle unreadable")
	}
	if err := record.Validate(); err != nil {
		return fail("reassessment cycle invalid: " + err.Error())
	}
	if record.Cycle.State != factory.ReassessmentReassessing || record.Cycle.Decision != factory.ReassessmentOptionReassess {
		return fail("reassessment answers a cycle without its reassess decision")
	}
	if !record.Local.Sealed() || record.InventoryDigest != subject.Completion.InventoryDigest {
		return fail("reassessment answers an unsealed cycle")
	}
	policy, err := deps.Store.RepositoryPolicy(ctx, repository)
	if err != nil {
		return fail("policy unreadable")
	}
	return record.EvidenceRevision, factory.EffectiveReassessmentMinutes(policy.ReassessmentMinutes), nil, nil
}

// stageTimeOnlyLaunch stages the host launch for one recorded root:
// the intent-bound joint for decompose roots, the phase-tagged generic
// launch for reassess roots. Both validate before the host call.
func stageTimeOnlyLaunch(plan *attemptPlan, model string, assignment factory.Assignment, run factory.Run, deadline time.Time) (project.FactoryLaunch, *planWait) {
	if assignment.Subject.Kind == factory.SubjectIntent {
		staged, err := StageIntentBoundLaunch(assignment.Subject, IntentBoundLaunchInput{
			Prompt: promptOf(plan, assignment), Deadline: deadline, RunID: run.ID,
			ProjectID: assignment.ProjectID, PreparationID: assignment.Preparation,
			Harness: plan.pin.Harness, HarnessVers: plan.pin.Version, Model: model,
			PromptSHA: assignment.PromptSHA, SourceCommit: assignment.SourceCommit,
			Connection: assignment.Connection, HarnessSHA256: plan.pin.SHA256, ActorID: assignment.ActorID,
		})
		if err != nil {
			return project.FactoryLaunch{}, waitFor(DispatchErrStore, "intent launch unstaged: "+err.Error())
		}
		return staged, nil
	}
	launch := project.FactoryLaunch{
		Run: project.FactoryRun{
			ID: run.ID, Project: assignment.ProjectID, Role: project.RoleCoder,
			Preparation: assignment.Preparation,
			Harness:     plan.pin.Harness, HarnessVers: plan.pin.Version,
			Model:      model,
			Assignment: assignment.PromptSHA, SourceCommit: assignment.SourceCommit,
			Connection: assignment.Connection, Actor: assignment.ActorID,
			Deadline: deadline, ExecutionPhase: assignment.Phase,
		},
		Prompt: promptOf(plan, assignment), HarnessSHA256: plan.pin.SHA256,
	}
	if err := launch.Validate(); err != nil {
		return project.FactoryLaunch{}, waitFor(DispatchErrStore, "reassessment launch invalid: "+err.Error())
	}
	return launch, nil
}

func promptOf(plan *attemptPlan, assignment factory.Assignment) []byte {
	if len(plan.prompt) != 0 {
		return plan.prompt
	}
	return assignment.Prompt
}
