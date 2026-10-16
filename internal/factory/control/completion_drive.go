package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// HoldCompletionWiring reports a collecting cycle that cannot advance
// because the native-page or local-capture binding is unwired. The
// FW32 handoff binds both; until then the fence holds honestly instead
// of sealing from incomplete evidence.
const HoldCompletionWiring = "completion_reads_unwired"

// CompletionDriveDeps carries the two FW32-handoff bindings behind one
// completion drive: native inventory pages and the local capture. A
// nil binding holds collecting cycles with HoldCompletionWiring; it
// never seals, decides or releases.
type CompletionDriveDeps struct {
	Enumerator   CompletionIssueEnumerator
	CaptureLocal func(ctx context.Context, repository int64) (factory.CompletionLocal, error)
}

// CompletionDriveReport is the durable outcome of one completion
// drive: cycle IDs sealed, decided, released or held for FW32, plus
// resumable hold reasons. It spends nothing by itself; funded calls
// flow through the decision path with their own receipts.
type CompletionDriveReport struct {
	Sealed        []string
	Decided       []string
	Released      []string
	AwaitingCoder []string
	Held          []string
}

// SealEvidenceFingerprint binds the funded-decision cache to the exact
// sealed evidence: the sealed inventory digest plus the distinct
// completion and cycle IDs. Two completions never share a decision
// cache entry, even with identical inventories.
func SealEvidenceFingerprint(inventoryDigest, completionID, cycleID string) string {
	sum := sha256.Sum256([]byte(inventoryDigest + "\x00" + completionID + "\x00" + cycleID))
	return hex.EncodeToString(sum[:])
}

// DriveCompletions advances every unresolved completion cycle once, in
// repository and generation order. Collecting cycles seal their full
// inventory; deciding cycles take exactly one funded decision each;
// valid continue decisions release after freshness, recheck and
// snapshot invalidation; valid reassess decisions rest recorded for
// the FW32 coder admission. Attention and superseded cycles are never
// touched: no poll, input edit or policy edit resumes them, and no
// automatic re-run ever starts. Reassessing, assessing and applying
// cycles belong to FW32 and rest reported. Every failure holds with
// its resumable step; a later drive retries from persisted progress.
func (c *Coordinator) DriveCompletions(ctx context.Context, deps CompletionDriveDeps) CompletionDriveReport {
	var report CompletionDriveReport
	if c.Store == nil {
		return report
	}
	repositories, err := c.Store.PendingCompletionRepositories(ctx)
	if err != nil {
		return report
	}
	for _, repository := range repositories {
		cycles, err := c.Store.PendingReassessmentCycles(ctx, repository)
		if err != nil {
			continue
		}
		for _, record := range cycles {
			switch routeCompletionCycle(record.Cycle.State) {
			case driveCollect:
				c.driveCollecting(ctx, deps, record, &report)
			case driveDecide:
				c.driveDeciding(ctx, deps, record, &report)
			case driveCoder:
				report.AwaitingCoder = append(report.AwaitingCoder, record.Cycle.CycleID)
			default:
				report.Held = append(report.Held, record.Cycle.CycleID+":awaiting-retry")
			}
		}
	}
	return report
}

// driveRoute names one cycle's handling in a drive. Collecting seals,
// deciding decides, reassessing and later rest for the FW32 coder
// admission, and attention and superseded rest for explicit Retry:
// the drive never resumes them and never starts an automatic re-run.
type driveRoute int

const (
	driveHold driveRoute = iota
	driveCollect
	driveDecide
	driveCoder
)

func routeCompletionCycle(state string) driveRoute {
	switch state {
	case factory.ReassessmentCollecting:
		return driveCollect
	case factory.ReassessmentDeciding:
		return driveDecide
	case factory.ReassessmentReassessing, factory.ReassessmentAssessing, factory.ReassessmentApplying:
		return driveCoder
	default:
		return driveHold
	}
}

