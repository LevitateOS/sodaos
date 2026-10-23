package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// PendingAppVerifications includes finished Test Pilot reports until a trusted
// whole-app completion is recorded. Worker receipts alone never remove them.
func (s *Store) PendingAppVerifications(ctx context.Context, limit int) ([]factory.Assignment, error) {
	if limit < 1 || limit > 64 {
		return nil, errors.New("invalid verification page")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.data FROM factory_assignments a WHERE a.stage='finished'
		AND a.data->>'phase'='verify' AND a.data->>'outcome'='succeeded'
		AND a.data->'result'->'phase_result'->'output'->'verification'->>'outcome'='verified'
		AND NOT EXISTS(SELECT 1 FROM factory_feed_comments p WHERE p.repository=a.repository AND p.data->>'event_identity'=a.data->'subject'->'delivery'->>'delivery_id' AND p.data->'communication'->>'component_id'='F10')
		ORDER BY a.seq LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var assignments []factory.Assignment
	for rows.Next() {
		var raw []byte
		var a factory.Assignment
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &a); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

func applicationVerificationInputsTx(ctx context.Context, tx *sql.Tx, a factory.Assignment) (factory.IntentRevision, int64, error) {
	var revision factory.IntentRevision
	if a.Phase != project.ExecutionVerify || a.Subject.Delivery == nil || a.Subject.Delivery.Validate() != nil || a.SourceCommit != a.Subject.Delivery.SourceCommit {
		return revision, 0, errors.New("verification lacks its exact delivery subject")
	}
	subject := a.Subject.Delivery
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT r.data FROM factory_intent_heads h JOIN factory_intent_revisions r ON r.id=h.revision WHERE h.id=$1 AND h.revision=$2 FOR SHARE OF h`, subject.IntentID, subject.AcceptedRevision).Scan(&raw); err != nil {
		return revision, 0, err
	}
	if err := json.Unmarshal(raw, &revision); err != nil {
		return revision, 0, err
	}
	if revision.Repository != a.Repository || revision.State != "accepted" || revision.Validate() != nil || revision.Digest() != subject.ScopeDigest {
		return revision, 0, errors.New("accepted application scope changed")
	}
	firstIssue, err := deliveredIntentTasksTx(ctx, tx, revision)
	return revision, firstIssue, err
}

func deliveredIntentTasksTx(ctx context.Context, tx *sql.Tx, revision factory.IntentRevision) (int64, error) {
	var raw []byte
	tasks := revision.Tasks
	err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_generations WHERE revision=$1 FOR SHARE`, revision.ID).Scan(&raw)
	if err == nil {
		var generation factory.IntentGeneration
		if err := json.Unmarshal(raw, &generation); err != nil {
			return 0, err
		}
		if generation.State != "completed" || generation.RevisionID != revision.ID || generation.IntentID != revision.Intent || generation.IntentDigest != revision.Digest() {
			return 0, errors.New("accepted application generation is not complete and current")
		}
		if err := factory.AssessProposedTasks(revision, generation.SubmittedTasks, revision.MaxIssues); err != nil {
			return 0, err
		}
		// Native adoption binds the recorded assessed graph, whose task keys
		// can differ from the owner's accepted planning sketch.
		tasks = generation.SubmittedTasks
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	var firstIssue int64
	for _, task := range tasks {
		var binding factory.IntentTaskBinding
		if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_tasks WHERE revision=$1 AND task=$2`, revision.ID, task.Key).Scan(&raw); err != nil {
			return 0, err
		}
		if err := json.Unmarshal(raw, &binding); err != nil {
			return 0, err
		}
		if binding.RevisionID != revision.ID || binding.TaskKey != task.Key || binding.IssueID <= 0 || binding.IssueIndex <= 0 || !project.ValidDecisionID(binding.AcceptanceID) {
			return 0, errors.New("accepted application task lacks its exact native binding")
		}
		var delivered bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_completions WHERE repository=$1 AND data->>'issue'=$2 AND data->>'acceptance_id'=$3)`, revision.Repository, fmt.Sprint(binding.IssueIndex), binding.AcceptanceID).Scan(&delivered); err != nil {
			return 0, err
		}
		if !delivered {
			return 0, errors.New("accepted application work is not integrated")
		}
		if firstIssue == 0 {
			firstIssue = binding.IssueIndex
		}
	}
	if firstIssue == 0 {
		return 0, errors.New("accepted application has no delivered work")
	}
	return firstIssue, nil
}

func (s *Store) ApplicationVerificationInputs(ctx context.Context, a factory.Assignment) (factory.IntentRevision, int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.IntentRevision{}, 0, err
	}
	defer tx.Rollback()
	revision, issue, err := applicationVerificationInputsTx(ctx, tx, a)
	if err != nil {
		return revision, issue, err
	}
	return revision, issue, tx.Commit()
}

