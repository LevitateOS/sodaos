package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// ErrCompletionFenced reports a dispatch packet for a repository whose
// completion fence is closed: at least one completion event has no
// released cycle yet. New issue investigation/implementation waits with
// reassessment_pending; already-running work, review/correction,
// cancellation and reconciliation are unaffected. Other repositories
// continue normally: a fence never turns the global tournament into a
// wait.
var ErrCompletionFenced = errors.New("repository completion fence is closed")

// ErrReassessmentBudgetExhausted reports a funded reassessment request
// the existing global consultation ceiling refused. Nothing was
// charged; the event holds with attention and waits for explicit
// Retry, never an automatic repeat.
var ErrReassessmentBudgetExhausted = errors.New("reassessment consultation budget is exhausted")

// ReassessmentRecord is one persisted bounded cycle: the FW30 domain
// cycle plus the sealed completion event, the complete sealed issue
// bodies with their inventory digest and native revision bracket, the
// funded-decision evidence fingerprint, the wait/retry reason for read
// models and the opaque resumable collector progress. Sealed evidence
// never changes: once an inventory digest is recorded, later writes
// must carry it with its bodies unchanged.
type ReassessmentRecord struct {
	Cycle               factory.ReassessmentCycle         `json:"cycle"`
	Completion          factory.CompletionEvent           `json:"completion,omitempty"`
	Local               factory.CompletionLocal           `json:"local,omitempty"`
	Evidence            []factory.CompletionIssueEvidence `json:"evidence,omitempty"`
	InventoryDigest     string                            `json:"inventory_digest,omitempty"`
	EvidenceFingerprint string                            `json:"evidence_fingerprint,omitempty"`
	WaitReason          string                            `json:"wait_reason,omitempty"`
	CollectorProgress   json.RawMessage                   `json:"collector_progress,omitempty"`
	Issue               int64                             `json:"issue,omitempty"`
	EvidenceIssues      int                               `json:"evidence_issues,omitempty"`
	EvidenceRevision    int64                             `json:"evidence_revision,omitempty"`
}

// evidenceBytes totals the canonical sealed body bytes for the bound
// check: the same bytes the collector bounded before the seal.
func (r ReassessmentRecord) evidenceBytes() int {
	total := 0
	for _, body := range r.Evidence {
		raw, err := body.CanonicalData()
		if err != nil {
			return factory.MaxReassessmentEvidenceBytes + 1
		}
		total += len(raw)
	}
	return total
}

// Validate checks the record shape without enforcing liveness: the
// cycle validates itself, digests validate when present, sealed bodies
// validate within their count and byte bounds with indexes matching
// the sealed count, and the wait reason stays a short resumable-step
// name, never evidence bytes.
func (r ReassessmentRecord) Validate() error {
	if err := r.Cycle.Validate(); err != nil {
		return err
	}
	if r.InventoryDigest != "" && !factory.ValidDigest(r.InventoryDigest) {
		return errors.New("invalid sealed inventory digest")
	}
	if r.EvidenceFingerprint != "" && !factory.ValidDigest(r.EvidenceFingerprint) {
		return errors.New("invalid decision evidence fingerprint")
	}
	if r.InventoryDigest != "" {
		if r.Evidence == nil {
			return errors.New("sealed reassessment lacks its issue bodies")
		}
		if len(r.Evidence) != r.EvidenceIssues || len(r.Evidence) > factory.MaxReassessmentIssues {
			return errors.New("sealed reassessment bodies differ from their sealed count")
		}
		if r.EvidenceRevision < 1 {
			return errors.New("sealed reassessment lacks its revision bracket")
		}
		if r.evidenceBytes() > factory.MaxReassessmentEvidenceBytes {
			return errors.New("sealed reassessment bodies exceed their bound")
		}
		for _, body := range r.Evidence {
			if err := body.Validate(); err != nil {
				return err
			}
		}
	}
	if len(r.WaitReason) > factory.MaxReassessmentReasonChars {
		return errors.New("invalid reassessment wait reason")
	}
	return nil
}

