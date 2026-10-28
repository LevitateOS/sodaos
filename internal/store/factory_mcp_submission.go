package store

// D14 immutable-submission persistence and receipt recovery (FW38).
//
// factory_mcp_submissions records one row per tool call. Unique
// (assignment, submission) is the idempotency key: exact replay
// returns the existing receipt, changed bytes under that ID
// conflict. One terminal proposal per assignment freezes new
// identities; the same terminal content under another ID returns the
// existing receipt without another record. A lost response recovers
// through MCPReceipt, never a newly identified action. Refused
// malformed/unauthorized calls record no row. Schema registration
// rides the coordinator-serialized proposal
// (see evidence/fw38/registration-proposal.md); this file owns the
// DDL constants and the transaction methods.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

const (
	// TableFactoryMCPSubmissions is the submission/receipt owner.
	TableFactoryMCPSubmissions = "factory_mcp_submissions"
)

// FactoryMCPSubmissionsDDL creates the submission table. data carries
// the factory.MCPSubmission payload bytes beside the compared
// digests; receipt carries the durable factory.MCPReceipt. capability
// durably binds the authorizing capability (author/run, role profile
// and subject live on the capability row), so the record names its
// author without assuming one capability per assignment.
const FactoryMCPSubmissionsDDL = `CREATE TABLE factory_mcp_submissions(
assignment TEXT NOT NULL CHECK(assignment ~ '^[a-f0-9]{32}$'),
submission TEXT NOT NULL CHECK(submission ~ '^[a-f0-9]{32}$'),
capability TEXT NOT NULL CHECK(capability ~ '^[a-f0-9]{32}$'),
tool TEXT NOT NULL,
input_digest TEXT NOT NULL CHECK(input_digest ~ '^[a-f0-9]{64}$'),
payload_hash TEXT NOT NULL CHECK(payload_hash ~ '^[a-f0-9]{64}$'),
payload BYTEA NOT NULL CHECK(octet_length(payload)>0 AND octet_length(payload)<=65536),
state TEXT NOT NULL, reason TEXT NOT NULL DEFAULT '',
receipt JSONB NOT NULL,
PRIMARY KEY(assignment,submission))`

// FactoryMCPSubmissionTriggers freezes recorded history: rows never
// change except through the receipt-state CAS FW40 consumers use,
// and rows are never deleted.
const FactoryMCPSubmissionTriggers = `CREATE TRIGGER factory_mcp_submission_no_delete BEFORE DELETE ON factory_mcp_submissions FOR EACH ROW EXECUTE FUNCTION soda_reject_immutable()`

// FactoryMCPAssignmentGuardDDL is the schema-registration proposal
// for the per-assignment serialization guard (coordinator-owned;
// see the fw38-f2f3 handoff). One row per assignment will live here;
// RecordMCPSubmission locks the assignment's guard row inside its
// transaction. This statement NEVER executes at request time: the
// table arrives through normal schema startup, and tests apply this
// constant as fixture setup until registration lands.
const FactoryMCPAssignmentGuardDDL = `CREATE TABLE factory_mcp_submission_guards(assignment TEXT NOT NULL CHECK(assignment ~ '^[a-f0-9]{32}$'), PRIMARY KEY(assignment))`

