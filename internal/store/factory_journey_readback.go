package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// journeyReassignmentLimit bounds one per-issue reassessment listing; the
// caller reads one extra record past its render cap as the truncation
// signal, so a bounded read never reports a cut chain as whole.
const journeyReassignmentLimit = 256

// journeyPhaseGroupLimit bounds one selected-root latest-per-phase read:
// one newest row per distinct recorded phase value plus one extra group
// as the truncation signal. Admitted execution phases are a small frozen
// vocabulary, so a cut here only ever covers pathological junk phase
// values; the caller still names the cut instead of passing a partial
// group set as the whole lineage.
const journeyPhaseGroupLimit = 32

// PhaseAttempt carries one phase group's newest recorded assignment
// behind the selected owner AttemptRoot, with its stable seq order.
// The read layer classifies the recorded phase value (canonical,
// pre-phase empty, or unknown); the store only scopes and orders.
type PhaseAttempt struct {
	Assignment factory.Assignment
	Seq        int64
}

// LatestPhaseAttempts returns the newest recorded assignment per
// distinct recorded phase value for one repository issue under the
// selected owner AttemptRoot and acceptance, newest first per group by
// stable seq order. One current state per phase means an old capped
// prefix can never stand in for a newer failed or current row, and a
// sibling AttemptRoot sharing the acceptance cannot leak in. The
// truncated flag reports whether more phase groups name the scope than
// the limit carries, so the caller marks its read incomplete instead
// of rendering a cut group set as the whole lineage. Rows that decode
// or scope-check refuse as storage errors: a corrupt recorded row is
// an explicit failure, never a skipped candidate.
func (s *Store) LatestPhaseAttempts(ctx context.Context, repository, issue int64, acceptance, attemptRoot string) ([]PhaseAttempt, bool, error) {
	if repository <= 0 || issue <= 0 || acceptance == "" || attemptRoot == "" {
		return nil, false, errors.New("invalid latest phase scope")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT ON ((data->>'phase')) seq, data FROM factory_assignments
		WHERE repository=$1 AND issue=$2 AND data->>'acceptance'=$3 AND data->>'attempt_root'=$4
		ORDER BY (data->>'phase'), seq DESC LIMIT $5`,
		repository, issue, acceptance, attemptRoot, journeyPhaseGroupLimit+1)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = rows.Close() }()
	groups := make([]PhaseAttempt, 0, journeyPhaseGroupLimit+1)
	for rows.Next() {
		var group PhaseAttempt
		var raw []byte
		if err = rows.Scan(&group.Seq, &raw); err != nil {
			return nil, false, err
		}
		if err = json.Unmarshal(raw, &group.Assignment); err != nil {
			return nil, false, err
		}
		a := group.Assignment
		if a.Repository != repository || a.Issue != issue || a.Acceptance != acceptance || a.AttemptRoot != attemptRoot {
			return nil, false, errors.New("latest phase scope mismatch")
		}
		groups = append(groups, group)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	if len(groups) > journeyPhaseGroupLimit {
		return groups[:journeyPhaseGroupLimit], true, nil
	}
	return groups, false, nil
}

// JourneyReassignmentCycles returns up to limit unresolved reassessment
// cycles naming one repository issue in stable generation order, oldest
// first. Released and superseded cycles are history, not work. The
// truncated flag reports whether more cycles name the issue than the
// limit carries, so the caller marks its read incomplete instead of
// rendering a cut set as the whole lineage. Rows that decode or validate
// refuse as storage errors: a corrupt recorded row is an explicit
// failure, never a skipped candidate.
func (s *Store) JourneyReassignmentCycles(ctx context.Context, repository, issue int64, limit int) ([]ReassessmentRecord, bool, error) {
	if repository <= 0 || issue <= 0 || limit <= 0 || limit > journeyReassignmentLimit {
		return nil, false, errors.New("invalid journey reassessment listing")
	}
	// The issue lives inside the recorded JSON (factory_reassessments
	// carries no issue column): the digit guard keeps unparseable rows
	// out of the cast without failing the scoped read, and decoded
	// records re-assert their scope below.
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_reassessments
		WHERE repository=$1 AND state NOT IN ('released','superseded')
		AND data->>'issue' ~ '^[0-9]+$' AND (data->>'issue')::bigint=$2
		ORDER BY generation LIMIT $3`, repository, issue, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = rows.Close() }()
	records := make([]ReassessmentRecord, 0, limit+1)
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return nil, false, err
		}
		var record ReassessmentRecord
		if err = json.Unmarshal(raw, &record); err != nil {
			return nil, false, err
		}
		if err = record.Validate(); err != nil {
			return nil, false, err
		}
		if record.Issue != issue {
			return nil, false, errors.New("journey reassessment scope mismatch")
		}
		records = append(records, record)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	if len(records) > limit {
		return records[:limit], true, nil
	}
	return records, false, nil
}

// LatestIssueAttempt returns the newest recorded assignment for one
// repository issue, whatever its ownership. The newest row may be a
// child correction or review attempt: callers keep its AttemptRoot as
// the lineage root instead of implicitly re-rooting the journey on the
// child.
func (s *Store) LatestIssueAttempt(ctx context.Context, repository, issue int64) (factory.Assignment, error) {
	var a factory.Assignment
	if repository <= 0 || issue <= 0 {
		return a, errors.New("invalid latest attempt scope")
	}
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_assignments
		WHERE repository=$1 AND issue=$2 ORDER BY seq DESC LIMIT 1`, repository, issue).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &a)
	}
	return a, err
}