// recordCompletionTx inserts one verified completion and seeds its first
// bounded cycle inside the caller's transaction. The caller is the
// first UpdateMerge recording this merge as merged: the merge already
// carries its confirmed merged commit, PR linkage, acceptance and
// original issue-closing occurrence (Merge.Validate enforces them for
// merged merges), and the same transaction already closed the issue
// allowance and retained the readiness enqueue, so rollback rolls back
// all four together with this insert.
//
// Exact-once: the (publication, occurrence) unique key keeps recovery
// re-observation from recording the event twice; a replay returns
// inserted=false and leaves the first cycle standing. Ordering: the
// repository dispatch-gate row is locked here, the same row packet
// admission locks, so whichever commits first defines the order and the
// fence check inside packet transactions always observes this insert.
func recordCompletionTx(ctx context.Context, tx *sql.Tx, m factory.Merge) (bool, error) {
	if m.Stage != factory.MergeMerged {
		return false, errors.New("completion needs its merged merge")
	}
	if !factory.ValidNativeOccurrence(m.OriginalClosingOccurrence) {
		return false, errors.New("completion needs its original issue closure")
	}
	if !project.ValidDecisionID(m.Acceptance) {
		return false, errors.New("completion needs its acceptance binding")
	}
	if !factory.ValidCommit(m.MergedCommit) {
		return false, errors.New("completion needs its merged commit")
	}
	if _, _, _, err := ensureDispatchGateTx(ctx, tx, m.Repository); err != nil {
		return false, err
	}
	var generation int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(generation),0)+1 FROM factory_completions WHERE repository=$1`, m.Repository).Scan(&generation); err != nil {
		return false, err
	}
	event := factory.CompletionEvent{
		PublicationID: m.PublicationID, MergedCommit: m.MergedCommit,
		ClosingOccurrence: m.OriginalClosingOccurrence, AcceptanceID: m.Acceptance,
		ID: factory.NewID(), Repository: m.Repository, Issue: m.Issue, Generation: generation,
	}
	if err := event.Validate(); err != nil {
		return false, err
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		return false, err
	}
	var insertedID string
	err = tx.QueryRowContext(ctx, `INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data)
		VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(publication,occurrence) DO NOTHING RETURNING event`,
		event.Repository, event.Generation, event.ID, event.PublicationID, event.ClosingOccurrence, string(eventData)).Scan(&insertedID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	cycle := factory.ReassessmentCycle{
		EventID: event.ID, CycleID: factory.NewID(),
		State: factory.ReassessmentCollecting, Repository: event.Repository, Generation: event.Generation,
	}
	if err := cycle.Validate(); err != nil {
		return false, err
	}
	record := ReassessmentRecord{Cycle: cycle, Completion: event, Issue: m.Issue}
	recordData, err := json.Marshal(record)
	if err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data)
		VALUES($1,$2,$3,$4,$5,$6)`, cycle.CycleID, cycle.Repository, cycle.Generation, cycle.EventID, cycle.State, string(recordData)); err != nil {
		return false, err
	}
	return true, nil
}

// checkCompletionFenceTx refuses packet admission while the repository
// fence is closed. The caller holds the repository dispatch-gate row
// lock (the same row completion insertion locks), so this read observes
// every committed completion event. Only released or superseded cycles
// rest the fence: collecting, deciding, reassessing, assessing,
// applying and attention all keep it closed.
func checkCompletionFenceTx(ctx context.Context, tx *sql.Tx, repository int64) error {
	var pending bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_reassessments
		WHERE repository=$1 AND state NOT IN ('released','superseded'))`, repository).Scan(&pending); err != nil {
		return err
	}
	if pending {
		return ErrCompletionFenced
	}
	return nil
}

// CompletionPending lists the pending completion generations for one
// repository in processing order: every generation whose cycle has not
// reached a terminal state. An empty list means the fence is open.
func (s *Store) CompletionPending(ctx context.Context, repository int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT generation FROM factory_reassessments
		WHERE repository=$1 AND state NOT IN ('released','superseded') ORDER BY generation`, repository)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var pending []int64
	for rows.Next() {
		var generation int64
		if err = rows.Scan(&generation); err != nil {
			_ = rows.Close()
			return nil, err
		}
		pending = append(pending, generation)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	return pending, nil
}