var (
	// ErrMCPSubmissionConflict reports changed bytes under a recorded
	// submission ID: same identity, different tool, input digest or
	// payload hash.
	ErrMCPSubmissionConflict = errors.New("factory submission conflicts with the recorded bytes: this submission_id already recorded a different call; mint a new submission_id for new content, or resend the same bytes to replay")
	// ErrMCPSubmissionFrozen reports a new submission identity after
	// the assignment recorded a terminal proposal.
	ErrMCPSubmissionFrozen = errors.New("assignment already recorded a terminal proposal: do not finish again under a new submission_id; read the result with factory.receipt and the first submission_id")
	// ErrMCPSubmissionHandoffCap reports a handoff past the 256-edge
	// bound: terminal receipt capacity stays reserved regardless of
	// task traffic.
	ErrMCPSubmissionHandoffCap = errors.New("assignment handoff bound reached (256 task handoffs): finish the assignment with its finish tool")
	// ErrMCPSubmissionCapabilityInactive reports a submission whose
	// bound capability is not active for its assignment at insert
	// time: unknown, staged, revoked or retired capabilities, an
	// assignment mismatch, or inputs that no longer match the
	// recorded scope. A revocation racing an in-flight insert lands
	// here once it commits first.
	ErrMCPSubmissionCapabilityInactive = errors.New("factory submission capability is not active for this assignment")
	// ErrMCPSubmissionCapabilityExpired reports a submission inserted
	// past its bound capability's deadline. Admission checks the
	// deadline once; queueing after admission can cross it, so the
	// insert transaction rechecks the deadline against insertion
	// time. Recorded receipt reads stay on the separate MCPReceipt
	// route: this refusal records nothing and grants nothing.
	ErrMCPSubmissionCapabilityExpired = errors.New("factory submission capability is expired at insert")
	// ErrMCPSubmissionAssignmentFinished reports a new submission
	// against a finished assignment: a late answer requires fresh
	// authorized admission, never revival of the finished attempt's
	// capability. Exact replay of a recorded submission still returns
	// its receipt above, so lost replies keep recovering; only new
	// identities refuse here.
	ErrMCPSubmissionAssignmentFinished = errors.New("assignment already finished")
)

