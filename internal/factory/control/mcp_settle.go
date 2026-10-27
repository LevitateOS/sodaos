package control

// MCP terminal-proposal settlement (Lane A execution join).
//
// FW38 records typed worker finishes as immutable terminal proposals
// with a recorded receipt, but no consumer ever settles them: the
// assignment stays assigned forever, holding its reservation and
// blocking redispatch, while the proposal's outcome never reaches a
// finished row Lane B feed producers can project. This settler closes
// that join: it maps one recorded terminal proposal to the existing
// FinishAssignment store owner and advances the receipt
// recorded→accepted, reusing the repair-chain and readiness joints
// the run-output settlement path already owns.
//
// Only proposals the settler can prove settle: worker-reported
// terminal states (finish_incomplete), Builder/Fixer nominations
// carrying a verifiable commit candidate (finish_implementation,
// finish_correction), and FW40 role finishes whose structured
// carrier strictly decodes into the typed phase result for their
// assignment (mcp_fw40.go). Planning, review, reassessment and app
// verification finishes without that carrier stay recorded and
// parked instead of settling as fabricated success. Parking is
// explicit: the settler reports settled=false and touches nothing.
//
// Ordering is crash-safe: FinishAssignment commits first, then the
// receipt advances recorded→accepted under its expected-state guard.
// A crash between the two recovers on the next pass (finished
// assignment plus recorded receipt advances with an already-finished
// note). ErrNotFound from FinishAssignment is ambiguous — a concurrent
// foreign finish, a missing allowance, or a stale input rollback — so
// the settler re-reads the assignment and settles only a genuinely
// terminal one; still-assigned work returns the refusal with its
// receipt left recorded for a later pass. Other transient store errors
// return without touching the receipt, so a later pass retries.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// MCPTerminalSettleStore is the narrow settlement joint. *store.Store
// implements it; tests stub it.
type MCPTerminalSettleStore interface {
	FactoryRun(ctx context.Context, id string) (factory.Run, error)
	Assignment(ctx context.Context, id string) (factory.Assignment, error)
	AssignmentByRun(ctx context.Context, runID string) (factory.Assignment, error)
	TerminalMCPSubmission(ctx context.Context, assignmentID string) (factory.MCPSubmission, bool, error)
	TerminalMCPProposal(ctx context.Context, assignmentID string) (factory.MCPReceipt, bool, error)
	AdvanceMCPReceipt(ctx context.Context, assignmentID, submissionID, expectedState string, receipt factory.MCPReceipt) error
	ResolveArtifactDigest(ctx context.Context, repository int64, assignment, digest string) (string, error)
	FinishAssignment(ctx context.Context, a factory.Assignment) error
}

// checkSettledArtifactRefs resolves every artifact a finished
// verification report cites against complete artifacts of the
// finishing assignment. The read spends nothing and publishes
// nothing; an unknown, foreign or unfinished digest fences the
// finish for human attention instead of recording a bogus citation.
func checkSettledArtifactRefs(ctx context.Context, st MCPTerminalSettleStore, finished factory.Assignment) error {
	if finished.Result == nil || finished.Result.PhaseResult == nil ||
		finished.Result.PhaseResult.Output == nil || finished.Result.PhaseResult.Output.Verification == nil {
		return nil
	}
	for _, ref := range finished.Result.PhaseResult.Output.Verification.ArtifactRefs {
		if _, err := st.ResolveArtifactDigest(ctx, finished.Repository, finished.ID, ref); err != nil {
			return errors.New("verification artifact is not a complete artifact of this assignment")
		}
	}
	return nil
}

// mcpDriveStore adds the recorded-terminal work list the coordinator
// drive enumerates. *store.Store implements it; tests stub it.
type mcpDriveStore interface {
	MCPTerminalSettleStore
	RecordedTerminalMCPAssignments(ctx context.Context, limit int) ([]string, error)
}

var _ mcpDriveStore = (*store.Store)(nil)

var _ MCPTerminalSettleStore = (*store.Store)(nil)

// mcpSettleNote bounds the receipt reason the settler records.
const mcpSettleNoteChars = 200

