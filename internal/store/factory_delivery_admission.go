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

type DeliveredIntentCandidate struct {
	Revision   factory.IntentRevision
	FirstIssue int64
}

func (s *Store) DeliveredIntentCandidates(ctx context.Context, limit int) ([]DeliveredIntentCandidate, error) {
	if limit < 1 || limit > 64 {
		return nil, errors.New("invalid delivery page")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.data FROM factory_intent_heads h JOIN factory_intent_revisions r ON r.id=h.revision WHERE r.data->>'state'='accepted' ORDER BY h.id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	var revisions []factory.IntentRevision
	for rows.Next() {
		var raw []byte
		var revision factory.IntentRevision
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(raw, &revision); err != nil {
			rows.Close()
			return nil, err
		}
		revisions = append(revisions, revision)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	var candidates []DeliveredIntentCandidate
	for _, revision := range revisions {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		issue, err := deliveredIntentTasksTx(ctx, tx, revision)
		tx.Rollback()
		if err == nil {
			candidates = append(candidates, DeliveredIntentCandidate{Revision: revision, FirstIssue: issue})
		}
	}
	return candidates, nil
}

func (s *Store) DeliveryAlreadyAssigned(ctx context.Context, repository int64, delivery string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_assignments WHERE repository=$1 AND data->'subject'->'delivery'->>'delivery_id'=$2)`, repository, delivery).Scan(&exists)
	return exists, err
}

func validateDeliveryRootTx(ctx context.Context, tx *sql.Tx, a factory.Assignment) error {
	if a.Phase != project.ExecutionVerify || a.Role != project.RoleReviewer || a.Issue != 0 || a.Acceptance != "" || a.ParentAssignment != "" || a.PublicationAssignment != "" || a.AttemptRoot != a.ID || a.Subject.Delivery == nil {
		return errors.New("application verification requires its independent delivery root")
	}
	var head string
	if err := tx.QueryRowContext(ctx, `SELECT revision FROM factory_intent_heads WHERE id=$1 FOR UPDATE`, a.Subject.Delivery.IntentID).Scan(&head); err != nil {
		return err
	}
	if head != a.Subject.Delivery.AcceptedRevision {
		return ErrStaleRevision
	}
	if _, _, err := applicationVerificationInputsTx(ctx, tx, a); err != nil {
		return err
	}
	var prior string
	err := tx.QueryRowContext(ctx, `SELECT id FROM factory_assignments WHERE repository=$1 AND data->'subject'->'delivery'->>'delivery_id'=$2 LIMIT 1`, a.Repository, a.Subject.Delivery.DeliveryID).Scan(&prior)
	if err == nil {
		return factory.ErrAttemptClosed
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err = validateDeliveryRetryPacketTx(ctx, tx, a); err != nil {
		return err
	}
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT p.data FROM project_preparations p JOIN project_requirement_heads r ON r.project_id=p.project_id AND r.decision=p.requirements JOIN project_approval_heads h ON h.project_id=p.project_id AND h.decision=p.approval WHERE p.id=$1 FOR SHARE OF p,r,h`, a.Preparation).Scan(&raw); err != nil {
		return err
	}
	preparation, _, err := unmarshalCandidatePreparation(raw)
	if err != nil {
		return err
	}
	if preparation.Preparation.Role != project.RoleReviewer || preparation.Preparation.SourceCommit != a.SourceCommit || !preparation.State.Ready || factory.DeliveryPreparationDigest(preparation.Preparation) != a.Subject.Delivery.PreparationDigest {
		return errors.New("verification preparation changed")
	}
	return nil
}

func insertDeliveryAttemptAllowanceTx(ctx context.Context, tx *sql.Tx, a factory.Assignment, minutes int, deadline, now time.Time) error {
	allowance := factory.AttemptAllowance{Limits: factory.AttemptLimits{ActiveMinutes: minutes}, Delivery: a.Subject.Delivery, RootAssignment: a.ID, Repository: a.Repository, Issue: 0, CheckpointUnix: now.Unix(), Active: true}
	if err := allowance.Validate(); err != nil {
		return err
	}
	remaining := allowance.RemainingSeconds(now)
	if remaining <= 0 || !deadline.After(now) || deadline.After(time.Unix(now.Unix()+remaining, 0)) {
		return factory.ErrAttemptTimeExhausted
	}
	raw, err := json.Marshal(allowance)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO factory_attempt_allowances(repository,issue,root_assignment,revision,data) VALUES($1,0,$2,0,$3)`, a.Repository, a.ID, string(raw))
	return err
}

func (s *Store) RecordDeliveryAdmissionWait(ctx context.Context, revision factory.IntentRevision, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = recordFactoryFeedEventTx(ctx, tx, revision.Repository, 0, "", revision.ID+":"+factory.MCPInputDigest([]byte(reason)), "verification:waiting", factory.FeedCommunication{ComponentID: "F09", Title: "Application verification needs evidence", Summary: feedText(reason), Severity: "warning"}, 0); err != nil {
		return err
	}
	return tx.Commit()
}
