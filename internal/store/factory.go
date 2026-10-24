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
	if r.Admission != nil || r.FinalCheckpoint != nil || len(r.VerificationManifest) > 0 || r.VerificationChecks != nil {
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
data=($3::jsonb - 'admission' - 'execution_manifest' - 'verification_manifest' - 'verification_checks' - 'final_checkpoint' - 'settled_at') || jsonb_build_object('admission',data->'admission','execution_manifest',data->'execution_manifest','verification_manifest',data->'verification_manifest','verification_checks',COALESCE(NULLIF($3::jsonb->'verification_checks','null'::jsonb),data->'verification_checks'),'final_checkpoint',data->'final_checkpoint','settled_at',COALESCE(NULLIF(data->'settled_at','null'::jsonb),CASE WHEN $11 AND NOT settled THEN to_jsonb(clock_timestamp()) END)) WHERE id=$4
AND (($7::jsonb->'admission') IS NULL OR (data->'admission')=($7::jsonb->'admission'))
AND (($7::jsonb->'execution_manifest') IS NULL OR (data->'execution_manifest')=($7::jsonb->'execution_manifest'))
AND (($7::jsonb->'verification_manifest') IS NULL OR (data->'verification_manifest')=($7::jsonb->'verification_manifest'))
AND (($7::jsonb->'verification_checks') IS NULL OR coalesce(data#>>'{verification_checks,phase}','') NOT IN ('passed','failed','stopped') OR data->'verification_checks'=$7::jsonb->'verification_checks')
AND (($7::jsonb->'final_checkpoint') IS NULL OR (data->'final_checkpoint')=($7::jsonb->'final_checkpoint'))
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
		repository, issue, assignment, scopeErr := factoryRunFeedScopeTx(ctx, tx, r.ID, r.ProjectID)
		if scopeErr != nil {
			return scopeErr
		}
		if repository > 0 {
			content := factory.FeedCommunication{ComponentID: "F02", Title: "Worker run finished", Summary: "The run has stopped and its recorded resources have settled.",
				Items: []factory.FeedCommunicationItem{{Label: "Run", State: string(r.Outcome)}}}
			if r.Outcome != factory.Succeeded {
				content.ComponentID, content.Title, content.Severity = "F09", "Worker run "+string(r.Outcome), "warning"
				content.Summary = feedText(r.Summary)
				if r.VerificationChecks != nil && (r.VerificationChecks.Phase == "failed" || r.VerificationChecks.Phase == "stopped") {
					content.Title = "Application verification checks " + r.VerificationChecks.Phase
					content.Summary = "The trusted native verification manifest did not permit worker startup. Its original run and spending remain recorded."
				}
			}
			if r.Outcome == factory.Cancelled {
				content.ComponentID, content.Title = "F08", "Worker run stopped"
			}
			if err = recordFactoryFeedEventTx(ctx, tx, repository, issue, assignment, "run:"+r.ID, "run:"+string(r.Outcome), content, 0); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// gateSettleRootTx locks the attempt allowance root and resolves its
// current owner. Missing or inactive roots do not proceed; every
// other failure aborts.
func gateSettleRootTx(ctx context.Context, tx *sql.Tx, repository, issue int64, root string) (factory.AttemptAllowance, string, bool, error) {
	var allowance factory.AttemptAllowance
	locked, err := lockAttemptAllowanceRootTx(ctx, tx, repository, issue, root)
	if errors.Is(err, sql.ErrNoRows) || err == nil && !locked.Active {
		return allowance, "", false, nil
	}
	if err != nil {
		return allowance, "", false, err
	}
	owner, err := currentAttemptOwnerTx(ctx, tx, repository, issue, root)
	if errors.Is(err, sql.ErrNoRows) {
		return allowance, "", false, nil
	}
	if err != nil {
		return allowance, "", false, err
	}
	return locked, owner, true, nil
}

// loadSettleAttemptTx reads the run view and its attempt assignment
// for one joint settlement. A missing view yields an empty assignment,
// which the caller skips; every other failure aborts.
func loadSettleAttemptTx(ctx context.Context, tx *sql.Tx, runID string) (int64, int64, factory.Assignment, error) {
	var repository, issue int64
	var attempt string
	var assignment factory.Assignment
	err := tx.QueryRowContext(ctx, `SELECT v.repository,v.issue,v.attempt FROM factory_run_views v WHERE v.run=$1`, runID).Scan(&repository, &issue, &attempt)
	if errors.Is(err, sql.ErrNoRows) {
		return repository, issue, assignment, nil
	}
	if err != nil {
		return repository, issue, assignment, err
	}
	var assignmentData []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1`, attempt).Scan(&assignmentData); err != nil {
		return repository, issue, assignment, err
	}
	if err = json.Unmarshal(assignmentData, &assignment); err != nil {
		return repository, issue, assignment, err
	}
	return repository, issue, assignment, nil
}

func settleClosedAttemptForRunTx(ctx context.Context, tx *sql.Tx, runID string, now time.Time) error {
	repository, issue, assignment, err := loadSettleAttemptTx(ctx, tx, runID)
	if err != nil {
		return err
	}
	if assignment.ID == "" {
		return nil
	}
	// Typed completion branch: completion roots carry no publication
	// owner, so the ordinary latest(repository, issue) owner lookup
	// below cannot address them and is never consulted here. The run
	// view's attempt assignment binds the exact root instead: its typed
	// allowance locks by exact subject IDs and digest, and the shared
	// joint checkpoints only once actual recorded effects settle. A
	// missing or mismatched typed scope fails the transaction instead
	// of silently skipping, keeping SaveFactoryRun and its accounting
	// atomic. Closed stays closed; terminal assignment state alone
	// never stands in for settled native effects.
	if assignment.Subject.Kind == factory.SubjectCompletion {
		return settleTypedCompletionRootTx(ctx, tx, repository, issue, assignment, now)
	}
	if assignment.Subject.Kind == factory.SubjectStall {
		return settleTypedStallRootTx(ctx, tx, repository, issue, assignment, now)
	}
	if assignment.Subject.Kind == factory.SubjectIntent {
		return settleTypedIntentRootTx(ctx, tx, repository, issue, assignment, now)
	}
	if assignment.Subject.Kind == factory.SubjectDelivery {
		return settleTypedDeliveryRootTx(ctx, tx, repository, issue, assignment, now)
	}
	// Accepted inquiries and Library maintenance have no publication
	// owner. Settle their own recorded issue root after its native effects
	// retire, preserving the typed terminal's existing closed posture.
	if factory.NonCodeIssueWork(assignment.Phase, assignment.Role, assignment.Subject, assignment.Issue, assignment.Acceptance) {
		if err := assignment.Validate(); err != nil {
			return err
		}
		if assignment.Repository != repository || assignment.Issue != issue {
			return errors.New("non-code run view does not match its issue scope")
		}
		allowance, err := lockAttemptAllowanceRootTx(ctx, tx, repository, issue, assignment.AttemptRoot)
		if err != nil {
			return err
		}
		if !allowance.Closed && assignment.Stage != factory.AssignmentFinished {
			return nil
		}
		return transitionLockedAttemptRootTx(ctx, tx, allowance, false, now)
	}
	allowance, owner, proceed, err := gateSettleRootTx(ctx, tx, repository, issue, assignment.AttemptRoot)
	if err != nil || !proceed {
		return err
	}
	latest, latestID, err := lockedLatestAttemptTx(ctx, tx, repository, issue)
	if err != nil {
		return err
	}
	if latest.AttemptRoot != assignment.AttemptRoot {
		return nil
	}
	flags := settleFlags{transition: allowance.Closed, closeRoot: allowance.Closed}
	if latest.Stage == factory.AssignmentFinished && !(latest.Outcome == factory.Succeeded && latest.Reason == factory.AssignReasonReported) {
		flags.transition = true
	}
	if latest.Stage == factory.AssignmentAssigned {
		if released, err := latestReservationReleasedTx(ctx, tx, latestID); err != nil {
			return err
		} else if released {
			flags.transition = true
		}
	}
	if err = accumulatePublicationSettleTx(ctx, tx, owner, &flags); err != nil {
		return err
	}
	if err = accumulateMergeSettleTx(ctx, tx, owner, &flags); err != nil {
		return err
	}
	if !flags.transition {
		return nil
	}
	return transitionAttemptAssignmentTx(ctx, tx, owner, flags.closeRoot, now)
}

// settleFlags accumulates one joint-settlement decision: whether the
// attempt root transitions and whether it closes with it.
type settleFlags struct {
	transition bool
	closeRoot  bool
}

// settleTypedCompletionRootTx settles one typed completion root
// through its exact-subject allowance. Closed OR finished completion
// assignments both reach the shared joint once actual effects
// settle: a terminal cancelled/withdrawn/superseded root keeps
// Closed false per policy, and a closed-but-still-assigned root
// (closed by the terminal transition while its run was unsettled)
// must not wait on a finish row that may never come. closeRoot stays
// false here to preserve existing Closed posture — the joint
// checkpoints settled active clocks and never forces Closed true,
// reopens, or refunds. Neither closed nor finished waits; the
// joint's actual-effects predicate still prevents early freeze.
func settleTypedCompletionRootTx(ctx context.Context, tx *sql.Tx, repository, issue int64, assignment factory.Assignment, now time.Time) error {
	if err := assignment.Validate(); err != nil {
		return err
	}
	if assignment.Repository != repository || issue != 0 || assignment.Issue != 0 {
		return errors.New("completion run view does not match its typed scope")
	}
	if assignment.Subject.Completion == nil {
		return errors.New("reassessment root lacks its completion subject")
	}
	typed, err := lockReassessmentAttemptAllowanceTx(ctx, tx, repository, *assignment.Subject.Completion, assignment.AttemptRoot)
	if err != nil {
		return err
	}
	if !typed.Closed && assignment.Stage != factory.AssignmentFinished {
		return nil
	}
	return transitionLockedAttemptRootTx(ctx, tx, typed, false, now)
}

// A fenced Strategist terminal can close its independent stall root before
// native retirement. Its issue-zero scope has no publication owner: settle
// the exact retained subject/root and checkpoint only confirmed effects.
func settleTypedStallRootTx(ctx context.Context, tx *sql.Tx, repository, issue int64, assignment factory.Assignment, now time.Time) error {
	if err := assignment.Validate(); err != nil {
		return err
	}
	if assignment.Repository != repository || issue != 0 || assignment.Issue != 0 || assignment.Subject.Stall == nil {
		return errors.New("stall run view does not match its typed scope")
	}
	typed, err := lockAttemptAllowanceRootTx(ctx, tx, repository, 0, assignment.AttemptRoot)
	if err != nil {
		return err
	}
	if typed.Stall == nil || *typed.Stall != *assignment.Subject.Stall {
		return ErrAttemptRootMismatch
	}
	if !typed.Closed && assignment.Stage != factory.AssignmentFinished {
		return nil
	}
	return transitionLockedAttemptRootTx(ctx, tx, typed, false, now)
}

// Intent planning and app verification can report incomplete before native
// retirement. Both own independent issue-zero roots without publication owners;
// resolve their exact typed allowance before the shared settled-effects joint.
func settleTypedIntentRootTx(ctx context.Context, tx *sql.Tx, repository, issue int64, assignment factory.Assignment, now time.Time) error {
	if err := assignment.Validate(); err != nil {
		return err
	}
	if assignment.Repository != repository || issue != 0 || assignment.Issue != 0 || assignment.Subject.Intent == nil {
		return errors.New("intent run view does not match its typed scope")
	}
	scope := factory.IntentAttemptScope{IntentID: assignment.Subject.Intent.IntentID, OwnerID: assignment.Subject.Intent.OwnerID}
	typed, err := lockIntentAttemptAllowanceTx(ctx, tx, repository, scope, assignment.AttemptRoot)
	if err != nil {
		return err
	}
	if !typed.Closed && assignment.Stage != factory.AssignmentFinished {
		return nil
	}
	return transitionLockedAttemptRootTx(ctx, tx, typed, false, now)
}

func settleTypedDeliveryRootTx(ctx context.Context, tx *sql.Tx, repository, issue int64, assignment factory.Assignment, now time.Time) error {
	if err := assignment.Validate(); err != nil {
		return err
	}
	if assignment.Repository != repository || issue != 0 || assignment.Issue != 0 || assignment.Subject.Delivery == nil {
		return errors.New("delivery run view does not match its typed scope")
	}
	typed, err := lockAttemptAllowanceRootTx(ctx, tx, repository, 0, assignment.AttemptRoot)
	if err != nil {
		return err
	}
	if typed.Delivery == nil || *typed.Delivery != *assignment.Subject.Delivery {
		return ErrAttemptRootMismatch
	}
	if !typed.Closed && assignment.Stage != factory.AssignmentFinished {
		return nil
	}
	return transitionLockedAttemptRootTx(ctx, tx, typed, false, now)
}

// lockedLatestAttemptTx reads the latest assignment for one
// repository issue.
func lockedLatestAttemptTx(ctx context.Context, tx *sql.Tx, repository, issue int64) (factory.Assignment, string, error) {
	var latestData []byte
	var latestID string
	var latest factory.Assignment
	err := tx.QueryRowContext(ctx, `SELECT id,data FROM factory_assignments
		WHERE repository=$1 AND issue=$2 ORDER BY seq DESC LIMIT 1`, repository, issue).Scan(&latestID, &latestData)
	if err != nil {
		return latest, "", err
	}
	if err = json.Unmarshal(latestData, &latest); err != nil {
		return latest, "", err
	}
	return latest, latestID, nil
}

// latestReservationReleasedTx reports whether the latest attempt's
// reservation released its capacity.
func latestReservationReleasedTx(ctx context.Context, tx *sql.Tx, latestID string) (bool, error) {
	var reservationState string
	err := tx.QueryRowContext(ctx, `SELECT state FROM factory_reservations WHERE assignment=$1`, latestID).Scan(&reservationState)
	if err == nil {
		return reservationState == factory.ReservationReleased, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

// accumulatePublicationSettleTx folds one publication's terminal
// effect into the joint-settlement flags.
func accumulatePublicationSettleTx(ctx context.Context, tx *sql.Tx, owner string, flags *settleFlags) error {
	var pubData []byte
	err := tx.QueryRowContext(ctx, `SELECT data FROM factory_publications WHERE assignment=$1`, owner).Scan(&pubData)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var publication factory.Publication
	if err = json.Unmarshal(pubData, &publication); err != nil {
		return err
	}
	switch publication.Stage {
	case factory.PublicationFailed, factory.PublicationFenced, factory.PublicationWithdrawn:
		flags.transition = true
		if publication.Reason != factory.PublishReasonSuperseded && publication.Reason != factory.PublishReasonWithdrawn {
			flags.closeRoot = true
		}
	}
	return nil
}

// accumulateMergeSettleTx folds one merge's terminal effect into the
// joint-settlement flags.
func accumulateMergeSettleTx(ctx context.Context, tx *sql.Tx, owner string, flags *settleFlags) error {
	var mergeData []byte
	err := tx.QueryRowContext(ctx, `SELECT data FROM factory_merges WHERE publication=$1`, owner).Scan(&mergeData)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var merge factory.Merge
	if err = json.Unmarshal(mergeData, &merge); err != nil {
		return err
	}
	if merge.Stage != factory.MergeOpen {
		flags.transition = true
		if merge.Stage == factory.MergeMerged || merge.Stage == factory.MergeFenced ||
			merge.Stage == factory.MergeFailed && merge.Reason != factory.MergeReasonSuperseded && merge.Reason != factory.MergeReasonWithdrawn {
			flags.closeRoot = true
		}
	}
	return nil
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
	rows, err := s.db.QueryContext(ctx, `SELECT id,type,target,principal,digest,payload,outcome,created,finished FROM factory_commands WHERE finished='' AND type<>$2 ORDER BY created,id LIMIT $1`,
		min(max(limit, 1), 101), factory.CommandDeliveryRepair)
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
