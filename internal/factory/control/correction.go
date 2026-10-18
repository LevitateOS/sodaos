package control

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// CorrectionLink is one publication advanced to a corrected head: the new
// exact head, the committed correction operation and the stored revision.
type CorrectionLink struct {
	AssignmentID  string `json:"assignment_id"`
	PublicationID string `json:"publication_id"`
	HeadOID       string `json:"head_oid"`
	OperationID   string `json:"operation_id"`
	Revision      int64  `json:"revision"`
}

// CorrectionWait is one correction that did not advance this call and why.
type CorrectionWait struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Detail string `json:"Detail,omitempty"`
}

// CorrectionError is one unexpected correction infrastructure failure. The
// affected publication keeps its recorded state; a later call retries.
type CorrectionError struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Detail string `json:"Detail,omitempty"`
}

// CorrectionReport is the durable outcome of one correction call: the
// publication it advanced and every publication it fenced, or why the
// correction waited or errored.
type CorrectionReport struct {
	Corrected []CorrectionLink  `json:"corrected"`
	Fenced    []string          `json:"fenced,omitempty"`
	Waits     []CorrectionWait  `json:"waits,omitempty"`
	Errors    []CorrectionError `json:"errors,omitempty"`
}

func correctionError(report *CorrectionReport, id, reason string) {
	report.Errors = append(report.Errors, CorrectionError{ID: id, Reason: reason})
}

func correctionWait(report *CorrectionReport, id, reason string) {
	report.Waits = append(report.Waits, CorrectionWait{ID: id, Reason: reason})
}

