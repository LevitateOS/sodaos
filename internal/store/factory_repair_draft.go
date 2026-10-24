package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
)

func recordMCPRepairDraftTx(ctx context.Context, tx *sql.Tx, cap factory.MCPCapability, sub factory.MCPSubmission, receipt *factory.MCPReceipt) error {
	if sub.Tool != factory.MCPToolDraftRepair {
		return nil
	}
	draft, err := factory.DecodeRepairDraft(sub.Payload)
	if err != nil {
		return err
	}
	a, current, err := repairObservationScopeTx(ctx, tx, cap)
	if err != nil {
		return err
	}
	proposal, err := draft.ValidateFor(cap, a, current)
	if err != nil {
		return err
	}
	ids, err := draft.DiagnosisIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		var raw []byte
		var capability, input, digest string
		if err := tx.QueryRowContext(ctx, `SELECT capability,input_digest,payload_hash,payload FROM factory_mcp_submissions WHERE assignment=$1 AND submission=$2 AND tool=$3 FOR SHARE`, a.ID, id, string(factory.MCPToolRecordDiagnosis)).Scan(&capability, &input, &digest, &raw); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("draft diagnosis is not a recorded observation of this assignment")
			}
			return err
		}
		diagnosis, err := factory.DecodeRepairDiagnosis(raw)
		if err != nil || capability != cap.ID || input != cap.ScopeDigest() || digest != factory.MCPInputDigest(raw) || diagnosis.ValidateFor(cap, a, current) != nil {
			return errors.New("draft diagnosis differs from its canonical cause and input")
		}
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM factory_mcp_submissions WHERE assignment=$1 AND tool=$2`, a.ID, string(sub.Tool)).Scan(&count); err != nil {
		return err
	}
	if count >= factory.MaxRepairObservations {
		return errors.New("repair draft limit reached; finish remains available")
	}
	content := factory.FeedCommunication{Kind: "repair_draft", ComponentID: "ST06", Title: "Proposed repair graph", Summary: "Recorded draft (" + proposal.Outcome + "). Not submitted for judgment or adopted; graph and readiness are unchanged.", Scope: cap.Subject, Before: "Sealed inventory " + proposal.InventoryDigest, After: "Proposed tasks only; use preview for deterministic adoption feedback.", EvidenceRefs: append([]string{}, draft.DiagnosisRefs...)}
	content.EvidenceRefs = append(content.EvidenceRefs, proposal.SourceCommit, "input:"+cap.ScopeDigest(), "inventory:"+proposal.InventoryDigest, "draft:"+sub.SubmissionID)
	rows, abbreviated := 0, false
	appendPreview := func(item factory.FeedCommunicationItem) {
		rows++
		if len(content.Items) >= feedPayloadCap {
			return
		}
		label, detail := feedText(item.Label), feedText(item.Detail)
		abbreviated = abbreviated || label != item.Label || detail != item.Detail || len(item.DependsOn) > feedPayloadCap
		item.Label, item.Detail = label, detail
		if len(item.DependsOn) > feedPayloadCap {
			item.DependsOn = item.DependsOn[:feedPayloadCap]
		}
		content.Items = append(content.Items, item)
	}
	for _, change := range proposal.Changes {
		appendPreview(factory.FeedCommunicationItem{ID: change.ExpectedRevision, Label: change.IntentID, Detail: change.Reason, State: "proposed_revision"})
		for _, task := range change.Tasks {
			appendPreview(factory.FeedCommunicationItem{ID: task.Key, DependsOn: append([]string{}, task.DependsOn...), Label: task.Objective, Detail: "Outcomes: " + strings.Join(task.Outcomes, "; ") + "\nAccepted ideas: " + strings.Join(task.IdeaIDs, ", "), State: "proposed"})
		}
	}
	content.Remaining = []string{fmt.Sprintf("Preview shows %d of %d proposed graph rows. Recover the complete immutable report through draft:%s.", len(content.Items), rows, sub.SubmissionID)}
	if abbreviated {
		content.Remaining = append(content.Remaining, "Long text or prerequisite lists are abbreviated in this preview; the recorded draft retains every byte.")
	}
	if err := factory.ValidateFeedComponent(content.ComponentID, cap.Profile, sub.Tool); err != nil {
		return err
	}
	return recordMCPFeedContentTx(ctx, tx, cap, sub, content, receipt)
}