// RecordMCPSubmission persists one admitted submission with its
// recorded receipt. Exact replay returns the existing receipt;
// changed bytes under the same ID conflict; a new identity after a
// recorded terminal refuses, except the same terminal content under
// another ID, which returns the existing terminal receipt without
// another record.
//
// The transaction locks in a fixed order shared with the
// capability state machine: first the bound capability row (the
// same row lock revoke/retire takes), then the assignment guard
// row, then the submission rows. Against those locks it rechecks
// the authentic capability (active, same assignment, current
// scope), the exact ID, the terminal freeze and the handoff cap,
// so a revocation committing ahead of an in-flight insert refuses
// that insert instead of racing it. The deadline rechecks last, at
// the new-record admission point, so queueing on any of those locks
// past expiry still refuses. The active check stays before the
// exact-ID replay: replay of new authority is an insert, and
// recorded history is recovered through the separate MCPReceipt
// read, never through this path.
func (s *Store) RecordMCPSubmission(ctx context.Context, submission factory.MCPSubmission, receipt factory.MCPReceipt) (factory.MCPReceipt, error) {
	raw, err := admitMCPSubmission(submission, receipt)
	if err != nil {
		return factory.MCPReceipt{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.MCPReceipt{}, err
	}
	defer func() { _ = tx.Rollback() }()
	capability, err := lockMCPCapabilityTx(ctx, tx, submission)
	if err != nil {
		return factory.MCPReceipt{}, err
	}
	if err = lockMCPSubmissionGuardTx(ctx, tx, submission.AssignmentID); err != nil {
		return factory.MCPReceipt{}, err
	}
	if recorded, done, err := replayMCPSubmissionTx(ctx, tx, submission); done || err != nil {
		return recorded, err
	}
	if terminal, done, err := freezeTerminalMCPSubmissionTx(ctx, tx, submission); done || err != nil {
		return terminal, err
	}
	if err = checkMCPHandoffCapTx(ctx, tx, submission); err != nil {
		return factory.MCPReceipt{}, err
	}
	// A finished assignment admits no new submission identities: the
	// attempt is over and a late answer needs fresh authorized
	// admission. Assignments without a recorded row predate the
	// dispatch packet and keep their current behavior; replay above
	// already returned recorded receipts before this gate.
	if finished, err := assignmentFinishedTx(ctx, tx, submission.AssignmentID); err != nil {
		return factory.MCPReceipt{}, err
	} else if finished {
		return factory.MCPReceipt{}, ErrMCPSubmissionAssignmentFinished
	}
	// Deadline binds at the new-record admission point, after every
	// potentially blocking lock (capability row, assignment guard,
	// submission row) and after the replay/freeze/cap decisions: a
	// request that locks the capability while live but queues on the
	// guard past the deadline still refuses here instead of
	// inserting. The clock reads last, so it names insertion time.
	// Replay and same-terminal-content returns above record nothing
	// and grant no new authority, so they never reach this gate.
	if capability.Expired(time.Now()) {
		return factory.MCPReceipt{}, ErrMCPSubmissionCapabilityExpired
	}
	if err = recordMCPArtifactTx(ctx, tx, capability, submission, &receipt); err != nil {
		return factory.MCPReceipt{}, err
	}
	if err = recordMCPLibraryTx(ctx, tx, capability, submission, &receipt); err != nil {
		return factory.MCPReceipt{}, err
	}
	if err = recordMCPResearchTx(ctx, tx, capability, submission, &receipt); err != nil {
		return factory.MCPReceipt{}, err
	}
	if err = recordMCPVerificationJourneyTx(ctx, tx, capability, submission, &receipt); err != nil {
		return factory.MCPReceipt{}, err
	}
	if err = recordMCPRepairDiagnosisTx(ctx, tx, capability, submission, &receipt); err != nil {
		return factory.MCPReceipt{}, err
	}
	if err = recordMCPRepairDraftTx(ctx, tx, capability, submission, &receipt); err != nil {
		return factory.MCPReceipt{}, err
	}
	if err = recordMCPVerificationJourneyReferencesTx(ctx, tx, capability, submission); err != nil {
		return factory.MCPReceipt{}, err
	}
	if err = recordMCPPostResolutionTx(ctx, tx, capability, submission, &receipt); err != nil {
		return factory.MCPReceipt{}, err
	}
	if err = recordMCPCommunicationTx(ctx, tx, capability, submission, &receipt); err != nil {
		return factory.MCPReceipt{}, err
	}
	raw, err = json.Marshal(receipt)
	if err != nil {
		return factory.MCPReceipt{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_mcp_submissions(assignment,submission,capability,tool,input_digest,payload_hash,payload,state,reason,receipt) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		submission.AssignmentID, submission.SubmissionID, submission.CapabilityID, string(submission.Tool),
		submission.ExpectedInputDigest, submission.PayloadHash, submission.Payload,
		receipt.State, receipt.Reason, string(raw)); err != nil {
		return factory.MCPReceipt{}, err
	}
	return receipt, tx.Commit()
}

// admitMCPSubmission validates one submission against its receipt,
// returning the encoded receipt.
func admitMCPSubmission(submission factory.MCPSubmission, receipt factory.MCPReceipt) ([]byte, error) {
	if err := submission.Validate(); err != nil {
		return nil, err
	}
	if err := receipt.Validate(); err != nil {
		return nil, err
	}
	if receipt.SubmissionID != submission.SubmissionID || receipt.Tool != submission.Tool ||
		receipt.InputDigest != submission.PayloadHash {
		return nil, errors.New("receipt does not match its submission")
	}
	return json.Marshal(receipt)
}

// lockMCPCapabilityTx locks the authentic capability behind one
// submission under the same row lock the revoke/retire CAS takes:
// only an active capability bound to this assignment and current
// scope admits the insert. Admission checked the same facts earlier;
// this recheck closes the revoke-vs-insert race inside the
// transaction.
func lockMCPCapabilityTx(ctx context.Context, tx *sql.Tx, submission factory.MCPSubmission) (factory.MCPCapability, error) {
	var capability factory.MCPCapability
	var capState string
	var capRaw []byte
	if err := tx.QueryRowContext(ctx, `SELECT state,data FROM factory_mcp_capabilities WHERE id=$1 FOR UPDATE`,
		submission.CapabilityID).Scan(&capState, &capRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return capability, ErrMCPSubmissionCapabilityInactive
		}
		return capability, err
	}
	if err := json.Unmarshal(capRaw, &capability); err != nil {
		return capability, err
	}
	capability.State = capState
	var observed struct {
		Live bool `json:"live_observed"`
	}
	if err := json.Unmarshal(capRaw, &observed); err != nil {
		return capability, err
	}
	if observed.Live {
		if err := checkMCPWorkActiveTx(ctx, tx, capability); err != nil {
			return capability, err
		}
	}
	if err := capability.Validate(); err != nil {
		return capability, ErrMCPSubmissionCapabilityInactive
	}
	if capability.State != factory.MCPCapabilityActive ||
		capability.AssignmentID != submission.AssignmentID ||
		capability.ScopeDigest() != submission.ExpectedInputDigest {
		return capability, ErrMCPSubmissionCapabilityInactive
	}
	return capability, nil
}

// lockMCPSubmissionGuardTx serializes one assignment's submitters:
// the guard row exists once per assignment, so concurrent submitters
// for one assignment queue on this row lock instead of racing an
// unlocked terminal scan past each other. Assignments never block
// each other, and capability rows always lock before guard rows.
// assignmentFinishedTx reports whether the assignment row exists and
// already finished. A missing row predates the dispatch packet and
// keeps its current behavior; only a recorded finish refuses.
func assignmentFinishedTx(ctx context.Context, tx *sql.Tx, assignmentID string) (bool, error) {
	var stage string
	err := tx.QueryRowContext(ctx, `SELECT stage FROM factory_assignments WHERE id=$1`, assignmentID).Scan(&stage)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return stage != "" && stage != string(factory.AssignmentAssigned), nil
}

func lockMCPSubmissionGuardTx(ctx context.Context, tx *sql.Tx, assignmentID string) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO factory_mcp_submission_guards(assignment) VALUES($1) ON CONFLICT DO NOTHING`,
		assignmentID); err != nil {
		return err
	}
	var guard string
	return tx.QueryRowContext(ctx, `SELECT assignment FROM factory_mcp_submission_guards WHERE assignment=$1 FOR UPDATE`,
		assignmentID).Scan(&guard)
}

// replayMCPSubmissionTx replays the recorded receipt behind an exact
// identity replay, refusing conflicting rewrites.
func replayMCPSubmissionTx(ctx context.Context, tx *sql.Tx, submission factory.MCPSubmission) (factory.MCPReceipt, bool, error) {
	var currentTool, currentInput, currentHash, currentState string
	var currentReceipt []byte
	err := tx.QueryRowContext(ctx, `SELECT tool,input_digest,payload_hash,state,receipt FROM factory_mcp_submissions WHERE assignment=$1 AND submission=$2 FOR UPDATE`,
		submission.AssignmentID, submission.SubmissionID).Scan(&currentTool, &currentInput, &currentHash, &currentState, &currentReceipt)
	if err == nil {
		if currentTool != string(submission.Tool) || currentInput != submission.ExpectedInputDigest || currentHash != submission.PayloadHash {
			return factory.MCPReceipt{}, true, ErrMCPSubmissionConflict
		}
		var recorded factory.MCPReceipt
		if err = json.Unmarshal(currentReceipt, &recorded); err != nil {
			return factory.MCPReceipt{}, true, err
		}
		return recorded, true, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return factory.MCPReceipt{}, true, err
	}
	return factory.MCPReceipt{}, false, nil
}

// freezeTerminalMCPSubmissionTx freezes new identities once the
// assignment records a terminal proposal: only exact replay of a
// recorded ID (handled above) and the same terminal content under
// another ID record nothing new. Admission refuses these calls
// first; persistence refuses them anyway so races cannot slip a
// second proposal past the freeze.
func freezeTerminalMCPSubmissionTx(ctx context.Context, tx *sql.Tx, submission factory.MCPSubmission) (factory.MCPReceipt, bool, error) {
	terminal, frozen, err := terminalMCPProposalTx(ctx, tx, submission.AssignmentID)
	if err != nil {
		return factory.MCPReceipt{}, true, err
	}
	if !frozen {
		return factory.MCPReceipt{}, false, nil
	}
	if terminal.Tool == submission.Tool && terminal.InputDigest == submission.PayloadHash {
		return terminal, true, tx.Commit()
	}
	return factory.MCPReceipt{}, true, ErrMCPSubmissionFrozen
}

// checkMCPHandoffCapTx binds the handoff cap at insert under the
// same guard, so a burst of concurrent handoffs cannot overshoot 256
// before any of them commits.
func checkMCPHandoffCapTx(ctx context.Context, tx *sql.Tx, submission factory.MCPSubmission) error {
	if submission.Tool != factory.MCPToolRequestHandoff {
		return nil
	}
	var handoffs int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM factory_mcp_submissions WHERE assignment=$1 AND tool=$2`,
		submission.AssignmentID, string(factory.MCPToolRequestHandoff)).Scan(&handoffs); err != nil {
		return err
	}
	if handoffs >= factory.MCPMaxHandoffs {
		return ErrMCPSubmissionHandoffCap
	}
	return nil
}