// PublishCorrection advances one published publication to the exact head a
// reconciled coder child reported for its parent publication. The
// run output nominates the candidate; the host export attests it, and the
// conditional branch operation commits it to the same branch and PR. One
// correction identity per run: a finished correction for a run never
// reopens, and only a newer run's different candidate opens the next
// identity. The initial publication receipts stay untouched; corrections
// chain behind them. The stored head always follows the latest committed
// correction, so every store point satisfies publication validation.
func (c *Coordinator) PublishCorrection(ctx context.Context, runID string) CorrectionReport {
	report := CorrectionReport{Corrected: []CorrectionLink{}}
	if c.Publication == nil {
		correctionError(&report, runID, "operations_unavailable")
		return report
	}
	run, err := c.Store.FactoryRun(ctx, runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			correctionWait(&report, runID, "run_missing")
		} else {
			correctionError(&report, runID, "store_unavailable")
		}
		return report
	}
	if !run.Reconciled {
		correctionWait(&report, runID, "run_unsettled")
		return report
	}
	assignment, a, err := c.publicationAssignmentsForRun(ctx, run)
	if err != nil || assignment.ID == a.ID || assignment.Role != project.RoleCoder {
		correctionWait(&report, runID, "run_unlinked")
		return report
	}
	p, err := c.Store.PublicationByAssignment(ctx, a.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			correctionWait(&report, a.ID, "publication_missing")
		} else {
			correctionError(&report, a.ID, "store_unavailable")
		}
		return report
	}
	if p.Stage != factory.PublicationPublished || p.PRNumber <= 0 || p.PRID <= 0 || p.PRCreate.Work == nil || p.Publish.Work == nil {
		correctionWait(&report, p.ID, "publication_unpublished")
		return report
	}
	if assignment.Stage != factory.AssignmentFinished || assignment.Outcome != factory.Succeeded ||
		assignment.Result == nil || !assignment.Result.Reported || assignment.Result.Status != "completed" ||
		!factory.ValidCommit(assignment.Result.Candidate) {
		correctionWait(&report, p.ID, "candidate_invalid")
		return report
	}
	reported := assignment.Result.ToResult()
	open, _, _, err := c.Store.DispatchState(ctx, p.Repository)
	if err != nil {
		correctionError(&report, p.ID, "store_unavailable")
		return report
	}
	if p.WithdrawRequested || !open {
		correctionWait(&report, p.ID, "correction_withdrawn")
		return report
	}
	c.reconcileRecordedCorrection(ctx, &p, a, &report)
	if len(report.Errors) != 0 || len(report.Fenced) != 0 || len(report.Waits) != 0 || len(report.Corrected) != 0 {
		return report
	}
	// A child admitted for an older publication head may finish after a
	// different correction advances the branch. Its recorded intent above
	// still gets reconciled, but stale source work cannot open a new write.
	if p.Run != runID && assignment.SourceCommit != p.Candidate {
		correctionWait(&report, p.ID, "candidate_superseded")
		return report
	}
	// One identity per run: a recorded correction for this same candidate
	// either continues under its identity or stays finished. Only a
	// different candidate opens the next identity.
	continuing := false
	if latest := len(p.Corrections); latest > 0 && p.Corrections[latest-1].Work != nil &&
		p.Corrections[latest-1].Work.Candidate == reported.Candidate {
		switch p.Corrections[latest-1].Effect {
		case "", factory.OpEffectPending:
			continuing = true
		case factory.OpEffectCommitted:
			correctionWait(&report, p.ID, "correction_recorded")
			return report
		default:
			correctionWait(&report, p.ID, "correction_refused")
			return report
		}
	}
	// An unchanged head is not a correction and fences nothing: the run
	// reported no newer candidate, so correction waits without mutating
	// the publication. Fencing stays for genuinely invalid observations.
	if reported.Candidate == p.Candidate || reported.Candidate == p.BaseSHA {
		correctionWait(&report, p.ID, "candidate_invalid")
		return report
	}
	if err := recordConfirmedUsage(ctx, c.Store, assignment, run, time.Now()); err != nil {
		correctionError(&report, p.ID, "store_unavailable")
		return report
	}
	q := p
	q.Run, q.Candidate = runID, reported.Candidate
	bundle, terminal, wait := c.exportCandidate(ctx, q, run)
	if wait != nil {
		correctionWait(&report, p.ID, wait.Reason)
		return report
	}
	if terminal != nil {
		correctionError(&report, p.ID, terminal.Reason)
		return report
	}
	scratch := PublishReport{}
	policy, revision, allowed := c.publicationAuthority(ctx, p, &scratch)
	if !allowed {
		// The authority check reports exactly one wait or error.
		for _, w := range scratch.Waits {
			correctionWait(&report, p.ID, w.Reason)
		}
		for _, e := range scratch.Errors {
			correctionError(&report, p.ID, e.Reason)
		}
		return report
	}
	work := factory.PublicationWork{
		Bundle: bundle, AssignmentID: a.ID, Publication: p.ID, RunID: runID, Run: run,
		Candidate: reported.Candidate, BaseSHA: p.BaseSHA, TargetBranch: p.TargetBranch,
		OperationID:  factory.PublicationOperationID(p.ID, factory.OpRefPublish, len(p.Corrections)+2),
		AuthRevision: factory.AuthRevisionFor(a.ID, p.ID, p.Revision),
		ExpectedOld:  p.Candidate, ComparisonRef: p.TargetBranch,
		// The PR title/body stay the linked PR's own: a correction moves
		// the branch, never the recorded PR description.
		PRTitle:    factory.PRTitleFor(a.Issue),
		PRBody:     factory.PRBodyFor(a.Issue, a.ID, a.Acceptance, a.Run, p.Candidate, a.SourceCommit),
		Repository: p.Repository, Issue: p.Issue, ActorID: policy.Publish.ActorID,
		CorrectionNumber: p.PRNumber, CorrectionAuthor: p.PRCreate.Work.ActorID,
	}
	if continuing {
		// A crash between record and push resumes under the recorded
		// identity with a fresh bundle; the recorded intent stands.
		recorded := p.Corrections[len(p.Corrections)-1]
		work = recorded.Work.Apply(work)
	} else {
		observation, err := c.Publication.ObservePublication(ctx, work)
		if err != nil {
			c.correctionCallError(&report, p.ID, err)
			return report
		}
		if observation.NativeRev != revision {
			correctionWait(&report, p.ID, "native_inputs_changed")
			return report
		}
		if observation.TargetRef != factory.PublishBranchName(a.ID) || !factory.ValidCommit(observation.Comparison) {
			correctionWait(&report, p.ID, "observation_invalid")
			return report
		}
		if observation.TargetTip != p.Candidate {
			c.fenceCorrection(ctx, &report, &p, factory.PublishReasonBadObservation)
			return report
		}
		now := time.Now()
		notAfter, err := c.Store.CapAttemptDeadline(ctx, p.AssignmentID, now.Add(10*time.Minute).Unix())
		if err != nil {
			correctionWait(&report, p.ID, "attempt_deadline_unavailable")
			return report
		}
		work.NativeRev, work.ComparisonOID, work.NotAfter = observation.NativeRev, observation.Comparison, notAfter
		intent := work.Intent()
		op := factory.PublicationOperation{
			Work: &intent, RunID: runID, OperationID: work.OperationID, Kind: factory.OpRefPublish,
			Attempts: 1, UpdatedUnix: now.Unix(),
		}
		p.Corrections = append(p.Corrections, op)
		if !c.storeCorrection(ctx, &p, &report) {
			return report
		}
		work = p.Corrections[len(p.Corrections)-1].Work.Apply(work)
	}
	op := p.Corrections[len(p.Corrections)-1]
	if op.Cancellation != "" && op.Cancellation != factory.OpCancelNone {
		correctionWait(&report, p.ID, "cancellation_pending")
		return report
	}
	// Recheck the local gate after recording. A concurrent withdrawal
	// cancels this same recorded identity.
	open, _, _, err = c.Store.DispatchState(ctx, p.Repository)
	if err != nil {
		correctionError(&report, p.ID, "store_unavailable")
		return report
	}
	if p.WithdrawRequested || !open {
		c.cancelRecordedCorrection(ctx, &p, &report)
		return report
	}
	if op.Effect == "" {
		var outcome factory.OperationOutcome
		outcome, err = c.Publication.SubmitPublish(ctx, work)
		if err != nil {
			c.correctionCallError(&report, p.ID, err)
			return report
		}
		if outcome.NotObserved {
			correctionWait(&report, p.ID, "submit_unconfirmed")
			return report
		}
		if !c.adoptCorrectionOutcome(ctx, &p, outcome, &report) {
			return report
		}
		op = p.Corrections[len(p.Corrections)-1]
	}
	if op.Effect == factory.OpEffectPending && (op.Cancellation == "" || op.Cancellation == factory.OpCancelNone) {
		outcome, err := c.Publication.PushBranch(ctx, work)
		if err != nil {
			c.correctionCallError(&report, p.ID, err)
			return report
		}
		if outcome.NotObserved {
			correctionWait(&report, p.ID, "push_unconfirmed")
			return report
		}
		if !c.adoptCorrectionOutcome(ctx, &p, outcome, &report) {
			return report
		}
		op = p.Corrections[len(p.Corrections)-1]
	}
	switch op.Effect {
	case factory.OpEffectCommitted:
		c.linkCorrection(ctx, &p, a, &report)
	case factory.OpEffectNotCommitted:
		correctionWait(&report, p.ID, "correction_refused")
	case factory.OpEffectIndeterminate:
		c.fenceCorrection(ctx, &report, &p, factory.PublishReasonFenced)
	default:
		correctionWait(&report, p.ID, "operation_pending")
	}
	return report
}

