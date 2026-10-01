package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
)

// storeMergeLimit bounds one outstanding-merge listing.
const storeMergeLimit = 1024

// storeMergeableLimit bounds one mergeable-publication listing.
const storeMergeableLimit = 256

// ErrMergeConflict rejects replacement of a recorded merge intent.
var ErrMergeConflict = errors.New("merge immutable input changed")

// RecordMerge stores one publication's fresh open merge. One publication
// merges at most once: a second record for the publication refuses
// instead of merging twice.
func (s *Store) RecordMerge(ctx context.Context, m factory.Merge) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if m.Stage != factory.MergeOpen || m.Outcome != "" || m.Reason != "" || m.FinishedUnix != 0 || m.Revision != 0 || m.Operation.Attempts != 0 || m.WithdrawRequested {
		return errors.New("merge packet carries fresh open work")
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_merges(publication,repository,issue,pr,stage,revision,data) VALUES(?,?,?,?,?,?,?)`,
		m.PublicationID, m.Repository, m.Issue, m.PRNumber, m.Stage, m.Revision, string(data)); err != nil {
		return fmt.Errorf("merge record failed: %w", err)
	}
	return nil
}

// MergeByPublication returns one publication's merge.
func (s *Store) MergeByPublication(ctx context.Context, publicationID string) (factory.Merge, error) {
	var m factory.Merge
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_merges WHERE publication=?`, publicationID).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &m)
	}
	return m, err
}

