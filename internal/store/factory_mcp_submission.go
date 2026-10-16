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

	"github.com/levitateos/sodaos/internal/factory"
)

const (
	// TableFactoryMCPSubmissions is the submission/receipt owner.
	TableFactoryMCPSubmissions = "factory_mcp_submissions"
)

// FactoryMCPSubmissionsDDL creates the submission table. data carries
// the factory.MCPSubmission payload bytes beside the compared
// digests; receipt carries the durable factory.MCPReceipt.
const FactoryMCPSubmissionsDDL = `CREATE TABLE factory_mcp_submissions(
assignment TEXT NOT NULL CHECK(assignment ~ '^[a-f0-9]{32}$'),
submission TEXT NOT NULL CHECK(submission ~ '^[a-f0-9]{32}$'),
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

var (
	// ErrMCPSubmissionConflict reports changed bytes under a recorded
	// submission ID: same identity, different tool, input digest or
	// payload hash.
	ErrMCPSubmissionConflict = errors.New("factory submission conflicts with the recorded bytes")
	// ErrMCPSubmissionFrozen reports a new submission identity after
	// the assignment recorded a terminal proposal.
	ErrMCPSubmissionFrozen = errors.New("assignment already recorded a terminal proposal")
)

// RecordMCPSubmission persists one admitted submission with its
// recorded receipt. Exact replay returns the existing receipt;
// changed bytes under the same ID conflict; a new identity after a
// recorded terminal refuses, except the same terminal content under
// another ID, which returns the existing terminal receipt without
// another record.
func (s *Store) RecordMCPSubmission(ctx context.Context, submission factory.MCPSubmission, receipt factory.MCPReceipt) (factory.MCPReceipt, error) {
	if err := submission.Validate(); err != nil {
		return factory.MCPReceipt{}, err
	}
	if err := receipt.Validate(); err != nil {
		return factory.MCPReceipt{}, err
	}
	if receipt.SubmissionID != submission.SubmissionID || receipt.Tool != submission.Tool ||
		receipt.InputDigest != submission.PayloadHash {
		return factory.MCPReceipt{}, errors.New("receipt does not match its submission")
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return factory.MCPReceipt{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.MCPReceipt{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var currentTool, currentInput, currentHash, currentState string
	var currentReceipt []byte
	err = tx.QueryRowContext(ctx, `SELECT tool,input_digest,payload_hash,state,receipt FROM factory_mcp_submissions WHERE assignment=$1 AND submission=$2 FOR UPDATE`,
		submission.AssignmentID, submission.SubmissionID).Scan(&currentTool, &currentInput, &currentHash, &currentState, &currentReceipt)
	if err == nil {
		if currentTool != string(submission.Tool) || currentInput != submission.ExpectedInputDigest || currentHash != submission.PayloadHash {
			return factory.MCPReceipt{}, ErrMCPSubmissionConflict
		}
		var recorded factory.MCPReceipt
		if err = json.Unmarshal(currentReceipt, &recorded); err != nil {
			return factory.MCPReceipt{}, err
		}
		return recorded, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return factory.MCPReceipt{}, err
	}
	// The terminal atomically freezes new identities: once the
	// assignment records a terminal proposal, only exact replay of a
	// recorded ID (handled above) and the same terminal content
	// under another ID record nothing new. Admission refuses these
	// calls first; persistence refuses them anyway so races cannot
	// slip a second proposal past the freeze.
	terminal, frozen, err := terminalMCPProposalTx(ctx, tx, submission.AssignmentID)
	if err != nil {
		return factory.MCPReceipt{}, err
	}
	if frozen {
		if terminal.Tool == submission.Tool && terminal.InputDigest == submission.PayloadHash {
			return terminal, tx.Commit()
		}
		return factory.MCPReceipt{}, ErrMCPSubmissionFrozen
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_mcp_submissions(assignment,submission,tool,input_digest,payload_hash,payload,state,reason,receipt) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		submission.AssignmentID, submission.SubmissionID, string(submission.Tool),
		submission.ExpectedInputDigest, submission.PayloadHash, submission.Payload,
		receipt.State, receipt.Reason, string(raw)); err != nil {
		return factory.MCPReceipt{}, err
	}
	return receipt, tx.Commit()
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
	if err = tx.QueryRowContext(ctx, `SELECT state FROM factory_mcp_submissions WHERE assignment=$1 AND submission=$2 FOR UPDATE`,
		assignmentID, submissionID).Scan(&state); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if state != expectedState {
		return ErrStaleRevision
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_mcp_submissions SET state=$3,reason=$4,receipt=$5 WHERE assignment=$1 AND submission=$2`,
		assignmentID, submissionID, receipt.State, receipt.Reason, string(raw)); err != nil {
		return err
	}
	return tx.Commit()
}
