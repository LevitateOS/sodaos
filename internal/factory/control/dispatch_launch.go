package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// executeFreshAttempt records one fresh assignment under the open gate and
// launches its first run. Registration, packet and the admission-gate
// limit recheck land in one transaction before any host call, so a
// concurrent pass either wins the issue or consumes the room first; the
// loser waits on a refreshed snapshot instead of admitting stale work.
func executeFreshAttempt(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, plan *attemptPlan, repository, issue int64, report *DispatchReport, retry *factory.RetryDecision) {
	now := time.Now()
	assignmentID, runID := factory.NewID(), factory.NewID()
	if plan.preparationOwner != "" {
		assignmentID = plan.preparationOwner
	}
	if retry != nil {
		assignmentID = retry.CommandID
	}
	attemptRoot := plan.attemptRoot
	if attemptRoot == "" {
		attemptRoot = assignmentID
	}
	authority := plan.effective.Authority
	authority.RequirementsID, authority.ApprovalID = plan.requirement, plan.approval
	assignment := factory.Assignment{
		Authority: authority,
		ID:        assignmentID, AttemptRoot: attemptRoot, PublicationAssignment: assignmentID,
		Subject: plan.subject, Phase: plan.phase, ParentAssignment: plan.parent,
		ProjectID: plan.projectID, Role: project.RoleCoder,
		Repository: repository, Issue: issue, NativeRev: plan.inputs.Revision,
		Acceptance: plan.acceptance.ID, Preparation: plan.prep.Preparation.ID,
		Harness: plan.pin.Harness + "-" + plan.pin.Version, HarnessVers: plan.pin.Version,
		Model: plan.policy.Roles[project.RoleCoder].Model, Connection: plan.sponsorship.Connection,
		SourceCommit: plan.inputs.Tip, Prompt: plan.prompt, PromptSHA: plan.promptSHA,
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: now.Unix(), ActorID: plan.sponsorship.ActorID,
	}
	if err := assignment.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "dispatch assignment invalid"})
		return
	}
	registration := factory.DispatchRegistration{
		ID: assignmentID, Repository: repository, Revision: plan.gateRev, Authority: authority,
	}
	reservation := factory.Reservation{
		AssignmentID: assignmentID, Repository: repository,
		Connection: plan.sponsorship.Connection, State: factory.ReservationHeld, PlannedMinutes: plan.planned,
	}
	run := factory.Run{
		ID: runID, ProjectID: plan.projectID, Role: project.RoleCoder, InputSHA: plan.inputs.Tip,
		Started: now, Deadline: plan.deadline,
		Image: plan.pin.Image, Harness: assignment.Harness, Model: assignment.Model,
	}
	view := factory.RunView{RunID: runID, Repository: repository, Issue: issue, Attempt: assignmentID}
	var packetErr error
	if retry == nil {
		packetErr = deps.Store.RecordDispatchPacket(ctx, registration, plan.control, assignment, reservation, run, view)
	} else {
		packetErr = deps.Store.RecordExplicitRetryPacket(ctx, *retry, registration, plan.control, assignment, reservation, run, view)
	}
	if err := packetErr; err != nil {
		if plan.preparationOwner != "" && deps.RetireRetry != nil && retryPreparationInvalidated(err) {
			// A lost commit reply may still have admitted the packet. Retire
			// only after the Store confirms that no assignment owns it.
			if _, readErr := deps.Store.Assignment(ctx, assignmentID); errors.Is(readErr, store.ErrNotFound) {
				if retireErr := deps.RetireRetry(ctx, plan); retireErr != nil {
					report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrHost, Detail: "rejected retry preparation retirement is unconfirmed"})
				}
			}
		}
		reportPacketRefusal(ctx, deps, occupancy, repository, issue, err, report)
		return
	}
	occupancy.hold(reservation)
	launch := project.FactoryLaunch{
		Run: project.FactoryRun{
			ID: runID, Project: plan.projectID, Role: project.RoleCoder,
			Preparation: plan.prep.Preparation.ID,
			Harness:     plan.pin.Harness, HarnessVers: plan.pin.Version,
			Model:      assignment.Model,
			Assignment: plan.promptSHA, SourceCommit: plan.inputs.Tip,
			Connection: plan.sponsorship.Connection, Actor: assignment.ActorID,
			Deadline: plan.deadline, ExecutionPhase: plan.phase,
		},
		Prompt: plan.prompt, HarnessSHA256: plan.pin.SHA256,
	}
	if err := launch.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrHost, Detail: "dispatch launch invalid"})
		return
	}
	launchTail(ctx, deps, assignment, run, launch, report)
}

