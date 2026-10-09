package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// RecordFactoryRun stores a fresh supervised run before any host call. A
// consumed run ID is never reused; only the supervisor mutates the record.
func (s *Store) RecordFactoryRun(ctx context.Context, r factory.Run) error {
	if r.Admission != nil {
		return errors.New("run admission requires an atomic dispatch packet")
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if r.Outcome != "" || r.Reconciled {
		return errors.New("new run must be active and unsettled")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO factory_runs(id,active,settled,data) VALUES($1,TRUE,FALSE,$2)`, r.ID, string(data))
	if err != nil {
		return fmt.Errorf("factory run record failed: %w", err)
	}
	return nil
}

func (s *Store) FactoryRun(ctx context.Context, id string) (factory.Run, error) {
	var r factory.Run
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_runs WHERE id=$1`, id).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &r)
	}
	return r, err
}

// SaveFactoryRun persists supervisor observations. Credential facts and the
// settled marker only advance; a terminal outcome never changes. Admission
// metadata belongs to the dispatch transaction and is preserved when an
// observation omits it; an attempted replacement refuses.
func (s *Store) SaveFactoryRun(ctx context.Context, r factory.Run) error {
	if err := r.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE factory_runs SET active=$1,settled=$2,
data=($3::jsonb - 'admission') || jsonb_build_object('admission',data->'admission') WHERE id=$4
AND (($7::jsonb->'admission') IS NULL OR (data->'admission')=($7::jsonb->'admission'))
AND (coalesce(data->>'identity_lease_id','')='' OR (data->>'identity_lease_id'=$5 AND (data->>'identity_generation')::bigint=$6))
AND ((data->'identity_binding') IS NULL OR (data->'identity_binding')=($7::jsonb->'identity_binding'))
AND (coalesce((data->>'credential_delegated')::boolean,FALSE)=FALSE OR $8)
AND (coalesce((data->>'credential_returned')::boolean,FALSE)=FALSE OR $9)
AND (coalesce(data->>'outcome','')='' OR data->>'outcome'=$10)
AND ((data->>'reconciled')::boolean=FALSE OR $11)`, r.Outcome == "", r.Reconciled, string(data), r.ID, r.IdentityLeaseID, r.IdentityGeneration, string(data), r.CredentialDelegated, r.CredentialReturned, r.Outcome, r.Reconciled)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrNotFound
	}
	if r.Reconciled {
		if err = settleClosedAttemptForRunTx(ctx, tx, r.ID, time.Now()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func settleClosedAttemptForRunTx(ctx context.Context, tx *sql.Tx, runID string, now time.Time) error {
	var repository, issue int64
	var attempt string
	err := tx.QueryRowContext(ctx, `SELECT v.repository,v.issue,v.attempt FROM factory_run_views v WHERE v.run=$1`, runID).Scan(&repository, &issue, &attempt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var assignmentData []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1`, attempt).Scan(&assignmentData); err != nil {
		return err
	}
	var assignment factory.Assignment
	if err = json.Unmarshal(assignmentData, &assignment); err != nil {
		return err
	}
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, repository, issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) || err == nil && !allowance.Active {
		return nil
	}
	if err != nil {
		return err
	}
	owner, err := currentAttemptOwnerTx(ctx, tx, repository, issue, assignment.AttemptRoot)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var latestData []byte
	var latestID string
	err = tx.QueryRowContext(ctx, `SELECT id,data FROM factory_assignments
		WHERE repository=$1 AND issue=$2 ORDER BY seq DESC LIMIT 1`, repository, issue).Scan(&latestID, &latestData)
	if err != nil {
		return err
	}
	var latest factory.Assignment
	if err = json.Unmarshal(latestData, &latest); err != nil {
		return err
	}
	if latest.AttemptRoot != assignment.AttemptRoot {
		return nil
	}
	closeRoot := allowance.Closed
	shouldTransition := allowance.Closed
	if latest.Stage == factory.AssignmentFinished && !(latest.Outcome == factory.Succeeded && latest.Reason == factory.AssignReasonReported) {
		shouldTransition = true
	}
	if latest.Stage == factory.AssignmentAssigned {
		var reservationState string
		err = tx.QueryRowContext(ctx, `SELECT state FROM factory_reservations WHERE assignment=$1`, latestID).Scan(&reservationState)
		if err == nil && reservationState == factory.ReservationReleased {
			shouldTransition = true
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	var pubData []byte
	err = tx.QueryRowContext(ctx, `SELECT data FROM factory_publications WHERE assignment=$1`, owner).Scan(&pubData)
	if err == nil {
		var publication factory.Publication
		if err = json.Unmarshal(pubData, &publication); err != nil {
			return err
		}
		switch publication.Stage {
		case factory.PublicationFailed, factory.PublicationFenced, factory.PublicationWithdrawn:
			shouldTransition = true
			if publication.Reason != factory.PublishReasonSuperseded && publication.Reason != factory.PublishReasonWithdrawn {
				closeRoot = true
			}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var mergeData []byte
	err = tx.QueryRowContext(ctx, `SELECT data FROM factory_merges WHERE publication=$1`, owner).Scan(&mergeData)
	if err == nil {
		var merge factory.Merge
		if err = json.Unmarshal(mergeData, &merge); err != nil {
			return err
		}
		if merge.Stage != factory.MergeOpen {
			shouldTransition = true
			if merge.Stage == factory.MergeMerged || merge.Stage == factory.MergeFenced ||
				merge.Stage == factory.MergeFailed && merge.Reason != factory.MergeReasonSuperseded && merge.Reason != factory.MergeReasonWithdrawn {
				closeRoot = true
			}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if !shouldTransition {
		return nil
	}
	return transitionAttemptAssignmentTx(ctx, tx, owner, closeRoot, now)
}

// FactoryRuns returns recorded runs, newest first, bounded for the private
// operator status read. Reconciliation never adopts host state it did not
// record; unknown runs stay invisible until recorded.
func (s *Store) FactoryRuns(ctx context.Context, limit int) ([]factory.Run, error) {
	if limit < 1 || limit > 1000 {
		return nil, errors.New("invalid run list bound")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_runs ORDER BY seq DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var runs []factory.Run
	for rows.Next() {
		var data []byte
		var r factory.Run
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// RecordFactoryCommand stores an operator command exactly once. The same ID
// with the same digest returns the stored outcome; a changed payload
// conflicts instead of executing twice.
func (s *Store) RecordFactoryCommand(ctx context.Context, c factory.Command, now time.Time) (factory.Command, bool, error) {
	if err := c.Validate(); err != nil {
		return factory.Command{}, false, err
	}
	if c.Outcome != "" {
		return factory.Command{}, false, errors.New("new command must not carry an outcome")
	}
	created := now.UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_commands(id,type,target,principal,digest,payload,created) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO NOTHING`, c.ID, c.Type, c.Target, c.Principal, c.Digest, c.Payload, created)
	if err != nil {
		return factory.Command{}, false, fmt.Errorf("factory command record failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.Command{}, false, err
	}
	stored, err := s.FactoryCommand(ctx, c.ID)
	if err != nil {
		return factory.Command{}, false, err
	}
	if stored.Digest != c.Digest {
		return factory.Command{}, false, ErrCommandConflict
	}
	return stored, n == 1, nil
}

func (s *Store) FactoryCommand(ctx context.Context, id string) (factory.Command, error) {
	var c factory.Command
	var created, finished string
	err := s.db.QueryRowContext(ctx, `SELECT id,type,target,principal,digest,payload,outcome,created,finished FROM factory_commands WHERE id=$1`, id).Scan(&c.ID, &c.Type, &c.Target, &c.Principal, &c.Digest, &c.Payload, &c.Outcome, &created, &finished)
	if err != nil {
		return c, err
	}
	c.Created, c.Finished = created, finished
	return c, nil
}

// FinishFactoryCommand records the durable outcome exactly once. A lost reply
// replays the stored outcome; it never re-executes the command.
// UnfinishedCommands lists recorded commands without a finish, oldest
// first, for restart recovery. A crash between record and finish leaves
// exactly this state; the coordinator settles what it can from durable
// target state and leaves the rest running.
func (s *Store) UnfinishedCommands(ctx context.Context, limit int) ([]factory.Command, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,type,target,principal,digest,payload,outcome,created,finished FROM factory_commands WHERE finished='' ORDER BY created,id LIMIT $1`,
		min(max(limit, 1), 101))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []factory.Command
	for rows.Next() {
		var c factory.Command
		if err = rows.Scan(&c.ID, &c.Type, &c.Target, &c.Principal, &c.Digest, &c.Payload, &c.Outcome, &c.Created, &c.Finished); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) FinishFactoryCommand(ctx context.Context, id, outcome string, now time.Time) error {
	if len(outcome) > 64<<10 {
		return errors.New("command outcome exceeds retained output limit")
	}
	finished := now.UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `UPDATE factory_commands SET outcome=$1,finished=$2 WHERE id=$3 AND finished=''`, outcome, finished, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrNotFound
	}
	return nil
}