// MCPReceipt recovers the durable receipt for one submission: the
// lost-response path. Unknown IDs report not-found without
// disclosing other assignments.
func (s *Store) MCPReceipt(ctx context.Context, assignmentID, submissionID string) (factory.MCPReceipt, error) {
	var raw []byte
	if err := s.db.QueryRowContext(ctx, `SELECT receipt FROM factory_mcp_submissions WHERE assignment=$1 AND submission=$2`,
		assignmentID, submissionID).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return factory.MCPReceipt{}, ErrNotFound
		}
		return factory.MCPReceipt{}, err
	}
	var receipt factory.MCPReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return factory.MCPReceipt{}, err
	}
	return receipt, nil
}

// TerminalMCPProposal returns the assignment's recorded terminal
// receipt, or false when none is recorded yet. Post-retirement
// receipt recovery uses this owner read, never a reactivated run.
func (s *Store) TerminalMCPProposal(ctx context.Context, assignmentID string) (factory.MCPReceipt, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return factory.MCPReceipt{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	return terminalMCPProposalTx(ctx, tx, assignmentID)
}

func terminalMCPProposalTx(ctx context.Context, tx *sql.Tx, assignmentID string) (factory.MCPReceipt, bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT tool,receipt FROM factory_mcp_submissions WHERE assignment=$1`, assignmentID)
	if err != nil {
		return factory.MCPReceipt{}, false, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var tool string
		var raw []byte
		if err = rows.Scan(&tool, &raw); err != nil {
			return factory.MCPReceipt{}, false, err
		}
		if !factory.MCPIsTerminal(factory.MCPTool(tool)) {
			continue
		}
		var receipt factory.MCPReceipt
		if err = json.Unmarshal(raw, &receipt); err != nil {
			return factory.MCPReceipt{}, false, err
		}
		return receipt, true, rows.Close()
	}
	if err = rows.Err(); err != nil {
		return factory.MCPReceipt{}, false, err
	}
	return factory.MCPReceipt{}, false, nil
}

// TerminalMCPSubmission returns the assignment's recorded terminal
// submission with its immutable payload bytes, or false when none is
// recorded yet. FW40 settlement consumers read the typed worker
// report through this owner; the bytes stay immutable and receipt
// state still moves only through AdvanceMCPReceipt.
func (s *Store) TerminalMCPSubmission(ctx context.Context, assignmentID string) (factory.MCPSubmission, bool, error) {
	var submission, capabilityID, tool, inputDigest, payloadHash string
	var payload []byte
	rows, err := s.db.QueryContext(ctx, `SELECT submission,capability,tool,input_digest,payload_hash,payload FROM factory_mcp_submissions WHERE assignment=$1`, assignmentID)
	if err != nil {
		return factory.MCPSubmission{}, false, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var rowSubmission, rowCapability, rowTool, rowInput, rowHash string
		var rowPayload []byte
		if err = rows.Scan(&rowSubmission, &rowCapability, &rowTool, &rowInput, &rowHash, &rowPayload); err != nil {
			return factory.MCPSubmission{}, false, err
		}
		if !factory.MCPIsTerminal(factory.MCPTool(rowTool)) {
			continue
		}
		submission, capabilityID, tool, inputDigest, payloadHash, payload = rowSubmission, rowCapability, rowTool, rowInput, rowHash, rowPayload
		break
	}
	if err = rows.Err(); err != nil {
		return factory.MCPSubmission{}, false, err
	}
	if submission == "" {
		return factory.MCPSubmission{}, false, nil
	}
	// The row durably binds the authorizing capability alongside the
	// identities, the tool and the payload binding the settler
	// consumes.
	if !factory.ValidID(submission) || !factory.ValidID(capabilityID) || !factory.MCPToolKnown(factory.MCPTool(tool)) ||
		!factory.ValidDigest(inputDigest) || !factory.ValidDigest(payloadHash) ||
		len(payload) == 0 || factory.MCPInputDigest(payload) != payloadHash {
		return factory.MCPSubmission{}, false, errors.New("recorded terminal submission fails its payload binding")
	}
	out := factory.MCPSubmission{
		AssignmentID: assignmentID, SubmissionID: submission, CapabilityID: capabilityID,
		Tool: factory.MCPTool(tool), ExpectedInputDigest: inputDigest,
		PayloadHash: payloadHash, Payload: payload,
	}
	return out, true, rows.Close()
}

// MCPHandoffCount counts recorded handoff submissions for one
// assignment so admission reserves terminal capacity.
func (s *Store) MCPHandoffCount(ctx context.Context, assignmentID string) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM factory_mcp_submissions WHERE assignment=$1 AND tool=$2`,
		assignmentID, string(factory.MCPToolRequestHandoff)).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// MCPSubmissionCount counts recorded submissions for one assignment
// for capability-bound status reads.
func (s *Store) MCPSubmissionCount(ctx context.Context, assignmentID string) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM factory_mcp_submissions WHERE assignment=$1`, assignmentID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// AdvanceMCPReceipt moves one receipt under an expected-state guard
// for FW40 consumers (evidence verification, acceptance): recorded
// advances, never rewrites history. The submission bytes stay
// immutable; only state, reason and the receipt envelope move.
func (s *Store) AdvanceMCPReceipt(ctx context.Context, assignmentID, submissionID, expectedState string, receipt factory.MCPReceipt) error {
	if err := receipt.Validate(); err != nil {
		return err
	}
	if receipt.SubmissionID != submissionID {
		return errors.New("receipt does not match its submission")
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var state string
	var storedRaw []byte
	if err = tx.QueryRowContext(ctx, `SELECT state,receipt FROM factory_mcp_submissions WHERE assignment=$1 AND submission=$2 FOR UPDATE`,
		assignmentID, submissionID).Scan(&state, &storedRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if state != expectedState {
		return ErrStaleRevision
	}
	var stored factory.MCPReceipt
	if err = json.Unmarshal(storedRaw, &stored); err != nil {
		return err
	}
	receipt.FeedPostID, receipt.LibraryItemID = stored.FeedPostID, stored.LibraryItemID
	receipt.ArtifactID = stored.ArtifactID
	raw, err = json.Marshal(receipt)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_mcp_submissions SET state=$3,reason=$4,receipt=$5 WHERE assignment=$1 AND submission=$2`,
		assignmentID, submissionID, receipt.State, receipt.Reason, string(raw)); err != nil {
		return err
	}
	return tx.Commit()
}