// PublishCorrectionWithReport advances one corrector run's candidate like
// PublishCorrection after journaling the run's parsed correction report
// through the canonical gate. The caller parses the corrector output's
// correction block (ParseCorrectionReport) and supplies the report with
// its fix evidence and check assessment; this entry resolves the review
// cycle the corrector answered, admits the report, and records exactly
// one journal entry per run over the cycle's exact finding-set digest.
// The canonical binding applies when the cycle already carries a journal:
// a retry must resubmit byte-identical findings, so a rewritten defect
// refuses instead of replacing the recorded set. A disputed report still
// journals — the dispute is evidence for the bounded D08 decision — but
// the head does not advance on contested feedback: the entry waits for
// the funded decision instead of launching another run, and the routing
// leg reads the journaled dispositions live. Any other refused report
// waits without mutating; admitted reports delegate the head advance to
// PublishCorrection. Runs without a parsed report keep the legacy entry;
// this entry never guesses dispositions.
func (c *Coordinator) PublishCorrectionWithReport(ctx context.Context, runID string, report factory.CorrectionReport, evidence []factory.CorrectionFixEvidence, checks *factory.CheckAssessment) CorrectionReport {
	served := CorrectionReport{Corrected: []CorrectionLink{}}
	if c.Store == nil {
		correctionError(&served, runID, "operations_unavailable")
		return served
	}
	run, err := c.Store.FactoryRun(ctx, runID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			correctionWait(&served, runID, "run_missing")
		} else {
			correctionError(&served, runID, "store_unavailable")
		}
		return served
	}
	if !run.Reconciled {
		correctionWait(&served, runID, "run_unsettled")
		return served
	}
	assignment, owner, err := c.publicationAssignmentsForRun(ctx, run)
	if err != nil || assignment.ID == owner.ID || assignment.Role != project.RoleCoder {
		correctionWait(&served, runID, "run_unlinked")
		return served
	}
	p, err := c.Store.PublicationByAssignment(ctx, owner.ID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			correctionWait(&served, owner.ID, "publication_missing")
		} else {
			correctionError(&served, owner.ID, "store_unavailable")
		}
		return served
	}
	// Replay reuses the recorded journal: a run that already reported
	// never journals twice, and the head advance below replays too. A
	// replay must preserve the entire recorded report, including its
	// dispositions, candidate and evidence. A journaled dispute remains
	// held; recording the report did not authorize its candidate.
	for _, journal := range p.CorrectionJournals {
		if journal.RunID == runID {
			if !reflect.DeepEqual(journal.Report, report) {
				correctionWait(&served, p.ID, "correction_refused")
				served.Waits[len(served.Waits)-1].Detail = "correction replay differs from its recorded report"
				return served
			}
			for _, disposition := range journal.Report.Dispositions {
				if disposition.Disposition == factory.DispositionDisputed {
					correctionWait(&served, p.ID, "delivery_judgment_required")
					return served
				}
			}
			return c.PublishCorrection(ctx, runID)
		}
	}
	if assignment.Stage != factory.AssignmentFinished || assignment.Outcome != factory.Succeeded ||
		assignment.Result == nil || !assignment.Result.Reported || assignment.Result.Status != "completed" ||
		!factory.ValidCommit(assignment.Result.Candidate) {
		correctionWait(&served, p.ID, "candidate_invalid")
		return served
	}
	var cycle *factory.ReviewCycle
	for i := range p.ReviewCycles {
		if p.ReviewCycles[i].CorrectionAssignment == assignment.ID {
			cycle = &p.ReviewCycles[i]
			break
		}
	}
	if cycle == nil {
		correctionWait(&served, p.ID, "correction_cycle_unknown")
		return served
	}
	newCandidate := assignment.Result.Candidate
	// A superseded cycle answers another tree: the publication moved on
	// before this report arrived, so no new journal may append merely
	// because the eventual ref push would refuse. Already-recorded
	// operations keep their replay recovery above; only first effects
	// wait here.
	if cycle.Head != p.Candidate || cycle.BaseOID != p.PRCreate.BaseOID {
		correctionWait(&served, p.ID, "correction_cycle_superseded")
		return served
	}
	var canonical []factory.Finding
	for _, journal := range p.CorrectionJournals {
		if journal.CycleID == cycle.ID {
			canonical = journal.Report.Findings
			break
		}
	}
	if canonical != nil {
		_, err = AdmitCorrectionReportCanonical(*cycle, runID, newCandidate, canonical, report, evidence, checks)
	} else {
		// First admission binds the report to the owning history's
		// canonical carried findings, not just the round's finding-set
		// digest: same-key rewritten content, omissions and changed
		// origins refuse before any journal/ref effect.
		_, err = c.admitFirstCorrectionReport(ctx, p, *cycle, runID, newCandidate, report, evidence, checks)
	}
	if err != nil {
		var block *CorrectionBlock
		if errors.As(err, &block) {
			if journalErr := c.journalCorrectionReport(ctx, &p, *cycle, runID, report); journalErr != nil {
				correctionError(&served, p.ID, "store_unavailable")
				return served
			}
			correctionWait(&served, p.ID, "delivery_judgment_required")
			return served
		}
		correctionWait(&served, p.ID, "correction_refused")
		served.Waits[len(served.Waits)-1].Detail = err.Error()
		return served
	}
	if journalErr := c.journalCorrectionReport(ctx, &p, *cycle, runID, report); journalErr != nil {
		correctionError(&served, p.ID, "store_unavailable")
		return served
	}
	return c.PublishCorrection(ctx, runID)
}