// driveCollecting seals one collecting cycle or persists its resumable
// progress with the hold reason.
func (c *Coordinator) driveCollecting(ctx context.Context, deps CompletionDriveDeps, record store.ReassessmentRecord, report *CompletionDriveReport) {
	cycleID := record.Cycle.CycleID
	hold := func(reason string, progress *InventoryCollector) {
		record.WaitReason = reason
		if progress != nil {
			if raw, err := json.Marshal(progress); err == nil {
				record.CollectorProgress = raw
			}
		}
		if err := c.Store.CasReassessmentRecord(ctx, cycleID, factory.ReassessmentCollecting, record); err != nil {
			if !errors.Is(err, store.ErrStaleRevision) && !errors.Is(err, store.ErrNotFound) {
				report.Held = append(report.Held, cycleID+":collect_hold_unrecorded")
				return
			}
		}
		report.Held = append(report.Held, cycleID+":"+reason)
	}
	if deps.Enumerator == nil || deps.CaptureLocal == nil {
		hold(HoldCompletionWiring, nil)
		return
	}
	local, err := deps.CaptureLocal(ctx, record.Cycle.Repository)
	if err != nil {
		hold(HoldInventoryUnavailable, nil)
		return
	}
	collector := &InventoryCollector{Revision: 0}
	if len(record.CollectorProgress) > 0 {
		var saved InventoryCollector
		if err := json.Unmarshal(record.CollectorProgress, &saved); err == nil {
			collector = &saved
		}
	}
	sealed, progress, err := collector.Collect(ctx, deps.Enumerator, record.Cycle.Repository, record.Cycle.Generation, local)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			hold(HoldInventoryInProgress, progress)
			return
		}
		hold(err.Error(), progress)
		return
	}
	record.InventoryDigest = sealed.Digest
	record.EvidenceFingerprint = SealEvidenceFingerprint(sealed.Digest, record.Cycle.EventID, cycleID)
	record.Local = sealed.Local
	record.Evidence = sealed.Bodies
	record.EvidenceIssues = sealed.Issues
	record.EvidenceRevision = sealed.Revision
	record.CollectorProgress = nil
	record.WaitReason = ""
	if err := record.Cycle.AdvanceTo(factory.ReassessmentDeciding); err != nil {
		hold(HoldInventoryUnavailable, progress)
		return
	}
	next := record.Cycle
	next.State = factory.ReassessmentDeciding
	if err := next.Validate(); err != nil {
		hold(HoldInventoryUnavailable, progress)
		return
	}
	record.Cycle = next
	if err := c.Store.CasReassessmentRecord(ctx, cycleID, factory.ReassessmentCollecting, record); err != nil {
		if errors.Is(err, store.ErrStaleRevision) || errors.Is(err, store.ErrNotFound) {
			return
		}
		hold(HoldInventoryInProgress, progress)
		return
	}
	report.Sealed = append(report.Sealed, cycleID)
}

