package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

var errExplicitRetryIneligible = errors.New("explicit retry is no longer dispatchable")

// LatestIssueAssignment reads the newest recorded assignment directly; old
// issue history stays bounded and cannot hide a newer active or terminal row.
func (s *Store) LatestIssueAssignment(ctx context.Context, repository, issue int64) (factory.Assignment, error) {
	var assignment factory.Assignment
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE repository=$1 AND issue=$2 AND data->>'publication_assignment'=id ORDER BY seq DESC LIMIT 1`, repository, issue).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &assignment)
	}
	return assignment, err
}

// RecordExplicitRetryCommand atomically records a retry intent and its
// immutable receipt after verifying the failed assignment still names the
// issue's current accepted head and its prior run is accounted or confirmed
// unused.
func (s *Store) RecordExplicitRetryCommand(ctx context.Context, cmd factory.Command, decision factory.RetryDecision, repository, issue int64, acceptance string, now time.Time) (factory.Command, bool, error) {
	if err := cmd.Validate(); err != nil {
		return factory.Command{}, false, err
	}
	if err := decision.Validate(); err != nil || cmd.ID != decision.CommandID || cmd.Type != factory.CommandRetry || cmd.Target != "run/"+decision.Prior || repository <= 0 || issue <= 0 || acceptance == "" {
		return factory.Command{}, false, errExplicitRetryIneligible
	}
	// The payload includes the immutable prior outcome. Run truth below
	// verifies that value before the retry receipt is committed.
	var payload struct {
		PriorRun     string `json:"prior_run"`
		PriorOutcome string `json:"prior_outcome"`
	}
	if err := json.Unmarshal([]byte(cmd.Payload), &payload); err != nil || payload.PriorRun != decision.Prior || !retryableRunOutcome(payload.PriorOutcome) {
		return factory.Command{}, false, errExplicitRetryIneligible
	}
	if cmd.Digest != factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload) {
		return factory.Command{}, false, ErrCommandConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.Command{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, _, _, err = ensureDispatchGateTx(ctx, tx, repository); err != nil {
		return factory.Command{}, false, err
	}
	stored, created, err := recordFactoryCommandTx(ctx, tx, cmd, now)
	if err != nil || !created {
		if err != nil {
			return factory.Command{}, false, err
		}
		if err = tx.Commit(); err != nil {
			return factory.Command{}, false, err
		}
		return stored, false, nil
	}
	var collision bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_assignments WHERE id=$1)
		OR EXISTS(SELECT 1 FROM factory_dispatch_regs WHERE id=$1)`, cmd.ID).Scan(&collision); err != nil {
		return factory.Command{}, false, err
	}
	if collision {
		return factory.Command{}, false, errExplicitRetryIneligible
	}
	if err = validateExplicitRetryFactsTx(ctx, tx, decision.Prior, repository, issue, acceptance, payload.PriorOutcome); err != nil {
		return factory.Command{}, false, err
	}
	outcome, err := json.Marshal(decision)
	if err != nil {
		return factory.Command{}, false, err
	}
	if err = finishFactoryCommandTx(ctx, tx, cmd.ID, string(outcome), now); err != nil {
		return factory.Command{}, false, err
	}
	stored.Outcome = string(outcome)
	stored.Finished = now.UTC().Format(time.RFC3339Nano)
	if err = tx.Commit(); err != nil {
		return factory.Command{}, false, err
	}
	return stored, true, nil
}

