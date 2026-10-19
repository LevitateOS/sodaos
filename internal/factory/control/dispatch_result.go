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
	finished, err = settleRepairChain(ctx, db, finished, now)
	if err != nil {
		return err
	}
	return db.FinishAssignment(ctx, finished)
}

// repairChainSubject reports whether one finished assignment needs its
// delivery-repair result bound to a recorded failed verification: a
// reported successful repair carrying a completed delivery-repair
// report. Anything else passes through untouched.
func repairChainSubject(finished factory.Assignment) (factory.DeliveryRepairSubject, bool) {
	if finished.Subject.Kind != factory.SubjectDeliveryRepair || finished.Subject.DeliveryRepair == nil {
		return factory.DeliveryRepairSubject{}, false
	}
	if finished.Outcome != factory.Succeeded || finished.Result == nil || !finished.Result.Reported {
		return factory.DeliveryRepairSubject{}, false
	}
	report := finished.Result.PhaseResult
	if report == nil || report.Status != "completed" || report.Output == nil || report.Output.DeliveryRepair == nil {
		return factory.DeliveryRepairSubject{}, false
	}
	return *finished.Subject.DeliveryRepair, true
}

// checkRepairChain verifies one loaded failed-verification assignment
// against the repair subject citing it: the loaded assignment must be
// the cited run's delivery verification, and its recorded result must
// report that same delivery as failed. Anything else means the repair
// answers no recorded failure. The failure digest has no producer
// equation yet, so digest-to-report linkage stays open; run,
// assignment, delivery and failed-outcome linkage close here.
func checkRepairChain(failed factory.Assignment, runID string, subject factory.DeliveryRepairSubject) error {
	if failed.Run != runID {
		return errors.New("repair cites a run outside its recorded assignment")
	}
	if failed.Subject.Kind != factory.SubjectDelivery || failed.Subject.Delivery == nil {
		return errors.New("repair cites a run that never verified a delivery")
	}
	if failed.Subject.Delivery.DeliveryID != subject.DeliveryID {
		return errors.New("repair cites a failure recorded for another delivery")
	}
	result := failed.Result
	if result == nil || !result.Reported || result.RunID != runID {
		return errors.New("cited run recorded no reported result")
	}
	phased := result.PhaseResult
	if phased == nil || phased.Phase != project.ExecutionVerify || phased.Status != "completed" ||
		phased.Output == nil || phased.Output.Verification == nil {
		return errors.New("cited run recorded no completed verification")
	}
	if phased.Output.Verification.Outcome != factory.AppVerificationFailed {
		return errors.New("cited verification did not fail")
	}
	if phased.Output.Verification.DeliveryID != subject.DeliveryID {
		return errors.New("cited verification covered another delivery")
	}
	return nil
}

// settleRepairChain binds one derived delivery-repair result to its
// recorded failed verification before persistence. A proven miss (no
// such run, or a run that never recorded that delivery as failed)
// replaces the unproven repair with the blocked refusal the settle
// path persists instead of it. A load failure returns retryable
// without persisting anything: an unprovable chain is not a disproven
// one, and recovery replays it.
// repairChainStore is the narrow failed-verification lookup
// settleRepairChain binds against. *store.Store implements it; the
// MCP settler's stub stores do too.
type repairChainStore interface {
	AssignmentByRun(ctx context.Context, runID string) (factory.Assignment, error)
}

func settleRepairChain(ctx context.Context, db repairChainStore, finished factory.Assignment, now time.Time) (factory.Assignment, error) {
	subject, ok := repairChainSubject(finished)
	if !ok {
		return finished, nil
	}
	failed, err := db.AssignmentByRun(ctx, subject.FailedVerificationRunID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return refuseUnprovenRepair(finished, now), nil
		}
		return factory.Assignment{}, err
	}
	if err := checkRepairChain(failed, subject.FailedVerificationRunID, subject); err != nil {
		return refuseUnprovenRepair(finished, now), nil
	}
	return finished, nil
}

// refuseUnprovenRepair replaces an unbound delivery repair with the
// blocked refusal the settle path persists: the assignment finishes
// fenced for human attention instead of recording an unproven repair.
func refuseUnprovenRepair(finished factory.Assignment, now time.Time) factory.Assignment {
	refused := factory.ResultSynthesized(finished.ID, finished.Run, "blocked", "delivery repair answers no recorded failed verification", now.Unix())
	finished.Result = &refused
	finished.Outcome, finished.Reason = factory.NeedsHuman, factory.AssignReasonNoReport
	return finished
}

