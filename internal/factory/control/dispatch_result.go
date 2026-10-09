package control

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
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
	finished, err := finishAssignmentFromRun(current, run, output, now)
	if err != nil {
		return err
	}
	return db.FinishAssignment(ctx, finished)
}

func finishAssignmentFromRun(a factory.Assignment, run factory.Run, output string, now time.Time) (factory.Assignment, error) {
	if a.Stage != factory.AssignmentAssigned {
		return a, nil
	}
	result, outcome, reason := deriveAttemptResult(a, run, output, run.Summary, now.Unix())
	a.Stage, a.Outcome, a.Reason = factory.AssignmentFinished, outcome, reason
	a.FinishedUnix = now.Unix()
	a.Result = &result
	if err := a.Validate(); err != nil {
		return factory.Assignment{}, err
	}
	return a, nil
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
func deriveAttemptResult(a factory.Assignment, run factory.Run, output, summary string, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string) {
	switch run.Outcome {
	case factory.Succeeded:
		if run.ID != a.Run || run.Role != a.Role || run.InputSHA != a.SourceCommit {
			return factory.ResultSynthesized(a.ID, a.Run, "blocked",
				"settled run does not match its assignment", recordedUnix), factory.NeedsHuman, factory.AssignReasonNoReport
		}
		if a.Role == project.RoleReviewer {
			if report, ok := factory.ParseReviewReport(output); ok {
				return factory.AssignmentResult{
					AssignmentID: a.ID, RunID: a.Run, Status: "completed",
					Summary: report.Summary, Candidate: run.InputSHA,
					Findings: []string{}, Review: &report, Reported: true, RecordedUnix: recordedUnix,
				}, factory.Succeeded, factory.AssignReasonReported
			}
			return factory.ResultSynthesized(a.ID, a.Run, "blocked",
				"review completed without a parseable report", recordedUnix), factory.NeedsHuman, factory.AssignReasonNoReport
		}
		if reported, ok := factory.ParseHarnessResult(output); ok {
			return factory.ResultFromHarness(a.ID, a.Run, reported, recordedUnix), factory.Succeeded, factory.AssignReasonReported
		}
		return factory.ResultSynthesized(a.ID, a.Run, "blocked",
			"harness completed without a parseable result report", recordedUnix), factory.NeedsHuman, factory.AssignReasonNoReport
	case factory.Cancelled:
		return factory.ResultSynthesized(a.ID, a.Run, "cancelled",
			boundSummary(summary, "run cancelled"), recordedUnix), factory.Cancelled, factory.AssignReasonCancelled
	default:
		return factory.ResultSynthesized(a.ID, a.Run, "failed",
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
	finished, err := finishAssignmentFromRun(a, run, output, now)
	if err != nil {
		return factory.Assignment{}, false
	}
	a = finished
	if err := db.FinishAssignment(ctx, a); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return a, true
		}
		return factory.Assignment{}, false
	}
	return a, true
}