// PendingExplicitRetry returns one finished retry command for the latest
// finished assignment on the issue's unchanged acceptance. Its command ID
// becomes the fresh assignment identity, so a recorded packet consumes it.
func (s *Store) PendingExplicitRetry(ctx context.Context, repository, issue int64, acceptance string) (factory.RetryDecision, error) {
	var id, outcome string
	err := s.db.QueryRowContext(ctx, `SELECT c.id,c.outcome FROM factory_assignments owner
		JOIN factory_commands c ON c.type=$1 AND c.finished<>''
		JOIN factory_runs prior ON c.target='run/'||prior.id
		JOIN factory_run_views v ON v.run=prior.id
		JOIN factory_assignments source ON source.id=v.attempt
		WHERE owner.repository=$2 AND owner.issue=$3 AND owner.stage='finished' AND owner.data->>'acceptance'=$4
		AND owner.data->>'publication_assignment'=owner.id
		AND owner.id=(SELECT latest.id FROM factory_assignments latest WHERE latest.repository=$2 AND latest.issue=$3 AND latest.data->>'publication_assignment'=latest.id ORDER BY latest.seq DESC LIMIT 1)
		AND source.repository=$2 AND source.issue=$3 AND source.stage='finished' AND source.data->>'acceptance'=$4
		AND source.data->>'attempt_root'=owner.data->>'attempt_root' AND source.data->>'publication_assignment'=owner.id
		AND prior.settled AND COALESCE(prior.data->>'reconciled','false')='true'
		AND prior.data->>'outcome' IN ($5,$6,$7)
		AND ((source.id=owner.id AND source.data->>'role'=$8 AND source.data->>'outcome' IN ($6,$7))
			OR EXISTS(SELECT 1 FROM factory_attempt_allowances allowance
				WHERE allowance.repository=$2 AND allowance.issue=$3 AND allowance.root_assignment=owner.id
				AND COALESCE((allowance.data->>'closed')::boolean,FALSE)
				AND NOT COALESCE((allowance.data->>'active')::boolean,FALSE)))
		AND NOT EXISTS(SELECT 1 FROM factory_assignments consumed WHERE consumed.id=c.id)
		ORDER BY c.created,c.id LIMIT 1`, factory.CommandRetry, repository, issue, acceptance,
		string(factory.Succeeded), string(factory.Failed), string(factory.Cancelled), project.RoleCoder).Scan(&id, &outcome)
	if err != nil {
		return factory.RetryDecision{}, err
	}
	var decision factory.RetryDecision
	if err = json.Unmarshal([]byte(outcome), &decision); err != nil {
		return factory.RetryDecision{}, err
	}
	if err = decision.Validate(); err != nil || decision.CommandID != id {
		return factory.RetryDecision{}, errExplicitRetryIneligible
	}
	return decision, nil
}