func finishAssignmentFromRun(a factory.Assignment, run factory.Run, output string, now time.Time) (factory.Assignment, error) {
	if a.Stage != factory.AssignmentAssigned {
		return a, nil
	}
	result, outcome, reason := deriveAttemptResult(a, run, output, run.Summary, now.Unix())
	a.Stage, a.Outcome, a.Reason = factory.AssignmentFinished, outcome, reason
	a.FinishedUnix = now.Unix()
	a.Result = &result
	// Every reported decompose answer settles against its assignment's
	// recorded staged input on this one joint, which normal settlement
	// and recovery share: neither path keeps a semantic bypass.
	if result.Reported && a.Phase == project.ExecutionDecompose && a.Subject.Intent != nil {
		if settled, valid := settleDecomposeSemantic(a, now); !valid {
			a.Outcome, a.Reason, a.Result = settled.Outcome, settled.Reason, settled.Result
		}
	}
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
// recorded assignment result. Phased assignments bind the strict v2 phase
// result to their recorded phase, subject lineage and run; pre-phase
// assignments keep the legacy harness report contract. A completed run
// with a valid fenced report records it verbatim; anything else
// synthesizes an honest result with an empty candidate, leaving validation
// to publication.
func deriveAttemptResult(a factory.Assignment, run factory.Run, output, summary string, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string) {
	if a.Phase != "" {
		return derivePhasedAttemptResult(a, run, output, summary, recordedUnix)
	}
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

// derivePhasedAttemptResult maps one settled run to the recorded result
// for a phased assignment. The settled run must match its assignment; the
// role must be the phase's admitted account; and a reported result must be
// a strict v2 phase result bound to that same phase. Cross-phase payloads,
// extra JSON and legacy harness shapes refuse to a synthesized blocked
// result: planning outputs never authorize effects, and completed planning
// never names a publishable candidate. Reviewer runs report the same v2
// contract with the typed review verdict; only pre-phase reviewer runs
// keep the legacy review fence.
// refusePhasedAttempt maps one invalid phased run to a synthesized
// blocked result.
func refusePhasedAttempt(a factory.Assignment, recordedUnix int64, detail string) (factory.AssignmentResult, factory.Outcome, string) {
	phased := synthesizePhased(a, "blocked", detail, recordedUnix)
	return phased, factory.NeedsHuman, factory.AssignReasonNoReport
}

func derivePhasedAttemptResult(a factory.Assignment, run factory.Run, output, summary string, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string) {
	if run.ID != a.Run || run.Role != a.Role || run.InputSHA != a.SourceCommit {
		return refusePhasedAttempt(a, recordedUnix, "settled run does not match its assignment")
	}
	if !factory.ValidPhaseRole(a.Phase, a.Role) {
		return refusePhasedAttempt(a, recordedUnix, "assignment role does not match its execution phase")
	}
	// A non-successful settled run never reports: the supervisor maps the
	// observed outcome, and only a successful run may carry a v2 payload.
	if result, outcome, reason, done := mapPhasedRunOutcome(a, run, summary, recordedUnix); done {
		return result, outcome, reason
	}
	if a.Role == project.RoleReviewer {
		return reportPhasedReviewerResult(a, run, output, summary, recordedUnix)
	}
	parsed, ok := factory.ParsePhaseResult(output)
	if !ok {
		return refusePhasedAttempt(a, recordedUnix, "run completed without a parseable phase result")
	}
	if err := parsed.BindAssignment(a); err != nil {
		return refusePhasedAttempt(a, recordedUnix, "phase result does not match its assignment phase")
	}
	return buildPhasedCoderResult(a, parsed, recordedUnix)
}

// mapPhasedRunOutcome maps one non-successful settled run to its
// recorded result, reporting done when the outcome already settles it.
func mapPhasedRunOutcome(a factory.Assignment, run factory.Run, summary string, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string, bool) {
	var empty factory.AssignmentResult
	switch run.Outcome {
	case factory.Succeeded:
		return empty, "", "", false
	case factory.Cancelled:
		cancelled := synthesizePhased(a, "cancelled", boundSummary(summary, "run cancelled"), recordedUnix)
		return cancelled, factory.Cancelled, factory.AssignReasonCancelled, true
	default:
		failed := synthesizePhased(a, "failed", boundSummary(summary, "run failed"), recordedUnix)
		return failed, factory.Failed, factory.AssignReasonRunFailed, true
	}
}

// reportPhasedReviewerResult maps one settled reviewer run to its
// recorded result under the phased contract.
func reportPhasedReviewerResult(a factory.Assignment, run factory.Run, output, summary string, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string) {
	if a.Phase == project.ExecutionVerify {
		return derivePhasedVerifyAttemptResult(a, run, output, summary, recordedUnix)
	}
	if a.Phase != "" {
		return derivePhasedReviewAttemptResult(a, run, output, summary, recordedUnix)
	}
	if report, ok := factory.ParseReviewReport(output); ok {
		return factory.AssignmentResult{
			AssignmentID: a.ID, RunID: a.Run, Status: "completed",
			Summary: report.Summary, Candidate: run.InputSHA,
			Findings: []string{}, Review: &report, Reported: true, RecordedUnix: recordedUnix,
		}, factory.Succeeded, factory.AssignReasonReported
	}
	return refusePhasedAttempt(a, recordedUnix, "review completed without a parseable report")
}

// buildPhasedCoderResult builds one successful coder run's recorded
// result from its bound phase result.
func buildPhasedCoderResult(a factory.Assignment, parsed factory.PhaseResult, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string) {
	result := factory.AssignmentResult{
		AssignmentID: a.ID, RunID: a.Run, Phase: a.Phase, PhaseResult: &parsed,
		Summary: parsed.Summary, Findings: []string{},
		Reported: true, RecordedUnix: recordedUnix,
	}
	switch parsed.Status {
	case "completed":
		result.Status = "completed"
		if a.Phase == project.ExecutionImplement && parsed.Output != nil && parsed.Output.Implementation != nil {
			result.Candidate = parsed.Output.Implementation.Candidate
		}
		// A completed correction nominates its resulting tree: the
		// retained candidate binds the report's nomination so the
		// correction consumer advances exactly the accounted tree. The
		// nomination is not checkpoint proof or publish authorization.
		if a.Phase == project.ExecutionCorrect && parsed.Output != nil && parsed.Output.Correction != nil {
			result.Candidate = parsed.Output.Correction.NewCandidate
		}
	case "blocked":
		result.Status = "blocked"
	case "failed":
		result.Status = "failed"
		return result, factory.Failed, factory.AssignReasonRunFailed
	case "cancelled":
		result.Status = "cancelled"
		return result, factory.Cancelled, factory.AssignReasonCancelled
	default:
		return refusePhasedAttempt(a, recordedUnix, "phase result carries an unknown status")
	}
	if err := result.ValidateForAssignment(a.Role, a.SourceCommit); err != nil {
		return refusePhasedAttempt(a, recordedUnix, "phase result does not match its assignment role")
	}
	if result.PhaseResult != nil {
		if err := result.ValidatePhaseBinding(a.Role, a.Phase); err != nil {
			return refusePhasedAttempt(a, recordedUnix, "phased result does not bind its assignment phase")
		}
	}
	outcome := factory.Outcome(factory.Succeeded)
	reason := factory.AssignReasonReported
	if parsed.Status == "blocked" {
		outcome, reason = factory.NeedsHuman, factory.AssignReasonNoReport
	}
	return result, outcome, reason
}

// derivePhasedReviewAttemptResult maps one settled reviewer run to the
// recorded result for a phased review assignment. The report is the same
// strict v2 phase contract every other phase uses, carrying exactly one
// typed review verdict (findings verdict or explicit clear) for a
// completed run and no verdict otherwise. A completed review records no
// candidate and no legacy string findings: assessing findings never
// authorizes publication. Anything unparseable, cross-phase or legacy
// refuses to a synthesized blocked result; no legacy review reader is
// added on this path.
func derivePhasedReviewAttemptResult(a factory.Assignment, run factory.Run, output, summary string, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string) {
	refuse := func(detail string) (factory.AssignmentResult, factory.Outcome, string) {
		phased := synthesizePhased(a, "blocked", detail, recordedUnix)
		return phased, factory.NeedsHuman, factory.AssignReasonNoReport
	}
	if run.ID != a.Run || run.Role != a.Role || run.InputSHA != a.SourceCommit {
		return refuse("settled run does not match its assignment")
	}
	if a.Phase != project.ExecutionReview || !factory.ValidPhaseRole(a.Phase, a.Role) {
		return refuse("assignment role does not match its execution phase")
	}
	switch run.Outcome {
	case factory.Succeeded:
	case factory.Cancelled:
		cancelled := synthesizePhased(a, "cancelled", boundSummary(summary, "run cancelled"), recordedUnix)
		return cancelled, factory.Cancelled, factory.AssignReasonCancelled
	default:
		failed := synthesizePhased(a, "failed", boundSummary(summary, "run failed"), recordedUnix)
		return failed, factory.Failed, factory.AssignReasonRunFailed
	}
	parsed, ok := factory.ParsePhaseResult(output)
	if !ok {
		return refuse("review completed without a parseable phase result")
	}
	if err := parsed.BindAssignment(a); err != nil {
		return refuse("phase result does not match its assignment phase")
	}
	result := factory.AssignmentResult{
		AssignmentID: a.ID, RunID: a.Run, Phase: a.Phase, PhaseResult: &parsed,
		Summary: parsed.Summary, Findings: []string{},
		Reported: true, RecordedUnix: recordedUnix,
	}
	switch parsed.Status {
	case "completed":
		result.Status = "completed"
	case "blocked":
		result.Status = "blocked"
	case "failed":
		result.Status = "failed"
		return result, factory.Failed, factory.AssignReasonRunFailed
	case "cancelled":
		result.Status = "cancelled"
		return result, factory.Cancelled, factory.AssignReasonCancelled
	default:
		return refuse("phase result carries an unknown status")
	}
	if err := result.ValidatePhaseBinding(a.Role, a.Phase); err != nil {
		return refuse("phased result does not bind its assignment phase")
	}
	outcome := factory.Outcome(factory.Succeeded)
	reason := factory.AssignReasonReported
	if parsed.Status == "blocked" {
		outcome, reason = factory.NeedsHuman, factory.AssignReasonNoReport
	}
	return result, outcome, reason
}

// derivePhasedVerifyAttemptResult maps one settled reviewer run to the
// recorded result for a phased verify assignment. The report is the same
// strict v2 phase contract every other phase uses, carrying exactly the
// app verification report for a completed run and no report otherwise. A
// completed verification records no candidate and no legacy string
// findings: checking a delivery never authorizes publication. Anything
// unparseable, cross-phase or legacy refuses to a synthesized blocked
// result; no legacy verification reader is added on this path.
func derivePhasedVerifyAttemptResult(a factory.Assignment, run factory.Run, output, summary string, recordedUnix int64) (factory.AssignmentResult, factory.Outcome, string) {
	refuse := func(detail string) (factory.AssignmentResult, factory.Outcome, string) {
		phased := synthesizePhased(a, "blocked", detail, recordedUnix)
		return phased, factory.NeedsHuman, factory.AssignReasonNoReport
	}
	if run.ID != a.Run || run.Role != a.Role || run.InputSHA != a.SourceCommit {
		return refuse("settled run does not match its assignment")
	}
	if a.Phase != project.ExecutionVerify || !factory.ValidPhaseRole(a.Phase, a.Role) {
		return refuse("assignment role does not match its execution phase")
	}
	switch run.Outcome {
	case factory.Succeeded:
	case factory.Cancelled:
		cancelled := synthesizePhased(a, "cancelled", boundSummary(summary, "run cancelled"), recordedUnix)
		return cancelled, factory.Cancelled, factory.AssignReasonCancelled
	default:
		failed := synthesizePhased(a, "failed", boundSummary(summary, "run failed"), recordedUnix)
		return failed, factory.Failed, factory.AssignReasonRunFailed
	}
	parsed, ok := factory.ParsePhaseResult(output)
	if !ok {
		return refuse("verification completed without a parseable phase result")
	}
	if err := parsed.BindAssignment(a); err != nil {
		return refuse("phase result does not match its assignment phase")
	}
	result := factory.AssignmentResult{
		AssignmentID: a.ID, RunID: a.Run, Phase: a.Phase, PhaseResult: &parsed,
		Summary: parsed.Summary, Findings: []string{},
		Reported: true, RecordedUnix: recordedUnix,
	}
	switch parsed.Status {
	case "completed":
		result.Status = "completed"
	case "blocked":
		result.Status = "blocked"
	case "failed":
		result.Status = "failed"
		return result, factory.Failed, factory.AssignReasonRunFailed
	case "cancelled":
		result.Status = "cancelled"
		return result, factory.Cancelled, factory.AssignReasonCancelled
	default:
		return refuse("phase result carries an unknown status")
	}
	if err := result.ValidatePhaseBinding(a.Role, a.Phase); err != nil {
		return refuse("phased result does not bind its assignment phase")
	}
	outcome := factory.Outcome(factory.Succeeded)
	reason := factory.AssignReasonReported
	if parsed.Status == "blocked" {
		outcome, reason = factory.NeedsHuman, factory.AssignReasonNoReport
	}
	return result, outcome, reason
}

// settleDecomposeSemantic settles one reported decompose result
// against its assignment's recorded staged input. A missing or corrupt
// recorded input refuses the semantic result: there is no legacy
// compatibility path around the staged contract. The staged input must
// answer the funded subject triple, every inspected source must still
// be the admitted source commit, and the original answer must assess
// through AdmitIntentBoundResult — revision/operation binding plus full
// candidate coverage and graph validation, or attention-only admission.
// Anything failing the semantic contract converts to needs-human with
// no report, mirroring the derive path's refusal shape. Usage and
// consumption already recorded stay untouched.
func settleDecomposeSemantic(a factory.Assignment, now time.Time) (factory.Assignment, bool) {
	assess := func(detail string) (factory.Assignment, bool) {
		converted := a
		converted.Outcome, converted.Reason = factory.NeedsHuman, factory.AssignReasonNoReport
		blocked := synthesizePhased(a, "blocked", detail, now.Unix())
		converted.Result = &blocked
		if err := converted.Validate(); err != nil {
			return factory.Assignment{}, false
		}
		return converted, false
	}
	in, staged, err := ParseStagedPlanningInput(a.Prompt)
	if err != nil || !staged {
		return assess("staged planning input is missing or corrupt")
	}
	subject := a.Subject.Intent
	if subject == nil || in.Head.Digest() != subject.ScopeDigest || in.Head.ID != subject.RevisionID ||
		in.Head.Intent != subject.IntentID || in.Head.Owner != subject.OwnerID {
		return assess("staged input answers another funded operation")
	}
	if a.Result == nil || a.Result.PhaseResult == nil || a.Result.PhaseResult.Output == nil ||
		a.Result.PhaseResult.Output.Decomposition == nil {
		return assess("staged answer carries no decomposition")
	}
	for _, ref := range in.Inspected {
		if ref.Commit != a.SourceCommit {
			return assess("staged answer inspects another source commit")
		}
	}
	if err := AdmitIntentBoundResult(a.Subject, a.Phase, in, *a.Result.PhaseResult.Output.Decomposition); err != nil {
		return assess(err.Error())
	}
	return a, true
}

// synthesizePhased binds one supervisor-derived result to a phased
// assignment, carrying the typed phase result for non-completed terminal
// states. Planning and other completed states never synthesize: only a
// reported v2 payload completes a phase.
func synthesizePhased(a factory.Assignment, status, summary string, recordedUnix int64) factory.AssignmentResult {
	phased := factory.ResultSynthesized(a.ID, a.Run, status, summary, recordedUnix)
	phaseStatus := status
	switch status {
	case "failed", "cancelled", "blocked":
	default:
		phaseStatus = "blocked"
		phased.Status = "blocked"
	}
	phased.Phase, phased.PhaseResult = a.Phase, &factory.PhaseResult{
		Version: factory.PhaseResultVersion, Phase: a.Phase,
		Status: phaseStatus, Summary: summary,
	}
	if err := phased.ValidatePhaseBinding(a.Role, a.Phase); err != nil {
		legacy := factory.ResultSynthesized(a.ID, a.Run, status, summary, recordedUnix)
		return legacy
	}
	return phased
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
	finished, err = settleRepairChain(ctx, db, finished, now)
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