// SettleMCPTerminalProposal settles one assignment's recorded terminal
// proposal. It reports settled=true only when the assignment finished
// through this proposal and the receipt advanced to accepted. A
// parked proposal (settled=false, nil error) needs its FW40 role
// consumer; anything else either has no terminal yet or already
// settled.
func SettleMCPTerminalProposal(ctx context.Context, st MCPTerminalSettleStore, assignmentID string, now time.Time) (settled bool, err error) {
	if st == nil || !factory.ValidID(assignmentID) {
		return false, errors.New("mcp settlement lacks its assignment")
	}
	assigned, err := st.Assignment(ctx, assignmentID)
	if err != nil {
		return false, err
	}
	submission, ok, err := st.TerminalMCPSubmission(ctx, assignmentID)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	// A consumed receipt stays settled: only a recorded terminal
	// still owns its assignment finish. The FinishAssignment CAS
	// below remains the exactly-once guard against races.
	receipt, ok, err := st.TerminalMCPProposal(ctx, assignmentID)
	if err != nil {
		return false, err
	}
	if !ok || receipt.State != factory.MCPReceiptRecorded {
		return false, nil
	}
	if assigned.Stage != factory.AssignmentAssigned {
		return advanceSettledTerminal(ctx, st, submission, "assignment already finished; proposal recorded for inspection")
	}
	finished, note, ok, err := mapMCPTerminalFinish(assigned, submission, now)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	ready, refusal, err := checkpointTerminalReady(ctx, st, finished)
	if err != nil {
		return false, err
	}
	if refusal != "" {
		finished = fenceMCPTerminal(assigned, submission, now, refusal)
		note = "native final-candidate evidence refused"
	} else if !ready {
		return false, nil
	}
	// Cited artifacts revalidate before persistence: the fence only
	// shape-checked the refs, so an unknown, foreign or unfinished
	// digest fences here with the proposal preserved for inspection.
	if err := checkSettledArtifactRefs(ctx, st, finished); err != nil {
		finished = fenceMCPTerminal(assigned, submission, now, err.Error())
		note = "unresolvable verification artifact fenced for human attention"
	}
	// The delivery-repair chain binds before persistence, exactly as
	// on the run-output path: a proven miss fences the repair, a load
	// failure retries on a later pass with the receipt untouched.
	finished, err = settleRepairChain(ctx, st, finished, now)
	if err != nil {
		return false, err
	}
	if err := st.FinishAssignment(ctx, finished); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// ErrNotFound is ambiguous: a concurrent foreign finish
			// consumes the assignment, but a missing allowance or a
			// stale input rollback refuses while the assignment stays
			// assigned. Settle only a genuinely terminal re-read;
			// still-assigned work keeps its recorded receipt for a
			// later pass instead of stranding as accepted.
			current, rerr := st.Assignment(ctx, assignmentID)
			if rerr != nil {
				return false, rerr
			}
			if current.Stage != factory.AssignmentFinished {
				return false, fmt.Errorf("mcp terminal finish refused: %w", err)
			}
			return advanceSettledTerminal(ctx, st, submission, "assignment finished during settlement")
		}
		return false, err
	}
	return advanceSettledTerminal(ctx, st, submission, note)
}