// RecordVerifiedApplication is called only after independent native observation
// of the approved check. Recheck current authority and delivered scope inside
// the same transaction as the F10 event.
func (s *Store) RecordVerifiedApplication(ctx context.Context, a factory.Assignment, manifest factory.ApprovedAppCheck, state project.PrepareState) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	revision, _, err := applicationVerificationInputsTx(ctx, tx, a)
	if err != nil {
		return err
	}
	var currentRaw []byte
	if err = tx.QueryRowContext(ctx, `SELECT data FROM factory_assignments WHERE id=$1 AND stage='finished' FOR SHARE`, a.ID).Scan(&currentRaw); err != nil {
		return err
	}
	var current factory.Assignment
	if err = json.Unmarshal(currentRaw, &current); err != nil {
		return err
	}
	if current.Result == nil || a.Result == nil || factory.ResultDigest(*current.Result) != factory.ResultDigest(*a.Result) || current.Run != a.Run {
		return ErrStaleRevision
	}
	if err = tx.QueryRowContext(ctx, `SELECT p.data FROM project_preparations p JOIN project_requirement_heads r ON r.project_id=p.project_id AND r.decision=p.requirements JOIN project_approval_heads h ON h.project_id=p.project_id AND h.decision=p.approval WHERE p.id=$1 FOR SHARE OF p,r,h`, a.Preparation).Scan(&currentRaw); err != nil {
		return err
	}
	preparation, _, err := unmarshalCandidatePreparation(currentRaw)
	if err != nil {
		return err
	}
	if !manifest.BindsIntent(revision) || manifest.Digest() != a.Subject.Delivery.VerificationManifestDigest || factory.DeliveryPreparationDigest(preparation.Preparation) != a.Subject.Delivery.PreparationDigest || state.ID != a.Preparation || state.Project != a.ProjectID || state.Role != project.RoleReviewer || state.SourceCommit != a.SourceCommit || state.SetupDigest != preparation.Preparation.SetupDigest || !state.Ready || state.Stopped || state.Phase != project.PrepareReady || state.CheckExit == nil || *state.CheckExit != 0 {
		return errors.New("trusted application check does not match the delivery")
	}
	var approval project.ApprovalDecision
	if err = tx.QueryRowContext(ctx, `SELECT data FROM project_approval_decisions WHERE id=$1`, preparation.Preparation.Approval.ID).Scan(&currentRaw); err != nil {
		return err
	}
	if err = json.Unmarshal(currentRaw, &approval); err != nil {
		return err
	}
	if !approval.Verified || approval.Requirement != preparation.Preparation.Requirements.ID {
		return errors.New("application check approval is not current and verified")
	}
	items := make([]factory.FeedCommunicationItem, 0, len(manifest.Outcomes))
	for _, outcome := range manifest.Outcomes {
		label := outcome
		for _, idea := range revision.Ideas {
			if idea.ID == outcome {
				label = feedText(idea.Description)
				break
			}
		}
		items = append(items, factory.FeedCommunicationItem{Label: label, State: "passed"})
	}
	if len(items) > feedPayloadCap {
		items = items[:feedPayloadCap]
	}
	if err = recordFactoryFeedEventTx(ctx, tx, a.Repository, 0, a.ID, a.Subject.Delivery.DeliveryID, "application:verified", factory.FeedCommunication{
		ComponentID: "F10", Title: "Application outcomes verified", Summary: fmt.Sprintf("All %d accepted outcomes passed the approved application check at source %s.", len(manifest.Outcomes), a.SourceCommit[:12]), Items: items,
		EvidenceRefs: []string{a.SourceCommit, manifest.Digest(), factory.MCPInputDigest([]byte(state.Output))},
	}, time.Now().Unix()); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RecordAppVerificationWait(ctx context.Context, a factory.Assignment, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = recordFactoryFeedEventTx(ctx, tx, a.Repository, 0, a.ID, a.ID+":"+factory.MCPInputDigest([]byte(reason)), "verification:waiting", factory.FeedCommunication{ComponentID: "F09", Title: "Application verification needs evidence", Summary: feedText(reason), Severity: "warning"}, 0); err != nil {
		return err
	}
	return tx.Commit()
}