// journalCorrectionReport records one structurally valid corrector report
// as the publication's next immutable correction journal entry. Refused
// admissions never reach it; admitted and disputed reports do, so the
// D08 routing leg always reads serving-journaled dispositions.
func (c *Coordinator) journalCorrectionReport(ctx context.Context, p *factory.Publication, cycle factory.ReviewCycle, runID string, report factory.CorrectionReport) error {
	if err := appendCorrectionJournal(p, cycle, runID, report); err != nil {
		return err
	}
	p.Revision++
	if err := c.Store.UpdatePublication(ctx, *p); err != nil {
		return err
	}
	return nil
}

// reconcileRecordedCorrection reconciles one in-flight recorded correction
// by lookup before any new write, and links a recovered commit exactly
// like the live path. Anything unattributed fences the publication.
func (c *Coordinator) reconcileRecordedCorrection(ctx context.Context, p *factory.Publication, a factory.Assignment, report *CorrectionReport) {
	if len(p.Corrections) == 0 {
		return
	}
	op := &p.Corrections[len(p.Corrections)-1]
	if op.Attempts == 0 || op.Effect == factory.OpEffectNotCommitted || op.Effect == factory.OpEffectIndeterminate ||
		(op.Effect == factory.OpEffectCommitted && op.Completion == factory.OpCompletionComplete) {
		return
	}
	outcome, err := c.Publication.LookupOp(ctx, op.OperationID)
	if err != nil {
		c.correctionCallError(report, p.ID, err)
		return
	}
	if outcome.NotObserved {
		correctionWait(report, p.ID, "lookup_unconfirmed")
		return
	}
	// The correction row carries the child run independently of the
	// credential-free native intent, so recovery attributes the new head
	// to the same settled result that registered it.
	if !c.adoptCorrectionOutcome(ctx, p, outcome, report) {
		return
	}
	if p.Corrections[len(p.Corrections)-1].Effect == factory.OpEffectCommitted {
		c.linkCorrection(ctx, p, a, report)
	}
}