// implementationManifest stages the recorded investigation plan as the
// immutable execution manifest of its implementation run: the plan's
// canonical bytes, digest-bound to the launch. The executor verifies the
// supplied bytes; FW13 owns the child-control envelope around them.
func implementationManifest(plan *factory.InvestigationPlan) ([]byte, error) {
	if plan == nil {
		return nil, errors.New("implementation lacks its recorded plan")
	}
	if err := plan.Validate(); err != nil {
		return nil, err
	}
	if err := plan.Executable(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	if len(data) > project.MaxExecutionManifest {
		return nil, errors.New("recorded plan exceeds its manifest bound")
	}
	return data, nil
}

// stageExecutionManifest binds the recorded investigation plan as the full
// immutable execution envelope of one implement run (D05/FW14): exact plan
// bytes and digest, invocation, phase/run/deadline, the admitted profile
// snapshot and the child limit. An omitted profile resolves to the product
// defaults at staging, matching policy creation; running snapshots stay
// explicit. The returned bytes are parsed back before use so a construction
// drift refuses here instead of at the native boundary.
//
// Invocation reuses the fresh run identity. Dispatch observes no native
// invocation: the executor learns systemd's at launch and enforces it for
// task events (record_task_event binds the live binding, never this
// claim). The run identity is the only dispatch-known unique execution
// identity, so it doubles as the invocation claim; executor event
// authorization stays authoritative. FW14 joint for Codex D05 review:
// either accept this doubling with the live binding authoritative, or
// revise D05 so the executor adopts the staged invocation (deep change).
func stageExecutionManifest(plan *factory.InvestigationPlan, runID string, deadline time.Time, profile factory.ExecutionProfile, maxChildren int) ([]byte, error) {
	planBytes, err := implementationManifest(plan)
	if err != nil {
		return nil, err
	}
	effective := factory.EffectiveExecutionProfile(&profile)
	manifest := project.ExecutionManifest{
		Plan: planBytes, PlanDigest: project.PlanDigest(planBytes),
		Invocation: runID, Phase: project.ExecutionImplement, Run: runID,
		Deadline: deadline,
		Effort:   effective.Effort, Contributor: effective.Contributor,
		MaxChildren: maxChildren,
	}
	data, err := project.MarshalExecutionManifest(manifest)
	if err != nil {
		return nil, err
	}
	if _, err := project.ParseExecutionManifest(data); err != nil {
		return nil, err
	}
	return data, nil
}

// verifyStagedManifest checks the staged envelope binds the launch it
// rides: digest, phase, run, deadline and contributor rule against the
// recorded run. A construction drift refuses before any host call.
func verifyStagedManifest(launch project.FactoryLaunch) error {
	manifest, err := project.ParseExecutionManifest(launch.ExecutionManifest)
	if err != nil {
		return err
	}
	return manifest.ValidateManifestForRun(launch.Run)
}

// admitImplementChild records one advanced implementation child under the
// open gate and launches its first run. Registration, packet and the
// admission-gate limit recheck land in one transaction before any host
// call, exactly like a fresh attempt. The child keeps its parent's attempt
// root, publication slot, preparation and source: the allowance is retained
// rather than reset, and planning stages no publication capability.
func admitImplementChild(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, plan *attemptPlan, childID string, parent factory.Assignment, report *DispatchReport) {
	repository, issue := parent.Repository, parent.Issue
	now := time.Now()
	runID := factory.NewID()
	selection := plan.policy.Roles[project.RoleCoder]
	manifest, err := stageExecutionManifest(plan.advancePlan, runID, plan.deadline, selection.Profile, plan.attemptLimits.MaxChildren)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "recorded plan is unstaged"})
		return
	}
	manifestDigest := project.ExecutionManifestDigest(manifest)
	authority := plan.effective.Authority
	authority.RequirementsID, authority.ApprovalID = plan.requirement, plan.approval
	assignment := factory.Assignment{
		Authority: authority,
		ID:        childID, AttemptRoot: parent.AttemptRoot, ParentAssignment: parent.ID,
		PublicationAssignment: parent.PublicationAssignment,
		Subject:               plan.subject, Phase: project.ExecutionImplement,
		ProjectID: plan.projectID, Role: project.RoleCoder,
		Repository: repository, Issue: issue, NativeRev: plan.inputs.Revision,
		Acceptance: parent.Acceptance, Preparation: plan.prep.Preparation.ID,
		Harness: plan.pin.Harness + "-" + plan.pin.Version, HarnessVers: plan.pin.Version,
		Model: plan.policy.Roles[project.RoleCoder].Model, Connection: plan.sponsorship.Connection,
		SourceCommit: parent.SourceCommit, Prompt: plan.prompt, PromptSHA: plan.promptSHA,
		Run: runID, RunHistory: []string{runID}, Stage: factory.AssignmentAssigned,
		Attempts: 1, CreatedUnix: now.Unix(), ActorID: plan.sponsorship.ActorID,
	}
	if err := assignment.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "dispatch assignment invalid"})
		return
	}
	if assignment.Subject.Kind != parent.Subject.Kind ||
		assignment.Subject.SubjectKey(repository) != parent.Subject.SubjectKey(repository) ||
		assignment.Subject.Issue == nil || parent.Subject.Issue == nil ||
		assignment.Subject.Issue.Acceptance != parent.Subject.Issue.Acceptance ||
		assignment.Acceptance != parent.Subject.Issue.Acceptance {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "dispatch assignment does not match its subject"})
		return
	}
	registration := factory.DispatchRegistration{
		ID: childID, Repository: repository, Revision: plan.gateRev, Authority: authority,
	}
	reservation := factory.Reservation{
		AssignmentID: childID, Repository: repository,
		Connection: plan.sponsorship.Connection, State: factory.ReservationHeld, PlannedMinutes: plan.planned,
	}
	run := factory.Run{
		ID: runID, ProjectID: plan.projectID, Role: project.RoleCoder, InputSHA: parent.SourceCommit,
		Started: now, Deadline: plan.deadline,
		Image: plan.pin.Image, Harness: assignment.Harness, Model: assignment.Model,
	}
	view := factory.RunView{RunID: runID, Repository: repository, Issue: issue, Attempt: childID}
	if packetErr := deps.Store.RecordDispatchPacket(ctx, registration, plan.control, assignment, reservation, run, view); packetErr != nil {
		reportPacketRefusal(ctx, deps, occupancy, repository, issue, packetErr, report)
		return
	}
	occupancy.hold(reservation)
	launch := project.FactoryLaunch{
		Run: project.FactoryRun{
			ID: runID, Project: plan.projectID, Role: project.RoleCoder,
			Preparation: plan.prep.Preparation.ID,
			Harness:     plan.pin.Harness, HarnessVers: plan.pin.Version,
			Model:      assignment.Model,
			Assignment: plan.promptSHA, SourceCommit: parent.SourceCommit,
			Connection: plan.sponsorship.Connection, Actor: assignment.ActorID,
			Deadline: plan.deadline, ExecutionPhase: project.ExecutionImplement,
			ManifestDigest: manifestDigest,
		},
		Prompt: plan.prompt, HarnessSHA256: plan.pin.SHA256,
		ExecutionManifest: manifest,
	}
	if err := launch.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrHost, Detail: "dispatch launch invalid"})
		return
	}
	if err := verifyStagedManifest(launch); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrHost, Detail: "dispatch launch invalid"})
		return
	}
	launchTail(ctx, deps, assignment, run, launch, report)
}

