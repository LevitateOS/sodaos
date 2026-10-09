package control

import (
	"context"
	"errors"

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
	ReadPreparationContext(ctx context.Context, in project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error)
	FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error)
	FactoryHarness(ctx context.Context, family string) (project.FactoryHarnessPin, error)
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
	ctx, ownsPass, cancelPass := c.readinessPass(ctx)
	defer cancelPass()
	report := DispatchPass(ctx, c.dispatchDeps())
	c.publishAfterDispatch(ctx, report)
	if ownsPass {
		drain, err := c.drainReadinessWork(ctx, "")
		if (err != nil && !errors.Is(err, ErrReadinessPassPending)) || drain.failed {
			report.Errors = append(report.Errors, DispatchError{Reason: DispatchErrStore, Detail: "readiness work deferred"})
		}
	}
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