// PendingReassessmentCycles lists every unresolved cycle for one
// repository in generation order so the completion driver processes
// the oldest event first. Released and superseded cycles are history,
// not work.
func (s *Store) PendingReassessmentCycles(ctx context.Context, repository int64) ([]ReassessmentRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_reassessments
		WHERE repository=$1 AND state NOT IN ('released','superseded') ORDER BY generation`, repository)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var cycles []ReassessmentRecord
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			_ = rows.Close()
			return nil, err
		}
		var record ReassessmentRecord
		if err = json.Unmarshal(raw, &record); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if err = record.Validate(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		cycles = append(cycles, record)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	return cycles, nil
}

// CaptureCompletionLocal reads the local half of D13 sealed evidence
// for one repository in a single repeatable-read transaction: the
// requirement source commit, the approved-preparation digest, the
// accepted-scope digest (acceptance heads plus withdrawals plus the
// material issue-control state) and the
// authority fingerprint. It is the production CompletionLocalCapture
// behind the completion drive: stable while nothing material changes,
// sensitive to source, preparation, scope and authority changes, and
// honest about gaps — a repository without a project or requirement
// source refuses instead of sealing from partial bindings. Private
// per-issue candidate custody and runtime preparation state stay out:
// they are volatile execution detail, not the approved local bindings
// the seal interprets the inventory against.
func (s *Store) CaptureCompletionLocal(ctx context.Context, repository int64) (factory.CompletionLocal, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var projectID string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM projects WHERE repository_id=$1`, repository).Scan(&projectID); err != nil {
		return factory.CompletionLocal{}, err
	}
	var requirementID string
	if err = tx.QueryRowContext(ctx, `SELECT decision FROM project_requirement_heads WHERE project_id=$1`, projectID).Scan(&requirementID); err != nil {
		return factory.CompletionLocal{}, err
	}
	var requirementData []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM project_requirement_decisions WHERE id=$1`, requirementID).Scan(&requirementData); err != nil {
		return factory.CompletionLocal{}, err
	}
	var requirement project.RequirementDecision
	if err = json.Unmarshal(requirementData, &requirement); err != nil {
		return factory.CompletionLocal{}, err
	}
	if !factory.ValidCommit(requirement.SourceCommit) {
		return factory.CompletionLocal{}, errors.New("completion local lacks its source commit")
	}
	preparationRows, err := tx.QueryContext(ctx, `SELECT data FROM project_preparations
		WHERE project_id=$1 AND NOT (data ? 'factory_admission') ORDER BY id`, projectID)
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	type preparationBinding struct {
		ID                string `json:"id"`
		Revision          int64  `json:"revision"`
		SourceCommit      string `json:"source_commit"`
		SetupDigest       string `json:"setup_digest"`
		RequirementDigest string `json:"requirement_digest"`
		ApprovalDigest    string `json:"approval_digest"`
	}
	var preparations []preparationBinding
	for preparationRows.Next() {
		var raw []byte
		var stored project.StoredPreparation
		if err = preparationRows.Scan(&raw); err != nil {
			_ = preparationRows.Close()
			return factory.CompletionLocal{}, err
		}
		if err = json.Unmarshal(raw, &stored); err != nil {
			_ = preparationRows.Close()
			return factory.CompletionLocal{}, err
		}
		preparations = append(preparations, preparationBinding{
			ID: stored.Preparation.ID, Revision: stored.Preparation.Revision,
			SourceCommit: stored.Preparation.SourceCommit, SetupDigest: stored.Preparation.SetupDigest,
			RequirementDigest: stored.Preparation.Requirements.Digest,
			ApprovalDigest:    stored.Preparation.Approval.EffectsDigest,
		})
	}
	if err = preparationRows.Err(); err != nil {
		_ = preparationRows.Close()
		return factory.CompletionLocal{}, err
	}
	_ = preparationRows.Close()
	headRows, err := tx.QueryContext(ctx, `SELECT issue, decision FROM issue_acceptance_heads
		WHERE repository=$1 ORDER BY issue`, repository)
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	type scopeBinding struct {
		Issue    int64  `json:"issue"`
		Decision string `json:"decision"`
	}
	var scope []scopeBinding
	for headRows.Next() {
		var binding scopeBinding
		if err = headRows.Scan(&binding.Issue, &binding.Decision); err != nil {
			_ = headRows.Close()
			return factory.CompletionLocal{}, err
		}
		scope = append(scope, binding)
	}
	if err = headRows.Err(); err != nil {
		_ = headRows.Close()
		return factory.CompletionLocal{}, err
	}
	_ = headRows.Close()
	withdrawalRows, err := tx.QueryContext(ctx, `SELECT issue, decision FROM issue_acceptance_withdrawals
		WHERE repository=$1 ORDER BY issue, decision`, repository)
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	var withdrawals []scopeBinding
	for withdrawalRows.Next() {
		var binding scopeBinding
		if err = withdrawalRows.Scan(&binding.Issue, &binding.Decision); err != nil {
			_ = withdrawalRows.Close()
			return factory.CompletionLocal{}, err
		}
		withdrawals = append(withdrawals, binding)
	}
	if err = withdrawalRows.Err(); err != nil {
		_ = withdrawalRows.Close()
		return factory.CompletionLocal{}, err
	}
	_ = withdrawalRows.Close()
	controlRows, err := tx.QueryContext(ctx, `SELECT data FROM issue_controls
		WHERE repository=$1 ORDER BY issue`, repository)
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	var controls []factory.IssueControl
	for controlRows.Next() {
		var raw []byte
		var control factory.IssueControl
		if err = controlRows.Scan(&raw); err != nil {
			_ = controlRows.Close()
			return factory.CompletionLocal{}, err
		}
		if err = json.Unmarshal(raw, &control); err != nil {
			_ = controlRows.Close()
			return factory.CompletionLocal{}, err
		}
		if err = control.Validate(); err != nil {
			_ = controlRows.Close()
			return factory.CompletionLocal{}, err
		}
		control.AssessedUnix, control.FirstSeenUnix, control.NativeRev = 0, 0, 0
		controls = append(controls, control)
	}
	if err = controlRows.Err(); err != nil {
		_ = controlRows.Close()
		return factory.CompletionLocal{}, err
	}
	_ = controlRows.Close()
	authorityInput, err := factoryAuthorityInput(ctx, tx, repository)
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	preparationDoc, err := json.Marshal(struct {
		Preparations []preparationBinding `json:"preparations"`
	}{preparations})
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	scopeDoc, err := json.Marshal(struct {
		Heads       []scopeBinding         `json:"heads"`
		Withdrawals []scopeBinding         `json:"withdrawals"`
		Controls    []factory.IssueControl `json:"controls"`
	}{scope, withdrawals, controls})
	if err != nil {
		return factory.CompletionLocal{}, err
	}
	preparationSum := sha256.Sum256(preparationDoc)
	scopeSum := sha256.Sum256(scopeDoc)
	local := factory.CompletionLocal{
		SourceCommit:         requirement.SourceCommit,
		PreparationDigest:    hex.EncodeToString(preparationSum[:]),
		ScopeDigest:          hex.EncodeToString(scopeSum[:]),
		AuthorityFingerprint: factory.AuthorityFingerprint(factory.EvaluateAuthority(authorityInput)),
	}
	if err = local.Validate(); err != nil {
		return factory.CompletionLocal{}, err
	}
	if err = tx.Commit(); err != nil {
		return factory.CompletionLocal{}, err
	}
	return local, nil
}

// PendingCompletionRepositories lists every repository with at least
// one unresolved cycle, so the completion driver fences and releases
// per repository without scanning unrelated work.
func (s *Store) PendingCompletionRepositories(ctx context.Context) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT repository FROM factory_reassessments
		WHERE state NOT IN ('released','superseded') ORDER BY repository`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var repositories []int64
	for rows.Next() {
		var repository int64
		if err = rows.Scan(&repository); err != nil {
			_ = rows.Close()
			return nil, err
		}
		repositories = append(repositories, repository)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()
	return repositories, nil
}