// reportPacketRefusal maps one atomic packet refusal to its wait. A refusal
// proves a concurrent admission consumed the room first; the issue waits
// for a later pass instead of launching.
func reportPacketRefusal(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, repository, issue int64, packetErr error, report *DispatchReport) {
	if errors.Is(packetErr, store.ErrDispatchControlStale) {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsChanged, Detail: "queued readiness changed during dispatch"})
		return
	}
	if errors.Is(packetErr, store.ErrDispatchClosed) {
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitDispatchClosed, Detail: "dispatch gate closed during dispatch"})
		return
	}
	if errors.Is(packetErr, store.ErrCompletionFenced) {
		reportCompletionFence(repository, issue, report)
		return
	}
	if errors.Is(packetErr, store.ErrAssignmentActive) {
		refreshOccupancy(ctx, deps, occupancy, report)
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAssigned, Detail: "a concurrent dispatch won this issue"})
		return
	}
	if wait := admissionWait(packetErr); wait != nil {
		refreshOccupancy(ctx, deps, occupancy, report)
		report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: wait.Reason, Detail: wait.Detail})
		return
	}
	report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "dispatch packet unrecorded"})
}

// retryAttempt relaunches one released assignment after re-checking
// limits, role preparation, harness and inputs. Root prompts are rebuilt
// and compared; child prompts remain their recorded immutable bytes.
// Limit and preparation waits defer quietly: the fresh visit already
// reports the issue as assigned.
func retryAttempt(ctx context.Context, deps DispatchDeps, a factory.Assignment, report *DispatchReport) {
	occupancy, err := snapshotOccupancy(ctx, deps.Store)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "capacity accounting unavailable"})
		return
	}
	control, err := deps.Store.IssueControl(ctx, a.Repository, a.Issue)
	if err != nil {
		report.Waits = append(report.Waits, DispatchWait{Repository: a.Repository, Issue: a.Issue, Reason: WaitInputsChanged, Detail: "queued readiness is unavailable"})
		return
	}
	// An advanced implementation child rebinds against its finished
	// investigation parent: there is no publication yet, so the published
	// parent checks below do not apply.
	if a.Phase == project.ExecutionImplement && a.ParentAssignment != "" && a.PublicationAssignment != a.ID {
		retryPhasedChild(ctx, deps, &occupancy, control, a, report)
		return
	}
	child := a.PublicationAssignment != a.ID
	publicationTarget := ""
	if child {
		publication, publicationErr := deps.Store.PublicationByAssignment(ctx, a.PublicationAssignment)
		if publicationErr != nil {
			if errors.Is(publicationErr, store.ErrNotFound) {
				finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "published parent no longer holds", report)
				return
			}
			report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "published parent unreadable"})
			return
		}
		if publication.Stage != factory.PublicationPublished || publication.WithdrawRequested || publication.Candidate != a.SourceCommit ||
			publication.Acceptance != a.Acceptance || publication.Repository != a.Repository || publication.Issue != a.Issue ||
			publication.AssignmentID != a.PublicationAssignment {
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "published parent no longer holds", report)
			return
		}
		publicationTarget = publication.TargetBranch
		if a.Role != project.RoleCoder && a.Role != project.RoleReviewer {
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "child role no longer holds", report)
			return
		}
	}
	role, exactPreparation := project.RoleCoder, ""
	if child {
		role, exactPreparation = a.Role, a.Preparation
	}
	lineage := attemptLineage{}
	if a.Phase != "" {
		lineage = attemptLineage{phase: a.Phase, parent: a.ParentAssignment}
	}
	plan, wait, failed := planAttemptForRole(ctx, deps, &occupancy, a.Repository, a.Issue, a.Acceptance, control, false, role, exactPreparation, child, "", lineage)
	if failed != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: failed.Reason, Detail: failed.Detail})
		return
	}
	if wait != nil {
		switch wait.Reason {
		case WaitAuthority, WaitDispatchClosed, WaitSponsorshipRole:
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonWithdrawn, "grants no longer authorize dispatch", report)
		case WaitInputsChanged, WaitInputsHidden, WaitIssueClosed, WaitIssueLocked:
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "accepted inputs no longer hold", report)
		}
		return
	}
	if child {
		selection := plan.policy.Roles[a.Role]
		if plan.attemptRoot != a.AttemptRoot ||
			plan.sponsorship.Connection != a.Connection || selection.Model != a.Model ||
			a.Harness != plan.pin.Harness+"-"+plan.pin.Version || a.HarnessVers != plan.pin.Version ||
			plan.prep.Preparation.ID != a.Preparation || plan.policy.TargetBranch != publicationTarget {
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "recorded child profile or authority changed", report)
			return
		}
		previousRun, runErr := deps.Store.FactoryRun(ctx, a.Run)
		if runErr != nil {
			if errors.Is(runErr, store.ErrNotFound) {
				finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "prior child run profile is unavailable", report)
				return
			}
			report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "prior child run unreadable"})
			return
		}
		if previousRun.Admission == nil {
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "prior child run profile is unavailable", report)
			return
		}
		environment, envErr := deps.Store.EnvironmentGrant(ctx, a.Repository)
		if envErr != nil {
			if errors.Is(envErr, store.ErrNotFound) {
				finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonWithdrawn, "environment authorization no longer holds", report)
				return
			}
			report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "environment profile unreadable"})
			return
		}
		previous := previousRun.Admission
		oldSelection, hadOldSelection := previous.Policy.Roles[a.Role]
		if !environment.Active || environment.Profile == nil {
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonWithdrawn, "environment authorization no longer holds", report)
			return
		}
		if !hadOldSelection ||
			previous.Policy.TargetBranch != plan.policy.TargetBranch ||
			!slices.Equal(previous.Policy.Checks, plan.policy.Checks) ||
			oldSelection != selection || previous.Profile != *environment.Profile {
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "recorded child launch profile changed", report)
			return
		}
	} else if !bytes.Equal(plan.prompt, a.Prompt) {
		finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "accepted inputs no longer hold", report)
		return
	}
	runID := factory.NewID()
	now := time.Now()
	run := factory.Run{
		ID: runID, ProjectID: a.ProjectID, Role: a.Role, InputSHA: a.SourceCommit,
		Started: now, Deadline: plan.deadline,
		Image: plan.pin.Image, Harness: a.Harness, Model: a.Model,
	}
	view := factory.RunView{RunID: runID, Repository: a.Repository, Issue: a.Issue, Attempt: a.ID}
	next, err := deps.Store.RecordRetryPacket(ctx, a, plan.control, plan.effective.Authority, run, view, plan.planned)
	if err != nil {
		if errors.Is(err, store.ErrDispatchControlStale) {
			report.Waits = append(report.Waits, DispatchWait{Repository: a.Repository, Issue: a.Issue, Reason: WaitInputsChanged, Detail: "queued readiness changed during retry"})
			return
		}
		// A limit refusal defers quietly: the fresh visit already
		// reports the issue as assigned, and a later pass retries.
		if admissionWait(err) != nil {
			return
		}
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "dispatch attempt unrecorded"})
		return
	}
	launch := project.FactoryLaunch{
		Run: project.FactoryRun{
			ID: runID, Project: a.ProjectID, Role: a.Role,
			Preparation: a.Preparation,
			Harness:     plan.pin.Harness, HarnessVers: a.HarnessVers,
			Model:      a.Model,
			Assignment: a.PromptSHA, SourceCommit: a.SourceCommit,
			Connection: a.Connection, Actor: a.ActorID,
			Deadline: plan.deadline,
		},
		Prompt: a.Prompt, HarnessSHA256: plan.pin.SHA256,
	}
	if err := launch.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrHost, Detail: "dispatch launch invalid"})
		return
	}
	launchTail(ctx, deps, next, run, launch, report)
}

