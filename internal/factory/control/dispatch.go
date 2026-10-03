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

// DispatchHost is the dispatch slice of the host surface: launch starts
// one recorded run under supervision, inspect observes one run without
// mutating it, and harness reports the staged-harness pin. The host
// client implements it; the coordinator passes its own host through.
type DispatchHost interface {
	FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error)
	FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error)
	FactoryHarness(ctx context.Context) (project.FactoryHarnessPin, error)
}

// DispatchBroker is the dispatch slice of the broker surface: get reads
// one execution's nonsecret metadata. Dispatch never acquires,
// registers or handles credential delivery.
type DispatchBroker interface {
	GetExecution(ctx context.Context, kind, executionID string) (identity.Execution, error)
}

// DispatchComment is one verified accepted native text section: the
// comment identity plus the content the bracketed read verified against
// its digest.
type DispatchComment struct {
	ID         string
	Content    string
	Digest     string
	ContentVer int
	Visible    bool
}

// DispatchIssue is the verified accepted objective: the exact title and
// body at one idle native revision plus the lifecycle state dispatch
// must respect.
type DispatchIssue struct {
	Title         string
	Body          string
	TitleDigest   string
	ContentDigest string
	ContentVer    int
	Lifecycle     int
	ClosedUnix    int64
	Visible       bool
	Closed        bool
	Locked        bool
	IsPull        bool
}

// DispatchInputs is one revision-bound accepted-input read: the verified
// objective, every selected comment revision and the exact target tip
// the run must build on.
type DispatchInputs struct {
	Issue    DispatchIssue
	Comments []DispatchComment
	Tip      string
	TipRef   string
	Revision int64
}

// DispatchReads brackets one accepted-input read between equal idle
// native revision observations: the objective, the named comment
// revisions and the target branch tip. Transport failures arrive as
// acceptance refusals with a stale, incomplete, busy or unavailable
// reason; any other error reports an unusable source.
type DispatchReads interface {
	ReadDispatchInputs(ctx context.Context, repository, issue string, commentIDs []string, targetRef string) (DispatchInputs, error)
}

// Dispatch wait reasons. Bounded codes: reports expose why an issue did
// not launch, never the evidence bytes behind it. Terminal reasons
// (attempt_recorded, launch_exhausted) end the attempt for its head;
// every other reason retries on a later pass.
const (
	WaitAssigned          = "already_assigned"
	WaitAttemptRecorded   = "attempt_recorded"
	WaitAuthority         = "authority_ineffective"
	WaitDispatchClosed    = "dispatch_closed"
	WaitCapacity          = "capacity_full"
	WaitRepository        = "repository_full"
	WaitSponsorship       = "sponsorship_full"
	WaitSponsorshipRole   = "sponsorship_role_missing"
	WaitAllowance         = "allowance_exhausted"
	WaitPreparation       = "preparation_not_ready"
	WaitHarness           = "harness_unavailable"
	WaitInputsUnavailable = "inputs_unavailable"
	WaitInputsBusy        = "inputs_busy"
	WaitInputsStale       = "inputs_stale"
	WaitInputsIncomplete  = "inputs_incomplete"
	WaitInputsHidden      = "inputs_hidden"
	WaitInputsChanged     = "inputs_changed"
	WaitIssueClosed       = "issue_closed"
	WaitIssueLocked       = "issue_locked"
	WaitSource            = "source_unavailable"
	WaitPrompt            = "prompt_too_large"
	WaitLaunchRefused     = "launch_refused"
	WaitLaunchExhausted   = "launch_exhausted"
)

// Dispatch error reasons for unexpected infrastructure failures. The
// affected assignment keeps its held reservation; a later pass retries.
const (
	DispatchErrStore = "store_unavailable"
	DispatchErrHost  = "host_unavailable"
	DispatchErrFence = "launch_fenced"
)

// DispatchWait is one issue that did not launch this pass and why.
type DispatchWait struct {
	Repository int64  `json:"repository,string"`
	Issue      int64  `json:"issue,string"`
	Reason     string `json:"reason"`
	Detail     string `json:"detail,omitempty"`
}