// driveDeciding takes one funded decision for an undecided cycle,
// records a reassess decision for the FW32 coder admission, or
// releases a fresh continue decision. A recorded reassess decision
// rests: the sole run admission is FW32's joint.
func (c *Coordinator) driveDeciding(ctx context.Context, deps CompletionDriveDeps, record store.ReassessmentRecord, report *CompletionDriveReport) {
	cycleID := record.Cycle.CycleID
	attend := func(reason string) {
		if err := record.Cycle.AdvanceTo(factory.ReassessmentAttention); err != nil {
			report.Held = append(report.Held, cycleID+":attention_refused")
			return
		}
		record.WaitReason = reason
		next := record.Cycle
		next.State = factory.ReassessmentAttention
		record.Cycle = next
		if err := c.Store.CasReassessmentRecord(ctx, cycleID, factory.ReassessmentDeciding, record); err != nil {
			if errors.Is(err, store.ErrStaleRevision) || errors.Is(err, store.ErrNotFound) {
				return
			}
		}
		report.Held = append(report.Held, cycleID+":"+reason)
	}
	if record.Cycle.Decision == factory.ReassessmentOptionReassess {
		report.AwaitingCoder = append(report.AwaitingCoder, cycleID)
		return
	}
	if record.Cycle.Decision == "" {
		if c.Planning == nil || c.Planning.Chooser == nil {
			record.WaitReason = HoldCompletionWiring
			if err := c.Store.CasReassessmentRecord(ctx, cycleID, factory.ReassessmentDeciding, record); err == nil {
				report.Held = append(report.Held, cycleID+":"+HoldCompletionWiring)
			}
			return
		}
		state, err := DecisionStateFromRecord(record)
		if err != nil {
			if errors.Is(err, errReassessmentDecisionOversize) {
				attend("decision_evidence_oversize")
				return
			}
			attend("decision_evidence_missing")
			return
		}
		decided, err := c.DecideReassessment(ctx, record, state)
		if err != nil {
			if errors.Is(err, errReassessmentDeferred) {
				report.Held = append(report.Held, cycleID+":decision_in_flight")
				return
			}
			if errors.Is(err, errReassessmentExhausted) {
				attend("funding_exhausted")
				return
			}
			if decided.WaitReason != "" {
				record = decided
				attend(decided.WaitReason)
				return
			}
			report.Held = append(report.Held, cycleID+":decision_deferred")
			return
		}
		if decided.Cycle.Decision == factory.ReassessmentOptionReassess {
			if err := c.Store.CasReassessmentRecord(ctx, cycleID, factory.ReassessmentDeciding, decided); err != nil {
				if errors.Is(err, store.ErrStaleRevision) || errors.Is(err, store.ErrNotFound) {
					return
				}
				report.Held = append(report.Held, cycleID+":decision_hold_unrecorded")
				return
			}
			report.Decided = append(report.Decided, cycleID)
			report.AwaitingCoder = append(report.AwaitingCoder, cycleID)
			return
		}
		record = decided
	}
	// A continue decision releases only after freshness, recheck and
	// snapshot invalidation. Any failure before the release CAS defers
	// (the recorded decision stands; the next drive retries without
	// charging again). Material drift holds with attention.
	if record.Cycle.Decision != factory.ReassessmentOptionContinue {
		attend("decision_not_actionable")
		return
	}
	if deps.Enumerator == nil || deps.CaptureLocal == nil {
		attend("freshness_unavailable")
		return
	}
	current, err := deps.CaptureLocal(ctx, record.Cycle.Repository)
	if err != nil {
		attend("freshness_unavailable")
		return
	}
	// The release re-observes the material inventory through the
	// enumerator at a stable native revision: native edits and
	// insertions after the seal change the fresh digest and refuse
	// the stale release, while an unchanged world releases. The
	// sealed digest never appears on the current side.
	fresh, err := ReobserveCompletionInventory(ctx, deps.Enumerator, record.Cycle.Repository, record.Cycle.Generation, current)
	if err != nil {
		// A busy or shifting native state defers the release: the
		// recorded continue decision stands and the next drive
		// retries without charging again. Material or structural
		// drift holds with attention for explicit Retry.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			report.Held = append(report.Held, cycleID+":release_deferred")
			return
		}
		switch err.Error() {
		case HoldInventoryRevision, HoldInventoryInProgress:
			report.Held = append(report.Held, cycleID+":release_deferred")
			return
		default:
			attend(err.Error())
			return
		}
	}
	sealed := freshnessFromBindings(record.Local, record.InventoryDigest)
	if err := CheckCompletionFreshness(sealed, freshnessFromBindings(fresh.Local, fresh.Digest)); err != nil {
		attend(err.Error())
		return
	}
	if err := c.Store.DeletePriorityScan(ctx); err != nil {
		report.Held = append(report.Held, cycleID+":release_deferred")
		return
	}
	if err := c.Store.EnqueueReadinessRootEvent(ctx, record.Cycle.Repository, record.Issue); err != nil {
		report.Held = append(report.Held, cycleID+":release_deferred")
		return
	}
	if err := record.Cycle.AdvanceTo(factory.ReassessmentReleased); err != nil {
		report.Held = append(report.Held, cycleID+":release_refused")
		return
	}
	next := record.Cycle
	next.State = factory.ReassessmentReleased
	record.Cycle = next
	record.WaitReason = ""
	if err := c.Store.CasReassessmentRecord(ctx, cycleID, factory.ReassessmentDeciding, record); err != nil {
		if errors.Is(err, store.ErrStaleRevision) || errors.Is(err, store.ErrNotFound) {
			return
		}
		report.Held = append(report.Held, cycleID+":release_deferred")
		return
	}
	report.Released = append(report.Released, cycleID)
}

// StartCompletionRetry opens the one successor cycle for an attention
// cycle through its explicit authorized Retry. Decision-phase retries
// need only the attention state, the expected prior identity and
// reconciled decision calls: when the record carries an evidence
// fingerprint, its receipt must exist (completed or charged-failed);
// cycles with admitted coder runs refuse so FW32 reconciles runs and
// native operations first. Command replay reuses the recorded
// successor; no ordinary poll, input edit or policy edit creates one.
func (c *Coordinator) StartCompletionRetry(ctx context.Context, retry factory.ReassessmentRetry) (string, error) {
	if c.Store == nil {
		return "", store.ErrNotFound
	}
	record, err := c.Store.ReassessmentRecord(ctx, retry.ExpectedPriorCycle)
	if err != nil {
		return "", err
	}
	if record.Cycle.State == factory.ReassessmentSuperseded {
		return c.Store.StartReassessmentRetry(ctx, retry)
	}
	if record.Cycle.State != factory.ReassessmentAttention {
		return "", errors.New("only an attention cycle waits for retry")
	}
	if len(record.Cycle.Ledger.CoderRuns) != 0 {
		return "", errors.New("reassessment runs reconcile before retry")
	}
	if record.EvidenceFingerprint != "" {
		state, err := DecisionStateFromRecord(record)
		if err != nil {
			return "", err
		}
		if _, err := c.Store.PriorityJudgment(ctx, fingerprintOf(state)); err != nil {
			return "", errors.New("reassessment calls reconcile before retry")
		}
	}
	return c.Store.StartReassessmentRetry(ctx, retry)
}