// retryPhasedChild relaunches one interrupted advanced implementation
// child: the recorded prompt bytes are immutable, so planning only rechecks
// live limits, preparation, harness and inputs before recording a fresh run
// for the same assignment. Lineage rebinds against the finished
// investigation parent — same subject, root, publication slot, preparation
// and source — and the staged manifest is remarshaled from the recorded
// plan, so its digest still binds the launch.
func retryPhasedChild(ctx context.Context, deps DispatchDeps, occupancy *passOccupancy, control factory.IssueControl, a factory.Assignment, report *DispatchReport) {
	parent, err := deps.Store.Assignment(ctx, a.ParentAssignment)
	if err != nil || !validImplementLineage(a, parent) {
		finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "implementation parent no longer holds", report)
		return
	}
	plan, wait, failed := planAttemptForRole(ctx, deps, occupancy, a.Repository, a.Issue, a.Acceptance, control, false, a.Role, a.Preparation, true, "", attemptLineage{phase: a.Phase, parent: a.ParentAssignment})
	if failed != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: failed.Reason, Detail: failed.Detail})
		return
	}
	if wait != nil {
		switch wait.Reason {
		case WaitAuthority, WaitDispatchClosed, WaitSponsorshipRole:
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonWithdrawn, "grants no longer authorize dispatch", report)
		case WaitInputsChanged, WaitInputsHidden, WaitIssueClosed, WaitIssueLocked:
			finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "accepted inputs no longer hold", report)
		}
		return
	}
	runID := factory.NewID()
	now := time.Now()
	// The recorded prompt bytes are immutable, so the staged envelope is
	// remarshaled from the recorded plan with this retry's fresh run
	// identity, deadline and current limits: its digest still binds the
	// launch below.
	selection := plan.policy.Roles[a.Role]
	manifest, err := stageExecutionManifest(parent.Result.PhaseResult.Output.Plan, runID, plan.deadline, selection.Profile, plan.attemptLimits.MaxChildren)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "recorded plan is unstaged"})
		return
	}
	run := factory.Run{
		ID: runID, ProjectID: a.ProjectID, Role: a.Role, InputSHA: a.SourceCommit,
		Started: now, Deadline: plan.deadline,
		Image: plan.pin.Image, Harness: a.Harness, Model: a.Model,
	}
	view := factory.RunView{RunID: runID, Repository: a.Repository, Issue: a.Issue, Attempt: a.ID}
	next, err := deps.Store.RecordRetryPacket(ctx, a, plan.control, plan.effective.Authority, run, view, plan.planned)
	if err != nil {
		if errors.Is(err, store.ErrDispatchControlStale) {
			report.Waits = append(report.Waits, DispatchWait{Repository: a.Repository, Issue: a.Issue, Reason: WaitInputsChanged, Detail: "queued readiness changed during retry"})
			return
		}
		// A limit refusal defers quietly: the fresh visit already
		// reports the issue as assigned, and a later pass retries.
		if admissionWait(err) != nil {
			return
		}
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "dispatch attempt unrecorded"})
		return
	}
	launch := project.FactoryLaunch{
		Run: project.FactoryRun{
			ID: runID, Project: a.ProjectID, Role: a.Role,
			Preparation: a.Preparation,
			Harness:     plan.pin.Harness, HarnessVers: a.HarnessVers,
			Model:      a.Model,
			Assignment: a.PromptSHA, SourceCommit: a.SourceCommit,
			Connection: a.Connection, Actor: a.ActorID,
			Deadline: plan.deadline, ExecutionPhase: project.ExecutionImplement,
			ManifestDigest: project.ExecutionManifestDigest(manifest),
		},
		Prompt: a.Prompt, HarnessSHA256: plan.pin.SHA256,
		ExecutionManifest: manifest,
	}
	if err := launch.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrHost, Detail: "dispatch launch invalid"})
		return
	}
	if err := verifyStagedManifest(launch); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrHost, Detail: "dispatch launch invalid"})
		return
	}
	launchTail(ctx, deps, next, run, launch, report)
}