// advanceSettledTerminal advances one terminal receipt
// recorded→accepted with its settlement note. A stale guard means a
// concurrent settler already consumed the proposal.
func advanceSettledTerminal(ctx context.Context, st MCPTerminalSettleStore, submission factory.MCPSubmission, note string) (bool, error) {
	if len(note) > mcpSettleNoteChars {
		note = note[:mcpSettleNoteChars]
	}
	receipt := factory.MCPReceipt{
		SubmissionID: submission.SubmissionID, Tool: submission.Tool,
		State: factory.MCPReceiptAccepted, InputDigest: submission.PayloadHash,
		Reason: note,
	}
	if err := receipt.Validate(); err != nil {
		return false, err
	}
	if err := st.AdvanceMCPReceipt(ctx, submission.AssignmentID, submission.SubmissionID, factory.MCPReceiptRecorded, receipt); err != nil {
		if errors.Is(err, store.ErrStaleRevision) || errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// mapMCPTerminalFinish maps one recorded terminal proposal to its
// finished assignment. ok=false parks the proposal untouched: an
// FW40 role finish without its structured carrier, or an unknown
// tool. Mapping failures inside a handled tool fence the assignment
// as blocked for human attention, following the refusePhasedAttempt
// precedent: an invalid worker report never wedges its assignment.
func mapMCPTerminalFinish(assigned factory.Assignment, submission factory.MCPSubmission, now time.Time) (finished factory.Assignment, note string, ok bool, err error) {
	if assigned.Stage != factory.AssignmentAssigned {
		return factory.Assignment{}, "", false, errors.New("mcp settlement maps only assigned work")
	}
	var built factory.Assignment
	switch submission.Tool {
	case factory.MCPToolFinishLibraryMaintenance:
		built, note, ok, err = settleLibraryMaintenance(assigned, submission, now)
		if err != nil || !ok {
			return factory.Assignment{}, "", ok, err
		}
	case factory.MCPToolFinishResearch:
		built, note, ok, err = settleResearch(assigned, submission, now)
		if err != nil || !ok {
			return factory.Assignment{}, "", ok, err
		}
	case factory.MCPToolFinishProductPlan, factory.MCPToolFinishIssuePlan,
		factory.MCPToolFinishReview, factory.MCPToolFinishReassessment,
		factory.MCPToolFinishAppVerify:
		built, note, ok, err = mapFW40Terminal(assigned, submission, now)
		if err != nil || !ok {
			return factory.Assignment{}, "", ok, err
		}
	case factory.MCPToolFinishCorrection:
		built, note, ok, err = settleCorrectionFinish(assigned, submission, now)
		if err != nil || !ok {
			return factory.Assignment{}, "", ok, err
		}
	case factory.MCPToolFinishIncomplete, factory.MCPToolFinishImplement:
		payload, derr := decodeMCPTerminalPayload(submission)
		if derr != nil {
			built, note = fenceMCPTerminal(assigned, submission, now, "terminal payload refused: "+derr.Error()), "unparseable terminal fenced for human attention"
			break
		}
		switch submission.Tool {
		case factory.MCPToolFinishIncomplete:
			built, note = settleIncomplete(assigned, submission, payload, now), "worker-reported terminal settled"
		case factory.MCPToolFinishImplement:
			if assigned.Phase == project.ExecutionImplement && factory.ValidCommit(payload["candidate"]) {
				built, note = settleTypedImplementation(assigned, submission, payload["candidate"], now), "code finish settled"
			} else {
				built, note = settleCodeFinish(assigned, submission, payload, now), "code finish settled"
			}
		default:
			built, note = settleCodeFinish(assigned, submission, payload, now), "code finish settled"
		}
	default:
		return factory.Assignment{}, "", false, nil
	}
	// A reported success the assignment cannot carry (for example a
	// completion nominated against a reviewer-role assignment) fences
	// for attention instead of wedging on a store refusal: the worker
	// report stays inspectable through the accepted receipt.
	if err := built.Validate(); err != nil {
		built = fenceMCPTerminal(assigned, submission, now, "terminal report does not fit its assignment: "+err.Error())
		if verr := built.Validate(); verr != nil {
			return factory.Assignment{}, "", false, verr
		}
		note = "mismatched terminal fenced for human attention"
	}
	return built, note, true, nil
}

// decodeMCPTerminalPayload decodes one recorded terminal payload to
// its string fields. Non-string values refuse: the settler never
// coerces worker prose into settlement facts.
func decodeMCPTerminalPayload(submission factory.MCPSubmission) (map[string]string, error) {
	var raw map[string]any
	if err := json.Unmarshal(submission.Payload, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		text, isString := value.(string)
		if !isString {
			continue
		}
		out[key] = text
	}
	return out, nil
}

// mcpTerminalSummary bounds one worker text field into a result
// summary naming its tool and submission.
func mcpTerminalSummary(tool factory.MCPTool, submissionID, text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		text = fallback
	}
	summary := "mcp " + string(tool) + " " + submissionID + ": " + text
	if len(summary) > 16<<10 {
		summary = summary[:16<<10]
	}
	return summary
}

// fenceMCPTerminal finishes one assignment as blocked for human
// attention when its terminal proposal cannot settle: the recorded
// proposal stays inspectable through the accepted receipt.
func fenceMCPTerminal(assigned factory.Assignment, submission factory.MCPSubmission, now time.Time, detail string) factory.Assignment {
	finished := assigned
	result := factory.ResultSynthesized(assigned.ID, assigned.Run, "blocked",
		mcpTerminalSummary(submission.Tool, submission.SubmissionID, detail, "terminal needs human attention"), now.Unix())
	finished.Stage, finished.Outcome, finished.Reason = factory.AssignmentFinished, factory.NeedsHuman, factory.AssignReasonNoReport
	finished.FinishedUnix = now.Unix()
	finished.Result = &result
	return finished
}

// settleIncomplete settles a worker-reported terminal state: blocked
// fences for attention, failed records the failed run, cancelled
// records the cancellation.
func settleIncomplete(assigned factory.Assignment, submission factory.MCPSubmission, payload map[string]string, now time.Time) factory.Assignment {
	finished := assigned
	var status string
	switch payload["outcome"] {
	case "failed":
		status = "failed"
		finished.Outcome, finished.Reason = factory.Failed, factory.AssignReasonRunFailed
	case "cancelled":
		status = "cancelled"
		finished.Outcome, finished.Reason = factory.Cancelled, factory.AssignReasonCancelled
	default:
		status = "blocked"
		finished.Outcome, finished.Reason = factory.NeedsHuman, factory.AssignReasonNoReport
	}
	result := factory.ResultSynthesized(assigned.ID, assigned.Run, status,
		mcpTerminalSummary(submission.Tool, submission.SubmissionID, payload["cause"], "worker reports "+status), now.Unix())
	finished.Stage = factory.AssignmentFinished
	finished.FinishedUnix = now.Unix()
	finished.Result = &result
	return finished
}

// settleCodeFinish settles a Builder/Fixer finish. A verifiable commit
// candidate nominates the resulting tree as the reported harness
// result; publication still validates it before use. Anything else
// (unchanged, missing or malformed candidate) fences for attention:
// no-change work needs no fake commit and an unverifiable nomination
// authorizes nothing.
func settleCodeFinish(assigned factory.Assignment, submission factory.MCPSubmission, payload map[string]string, now time.Time) factory.Assignment {
	if candidate := payload["candidate"]; factory.ValidCommit(candidate) {
		finished := assigned
		reported := factory.Result{
			Status:    "completed",
			Summary:   mcpTerminalSummary(submission.Tool, submission.SubmissionID, "", "worker nominates "+candidate),
			Candidate: candidate,
		}
		result := factory.ResultFromHarness(assigned.ID, assigned.Run, reported, now.Unix())
		finished.Stage, finished.Outcome, finished.Reason = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported
		finished.FinishedUnix = now.Unix()
		finished.Result = &result
		return finished
	}
	return fenceMCPTerminal(assigned, submission, now, "code finish names no verifiable candidate")
}

// driveMCPTerminals settles one bounded page of recorded terminal
// proposals and reports how many settled. Individual failures keep
// their recorded receipt for a later pass; parked proposals stay
// listed and visible. Best effort like the other progress drives.
func driveMCPTerminals(ctx context.Context, st mcpDriveStore, now time.Time) int {
	if st == nil {
		return 0
	}
	ids, err := st.RecordedTerminalMCPAssignments(ctx, MaxDispatchRecovery)
	if err != nil {
		return 0
	}
	settled := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			break
		}
		if done, err := SettleMCPTerminalProposal(ctx, st, id, now); err == nil && done {
			settled++
		}
	}
	return settled
}

// settleMCPTerminals drives recorded MCP terminal settlement on the
// coordinator's store: the Start and progress-pass hook.
func (c *Coordinator) settleMCPTerminals(ctx context.Context) int {
	if c == nil || c.Store == nil {
		return 0
	}
	settled := driveMCPTerminals(ctx, c.Store, time.Now())
	c.driveAppVerifications(ctx)
	return settled
}