// RecordExplicitRetryPacket serializes competing dispatchers on the gate and
// command, rechecks prior accounting and the accepted head, then commits the
// retry assignment/run/reservation through the regular admission checks.
func (s *Store) RecordExplicitRetryPacket(ctx context.Context, decision factory.RetryDecision, d factory.DispatchRegistration, expected factory.IssueControl, a factory.Assignment, r factory.Reservation, run factory.Run, view factory.RunView) error {
	if err := decision.Validate(); err != nil || a.ID != decision.CommandID || a.Role != project.RoleCoder || a.AttemptRoot != a.ID || a.PublicationAssignment != a.ID {
		return errExplicitRetryIneligible
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, _, _, err = ensureDispatchGateTx(ctx, tx, a.Repository); err != nil {
		return err
	}
	var collision bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_assignments WHERE id=$1)
		OR EXISTS(SELECT 1 FROM factory_dispatch_regs WHERE id=$1)`, a.ID).Scan(&collision); err != nil {
		return err
	}
	if collision {
		return ErrAssignmentActive
	}
	var typ, target, payload, outcome, finished string
	if err = tx.QueryRowContext(ctx, `SELECT type,target,payload,outcome,finished FROM factory_commands WHERE id=$1 FOR UPDATE`, decision.CommandID).Scan(&typ, &target, &payload, &outcome, &finished); err != nil {
		return err
	}
	if typ != factory.CommandRetry || target != "run/"+decision.Prior || finished == "" {
		return errExplicitRetryIneligible
	}
	var storedDecision factory.RetryDecision
	if err = json.Unmarshal([]byte(outcome), &storedDecision); err != nil {
		return err
	}
	if storedDecision != decision {
		return ErrCommandConflict
	}
	var priorPayload struct {
		PriorRun     string `json:"prior_run"`
		PriorOutcome string `json:"prior_outcome"`
	}
	if err = json.Unmarshal([]byte(payload), &priorPayload); err != nil {
		return err
	}
	if priorPayload.PriorRun != decision.Prior || !retryableRunOutcome(priorPayload.PriorOutcome) {
		return errExplicitRetryIneligible
	}
	if err = validateExplicitRetryFactsTx(ctx, tx, decision.Prior, a.Repository, a.Issue, a.Acceptance, priorPayload.PriorOutcome); err != nil {
		return err
	}
	if err = registerDispatchTx(ctx, tx, d); err != nil {
		return err
	}
	if err = recordDispatchPacketTx(ctx, tx, d, expected, a, r, run, view, true); err != nil {
		return err
	}
	return tx.Commit()
}

func validateExplicitRetryFactsTx(ctx context.Context, tx *sql.Tx, priorRun string, repository, issue int64, acceptance, priorOutcome string) error {
	var head string
	if err := tx.QueryRowContext(ctx, `SELECT decision FROM issue_acceptance_heads WHERE repository=$1 AND issue=$2 FOR UPDATE`, repository, issue).Scan(&head); err != nil {
		return err
	}
	if head != acceptance {
		return errExplicitRetryIneligible
	}
	var priorAssignmentData []byte
	err := tx.QueryRowContext(ctx, `SELECT a.data FROM factory_assignments a
		JOIN factory_run_views v ON v.attempt=a.id WHERE v.run=$1 AND v.repository=$2 AND v.issue=$3 FOR UPDATE OF a`, priorRun, repository, issue).Scan(&priorAssignmentData)
	if err != nil {
		return err
	}
	var priorAssignment factory.Assignment
	if err = json.Unmarshal(priorAssignmentData, &priorAssignment); err != nil {
		return err
	}
	if err = priorAssignment.Validate(); err != nil {
		return errExplicitRetryIneligible
	}
	if priorAssignment.Run != priorRun || priorAssignment.Acceptance != acceptance || priorAssignment.Stage != factory.AssignmentFinished ||
		!project.ValidFactoryRole(priorAssignment.Role) || string(priorAssignment.Outcome) != priorOutcome || !retryableRunOutcome(priorOutcome) ||
		priorAssignment.Repository != repository || priorAssignment.Issue != issue {
		return errExplicitRetryIneligible
	}
	var latestData []byte
	err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE repository=$1 AND issue=$2 ORDER BY seq DESC LIMIT 1 FOR UPDATE`, repository, issue).Scan(&latestData)
	if err != nil {
		return err
	}
	var latest factory.Assignment
	if err = json.Unmarshal(latestData, &latest); err != nil || latest.Validate() != nil || latest.ID != priorAssignment.ID {
		return errExplicitRetryIneligible
	}
	var ownerData []byte
	err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE repository=$1 AND issue=$2
		AND data->>'publication_assignment'=id ORDER BY seq DESC LIMIT 1 FOR UPDATE`, repository, issue).Scan(&ownerData)
	if err != nil {
		return err
	}
	var owner factory.Assignment
	if err = json.Unmarshal(ownerData, &owner); err != nil {
		return err
	}
	if err = owner.Validate(); err != nil || owner.Stage != factory.AssignmentFinished || owner.Role != project.RoleCoder ||
		owner.Acceptance != acceptance || owner.AttemptRoot != owner.ID || priorAssignment.AttemptRoot != owner.ID ||
		priorAssignment.PublicationAssignment != owner.ID {
		return errExplicitRetryIneligible
	}
	var runData []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_runs WHERE id=$1 FOR UPDATE`, priorRun).Scan(&runData); err != nil {
		return err
	}
	var run factory.Run
	if err = json.Unmarshal(runData, &run); err != nil {
		return err
	}
	if !run.Reconciled || run.Role != priorAssignment.Role || string(run.Outcome) != priorOutcome || run.ProjectID != priorAssignment.ProjectID {
		return errExplicitRetryIneligible
	}
	allowance, err := lockAttemptAllowanceRootTx(ctx, tx, repository, issue, owner.ID)
	if err != nil || allowance.RootAssignment != owner.ID {
		return errExplicitRetryIneligible
	}
	// Preserve the original failed/cancelled coder-owner retry path: those
	// outcomes already end the owner attempt without closing its reusable
	// allowance. A successful owner or any child retry is eligible only after
	// its root is closed.
	stoppedOwnerFailure := priorAssignment.ID == owner.ID && priorAssignment.Role == project.RoleCoder &&
		(priorOutcome == string(factory.Failed) || priorOutcome == string(factory.Cancelled))
	if allowance.Active || (!allowance.Closed && !stoppedOwnerFailure) {
		return errExplicitRetryIneligible
	}
	settled, err := attemptRootRunsSettledTx(ctx, tx, repository, issue, owner.ID)
	if err != nil {
		return err
	}
	if !settled {
		return errExplicitRetryIneligible
	}
	if err = retryOperationsSettledTx(ctx, tx, owner); err != nil {
		return err
	}
	var state string
	var reservationData []byte
	if err = tx.QueryRowContext(ctx, `SELECT state,data FROM factory_reservations WHERE assignment=$1 FOR UPDATE`, priorAssignment.ID).Scan(&state, &reservationData); err != nil {
		return err
	}
	var reservation factory.Reservation
	if err = json.Unmarshal(reservationData, &reservation); err != nil {
		return err
	}
	if err = reservation.Validate(); err != nil || reservation.AssignmentID != priorAssignment.ID || reservation.Repository != repository || reservation.Connection != priorAssignment.Connection || reservation.State != state {
		return errExplicitRetryIneligible
	}
	var usageData []byte
	err = tx.QueryRowContext(ctx, `SELECT data FROM factory_usage WHERE run=$1`, priorRun).Scan(&usageData)
	if err == nil {
		var usage factory.Usage
		if err = json.Unmarshal(usageData, &usage); err != nil {
			return err
		}
		if err = usage.Validate(); err != nil || usage.RunID != priorRun || usage.Repository != repository || usage.Connection != priorAssignment.Connection || !usage.StartedAt.Equal(run.Started) || reservation.State != factory.ReservationConsumed {
			return errExplicitRetryIneligible
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if reservation.State != factory.ReservationReleased {
		return errExplicitRetryIneligible
	}
	return nil
}

func retryableRunOutcome(outcome string) bool {
	return outcome == string(factory.Succeeded) || outcome == string(factory.Failed) || outcome == string(factory.Cancelled)
}

func retryOperationNeedsResolution(attempts int, effect, cancellation, completion string) bool {
	if attempts == 0 {
		return false
	}
	return effect == "" || effect == factory.OpEffectPending || effect == factory.OpEffectIndeterminate ||
		cancellation == factory.OpCancelPending || cancellation == factory.OpCancelIndeterminate ||
		completion == factory.OpCompletionPending || completion == factory.OpCompletionNeedsIntervention ||
		effect == factory.OpEffectCommitted && completion != factory.OpCompletionComplete
}

func retryOperationsSettledTx(ctx context.Context, tx *sql.Tx, owner factory.Assignment) error {
	pubRows, err := tx.QueryContext(ctx, `SELECT data FROM factory_publications WHERE repository=$1 AND issue=$2 ORDER BY seq LIMIT $3 FOR UPDATE`, owner.Repository, owner.Issue, storePublicationLimit+1)
	if err != nil {
		return err
	}
	publications := make([]factory.Publication, 0)
	for pubRows.Next() {
		var data []byte
		var publication factory.Publication
		if err = pubRows.Scan(&data); err != nil || json.Unmarshal(data, &publication) != nil || publication.Validate() != nil {
			_ = pubRows.Close()
			return errExplicitRetryIneligible
		}
		publications = append(publications, publication)
	}
	if err = pubRows.Err(); err != nil {
		_ = pubRows.Close()
		return err
	}
	if err = pubRows.Close(); err != nil {
		return err
	}
	if len(publications) > storePublicationLimit {
		return errExplicitRetryIneligible
	}
	for _, publication := range publications {
		if publication.Stage == factory.PublicationOpen ||
			retryOperationNeedsResolution(publication.Publish.Attempts, publication.Publish.Effect, publication.Publish.Cancellation, publication.Publish.Completion) ||
			retryOperationNeedsResolution(publication.PRCreate.Attempts, publication.PRCreate.Effect, publication.PRCreate.Cancellation, publication.PRCreate.Completion) {
			return errExplicitRetryIneligible
		}
		for _, correction := range publication.Corrections {
			if retryOperationNeedsResolution(correction.Attempts, correction.Effect, correction.Cancellation, correction.Completion) {
				return errExplicitRetryIneligible
			}
		}
		for _, review := range publication.ReviewOperations {
			if review.Outcome.OperationID == "" || retryOperationNeedsResolution(1, review.Outcome.Effect, review.Outcome.Cancellation, review.Outcome.Completion) {
				return errExplicitRetryIneligible
			}
		}
	}
	mergeRows, err := tx.QueryContext(ctx, `SELECT data FROM factory_merges WHERE repository=$1 AND issue=$2 ORDER BY seq LIMIT $3 FOR UPDATE`, owner.Repository, owner.Issue, storeMergeLimit+1)
	if err != nil {
		return err
	}
	merges := make([]factory.Merge, 0)
	for mergeRows.Next() {
		var data []byte
		var merge factory.Merge
		if err = mergeRows.Scan(&data); err != nil || json.Unmarshal(data, &merge) != nil || merge.Validate() != nil {
			_ = mergeRows.Close()
			return errExplicitRetryIneligible
		}
		merges = append(merges, merge)
	}
	if err = mergeRows.Err(); err != nil {
		_ = mergeRows.Close()
		return err
	}
	if err = mergeRows.Close(); err != nil {
		return err
	}
	if len(merges) > storeMergeLimit {
		return errExplicitRetryIneligible
	}
	for _, merge := range merges {
		if merge.Stage == factory.MergeOpen || merge.Stage == factory.MergeMerged ||
			retryOperationNeedsResolution(merge.Operation.Attempts, merge.Operation.Effect, merge.Operation.Cancellation, merge.Operation.Completion) {
			return errExplicitRetryIneligible
		}
	}
	return nil
}