// DispatchLaunch is one run this pass started.
type DispatchLaunch struct {
	Repository   int64  `json:"repository,string"`
	Issue        int64  `json:"issue,string"`
	AssignmentID string `json:"assignment_id"`
	RunID        string `json:"run_id"`
	Phase        string `json:"phase"`
}

// DispatchError is one unexpected infrastructure failure.
type DispatchError struct {
	Repository int64  `json:"repository,string"`
	Issue      int64  `json:"issue,string"`
	Reason     string `json:"reason"`
	Detail     string `json:"detail,omitempty"`
}

// DispatchReport is the durable outcome of one dispatch pass: the runs
// it started, the interrupted assignments it resolved, and every issue
// that waited or errored. Triggers record it; nothing in it admits,
// spends or publishes.
type DispatchReport struct {
	Launched  []DispatchLaunch `json:"launched"`
	Recovered []string         `json:"recovered"`
	Waits     []DispatchWait   `json:"waits"`
	Errors    []DispatchError  `json:"errors"`
}

func newDispatchReport() DispatchReport {
	return DispatchReport{Launched: []DispatchLaunch{}, Recovered: []string{}, Waits: []DispatchWait{}, Errors: []DispatchError{}}
}

const (
	// MaxDispatchVisits bounds queued issues visited per pass. Visits run
	// oldest first in deterministic order; the rest wait for a later pass.
	MaxDispatchVisits = 64
	// MaxDispatchRecovery bounds interrupted assignments resolved per
	// pass. Each resolution is one bounded state transition.
	MaxDispatchRecovery = 64
	// MaxDispatchDeadline bounds one dispatched run in minutes. The host
	// refuses deadlines past three hours; the minute under keeps clock
	// skew from tipping a full allowance over the bound.
	MaxDispatchDeadline = 179
)

// DispatchDeps gathers one dispatch pass's dependencies. Authority
// derives the visible enablement verdict; the coordinator passes its own
// EffectiveAuthority so dispatch and assessment share one verdict. Queue
// carries fair-rotation progress across passes; a nil queue visits from
// the beginning without advancing.
type DispatchDeps struct {
	Store     *store.Store
	Host      DispatchHost
	Broker    DispatchBroker
	Reads     DispatchReads
	Authority func(ctx context.Context, repository int64) (factory.EffectiveAuthority, error)
	Queue     *DispatchQueueCursor
}

// dispatchReady reports whether one pass has its required dependencies.
// Reads stay optional: an unwired source waits every issue instead of
// failing the pass.
func dispatchReady(deps DispatchDeps) bool {
	return deps.Store != nil && deps.Host != nil && deps.Broker != nil && deps.Authority != nil
}

// RecoverDispatch resolves interrupted assignments without dispatching
// new work: held capacity behind settled runs is consumed, launches
// that never reached the host release and retry bounded, and fenced
// runs stay held for reconcile. Coordinator startup calls this after
// settling runs; failures report instead of failing the caller.
func RecoverDispatch(ctx context.Context, deps DispatchDeps) DispatchReport {
	report := newDispatchReport()
	if !dispatchReady(deps) {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "dispatch dependencies unavailable"})
		return report
	}
	recoverAssigned(ctx, deps, &report)
	return report
}

// DispatchPass resolves interrupted assignments, then dispatches the
// oldest queued issues within current limits. It never fails: every
// operational failure lands in the report, and a later pass retries.
// Inference stays out: waits release nothing because they hold nothing,
// and reservations release only capacity confirmed unused.
func DispatchPass(ctx context.Context, deps DispatchDeps) DispatchReport {
	report := RecoverDispatch(ctx, deps)
	if !dispatchReady(deps) {
		return report
	}
	firstSeen, repo, issue, hasCursor := deps.Queue.start()
	queued, err := deps.Store.QueuedControlsAfter(ctx, MaxDispatchVisits, firstSeen, repo, issue, hasCursor)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "queued listing unavailable"})
		return report
	}
	if hasCursor && len(queued) == 0 {
		// Rotation reached the end of the listing: wrap to the
		// beginning so this same pass still visits waiting work.
		deps.Queue.reset()
		queued, err = deps.Store.QueuedControlsAfter(ctx, MaxDispatchVisits, 0, 0, 0, false)
		if err != nil {
			report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "queued listing unavailable"})
			return report
		}
	}
	occupancy, err := snapshotOccupancy(ctx, deps.Store)
	if err != nil {
		report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "capacity accounting unavailable"})
		return report
	}
	for _, control := range queued {
		if err := ctx.Err(); err != nil {
			report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "dispatch pass cancelled"})
			return report
		}
		dispatchOne(ctx, deps, &occupancy, control, &report)
	}
	// A short batch observed the end of the listing, so rotation
	// wraps; a full batch continues after the last visited control.
	if len(queued) > 0 {
		deps.Queue.advance(queued[len(queued)-1], len(queued) < MaxDispatchVisits)
	}
	return report
}

