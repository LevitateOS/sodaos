package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
)

// storePublicationLimit bounds one outstanding-publication listing.
const storePublicationLimit = 1024

// storePublishableLimit bounds one publishable-assignment listing.
const storePublishableLimit = 256

// ErrPublicationConflict rejects replacement of a recorded publication intent.
var ErrPublicationConflict = errors.New("publication immutable input changed")

// RecordPublication stores one assignment's fresh open publication. One
// assignment publishes at most once: a second record for the assignment
// refuses instead of publishing twice.
func (s *Store) RecordPublication(ctx context.Context, p factory.Publication) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Stage != factory.PublicationOpen || p.Outcome != "" || p.Reason != "" || p.FinishedUnix != 0 || p.Revision != 0 || p.Publish.Attempts != 0 || p.PRCreate.Attempts != 0 || p.WithdrawRequested {
		return errors.New("publication packet carries fresh open work")
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_publications(assignment,repository,issue,run,stage,revision,data) VALUES(?,?,?,?,?,?,?)`,
		p.AssignmentID, p.Repository, p.Issue, p.Run, p.Stage, p.Revision, string(data)); err != nil {
		return fmt.Errorf("publication record failed: %w", err)
	}
	return nil
}

// PublicationByAssignment returns one assignment's publication.
func (s *Store) PublicationByAssignment(ctx context.Context, assignmentID string) (factory.Publication, error) {
	var p factory.Publication
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_publications WHERE assignment=?`, assignmentID).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &p)
	}
	return p, err
}

// UpdatePublication stores one publication's next revision exactly once.
// The caller increments Revision first; only the recorded predecessor
// advances, and anything else reports stale instead of overwriting a
// concurrent receipt.
func (s *Store) UpdatePublication(ctx context.Context, p factory.Publication) error {
	if err := p.Validate(); err != nil {
		return err
	}
	previous, err := s.PublicationByAssignment(ctx, p.AssignmentID)
	if err != nil {
		return err
	}
	if previous.Revision != p.Revision-1 {
		return ErrStaleRevision
	}
	if err := publicationUpdateAllowed(previous, p); err != nil {
		return err
	}
	registering := previous.Publish.Work == nil && p.Publish.Work != nil || previous.PRCreate.Work == nil && p.PRCreate.Work != nil
	if registering && (previous.WithdrawRequested || p.WithdrawRequested) {
		return ErrDispatchClosed
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	// Gate inspection and registration share one statement: a withdrawal
	// either sees this operation or closes the gate before it can register.
	result, err := s.db.ExecContext(ctx, `UPDATE factory_publications SET stage=?,revision=?,data=?
		WHERE assignment=? AND revision=?
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
		p.Stage, p.Revision, string(data), p.AssignmentID, p.Revision-1, registering,
		p.Repository, p.Repository, p.Issue, p.Acceptance, p.Repository, p.Issue, p.Acceptance,
		p.Repository, p.Authority.Policy, p.Repository, p.Authority.Operator, p.Repository, p.Authority.Environment,
		p.AssignmentID, p.Repository, p.Authority.Sponsorship,
		p.ProjectID, p.Authority.RequirementsID, p.ProjectID, p.Authority.ApprovalID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		if registering {
			if open, _, _, readErr := s.DispatchState(ctx, p.Repository); readErr != nil {
				return readErr
			} else if !open {
				return ErrDispatchClosed
			}
		}
		return ErrStaleRevision
	}
	return nil
}

func publicationUpdateAllowed(old, next factory.Publication) error {
	if old.ID != next.ID || old.AssignmentID != next.AssignmentID || old.ProjectID != next.ProjectID ||
		old.Role != next.Role || old.Acceptance != next.Acceptance || old.Preparation != next.Preparation ||
		old.Run != next.Run || old.Candidate != next.Candidate || old.BaseSHA != next.BaseSHA ||
		old.TargetBranch != next.TargetBranch || old.Repository != next.Repository || old.Issue != next.Issue ||
		old.CreatedUnix != next.CreatedUnix || old.Authority != next.Authority || old.WithdrawRequested && !next.WithdrawRequested {
		return ErrPublicationConflict
	}
	if old.Stage != factory.PublicationOpen && old.Stage != factory.PublicationFenced {
		return ErrPublicationConflict
	}
	if !publicationOperationUpdateAllowed(old.Publish, next.Publish) || !publicationOperationUpdateAllowed(old.PRCreate, next.PRCreate) {
		return ErrPublicationConflict
	}
	return nil
}

func publicationOperationUpdateAllowed(old, next factory.PublicationOperation) bool {
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

// OutstandingPublications lists every open or fenced publication for one
// repository, oldest first. Recovery reconciles each one against native
// state; withdrawal cancels each one's outstanding operations.
func (s *Store) OutstandingPublications(ctx context.Context, repository int64, limit int) ([]factory.Publication, error) {
	if repository <= 0 {
		return nil, errors.New("invalid publication repository")
	}
	if limit <= 0 || limit > storePublicationLimit {
		return nil, errors.New("invalid publication listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_publications
		WHERE repository=? AND stage IN ('open','fenced') ORDER BY rowid LIMIT ?`, repository, limit)
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

// OpenPublications lists every open or fenced publication across
// repositories, oldest first, for pass recovery. Deferred work waits for
// a later pass; nothing here submits without reconciling first.
func (s *Store) OpenPublications(ctx context.Context, limit int) ([]factory.Publication, error) {
	if limit <= 0 || limit > storePublicationLimit {
		return nil, errors.New("invalid publication listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_publications
		WHERE stage IN ('open','fenced') ORDER BY rowid LIMIT ?`, limit)
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

// PublishableAssignments lists finished assignments whose harness
// reported a completed exact candidate and that carry no publication
// yet, oldest first. Only succeeded work with an attributable report
// publishes; synthesized results never do.
func (s *Store) PublishableAssignments(ctx context.Context, limit int) ([]factory.Assignment, error) {
	if limit <= 0 || limit > storePublishableLimit {
		return nil, errors.New("invalid publishable listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.data FROM factory_assignments a
		LEFT JOIN factory_publications p ON p.assignment=a.id
		WHERE a.stage='finished' AND p.assignment IS NULL
		AND json_extract(a.data,'$.role')='soda-coder'
		AND json_extract(a.data,'$.outcome')='succeeded'
		AND json_extract(a.data,'$.result.reported')=1
		AND json_extract(a.data,'$.result.status')='completed'
		ORDER BY a.rowid LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []factory.Assignment
	for rows.Next() {
		var data []byte
		var a factory.Assignment
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
