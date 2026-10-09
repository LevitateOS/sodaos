package control

import (
	"bytes"
	"context"
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
		if errors.Is(err, store.ErrDispatchControlStale) {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitInputsChanged, Detail: "queued readiness changed during dispatch"})
			return
		}
		if errors.Is(err, store.ErrDispatchClosed) {
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitDispatchClosed, Detail: "dispatch gate closed during dispatch"})
			return
		}
		if errors.Is(err, store.ErrAssignmentActive) {
			refreshOccupancy(ctx, deps, occupancy, report)
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: WaitAssigned, Detail: "a concurrent dispatch won this issue"})
			return
		}
		if wait := admissionWait(err); wait != nil {
			refreshOccupancy(ctx, deps, occupancy, report)
			report.Waits = append(report.Waits, DispatchWait{Repository: repository, Issue: issue, Reason: wait.Reason, Detail: wait.Detail})
			return
		}
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrStore, Detail: "dispatch packet unrecorded"})
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
			Deadline: plan.deadline,
		},
		Prompt: plan.prompt, HarnessSHA256: plan.pin.SHA256,
	}
	if err := launch.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: repository, Issue: issue, Reason: DispatchErrHost, Detail: "dispatch launch invalid"})
		return
	}
	launchTail(ctx, deps, assignment, run, launch, report)
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
	plan, wait, failed := planAttemptForRole(ctx, deps, &occupancy, a.Repository, a.Issue, a.Acceptance, control, false, role, exactPreparation, child)
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

// launchTail drives one recorded attempt run: a detached launch bounded
// by the run deadline, then exactly one of launched, confirmed-unused
// settle-and-release, or fenced-and-held. A returned error never guesses
// the run's effects; the host and broker decide what is unused.
func launchTail(ctx context.Context, deps DispatchDeps, a factory.Assignment, run factory.Run, launch project.FactoryLaunch, report *DispatchReport) {
	launchCtx, cancel := context.WithDeadline(context.WithoutCancel(ctx), run.Deadline)
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