// passOccupancy snapshots held reservations and unattributed active
// runs once per pass. Attempts refresh it from the report they build so
// one pass never over-commits the limits it just consumed.
type passOccupancy struct {
	held         []factory.Reservation
	unattributed int
	byProject    map[string]int
}

func snapshotOccupancy(ctx context.Context, db *store.Store) (passOccupancy, error) {
	held, err := db.HeldReservations(ctx, store.MaxHeldReservations)
	if err != nil {
		return passOccupancy{}, err
	}
	total, byProject, err := db.ActiveRunCounts(ctx)
	if err != nil {
		return passOccupancy{}, err
	}
	return passOccupancy{held: held, unattributed: total, byProject: byProject}, nil
}

func (o *passOccupancy) hold(r factory.Reservation) {
	o.held = append(o.held, r)
}

func (o passOccupancy) heldTotal() int { return len(o.held) }

func (o passOccupancy) heldRepo(repository int64) int {
	n := 0
	for _, r := range o.held {
		if r.Repository == repository {
			n++
		}
	}
	return n
}

func (o passOccupancy) heldConn(repository int64, connection string) (slots, planned int) {
	for _, r := range o.held {
		if r.Repository == repository && r.Connection == connection {
			slots++
			planned += r.PlannedMinutes
		}
	}
	return slots, planned
}

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

// finishFromRun finishes one assigned assignment from its settled latest
// run, deriving the recorded result from the run outcome and output.
func finishFromRun(ctx context.Context, db *store.Store, a factory.Assignment, run factory.Run, output string, now time.Time) error {
	current, err := db.Assignment(ctx, a.ID)
	if err != nil {
		return err
	}
	if current.Stage != factory.AssignmentAssigned {
		return nil
	}
	result, outcome, reason := deriveAttemptResult(run.Outcome, output, run.Summary, current.ID, current.Run, now.Unix())
	current.Stage, current.Outcome, current.Reason = factory.AssignmentFinished, outcome, reason
	current.FinishedUnix = now.Unix()
	current.Result = &result
	if err := current.Validate(); err != nil {
		return err
	}
	return db.FinishAssignment(ctx, current)
}

// recordConfirmedUsage appends one settled run's confirmed consumption in
// whole minutes, rounding up. The first write wins; replays reuse it.
func recordConfirmedUsage(ctx context.Context, db *store.Store, a factory.Assignment, run factory.Run, now time.Time) error {
	minutes := 0
	if elapsed := now.Sub(run.Started); elapsed > 0 {
		minutes = int((elapsed + time.Minute - time.Nanosecond) / time.Minute)
	}
	return db.RecordRunUsage(ctx, factory.Usage{
		RunID: run.ID, Repository: a.Repository, Connection: a.Connection,
		Minutes: minutes, RecordedUnix: now.Unix(),
	})
}

