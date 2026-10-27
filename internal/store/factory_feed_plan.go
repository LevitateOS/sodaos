package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

func recordPlanningFeedTx(ctx context.Context, tx *sql.Tx, record factory.PlanningRecord, raw []byte) error {
	control := record.Input.Candidate.Control
	digest := sha256.Sum256(raw)
	recordDigest := hex.EncodeToString(digest[:])
	decision := "Implementation strategy: " + strings.ReplaceAll(record.Strategy, "_", " ")
	title := "Implementation strategy recorded"
	if record.Outcome == "unavailable" {
		title, decision = "Implementation strategy unavailable", "No implementation strategy was selected."
	} else if record.Provisional {
		title = "Provisional implementation strategy recorded"
	}
	content := factory.FeedCommunication{
		ComponentID: "F11", Title: title, Summary: decision, Decision: decision,
		Scope:          fmt.Sprintf("Issue %d · %s", control.Issue, record.Outcome),
		Rationale:      "The controller recorded this approach for the accepted scope. Execution still requires current admission; this record does not claim implementation or verification.",
		EvidenceRefs:   []string{"planning_snapshot:" + record.Snapshot, "planning_record:" + recordDigest, "acceptance:" + control.Acceptance},
		PlanningSource: &factory.FeedPlanningSource{Snapshot: record.Snapshot, RecordDigest: recordDigest, Strategy: record.Strategy, Outcome: record.Outcome, Provisional: record.Provisional, Judgment: record.Judgment, RecordedUnix: record.CreatedUnix},
	}
	if record.Judgment.Selected != "" && record.Judgment.Selected != record.Strategy {
		content.Rationale += " The controller's recorded strategy differs from the advisory selection; both are retained below."
	}
	return recordFactoryFeedEventTx(ctx, tx, control.Repository, control.Issue, "", "planning-record:"+recordDigest, "planning:"+record.Outcome, content, record.CreatedUnix)
}

// The native plan remains authoritative. Publication shares its transaction
// and exact revision identity; it does not admit the plan or claim execution.
func recordInvestigationPlanFeedTx(ctx context.Context, tx *sql.Tx, plan factory.InvestigationPlan) error {
	var repository, issue int64
	err := tx.QueryRowContext(ctx, `SELECT repository,issue FROM factory_assignments WHERE id=$1`, plan.AttemptRoot).Scan(&repository, &issue)
	if errors.Is(err, sql.ErrNoRows) {
		// Unbound historical/imported plans have no authorized feed scope.
		return nil
	}
	if err != nil {
		return err
	}
	content := factory.FeedCommunication{
		ComponentID: "F11", Title: "Architecture plan recorded", Summary: TruncateFeedSummary(plan.Approach),
		Decision: plan.Approach, Scope: fmt.Sprintf("Issue %d · plan generation %d", issue, plan.Generation),
		Rationale:    "Recorded by the native plan revision operation. Plan admission, implementation and verification remain separate records.",
		EvidenceRefs: []string{"plan:" + plan.ID, "plan_digest:" + plan.Digest, "assignment:" + plan.AttemptRoot, "commit:" + plan.SourceCommit},
		PlanSource:   &factory.FeedPlanSource{AttemptRoot: plan.AttemptRoot, Plan: plan.ID, Digest: plan.Digest, Source: plan.SourceCommit, Generation: plan.Generation},
	}
	if len(plan.Findings) > 0 {
		content.Rationale += "\n\n" + TruncateFeedSummary(strings.Join(plan.Findings, "\n"))
	}
	for _, alternative := range plan.Rejected {
		if len(content.Alternatives) == 24 {
			break
		}
		content.Alternatives = append(content.Alternatives, TruncateFeedSummary(alternative.Approach+": "+alternative.Reason))
	}
	for _, fact := range plan.Unresolved {
		if len(content.Remaining) == 24 {
			break
		}
		content.Remaining = append(content.Remaining, TruncateFeedSummary(fact))
	}
	return recordFactoryFeedEventTx(ctx, tx, repository, issue, plan.AttemptRoot, "plan-revision:"+plan.AttemptRoot+":"+plan.ID, "plan:recorded", content, 0)
}