// UpdateMerge stores one merge's next revision exactly once. The caller
// increments Revision first; only the recorded predecessor advances,
// and anything else reports stale instead of overwriting a concurrent
// receipt.
func (s *Store) UpdateMerge(ctx context.Context, m factory.Merge) error {
	if err := m.Validate(); err != nil {
		return err
	}
	previous, err := s.MergeByPublication(ctx, m.PublicationID)
	if err != nil {
		return err
	}
	if previous.Revision != m.Revision-1 {
		return ErrStaleRevision
	}
	if err := mergeUpdateAllowed(previous, m); err != nil {
		return err
	}
	registering := previous.Operation.Work == nil && m.Operation.Work != nil
	if registering && (previous.WithdrawRequested || m.WithdrawRequested) {
		return ErrDispatchClosed
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	// Gate inspection and registration share one statement: a withdrawal
	// either sees this operation or closes the gate before it can register.
	result, err := s.db.ExecContext(ctx, `UPDATE factory_merges SET stage=?,revision=?,data=?
		WHERE publication=? AND revision=?
		AND (?=0 OR (
		 NOT EXISTS(SELECT 1 FROM factory_dispatch WHERE repository=? AND open=0)
		 AND EXISTS(SELECT 1 FROM issue_acceptance_heads WHERE repository=? AND issue=? AND decision=?)
		 AND NOT EXISTS(SELECT 1 FROM issue_acceptance_withdrawals WHERE repository=? AND issue=? AND decision=?)
		 AND EXISTS(SELECT 1 FROM factory_policies WHERE repository=? AND revision=?)
		 AND EXISTS(SELECT 1 FROM factory_operator_grants WHERE repository=? AND revision=?)
		 AND EXISTS(SELECT 1 FROM project_environment_grants WHERE repository=? AND revision=?)
		 AND EXISTS(SELECT 1 FROM factory_sponsorships s JOIN factory_assignments a
		   ON a.id=? AND s.connection=json_extract(a.data,'$.connection')
		   WHERE s.repository=? AND s.revision=?)
		 AND EXISTS(SELECT 1 FROM project_requirement_heads WHERE project_id=? AND decision=?)
		 AND EXISTS(SELECT 1 FROM project_approval_heads WHERE project_id=? AND decision=?)
		))`,
		m.Stage, m.Revision, string(data), m.PublicationID, m.Revision-1, registering,
		m.Repository, m.Repository, m.Issue, m.Acceptance, m.Repository, m.Issue, m.Acceptance,
		m.Repository, m.Authority.Policy, m.Repository, m.Authority.Operator, m.Repository, m.Authority.Environment,
		m.AssignmentID, m.Repository, m.Authority.Sponsorship,
		m.ProjectID, m.Authority.RequirementsID, m.ProjectID, m.Authority.ApprovalID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		if registering {
			if open, _, _, readErr := s.DispatchState(ctx, m.Repository); readErr != nil {
				return readErr
			} else if !open {
				return ErrDispatchClosed
			}
		}
		return ErrStaleRevision
	}
	return nil
}

func mergeUpdateAllowed(old, next factory.Merge) error {
	if old.ID != next.ID || old.PublicationID != next.PublicationID || old.AssignmentID != next.AssignmentID ||
		old.ProjectID != next.ProjectID || old.Role != next.Role || old.Acceptance != next.Acceptance ||
		old.Repository != next.Repository || old.Issue != next.Issue || old.PRNumber != next.PRNumber ||
		old.PRID != next.PRID || old.IssueID != next.IssueID || old.PRAuthorID != next.PRAuthorID ||
		old.ReviewerID != next.ReviewerID || old.HeadRef != next.HeadRef || old.BaseRef != next.BaseRef ||
		old.HeadOID != next.HeadOID || old.BaseOID != next.BaseOID ||
		old.CreatedUnix != next.CreatedUnix || old.Authority != next.Authority || old.WithdrawRequested && !next.WithdrawRequested {
		return ErrMergeConflict
	}
	if old.Stage != factory.MergeOpen && old.Stage != factory.MergeFenced {
		return ErrMergeConflict
	}
	if !mergeOperationUpdateAllowed(old.Operation, next.Operation) {
		return ErrMergeConflict
	}
	return nil
}

func mergeOperationUpdateAllowed(old, next factory.MergeOperation) bool {
	if old.Kind != next.Kind {
		return false
	}
	if old.Work != nil && (next.Work == nil || *old.Work != *next.Work || old.OperationID != next.OperationID || old.Attempts != next.Attempts) {
		return false
	}
	if old.Receipt != "" && old.Receipt != next.Receipt {
		return false
	}
	if (old.Effect == factory.OpEffectCommitted || old.Effect == factory.OpEffectNotCommitted) && old.Effect != next.Effect {
		return false
	}
	return true
}

// OutstandingMerges lists every open or fenced merge for one
// repository, oldest first. Recovery reconciles each one against native
// state; withdrawal cancels each one's outstanding operation.
func (s *Store) OutstandingMerges(ctx context.Context, repository int64, limit int) ([]factory.Merge, error) {
	if repository <= 0 {
		return nil, errors.New("invalid merge repository")
	}
	if limit <= 0 || limit > storeMergeLimit {
		return nil, errors.New("invalid merge listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_merges
		WHERE repository=? AND stage IN ('open','fenced') ORDER BY rowid LIMIT ?`, repository, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []factory.Merge
	for rows.Next() {
		var data []byte
		var m factory.Merge
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// OpenMerges lists every open or fenced merge across repositories,
// oldest first, for pass recovery. Deferred work waits for a later
// pass; nothing here submits without reconciling first.
func (s *Store) OpenMerges(ctx context.Context, limit int) ([]factory.Merge, error) {
	if limit <= 0 || limit > storeMergeLimit {
		return nil, errors.New("invalid merge listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_merges
		WHERE stage IN ('open','fenced') ORDER BY rowid LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []factory.Merge
	for rows.Next() {
		var data []byte
		var m factory.Merge
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// MergeablePublications lists published publications that carry no
// merge yet, oldest first. Only natively completed exact PRs merge;
// anything else never reaches this listing.
func (s *Store) MergeablePublications(ctx context.Context, limit int) ([]factory.Publication, error) {
	if limit <= 0 || limit > storeMergeableLimit {
		return nil, errors.New("invalid mergeable listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT p.data FROM factory_publications p
		LEFT JOIN factory_merges m ON m.publication=p.data->>'$.id'
		WHERE p.stage='published' AND m.publication IS NULL
		ORDER BY p.rowid LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []factory.Publication
	for rows.Next() {
		var data []byte
		var p factory.Publication
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// MergeForIssue returns the latest merge for one repository issue, any
// stage. Absence reports ErrNotFound: no merge exists yet for it.
func (s *Store) MergeForIssue(ctx context.Context, repository, issue int64) (factory.Merge, error) {
	var m factory.Merge
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_merges WHERE repository=? AND issue=? ORDER BY rowid DESC LIMIT 1`, repository, issue).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &m)
	}
	return m, err
}

// IssueMergeCompletion returns the latest completed merge for one
// repository issue: the attributable factory completion behind code
// prerequisite satisfaction. Absence reports ErrNotFound.
func (s *Store) IssueMergeCompletion(ctx context.Context, repository, issue int64) (factory.Merge, error) {
	var m factory.Merge
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_merges WHERE repository=? AND issue=? AND stage='merged' ORDER BY rowid DESC LIMIT 1`, repository, issue).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &m)
	}
	return m, err
}
