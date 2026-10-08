package control

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

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
		Minutes: minutes, StartedAt: run.Started.UTC(), EndedAt: now.UTC(),
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
