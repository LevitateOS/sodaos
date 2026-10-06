package control

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// recoverAssigned resolves every interrupted assigned assignment: runs
// that settled without accounting consume their reservation and finish,
// launches that never reached the host release and retry bounded, and
// fenced runs stay held for reconcile to settle. Recovery launches only
// after re-checking authority, limits and inputs; anything uncertain
// defers to a later pass.
func recoverAssigned(ctx context.Context, deps DispatchDeps, report *DispatchReport) {
	assigned, err := deps.Store.AssignedAssignments(ctx, MaxDispatchRecovery)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "assignment recovery unavailable"})
		return
	}
	for _, a := range assigned {
		if err := ctx.Err(); err != nil {
			report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "dispatch recovery cancelled"})
			return
		}
		recoverOne(ctx, deps, a, report)
	}
}

// recoverOne resolves one assigned assignment and records the outcome in
// the report. Resolved means consumed, released-and-retried, or finished;
// deferred means untouched for a later pass.
func recoverOne(ctx context.Context, deps DispatchDeps, a factory.Assignment, report *DispatchReport) {
	reservation, err := deps.Store.Reservation(ctx, a.ID)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment reservation unreadable"})
		return
	}
	runs, err := dispatchHistoryRuns(ctx, deps.Store, a)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment runs unreadable"})
		return
	}
	active := activeHistoryRun(runs)
	if active == nil {
		recoverSettled(ctx, deps, a, reservation, report)
		return
	}
	unused, fenced := confirmUnused(ctx, deps, a, *active)
	switch {
	case fenced:
		ensureHeld(ctx, deps, a, reservation, report)
	case unused:
		recoverUnused(ctx, deps, a, reservation, *active, report)
	default:
		ensureHeld(ctx, deps, a, reservation, report)
	}
}

// dispatchHistoryRuns loads every attempt run of one assignment. A
// missing record reports nil at its position: record-before-call means a
// missing run never reached the host.
func dispatchHistoryRuns(ctx context.Context, db *store.Store, a factory.Assignment) ([]*factory.Run, error) {
	runs := make([]*factory.Run, 0, len(a.RunHistory))
	for _, id := range a.RunHistory {
		run, err := db.FactoryRun(ctx, id)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				runs = append(runs, nil)
				continue
			}
			return nil, err
		}
		runs = append(runs, &run)
	}
	return runs, nil
}

func activeHistoryRun(runs []*factory.Run) *factory.Run {
	for _, run := range runs {
		if run != nil && !run.Reconciled {
			return run
		}
	}
	return nil
}

// confirmUnused determines whether an active attempt run provably never
// consumed capacity. The host side holds no effects when the run was
// never recorded, or when its receipt never left approved (recorded
// intent, nothing started); anything later may have consumed. The broker
// side holds nothing when no execution exists for the run. Only both
// together confirm unused; a transport failure or any recorded effect
// fences instead and the reservation stays held.
func confirmUnused(ctx context.Context, deps DispatchDeps, a factory.Assignment, run factory.Run) (unused, fenced bool) {
	state, err := deps.Host.FactoryInspect(ctx, project.FactoryInspect{Project: a.ProjectID, ID: run.ID})
	if err != nil {
		if !isHostNotFound(err) {
			return false, true
		}
	} else if state.ID != "" && state.Phase != project.FactoryApproved {
		return false, false
	}
	_, err = deps.Broker.GetExecution(ctx, identity.Factory, run.ID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			return true, false
		}
		return false, true
	}
	return false, false
}

// hostRunNotFound is the host client's never-recorded miss text. Control
// cannot import the host package, so the coupling is pinned by a test
// against the host sentinel; the broker's typed unknown check gates every
// release alongside it.
const hostRunNotFound = "factory run not found"

func isHostNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), hostRunNotFound)
}

// ensureHeld re-holds a released reservation behind a still-active run so
// the in-flight attempt keeps counting against its limits.
func ensureHeld(ctx context.Context, deps DispatchDeps, a factory.Assignment, reservation factory.Reservation, report *DispatchReport) {
	if reservation.State == factory.ReservationHeld {
		return
	}
	if reservation.State != factory.ReservationReleased {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "reservation behind a live run is not held"})
		return
	}
	reservation.State = factory.ReservationHeld
	reservation.Revision++
	if err := deps.Store.ReholdReservation(ctx, reservation); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "reservation re-hold failed"})
	}
}