// EnqueueReadinessRootEvent coalesces one best-effort readiness root
// outside a domain transaction: after a continue release it forces a
// fresh assessment pass instead of reusing a stale snapshot.
func (s *Store) EnqueueReadinessRootEvent(ctx context.Context, repository, issue int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = enqueueReadinessRootEventTx(ctx, tx, repository, issue); err != nil {
		return err
	}
	return tx.Commit()
}

// ReassessmentRecord reads one cycle by its ID for the completion
// driver and explicit Retry validation.
func (s *Store) ReassessmentRecord(ctx context.Context, cycleID string) (ReassessmentRecord, error) {
	var record ReassessmentRecord
	var raw []byte
	if err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_reassessments WHERE cycle=$1`, cycleID).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReassessmentRecord{}, ErrNotFound
		}
		return ReassessmentRecord{}, err
	}
	if err := json.Unmarshal(raw, &record); err != nil {
		return ReassessmentRecord{}, err
	}
	if err := record.Validate(); err != nil {
		return ReassessmentRecord{}, err
	}
	return record, nil
}

// CasReassessmentRecord advances one cycle under an expected-state
// guard: the write lands only when the row still carries the state the
// caller sealed, decided or reconciled against, so concurrent drivers
// serialize instead of overwriting. Sealed evidence is immutable: once
// an inventory digest or evidence fingerprint is recorded, later writes
// must carry it unchanged. Only the state column and data move; rows
// are never deleted.
func (s *Store) CasReassessmentRecord(ctx context.Context, cycleID, expectedState string, record ReassessmentRecord) error {
	if !factory.ValidID(cycleID) {
		return errors.New("invalid reassessment cycle identity")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	if record.Cycle.CycleID != cycleID {
		return errors.New("reassessment record does not match its cycle")
	}
	if record.Issue <= 0 {
		return errors.New("reassessment record lacks its completed issue")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var raw []byte
	var state string
	if err = tx.QueryRowContext(ctx, `SELECT state,data FROM factory_reassessments WHERE cycle=$1 FOR UPDATE`, cycleID).Scan(&state, &raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if state != expectedState {
		return ErrStaleRevision
	}
	var current ReassessmentRecord
	if err = json.Unmarshal(raw, &current); err != nil {
		return err
	}
	if current.InventoryDigest != "" && record.InventoryDigest != current.InventoryDigest {
		return errors.New("sealed reassessment inventory is immutable")
	}
	if current.EvidenceFingerprint != "" && record.EvidenceFingerprint != current.EvidenceFingerprint {
		return errors.New("sealed reassessment evidence is immutable")
	}
	if current.Local.Sealed() && record.Local != current.Local {
		return errors.New("sealed reassessment local bindings are immutable")
	}
	if current.InventoryDigest != "" {
		if record.EvidenceRevision != current.EvidenceRevision || record.EvidenceIssues != current.EvidenceIssues ||
			len(record.Evidence) != len(current.Evidence) {
			return errors.New("sealed reassessment bodies are immutable")
		}
		for i := range record.Evidence {
			sealed, err := record.Evidence[i].CanonicalData()
			held, herr := current.Evidence[i].CanonicalData()
			if err != nil || herr != nil || sealed != held {
				return errors.New("sealed reassessment bodies are immutable")
			}
		}
		if record.Completion != current.Completion {
			return errors.New("sealed reassessment completion is immutable")
		}
	}
	if record.Issue != current.Issue {
		return errors.New("reassessment completed issue is immutable")
	}
	recordData, err := json.Marshal(record)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE factory_reassessments SET state=$1,data=$2 WHERE cycle=$3 AND state=$4`,
		record.Cycle.State, string(recordData), cycleID, expectedState)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrStaleRevision
	}
	return tx.Commit()
}