// validImplementLineage verifies one implementation child still binds its
// recorded investigation parent: a finished reported investigation with an
// executable plan, sharing subject, root, publication slot, preparation and
// source. Anything else supersedes the child instead of executing it.
func validImplementLineage(a, parent factory.Assignment) bool {
	if parent.Validate() != nil || parent.ID != a.ParentAssignment ||
		parent.Phase != project.ExecutionInvestigate || parent.Stage != factory.AssignmentFinished ||
		parent.Outcome != factory.Succeeded || parent.Reason != factory.AssignReasonReported ||
		parent.Result == nil || parent.Result.PhaseResult == nil {
		return false
	}
	phased := parent.Result.PhaseResult
	if phased.Phase != project.ExecutionInvestigate || phased.Status != "completed" ||
		phased.Output == nil || phased.Output.Plan == nil ||
		phased.Output.Plan.Validate() != nil || phased.Output.Plan.Executable() != nil {
		return false
	}
	return a.Role == project.RoleCoder && a.Repository == parent.Repository && a.Issue == parent.Issue &&
		a.Acceptance == parent.Acceptance && a.AttemptRoot == parent.AttemptRoot &&
		a.PublicationAssignment == parent.PublicationAssignment && a.Preparation == parent.Preparation &&
		a.SourceCommit == parent.SourceCommit &&
		a.Subject.Kind == parent.Subject.Kind &&
		a.Subject.SubjectKey(a.Repository) == parent.Subject.SubjectKey(parent.Repository)
}