// recoverSettled resolves an assigned assignment with no live attempt.
// A held reservation served its run: usage is recorded, the reservation
// is consumed and the assignment finishes from the run. A released
// reservation served nothing: attempts remain retryable, so recovery
// retries bounded or finishes terminally. A consumed reservation without
// a finish replays the missing finish.
func recoverSettled(ctx context.Context, deps DispatchDeps, a factory.Assignment, reservation factory.Reservation, report *DispatchReport) {
	latest, err := deps.Store.FactoryRun(ctx, a.Run)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			recoverMissingRun(ctx, deps, a, reservation, report)
			return
		}
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment run unreadable"})
		return
	}
	if !latest.Reconciled {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "settled recovery met a live run"})
		return
	}
	switch reservation.State {
	case factory.ReservationHeld:
		output := recoverOutput(ctx, deps, a, latest)
		if err := recordConfirmedUsage(ctx, deps.Store, a, latest, time.Now()); err != nil {
			report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "confirmed usage unrecorded"})
			return
		}
		if err := deps.Store.ConsumeReservation(ctx, a.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
			report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "reservation consume failed"})
			return
		}
		if err := finishFromRun(ctx, deps.Store, a, latest, output, time.Now()); err != nil {
			if !errors.Is(err, store.ErrNotFound) {
				report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment finish failed"})
			}
			return
		}
		report.Recovered = append(report.Recovered, a.ID)
	case factory.ReservationConsumed:
		output := recoverOutput(ctx, deps, a, latest)
		if err := finishFromRun(ctx, deps.Store, a, latest, output, time.Now()); err != nil {
			if !errors.Is(err, store.ErrNotFound) {
				report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment finish failed"})
			}
			return
		}
		report.Recovered = append(report.Recovered, a.ID)
	default:
		if latest.Outcome == factory.Succeeded {
			output := recoverOutput(ctx, deps, a, latest)
			if err := finishFromRun(ctx, deps.Store, a, latest, output, time.Now()); err != nil {
				if !errors.Is(err, store.ErrNotFound) {
					report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment finish failed"})
				}
				return
			}
			report.Recovered = append(report.Recovered, a.ID)
			return
		}
		retryOrFinish(ctx, deps, a, report)
	}
}

// recoverMissingRun resolves an assigned assignment whose latest run was
// never recorded: record-before-call proves the host never ran it, so
// the attempt released its capacity without consuming anything.
func recoverMissingRun(ctx context.Context, deps DispatchDeps, a factory.Assignment, reservation factory.Reservation, report *DispatchReport) {
	if reservation.State == factory.ReservationHeld {
		if err := deps.Store.ReleaseReservation(ctx, a.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
			report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "reservation release failed"})
			return
		}
	}
	retryOrFinish(ctx, deps, a, report)
}

// recoverUnused settles an active run the host and broker never saw,
// releases its confirmed-unused capacity, then retries bounded or
// finishes the assignment.
func recoverUnused(ctx context.Context, deps DispatchDeps, a factory.Assignment, reservation factory.Reservation, run factory.Run, report *DispatchReport) {
	settled := run
	settled.Outcome, settled.Summary, settled.Reconciled = factory.Failed, "dispatch launch never reached the host", true
	if err := deps.Store.SaveFactoryRun(ctx, settled); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "unlaunched run unsettled"})
		return
	}
	if reservation.State == factory.ReservationHeld {
		if err := deps.Store.ReleaseReservation(ctx, a.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
			report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "reservation release failed"})
			return
		}
	}
	retryOrFinish(ctx, deps, a, report)
}

// retryOrFinish retries a released assignment whose attempts remain and
// whose inputs still hold, or finishes it terminally. Limits and
// preparation defer quietly: the next pass retries, and the fresh visit
// already reports the issue as assigned.
func retryOrFinish(ctx context.Context, deps DispatchDeps, a factory.Assignment, report *DispatchReport) {
	if a.Attempts >= factory.MaxDispatchAttempts {
		finishTerminal(ctx, deps, a, factory.Failed, factory.AssignReasonExhausted, "dispatch launch refused", report)
		return
	}
	head, err := deps.Store.AcceptanceHead(ctx, a.Repository, a.Issue)
	if err != nil || head != a.Acceptance {
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "acceptance head unreadable"})
			return
		}
		finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonSuperseded, "acceptance no longer authorizes this attempt", report)
		return
	}
	effective, err := deps.Authority(ctx, a.Repository)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "authority unreadable"})
		return
	}
	if !effective.Effective {
		finishTerminal(ctx, deps, a, factory.Cancelled, factory.AssignReasonWithdrawn, "grants no longer authorize dispatch", report)
		return
	}
	retryAttempt(ctx, deps, a, report)
}

// finishTerminal finishes one assigned assignment with a synthesized
// result. A concurrent finish replays silently.
func finishTerminal(ctx context.Context, deps DispatchDeps, a factory.Assignment, outcome factory.Outcome, reason, summary string, report *DispatchReport) {
	current, err := deps.Store.Assignment(ctx, a.ID)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment unreadable"})
		return
	}
	if current.Stage != factory.AssignmentAssigned {
		return
	}
	now := time.Now()
	status := "failed"
	if outcome == factory.Cancelled {
		status = "cancelled"
	}
	result := factory.ResultSynthesized(current.ID, current.Run, status, summary, now.Unix())
	current.Stage, current.Outcome, current.Reason = factory.AssignmentFinished, outcome, reason
	current.FinishedUnix = now.Unix()
	current.Result = &result
	if err := current.Validate(); err != nil {
		report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "terminal assignment invalid"})
		return
	}
	if err := deps.Store.FinishAssignment(ctx, current); err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			report.Errors = append(report.Errors, DispatchError{Repository: a.Repository, Issue: a.Issue, Reason: DispatchErrStore, Detail: "assignment finish failed"})
		}
		return
	}
	report.Recovered = append(report.Recovered, a.ID)
}

// recoverOutput re-reads a settled run's output for result derivation.
// An unreadable run reports without output: the recorded outcome stays
// authoritative either way.
func recoverOutput(ctx context.Context, deps DispatchDeps, a factory.Assignment, run factory.Run) string {
	state, err := deps.Host.FactoryInspect(ctx, project.FactoryInspect{Project: a.ProjectID, ID: run.ID})
	if err != nil || state.ID != run.ID {
		return ""
	}
	return state.Output
}