// ReserveReassessmentCall charges one funded Jev request against the
// cycle ledger and the existing global consultation ceiling in a single
// transaction. The reservation lands before the provider is consulted
// and stays charged whatever happens: failed or indeterminate requests
// are never refunded and never silently repeated. A spent global budget
// or an already-reserved fingerprint refuses with reserved=false and no
// charge. Cached completed receipts bypass reservation entirely through
// the shared judgment read path, keyed by the fingerprint that already
// embeds the cycle, purpose, rubric, model and complete evidence. A
// spent global budget refuses with ErrReassessmentBudgetExhausted; an
// already-reserved fingerprint or a full ledger refuses with
// reserved=false so the caller reconciles instead of charging again.
func (s *Store) ReserveReassessmentCall(ctx context.Context, cycleID, expectedState, callID, purpose, decision, fingerprint string) (bool, error) {
	if !factory.ValidID(cycleID) || !factory.ValidID(callID) || !factory.ValidDigest(fingerprint) {
		return false, errors.New("invalid reassessment reservation")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var raw []byte
	var state string
	if err = tx.QueryRowContext(ctx, `SELECT state,data FROM factory_reassessments WHERE cycle=$1 FOR UPDATE`, cycleID).Scan(&state, &raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, err
	}
	if state != expectedState {
		return false, ErrStaleRevision
	}
	var record ReassessmentRecord
	if err = json.Unmarshal(raw, &record); err != nil {
		return false, err
	}
	if purpose != factory.ReassessmentDecisionPurpose && purpose != factory.ReassessmentProposalPurpose {
		return false, errors.New("invalid reassessment call purpose")
	}
	if len(record.Cycle.Ledger.DecisionCalls) >= factory.MaxReassessmentDecisionCalls {
		return false, nil
	}
	for _, id := range record.Cycle.Ledger.DecisionCalls {
		if id == callID {
			return false, nil
		}
	}
	ledger, err := record.Cycle.Ledger.ReserveDecisionCall(callID, purpose, decision)
	if err != nil {
		return false, err
	}
	var existing int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_priority_judgments WHERE snapshot=$1`, fingerprint).Scan(&existing); err != nil {
		return false, err
	}
	if existing > 0 {
		return false, nil
	}
	var spent int
	if err = tx.QueryRowContext(ctx, `SELECT spent FROM factory_judgment_budget WHERE id=1 FOR UPDATE`).Scan(&spent); err != nil {
		return false, err
	}
	if spent >= s.PriorityCallBudget {
		return false, ErrReassessmentBudgetExhausted
	}
	record.Cycle.Ledger = ledger
	recordData, err := json.Marshal(record)
	if err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_reassessments SET data=$1 WHERE cycle=$2 AND state=$3`, string(recordData), cycleID, expectedState); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_priority_judgments(snapshot,max_calls) VALUES($1,1)`, fingerprint); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_judgment_budget SET spent=spent+1 WHERE id=1`); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// StartReassessmentRetry creates the one successor cycle for an