// adoptCorrectionOutcome adopts one observed correction outcome into the
// latest recorded correction. A commit advances the stored head before
// the store, so the record always satisfies publication validation; an
// unattributed outcome fences the publication instead.
func (c *Coordinator) adoptCorrectionOutcome(ctx context.Context, p *factory.Publication, outcome factory.OperationOutcome, report *CorrectionReport) bool {
	op := &p.Corrections[len(p.Corrections)-1]
	if !c.adoptObserved(op, outcome, time.Now()) {
		c.fenceCorrection(ctx, report, p, factory.PublishReasonUnattributed)
		return false
	}
	if op.Effect == factory.OpEffectCommitted {
		if op.Work == nil || op.RunID == "" {
			c.fenceCorrection(ctx, report, p, factory.PublishReasonUnattributed)
			return false
		}
		p.Candidate = op.Work.Candidate
		p.Run = op.RunID
	}
	return c.storeCorrection(ctx, p, report)
}

// linkCorrection verifies one committed correction receipt against its
// recorded intent and links it. A receipt that does not match fences the
// publication instead of linking a moved head.
func (c *Coordinator) linkCorrection(ctx context.Context, p *factory.Publication, a factory.Assignment, report *CorrectionReport) {
	op := &p.Corrections[len(p.Corrections)-1]
	adopt := op.Work.Apply(factory.PublicationWork{AssignmentID: a.ID, Publication: p.ID})
	if _, err := c.Publication.AdoptBranch(adopt, operationOutcomeOf(*op)); err != nil {
		c.fenceCorrection(ctx, report, p, factory.PublishReasonUnattributed)
		return
	}
	report.Corrected = append(report.Corrected, CorrectionLink{
		AssignmentID: p.AssignmentID, PublicationID: p.ID,
		HeadOID: p.Candidate, OperationID: op.OperationID, Revision: p.Revision,
	})
}

