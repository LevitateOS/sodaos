package store

import (
	"context"
	"database/sql"
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
	if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_publications(assignment,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		p.AssignmentID, p.Repository, p.Issue, p.Run, p.Stage, p.Revision, string(data)); err != nil {
		return fmt.Errorf("publication record failed: %w", err)
	}
	return nil
}

// PublicationByAssignment returns one assignment's publication.
func (s *Store) PublicationByAssignment(ctx context.Context, assignmentID string) (factory.Publication, error) {
	var p factory.Publication
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_publications WHERE assignment=$1`, assignmentID).Scan(&data)
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var previous factory.Publication
	var previousData []byte
	err = tx.QueryRowContext(ctx, `SELECT data FROM factory_publications WHERE assignment=$1`, p.AssignmentID).Scan(&previousData)
	if err == nil {
		err = json.Unmarshal(previousData, &previous)
	}
	if err != nil {
		return err
	}
	if previous.Revision != p.Revision-1 {
		return ErrStaleRevision
	}
	if err := publicationUpdateAllowed(previous, p); err != nil {
		return err
	}
	registering := previous.Publish.Work == nil && p.Publish.Work != nil ||
		previous.PRCreate.Work == nil && p.PRCreate.Work != nil ||
		len(p.Corrections) > len(previous.Corrections) ||
		len(p.ReviewOperations) > len(previous.ReviewOperations)
	if registering && (previous.WithdrawRequested || p.WithdrawRequested) {
		return ErrDispatchClosed
	}
	if registering {
		// Materialize the logical default-open gate so registration and
		// WithdrawDispatch serialize on the same row even before its first
		// withdrawal. A missing row otherwise has no lockable identity.
		if _, err = tx.ExecContext(ctx, `INSERT INTO factory_dispatch(repository,revision,open,data)
			VALUES($1,0,TRUE,'{}'::jsonb) ON CONFLICT(repository) DO NOTHING`, p.Repository); err != nil {
			return err
		}
		var open bool
		if err = tx.QueryRowContext(ctx, `SELECT open FROM factory_dispatch WHERE repository=$1 FOR UPDATE`, p.Repository).Scan(&open); err != nil {
			return err
		}
		if !open {
			return ErrDispatchClosed
		}
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	// Gate inspection and registration share one statement: a withdrawal
	// either sees this operation or closes the gate before it can register.
	result, err := tx.ExecContext(ctx, `UPDATE factory_publications SET stage=$1,revision=$2,data=$3
		WHERE assignment=$4 AND revision=$5
		AND (NOT $6 OR (
		 NOT EXISTS(SELECT 1 FROM factory_dispatch WHERE repository=$7 AND NOT open)
		 AND EXISTS(SELECT 1 FROM issue_acceptance_heads WHERE repository=$8 AND issue=$9 AND decision=$10)
		 AND NOT EXISTS(SELECT 1 FROM issue_acceptance_withdrawals WHERE repository=$11 AND issue=$12 AND decision=$13)
		 AND EXISTS(SELECT 1 FROM factory_policies WHERE repository=$14 AND revision=$15)
		 AND EXISTS(SELECT 1 FROM factory_operator_grants WHERE repository=$16 AND revision=$17)
		 AND EXISTS(SELECT 1 FROM project_environment_grants WHERE repository=$18 AND revision=$19)
		 AND EXISTS(SELECT 1 FROM factory_sponsorships s JOIN factory_assignments a
		   ON a.id=$20 AND s.connection=a.data->>'connection'
		   WHERE s.repository=$21 AND s.revision=$22)
		 AND EXISTS(SELECT 1 FROM project_requirement_heads WHERE project_id=$23 AND decision=$24)
		 AND EXISTS(SELECT 1 FROM project_approval_heads WHERE project_id=$25 AND decision=$26)
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
		return ErrStaleRevision
	}
	return tx.Commit()
}

func publicationUpdateAllowed(old, next factory.Publication) error {
	if old.ID != next.ID || old.AssignmentID != next.AssignmentID || old.ProjectID != next.ProjectID ||
		old.Role != next.Role || old.Acceptance != next.Acceptance || old.Preparation != next.Preparation ||
		old.BaseSHA != next.BaseSHA ||
		old.TargetBranch != next.TargetBranch || old.Repository != next.Repository || old.Issue != next.Issue ||
		old.CreatedUnix != next.CreatedUnix || old.Authority != next.Authority || old.WithdrawRequested && !next.WithdrawRequested {
		return ErrPublicationConflict
	}
	// The run and head advance only along a recorded correction chain;
	// Publication.Validate pins the exact chain and head.
	if (old.Run != next.Run || old.Candidate != next.Candidate) && len(next.Corrections) == 0 {
		return ErrPublicationConflict
	}
	if len(next.Corrections) < len(old.Corrections) {
		return ErrPublicationConflict
	}
	for i := range old.Corrections {
		if !publicationOperationUpdateAllowed(old.Corrections[i], next.Corrections[i]) {
			return ErrPublicationConflict
		}
	}
	if len(next.ReviewOperations) < len(old.ReviewOperations) {
		return ErrPublicationConflict
	}
	for i := range old.ReviewOperations {
		if !reviewOperationUpdateAllowed(old.ReviewOperations[i], next.ReviewOperations[i]) {
			return ErrPublicationConflict
		}
	}
	switch {
	case old.Stage == factory.PublicationOpen || old.Stage == factory.PublicationFenced:
	case old.Stage == factory.PublicationPublished &&
		(next.Stage == factory.PublicationPublished || next.Stage == factory.PublicationFenced):
	default:
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

func reviewOperationUpdateAllowed(old, next factory.ReviewOperation) bool {
	if old.RunID != next.RunID || old.Work != next.Work {
		return false
	}
	prior, updated := old.Outcome, next.Outcome
	if prior.OperationID != "" && prior.OperationID != updated.OperationID ||
		prior.InstallationID != "" && prior.InstallationID != updated.InstallationID ||
		prior.Kind != "" && prior.Kind != updated.Kind || prior.ActorID != 0 && prior.ActorID != updated.ActorID ||
		prior.RepositoryID != 0 && prior.RepositoryID != updated.RepositoryID {
		return false
	}
	if len(prior.Receipt) > 0 && string(prior.Receipt) != string(updated.Receipt) {
		return false
	}
	if (prior.Effect == factory.OpEffectCommitted || prior.Effect == factory.OpEffectNotCommitted) && prior.Effect != updated.Effect {
		return false
	}
	if prior.Cancellation == factory.OpCancelCancelled && updated.Cancellation != prior.Cancellation {
		return false
	}
	if prior.Completion == factory.OpCompletionComplete && updated.Completion != prior.Completion {
		return false
	}
	return true
}

// OutstandingPublications lists every open or fenced publication and each
// published record with an unresolved correction for one repository, oldest
// first. Recovery reconciles each one against native state; withdrawal
// cancels each one's outstanding operations.
func (s *Store) OutstandingPublications(ctx context.Context, repository int64, limit int) ([]factory.Publication, error) {
	if repository <= 0 {
		return nil, errors.New("invalid publication repository")
	}
	if limit <= 0 || limit > storePublicationLimit {
		return nil, errors.New("invalid publication listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_publications
		WHERE repository=$1 AND (
			stage IN ('open','fenced') OR
			(stage='published' AND (EXISTS (
				SELECT 1 FROM jsonb_array_elements(COALESCE(data->'corrections','[]'::jsonb)) AS correction
				WHERE COALESCE(correction->>'effect','') IN ('','pending','indeterminate')
				   OR COALESCE(correction->>'cancellation','') IN ('pending','indeterminate')
				   OR (correction->>'effect'='committed' AND COALESCE(correction->>'completion','') <> 'complete')
			) OR EXISTS (
				SELECT 1 FROM jsonb_array_elements(COALESCE(data->'review_operations','[]'::jsonb)) AS review_op
				WHERE COALESCE(review_op->'outcome'->>'Effect','') IN ('','pending','indeterminate')
				   OR COALESCE(review_op->'outcome'->>'Cancellation','') IN ('pending','indeterminate')
				   OR (review_op->'outcome'->>'Effect'='committed' AND COALESCE(review_op->'outcome'->>'Completion','') <> 'complete')
			)))
		) ORDER BY seq LIMIT $2`, repository, limit)
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

// requestPublicationWithdrawalsTx durably latches every outstanding
// publication under the caller's repository gate lock. It refuses an
// oversized set rather than leaving later records uncancelled.
func requestPublicationWithdrawalsTx(ctx context.Context, tx *sql.Tx, repository int64) (bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT assignment,revision,COALESCE((data->>'withdraw_requested')::boolean,FALSE) FROM factory_publications
		WHERE repository=$1 AND (
			stage IN ('open','fenced') OR
			(stage='published' AND (EXISTS (
				SELECT 1 FROM jsonb_array_elements(COALESCE(data->'corrections','[]'::jsonb)) AS correction
				WHERE COALESCE(correction->>'effect','') IN ('','pending','indeterminate')
				   OR COALESCE(correction->>'cancellation','') IN ('pending','indeterminate')
				   OR (correction->>'effect'='committed' AND COALESCE(correction->>'completion','') <> 'complete')
			) OR EXISTS (
				SELECT 1 FROM jsonb_array_elements(COALESCE(data->'review_operations','[]'::jsonb)) AS review_op
				WHERE COALESCE(review_op->'outcome'->>'Effect','') IN ('','pending','indeterminate')
				   OR COALESCE(review_op->'outcome'->>'Cancellation','') IN ('pending','indeterminate')
				   OR (review_op->'outcome'->>'Effect'='committed' AND COALESCE(review_op->'outcome'->>'Completion','') <> 'complete')
			)))
		) ORDER BY seq LIMIT $2 FOR UPDATE`, repository, storePublicationLimit+1)
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	type publicationIntent struct {
		assignment string
		revision   int64
		requested  bool
	}
	var publications []publicationIntent
	for rows.Next() {
		var item publicationIntent
		if err = rows.Scan(&item.assignment, &item.revision, &item.requested); err != nil {
			return false, err
		}
		publications = append(publications, item)
	}
	if err = rows.Err(); err != nil {
		return false, err
	}
	if err = rows.Close(); err != nil {
		return false, err
	}
	if len(publications) > storePublicationLimit {
		return false, errors.New("too many outstanding publications to withdraw")
	}
	for _, p := range publications {
		if p.requested {
			continue
		}
		revision := p.revision + 1
		updated, updateErr := tx.ExecContext(ctx, `UPDATE factory_publications SET revision=$1,
			data=jsonb_set(jsonb_set(data,'{withdraw_requested}','true'::jsonb,true),'{revision}',to_jsonb($1::integer),true)
			WHERE assignment=$2 AND revision=$3`, revision, p.assignment, p.revision)
		if updateErr != nil {
			return false, updateErr
		}
		n, affectedErr := updated.RowsAffected()
		if affectedErr != nil {
			return false, affectedErr
		}
		if n != 1 {
			return false, ErrStaleRevision
		}
	}
	return len(publications) > 0, nil
}

// OpenPublications lists every open or fenced publication across
// repositories, oldest first, for pass recovery. Deferred work waits for
// a later pass; nothing here submits without reconciling first.
func (s *Store) OpenPublications(ctx context.Context, limit int) ([]factory.Publication, error) {
	if limit <= 0 || limit > storePublicationLimit {
		return nil, errors.New("invalid publication listing limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_publications
		WHERE stage IN ('open','fenced') OR (stage='published' AND (EXISTS (
			SELECT 1 FROM jsonb_array_elements(COALESCE(data->'corrections','[]'::jsonb)) AS correction
			WHERE COALESCE(correction->>'effect','') IN ('','pending','indeterminate')
			   OR COALESCE(correction->>'cancellation','') IN ('pending','indeterminate')
			   OR (correction->>'effect'='committed' AND COALESCE(correction->>'completion','') <> 'complete')
		) OR EXISTS (
			SELECT 1 FROM jsonb_array_elements(COALESCE(data->'review_operations','[]'::jsonb)) AS review_op
			WHERE COALESCE(review_op->'outcome'->>'Effect','') IN ('','pending','indeterminate')
			   OR COALESCE(review_op->'outcome'->>'Cancellation','') IN ('pending','indeterminate')
			   OR (review_op->'outcome'->>'Effect'='committed' AND COALESCE(review_op->'outcome'->>'Completion','') <> 'complete')
		))) ORDER BY seq LIMIT $1`, limit)
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
		AND a.data->>'role'='soda-coder'
		AND a.data->>'publication_assignment'=a.id
		AND a.data->>'outcome'='succeeded'
		AND (a.data#>>'{result,reported}')::boolean
		AND a.data#>>'{result,status}'='completed'
		ORDER BY a.seq LIMIT $1`, limit)
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
