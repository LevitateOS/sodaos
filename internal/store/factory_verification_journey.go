package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func verificationJourneyScopeTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability) (factory.Assignment, factory.IntentRevision, error) {
	var a factory.Assignment
	var r factory.IntentRevision
	if cap.Profile != factory.MCPProfileTestPilot {
		return a, r, errors.New("verification scope requires Test Pilot authority")
	}
	var recorded factory.MCPCapability
	var state string
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT state,data FROM factory_mcp_capabilities WHERE id=$1 FOR SHARE`, cap.ID).Scan(&state, &raw); err != nil {
		return a, r, err
	}
	if json.Unmarshal(raw, &recorded) != nil || state != factory.MCPCapabilityActive || recorded.ScopeDigest() != cap.ScopeDigest() || recorded.Expired(time.Now()) {
		return a, r, errors.New("verification scope capability is no longer active")
	}
	if err := contextJSONRow(ctx, tx, &a, `SELECT data FROM factory_assignments WHERE id=$1 FOR SHARE`, cap.AssignmentID); err != nil {
		return a, r, err
	}
	if a.ID != cap.AssignmentID || a.Subject.Delivery == nil || a.Phase != project.ExecutionVerify || a.Role != project.RoleReviewer || a.Issue != 0 || a.Run != cap.RunID || a.ProjectID != cap.Project || a.Repository != cap.Repository || a.Subject.SubjectKey(a.Repository) != cap.Subject || a.SourceCommit != cap.SourceDigest || strconv.FormatInt(a.NativeRev, 10) != cap.AcceptedRevision {
		return a, r, errors.New("verification scope differs from the admitted worker")
	}
	r, _, err := applicationVerificationPrerequisitesTx(ctx, tx, a)
	return a, r, err
}

// The submission, its observed journey and its attributed feed post commit
// together. This appends a report only; F10 and terminal results have other owners.
func recordMCPVerificationJourneyTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, sub factory.MCPSubmission, receipt *factory.MCPReceipt) error {
	if sub.Tool != factory.MCPToolRecordJourneyResult {
		return nil
	}
	o, err := factory.DecodeVerificationJourney(sub.Payload)
	if err != nil {
		return err
	}
	a, r, err := verificationJourneyScopeTx(ctx, tx, cap)
	if err != nil {
		return err
	}
	if err := o.ValidateFor(cap, a, r.EffectiveOutcomes); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_mcp_submissions WHERE assignment=$1 AND tool=$2`, a.ID, string(sub.Tool)).Scan(&count); err != nil {
		return err
	}
	if count >= factory.MaxVerificationJourneys {
		return errors.New("journey observation limit reached; finish remains available")
	}
	content := factory.FeedCommunication{Kind: "journey", ComponentID: "TP03", Title: o.Name, Summary: "Recorded journey observation: " + o.Result + ". Application verification is unchanged.", Scope: o.DeliveryID, Before: o.Expected, After: o.Observed, EvidenceRefs: append([]string(nil), o.EvidenceRefs...), Remaining: append([]string(nil), o.Limitations...)}
	for i, step := range o.Steps {
		content.Items = append(content.Items, factory.FeedCommunicationItem{Label: fmt.Sprintf("%d. %s", i+1, step), State: "reported"})
	}
	content.EvidenceRefs = append(content.EvidenceRefs, o.SourceCommit, "journey:"+sub.SubmissionID)
	for _, id := range o.ArtifactIDs {
		var owned bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_feed_artifacts WHERE id=$1 AND repository=$2 AND assignment=$3 AND complete)`, id, cap.Repository, a.ID).Scan(&owned); err != nil {
			return err
		}
		if !owned {
			return errors.New("journey artifact is not a complete artifact of this assignment")
		}
		content.Artifacts = append(content.Artifacts, factory.FeedArtifact{ID: id})
	}
	if err := resolveCommunicationArtifactsTx(ctx, tx, cap.Repository, &content); err != nil {
		return err
	}
	if err := factory.ValidateFeedComponent(content.ComponentID, cap.Profile, sub.Tool); err != nil {
		return err
	}
	return recordMCPFeedContentTx(ctx, tx, cap, sub, content, receipt)
}

// VerificationScope reads retained observations in canonical feed sequence.
// Keyset pages do not run models, append records, or change workflow state.
func (s *Store) VerificationScope(ctx context.Context, cap factory.MCPCapability, after int64) (factory.VerificationScopeView, error) {
	var out factory.VerificationScopeView
	if after < 0 {
		return out, errors.New("invalid journey observation cursor")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	a, r, err := verificationJourneyScopeTx(ctx, tx, cap)
	if err != nil {
		return out, err
	}
	out = factory.VerificationScopeView{Assignment: a.ID, Run: a.Run, InputDigest: cap.ScopeDigest(), Delivery: *a.Subject.Delivery, OutcomeIDs: append([]string(nil), r.EffectiveOutcomes...), Records: []factory.VerificationJourneyRecord{}, Limit: factory.VerificationJourneyPageSize, Maximum: factory.MaxVerificationJourneys}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_mcp_submissions WHERE assignment=$1 AND tool=$2`, a.ID, string(factory.MCPToolRecordJourneyResult)).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT s.submission,s.receipt->>'feed_post_id',p.seq,s.payload FROM factory_mcp_submissions s JOIN factory_feed_comments p ON p.id=s.receipt->>'feed_post_id' AND p.repository=$2 WHERE s.assignment=$1 AND s.tool=$3 AND p.seq>$4 ORDER BY p.seq LIMIT $5`, a.ID, a.Repository, string(factory.MCPToolRecordJourneyResult), after, factory.VerificationJourneyPageSize+1)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var record factory.VerificationJourneyRecord
		var raw []byte
		if err := rows.Scan(&record.ID, &record.FeedPostID, &record.Sequence, &raw); err != nil {
			rows.Close()
			return out, err
		}
		if len(out.Records) == out.Limit {
			out.NextAfter = out.Records[len(out.Records)-1].Sequence
			break
		}
		record.Observation, err = factory.DecodeVerificationJourney(raw)
		if err != nil || record.Observation.ValidateFor(cap, a, r.EffectiveOutcomes) != nil {
			rows.Close()
			return out, errors.New("recorded journey differs from its admitted scope")
		}
		out.Records = append(out.Records, record)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	// Missing atomic feed projections must not appear as a complete empty log.
	if after == 0 && out.NextAfter == 0 && len(out.Records) != out.Total {
		return out, errors.New("journey observation projection is incomplete")
	}
	out.Complete = out.NextAfter == 0
	if err := tx.Commit(); err != nil {
		return out, err
	}
	return out, nil
}