// cancelRecordedCorrection cancels the just-recorded correction after a
// concurrent withdrawal closed the gate.
func (c *Coordinator) cancelRecordedCorrection(ctx context.Context, p *factory.Publication, report *CorrectionReport) {
	op := &p.Corrections[len(p.Corrections)-1]
	outcome, err := c.Publication.CancelOp(ctx, op.OperationID)
	if err != nil {
		c.correctionCallError(report, p.ID, err)
		return
	}
	if !outcome.NotObserved && !c.adoptObserved(op, outcome, time.Now()) {
		c.fenceCorrection(ctx, report, p, factory.PublishReasonUnattributed)
		return
	}
	if !outcome.NotObserved && op.Effect == factory.OpEffectCommitted {
		if op.Work == nil {
			c.fenceCorrection(ctx, report, p, factory.PublishReasonUnattributed)
			return
		}
		p.Candidate = op.Work.Candidate
		p.Run = op.RunID
		work := op.Work.Apply(factory.PublicationWork{AssignmentID: p.AssignmentID, Publication: p.ID})
		if _, err := c.Publication.AdoptBranch(work, operationOutcomeOf(*op)); err != nil {
			c.fenceCorrection(ctx, report, p, factory.PublishReasonUnattributed)
			return
		}
	}
	if !c.storeCorrection(ctx, p, report) {
		return
	}
	correctionWait(report, p.ID, "correction_withdrawn")
}

func (c *Coordinator) fenceCorrection(ctx context.Context, report *CorrectionReport, p *factory.Publication, reason string) {
	p.Stage, p.Outcome, p.Reason, p.FinishedUnix = factory.PublicationFenced, factory.NeedsHuman, reason, time.Now().Unix()
	if !c.storeCorrection(ctx, p, report) {
		return
	}
	report.Fenced = append(report.Fenced, p.ID)
}

func (c *Coordinator) storeCorrection(ctx context.Context, p *factory.Publication, report *CorrectionReport) bool {
	p.Revision++
	if err := c.Store.UpdatePublication(ctx, *p); err != nil {
		correctionError(report, p.ID, "store_conflict")
		return false
	}
	return true
}

func (c *Coordinator) correctionCallError(report *CorrectionReport, id string, err error) {
	var wait *factory.PublicationWait
	var refused *factory.PublicationRefusal
	switch {
	case errors.As(err, &wait):
		correctionWait(report, id, wait.Reason)
	case errors.As(err, &refused):
		correctionWait(report, id, refused.Reason)
	default:
		correctionError(report, id, "native_unavailable")
	}
}
