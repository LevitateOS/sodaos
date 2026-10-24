package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// repairObservationScopeTx is shared by scope reads and the existing immutable
// submission transaction. It neither creates a guard row nor buys any work.
func repairObservationScopeTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability) (factory.Assignment, factory.CurrentContext, error) {
	var a factory.Assignment
	var current factory.CurrentContext
	if cap.Validate() != nil || cap.Profile != factory.MCPProfileStrategist || cap.State != factory.MCPCapabilityActive {
		return a, current, errors.New("repair scope requires an active bound Strategist")
	}
	var recorded factory.MCPCapability
	var state string
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT state,data FROM factory_mcp_capabilities WHERE id=$1 FOR SHARE`, cap.ID).Scan(&state, &raw); err != nil {
		return a, current, err
	}
	if json.Unmarshal(raw, &recorded) != nil || state != factory.MCPCapabilityActive || recorded.ScopeDigest() != cap.ScopeDigest() || recorded.Expired(time.Now()) {
		return a, current, errors.New("repair scope capability is no longer active")
	}
	if err := contextJSONRow(ctx, tx, &a, `SELECT data FROM factory_assignments WHERE id=$1 AND stage='assigned' FOR SHARE`, cap.AssignmentID); err != nil {
		return a, current, err
	}
	phaseMatches := a.Subject.Completion != nil && a.Phase == project.ExecutionReassess || (a.Subject.Stall != nil || a.Subject.DeliveryRepair != nil) && a.Phase == project.ExecutionRepair
	if a.Validate() != nil || a.ID != cap.AssignmentID || a.Run != cap.RunID || a.Repository != cap.Repository || a.ProjectID != cap.Project || a.SourceCommit != cap.SourceDigest || strconv.FormatInt(a.NativeRev, 10) != cap.AcceptedRevision || a.Subject.SubjectKey(a.Repository) != cap.Subject || !phaseMatches || a.Role != project.RoleCoder || a.Issue != 0 {
		return a, current, errors.New("repair scope differs from its admitted investigator")
	}
	var locked string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM factory_runs WHERE id=$1 AND active=TRUE AND settled=FALSE FOR SHARE`, a.Run).Scan(&locked); err != nil {
		return a, current, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT root_assignment FROM factory_attempt_allowances WHERE repository=$1 AND root_assignment=$2 FOR SHARE`, a.Repository, a.AttemptRoot).Scan(&locked); err != nil {
		return a, current, err
	}
	if _, frozen, err := terminalMCPProposalTx(ctx, tx, a.ID); err != nil || frozen {
		if err == nil {
			err = ErrMCPSubmissionFrozen
		}
		return a, current, err
	}
	current, err := factoryCurrentContextTx(ctx, tx, cap, time.Now())
	if err != nil {
		return a, current, err
	}
	if current.Run == nil || !current.Run.Active || current.Run.Settled || current.Run.Reconciled || current.Allowance == nil || current.Allowance.Closed || current.Allowance.DeadlineRemainingSeconds <= 0 || !factory.ValidDigest(factory.RepairInventoryDigest(current)) {
		return a, current, errors.New("repair scope lacks an active run, allowance or complete admitted cause")
	}
	return a, current, nil
}

// ReassessmentScope reads only this assignment's attributed intermediate
// report references. Full contents remain in the existing inbox/feed owner.
func (s *Store) ReassessmentScope(ctx context.Context, cap factory.MCPCapability, after int64) (factory.RepairScopeView, error) {
	var out factory.RepairScopeView
	if after < 0 {
		return out, errors.New("invalid repair observation cursor")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	a, current, err := repairObservationScopeTx(ctx, tx, cap)
	if err != nil {
		return out, err
	}
	out = factory.RepairScopeView{Assignment: a.ID, Run: a.Run, InputDigest: cap.ScopeDigest(), Subject: cap.Subject, SourceCommit: a.SourceCommit, InventoryDigest: factory.RepairInventoryDigest(current), Records: []factory.RepairScopeRecord{}, Limit: factory.RepairObservationPageSize, MaximumPerTool: factory.MaxRepairObservations}
	var projected int
	if err := tx.QueryRowContext(ctx, `SELECT count(*),count(p.id) FROM factory_mcp_submissions s LEFT JOIN factory_feed_comments p ON p.id=s.receipt->>'feed_post_id' AND p.repository=$2 AND p.data->>'assignment'=$1 AND s.capability=$3 AND s.input_digest=$4 WHERE s.assignment=$1 AND s.tool IN ($5,$6)`, a.ID, a.Repository, cap.ID, cap.ScopeDigest(), string(factory.MCPToolRecordDiagnosis), string(factory.MCPToolDraftRepair)).Scan(&out.Total, &projected); err != nil {
		return out, err
	}
	if projected != out.Total {
		return out, errors.New("repair observation projection is incomplete")
	}
	rows, err := tx.QueryContext(ctx, `SELECT s.submission,s.tool,s.receipt->>'feed_post_id',p.seq FROM factory_mcp_submissions s JOIN factory_feed_comments p ON p.id=s.receipt->>'feed_post_id' AND p.repository=$2 AND p.data->>'assignment'=$1 WHERE s.assignment=$1 AND s.capability=$3 AND s.input_digest=$4 AND s.tool IN ($5,$6) AND p.seq>$7 ORDER BY p.seq LIMIT $8`, a.ID, a.Repository, cap.ID, cap.ScopeDigest(), string(factory.MCPToolRecordDiagnosis), string(factory.MCPToolDraftRepair), after, out.Limit+1)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var record factory.RepairScopeRecord
		if err := rows.Scan(&record.ID, &record.Tool, &record.FeedPostID, &record.Sequence); err != nil {
			rows.Close()
			return out, err
		}
		if len(out.Records) == out.Limit {
			out.NextAfter = out.Records[len(out.Records)-1].Sequence
			break
		}
		out.Records = append(out.Records, record)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	out.Complete = out.NextAfter == 0
	return out, tx.Commit()
}

// ReadRepairObservation recovers only this active investigator's immutable
// diagnosis or draft. The submission and its attributed feed post must still
// agree with the current admitted cause; a reference grants no broader read.
func (s *Store) ReadRepairObservation(ctx context.Context, cap factory.MCPCapability, id string, offset int) (factory.RepairObservationPage, error) {
	var out factory.RepairObservationPage
	if !factory.ValidID(id) || offset < 0 {
		return out, errors.New("invalid repair observation reference or offset")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	a, _, err := repairObservationScopeTx(ctx, tx, cap)
	if err != nil {
		return out, err
	}
	var capability string
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT s.tool,s.capability,s.input_digest,s.payload_hash,s.payload FROM factory_mcp_submissions s JOIN factory_feed_comments p ON p.id=s.receipt->>'feed_post_id' AND p.repository=$2 AND p.data->>'assignment'=$1 WHERE s.assignment=$1 AND s.submission=$3 AND s.tool IN ($4,$5) FOR SHARE OF s,p`, a.ID, a.Repository, id, string(factory.MCPToolRecordDiagnosis), string(factory.MCPToolDraftRepair)).Scan(&out.Tool, &capability, &out.InputDigest, &out.PayloadHash, &raw); err != nil {
		return out, err
	}
	if capability != cap.ID || out.InputDigest != cap.ScopeDigest() || len(raw) == 0 || len(raw) > factory.MCPMaxSubmissionPayload || !utf8.Valid(raw) || !json.Valid(raw) || factory.MCPInputDigest(raw) != out.PayloadHash {
		return factory.RepairObservationPage{}, errors.New("repair observation differs from its canonical admitted report")
	}
	if offset >= len(raw) || !utf8.RuneStart(raw[offset]) {
		return factory.RepairObservationPage{}, errors.New("repair observation offset is outside its UTF-8 content")
	}
	end := min(len(raw), offset+factory.RepairObservationFragmentBytes)
	for end < len(raw) && !utf8.RuneStart(raw[end]) {
		end--
	}
	out.ID, out.Offset, out.TotalBytes, out.Encoding = id, offset, len(raw), "utf8-json-fragment"
	out.Content, out.Complete = string(raw[offset:end]), end == len(raw)
	if !out.Complete {
		out.NextOffset = end
	}
	return out, tx.Commit()
}