const launchObservationTimeout = 5 * time.Second

// launchTail briefly observes one recorded attempt's launch, then records
// exactly one of launched, confirmed-unused
// settle-and-release, or fenced-and-held. A returned error never guesses
// the run's effects; the host and broker decide what is unused. The native
// run retains its admitted deadline after this observation ends.
func launchTail(ctx context.Context, deps DispatchDeps, a factory.Assignment, run factory.Run, launch project.FactoryLaunch, report *DispatchReport) {
	launchCtx, cancel := context.WithTimeout(ctx, launchObservationTimeout)
	defer cancel()
	state, err := deps.Host.FactoryLaunch(launchCtx, launch)
	if err == nil {
		report.Launched = append(report.Launched, DispatchLaunch{
			Repository: a.Repository, Issue: a.Issue,
			AssignmentID: a.ID, RunID: run.ID, Phase: state.Phase,
		})
		return
	}
	unused, fenced := confirmUnused(ctx, deps, a, run)
	switch {
	case fenced:
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrHost, Detail: "launch outcome unconfirmed"})
		ensureHeldLaunch(ctx, deps, a, report)
	case !unused:
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrFence, Detail: "launch left effects for reconcile"})
		ensureHeldLaunch(ctx, deps, a, report)
	default:
		settleUnusedLaunch(ctx, deps, a, run, err, report)
	}
}