// attention cycle through its explicit authorized Retry. The prior
// cycle must still carry the expected attention state with no
// successor yet; command replay reuses the recorded successor instead
// of creating another. The successor preserves all earlier receipts
// and committed native effects as history via its retry lineage. Only
// explicit authorized Retry supersedes: no poll, input edit or policy
// edit creates a successor.
func (s *Store) StartReassessmentRetry(ctx context.Context, retry factory.ReassessmentRetry) (string, error) {
	if err := retry.Validate(); err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	var raw []byte
	var state string
	if err = tx.QueryRowContext(ctx, `SELECT state,data FROM factory_reassessments WHERE cycle=$1 FOR UPDATE`, retry.ExpectedPriorCycle).Scan(&state, &raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	var prior ReassessmentRecord
	if err = json.Unmarshal(raw, &prior); err != nil {
		return "", err
	}
	if state == factory.ReassessmentSuperseded {
		if prior.Cycle.Successor == "" || prior.Cycle.Successor != retry.SuccessorCycleID {
			return "", errors.New("reassessment retry names another successor")
		}
		return prior.Cycle.Successor, nil
	}
	superseded, err := prior.Cycle.Supersede(retry)
	if err != nil {
		return "", err
	}
	prior.Cycle = superseded
	prior.WaitReason = ""
	priorData, err := json.Marshal(prior)
	if err != nil {
		return "", err
	}
	result, err := tx.ExecContext(ctx, `UPDATE factory_reassessments SET state=$1,data=$2 WHERE cycle=$3 AND state=$4`,
		factory.ReassessmentSuperseded, string(priorData), retry.ExpectedPriorCycle, factory.ReassessmentAttention)
	if err != nil {
		return "", err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if n != 1 {
		return "", ErrStaleRevision
	}
	successor := factory.ReassessmentCycle{
		EventID: retry.EventID, CycleID: retry.SuccessorCycleID, RetryOf: retry.ExpectedPriorCycle,
		State: factory.ReassessmentCollecting, Repository: prior.Cycle.Repository, Generation: prior.Cycle.Generation,
	}
	if err := successor.Validate(); err != nil {
		return "", err
	}
	successorRecord := ReassessmentRecord{Cycle: successor, Issue: prior.Issue, Completion: prior.Completion}
	successorData, err := json.Marshal(successorRecord)
	if err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data)
		VALUES($1,$2,$3,$4,$5,$6)`, successor.CycleID, successor.Repository, successor.Generation,
		successor.EventID, successor.State, string(successorData)); err != nil {
		return "", err
	}
	return successor.CycleID, tx.Commit()
}
