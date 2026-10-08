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
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE repository=$1 AND issue=$2 ORDER BY seq DESC LIMIT 1`, repository, issue).Scan(&data)
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
	if err := json.Unmarshal([]byte(cmd.Payload), &payload); err != nil || payload.PriorRun != decision.Prior || (payload.PriorOutcome != string(factory.Failed) && payload.PriorOutcome != string(factory.Cancelled)) {
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
	err := s.db.QueryRowContext(ctx, `SELECT c.id,c.outcome FROM factory_assignments a
		JOIN factory_commands c ON c.type=$1 AND c.target='run/'||a.run AND c.finished<>''
		WHERE a.repository=$2 AND a.issue=$3 AND a.stage='finished' AND a.data->>'acceptance'=$4
		AND a.data->>'role'=$5 AND a.data->>'outcome' IN ($6,$7)
		AND a.id=(SELECT latest.id FROM factory_assignments latest WHERE latest.repository=$2 AND latest.issue=$3 ORDER BY latest.seq DESC LIMIT 1)
		AND NOT EXISTS(SELECT 1 FROM factory_assignments consumed WHERE consumed.id=c.id)
		ORDER BY c.created,c.id LIMIT 1`, factory.CommandRetry, repository, issue, acceptance,
		project.RoleCoder, string(factory.Failed), string(factory.Cancelled)).Scan(&id, &outcome)
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
func (s *Store) RecordExplicitRetryPacket(ctx context.Context, decision factory.RetryDecision, d factory.DispatchRegistration, a factory.Assignment, r factory.Reservation, run factory.Run, view factory.RunView) error {
	if err := decision.Validate(); err != nil || a.ID != decision.CommandID || a.Role != project.RoleCoder {
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
	if priorPayload.PriorRun != decision.Prior || (priorPayload.PriorOutcome != string(factory.Failed) && priorPayload.PriorOutcome != string(factory.Cancelled)) {
		return errExplicitRetryIneligible
	}
	if err = validateExplicitRetryFactsTx(ctx, tx, decision.Prior, a.Repository, a.Issue, a.Acceptance, priorPayload.PriorOutcome); err != nil {
		return err
	}
	if err = registerDispatchTx(ctx, tx, d); err != nil {
		return err
	}
	if err = recordDispatchPacketTx(ctx, tx, d, a, r, run, view); err != nil {
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
	var assignmentData []byte
	err := tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE repository=$1 AND issue=$2 ORDER BY seq DESC LIMIT 1 FOR UPDATE`, repository, issue).Scan(&assignmentData)
	if err != nil {
		return err
	}
	var assignment factory.Assignment
	if err = json.Unmarshal(assignmentData, &assignment); err != nil {
		return err
	}
	if assignment.Run != priorRun || assignment.Acceptance != acceptance || assignment.Stage != factory.AssignmentFinished || assignment.Role != project.RoleCoder || string(assignment.Outcome) != priorOutcome || (priorOutcome != string(factory.Failed) && priorOutcome != string(factory.Cancelled)) {
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
	if !run.Reconciled || run.Role != project.RoleCoder || string(run.Outcome) != priorOutcome || run.ProjectID != assignment.ProjectID {
		return errExplicitRetryIneligible
	}
	var linked bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_run_views WHERE run=$1 AND repository=$2 AND issue=$3 AND attempt=$4)`, priorRun, repository, issue, assignment.ID).Scan(&linked); err != nil {
		return err
	}
	if !linked {
		return errExplicitRetryIneligible
	}
	var state string
	var reservationData []byte
	if err = tx.QueryRowContext(ctx, `SELECT state,data FROM factory_reservations WHERE assignment=$1 FOR UPDATE`, assignment.ID).Scan(&state, &reservationData); err != nil {
		return err
	}
	var reservation factory.Reservation
	if err = json.Unmarshal(reservationData, &reservation); err != nil {
		return err
	}
	if err = reservation.Validate(); err != nil || reservation.AssignmentID != assignment.ID || reservation.Repository != repository || reservation.Connection != assignment.Connection || reservation.State != state {
		return errExplicitRetryIneligible
	}
	var usageData []byte
	err = tx.QueryRowContext(ctx, `SELECT data FROM factory_usage WHERE run=$1`, priorRun).Scan(&usageData)
	if err == nil {
		var usage factory.Usage
		if err = json.Unmarshal(usageData, &usage); err != nil {
			return err
		}
		if err = usage.Validate(); err != nil || usage.RunID != priorRun || usage.Repository != repository || usage.Connection != assignment.Connection || !usage.StartedAt.Equal(run.Started) || reservation.State != factory.ReservationConsumed {
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