// ensureHeldLaunch re-holds the reservation behind a fenced launch so
// the uncertain attempt keeps counting against its limits.
func ensureHeldLaunch(ctx context.Context, deps DispatchDeps, a factory.Assignment, report *DispatchReport) {
	reservation, err := deps.Store.Reservation(ctx, a.ID)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment reservation unreadable"})
		return
	}
	ensureHeld(ctx, deps, a, reservation, report)
}

// settleUnusedLaunch settles a launch the host and broker never saw and
// releases its confirmed-unused capacity. Attempts remain retryable
// until exhausted; an exhausted assignment finishes failed.
func settleUnusedLaunch(ctx context.Context, deps DispatchDeps, a factory.Assignment, run factory.Run, launchErr error, report *DispatchReport) {
	settled := run
	settled.Outcome, settled.Summary, settled.Reconciled = factory.Failed, boundSummary("dispatch launch refused: "+launchErr.Error(), "run failed"), true
	if err := deps.Store.SaveFactoryRun(ctx, settled); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "unlaunched run unsettled"})
		return
	}
	if err := deps.Store.ReleaseReservation(ctx, a.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "reservation release failed"})
		return
	}
	if a.Attempts >= factory.MaxDispatchAttempts {
		finishTerminal(ctx, deps, a, factory.Failed, factory.AssignReasonExhausted, "dispatch launch refused", report)
		report.Waits = append(report.Waits, DispatchWait{Repository: a.Repository, Issue: a.Issue, Reason: WaitLaunchExhausted, Detail: "dispatch launch refused"})
		return
	}
	report.Waits = append(report.Waits, DispatchWait{Repository: a.Repository, Issue: a.Issue, Reason: WaitLaunchRefused, Detail: "dispatch launch refused; retries remain"})
}
