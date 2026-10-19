package store

// Recorded-terminal enumeration for MCP settlement (Lane A2 drive).
//
// The coordinator settler works one assignment at a time through
// TerminalMCPSubmission/TerminalMCPProposal; this reader supplies
// the bounded work list: assignments whose terminal proposal receipt
// still sits at recorded. Finished assignments remain listed until
// their receipt advances, so a crash between FinishAssignment and
// the receipt CAS recovers on a later pass instead of stranding the
// proposal. Parked FW40 proposals stay listed and visible; the
// settler decides per assignment whether it can prove settlement.

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// RecordedTerminalMCPAssignments lists distinct assignments carrying
// a recorded terminal MCP proposal, in assignment order, bounded by
// limit. Non-terminal traffic and consumed receipts never list.
func (s *Store) RecordedTerminalMCPAssignments(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 || limit > storeAssignedLimit {
		return nil, errors.New("invalid recorded terminal listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT assignment,tool FROM factory_mcp_submissions WHERE state=$1 ORDER BY assignment,tool`, factory.MCPReceiptRecorded)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	seen := ""
	for rows.Next() {
		var assignment, tool string
		if err = rows.Scan(&assignment, &tool); err != nil {
			return nil, err
		}
		if assignment == seen || !factory.MCPIsTerminal(factory.MCPTool(tool)) {
			continue
		}
		seen = assignment
		out = append(out, assignment)
		if len(out) >= limit {
			break
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