// deriveAttemptResult maps one settled run outcome plus its output to the
// recorded assignment result. A completed run with a valid fenced report
// records it verbatim; anything else synthesizes an honest result with an
// empty candidate, leaving validation to publication.
func deriveAttemptResult(outcome factory.Outcome, output, summary, assignmentID, runID string, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string) {
	switch outcome {
	case factory.Succeeded:
		if reported, ok := factory.ParseHarnessResult(output); ok {
			return factory.ResultFromHarness(assignmentID, runID, reported, recordedUnix), factory.Succeeded, factory.AssignReasonReported
		}
		return factory.ResultSynthesized(assignmentID, runID, "blocked",
			"harness completed without a parseable result report", recordedUnix), factory.NeedsHuman, factory.AssignReasonNoReport
	case factory.Cancelled:
		return factory.ResultSynthesized(assignmentID, runID, "cancelled",
			boundSummary(summary, "run cancelled"), recordedUnix), factory.Cancelled, factory.AssignReasonCancelled
	default:
		return factory.ResultSynthesized(assignmentID, runID, "failed",
			boundSummary(summary, "run failed"), recordedUnix), factory.Failed, factory.AssignReasonRunFailed
	}
}

func boundSummary(summary, fallback string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return fallback
	}
	if len(summary) > 4096 {
		summary = summary[:4096-12] + "…[truncated]"
	}
	return summary
}

// AccountSettledRun accounts one run the supervisor just settled: it
// records confirmed usage, consumes the assignment's reservation and
// finishes the assignment from the settled outcome. Runs without an
// assignment, and assignments already finished, replay silently. The
// settle path calls this best-effort; recovery replays anything it
// misses, so a false return never loses accounting.
func AccountSettledRun(ctx context.Context, db *store.Store, run factory.Run, output string, now time.Time) (factory.Assignment, bool) {
	if db == nil || !run.Reconciled {
		return factory.Assignment{}, false
	}
	a, err := db.AssignmentByRun(ctx, run.ID)
	if err != nil || a.Stage != factory.AssignmentAssigned {
		return factory.Assignment{}, false
	}
	if err := recordConfirmedUsage(ctx, db, a, run, now); err != nil {
		return factory.Assignment{}, false
	}
	if err := db.ConsumeReservation(ctx, a.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
		return factory.Assignment{}, false
	}
	result, outcome, reason := deriveAttemptResult(run.Outcome, output, run.Summary, a.ID, a.Run, now.Unix())
	a.Stage, a.Outcome, a.Reason = factory.AssignmentFinished, outcome, reason
	a.FinishedUnix = now.Unix()
	a.Result = &result
	if err := a.Validate(); err != nil {
		return factory.Assignment{}, false
	}
	if err := db.FinishAssignment(ctx, a); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return a, true
		}
		return factory.Assignment{}, false
	}
	return a, true
}

// dispatchDeps binds one pass to this coordinator's dependencies.
func (c *Coordinator) dispatchDeps() DispatchDeps {
	return DispatchDeps{
		Store: c.Store, Host: c.Host, Broker: c.Broker,
		Reads: c.DispatchReads, Authority: c.EffectiveAuthority,
		Queue: &c.queue,
	}
}

// Dispatch resolves interrupted assignments, then dispatches the oldest
// queued issues within current limits. It never fails: every operational
// failure lands in the report, and a later pass retries.
func (c *Coordinator) Dispatch(ctx context.Context) DispatchReport {
	report := DispatchPass(ctx, c.dispatchDeps())
	c.publishAfterDispatch(ctx, report)
	return report
}

// dispatchAfterIntake runs one best-effort dispatch pass after a native
// intake assessment. Dispatch runs only while an accepted-input source
// is wired; an unwired coordinator keeps its assessment-only behavior.
func (c *Coordinator) dispatchAfterIntake(ctx context.Context) {
	if c.DispatchReads == nil {
		return
	}
	_ = c.Dispatch(ctx)
}

// assessDispatchDependants reassesses every recorded dependant of one
// completed issue exactly once. Completion arrival retriggers dependent
// assessment; unchanged dependents replay without recording.
func (c *Coordinator) assessDispatchDependants(ctx context.Context, repository, issue int64) {
	dependants, err := c.Store.AcceptanceDependants(ctx, repository, issue)
	if err != nil {
		return
	}
	visited := map[factory.DependenceRef]bool{{Repository: repository, Issue: issue}: true}
	for _, dependant := range dependants {
		_, _ = c.assessCascade(ctx, dependant.Repository, dependant.Issue, visited)
	}
}
