package control

import (
	"context"
	"errors"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// readAcceptanceEvidence brackets the objective, every selected source and
// resolution, and the complete edge set at one native revision.
func (c *Coordinator) readAcceptanceEvidence(ctx context.Context, decision factory.Acceptance) (AcceptanceEvidence, error) {
	if c.AcceptanceReads == nil {
		return AcceptanceEvidence{}, refuseAcceptance(RefusalSnapshotUnavailable)
	}
	seen := make(map[string]bool)
	var commentIDs []string
	for _, section := range [][]factory.SelectedSource{decision.Sources, decision.Resolutions} {
		for _, source := range section {
			if !seen[source.ID] {
				seen[source.ID] = true
				commentIDs = append(commentIDs, source.ID)
			}
		}
	}
	if err := c.recordReadinessEvidence(ctx); err != nil {
		return AcceptanceEvidence{}, err
	}
	evidence, err := c.AcceptanceReads.ReadAcceptanceEvidence(ctx, strconv.FormatInt(decision.Repository, 10), decision.IssueIndex, commentIDs)
	if err != nil {
		var refusal *AcceptanceRefusal
		if errors.As(err, &refusal) {
			return AcceptanceEvidence{}, refusal
		}
		return AcceptanceEvidence{}, err
	}
	return evidence, nil
}

// verifyAcceptance compares one decision against its bracketed evidence.
// A stale screen, hidden source or changed revision refuses instead of
// silently accepting newer material.
func (c *Coordinator) verifyAcceptance(ctx context.Context, decision factory.Acceptance, evidence AcceptanceEvidence) error {
	if evidence.Revision < 1 || decision.NativeRev != evidence.Revision {
		return refuseAcceptance(RefusalStaleEvidence)
	}
	issue := evidence.Issue
	if issue.Index != decision.IssueIndex {
		return refuseAcceptance(RefusalIncompleteEvidence)
	}
	if !issue.Visible {
		return refuseAcceptance(RefusalIssueHidden)
	}
	if issue.TitleDigest != decision.TitleDigest || issue.ContentDigest != decision.ContentDigest || issue.ContentVer != decision.ContentVersion {
		return refuseAcceptance(RefusalObjectiveChanged)
	}
	comments := make(map[string]AcceptanceComment, len(evidence.Comments))
	for _, comment := range evidence.Comments {
		comments[comment.ID] = comment
	}
	for _, section := range [][]factory.SelectedSource{decision.Sources, decision.Resolutions} {
		for _, source := range section {
			comment, ok := comments[source.ID]
			if !ok {
				return refuseAcceptance(RefusalSourceMissing)
			}
			if !comment.Visible {
				return refuseAcceptance(RefusalSourceHidden)
			}
			if comment.Digest != source.Digest || comment.ContentVer != source.ContentVersion {
				return refuseAcceptance(RefusalSourceChanged)
			}
		}
	}
	edges := make(map[string]AcceptanceEdge, len(evidence.Dependencies))
	for _, edge := range evidence.Dependencies {
		if !edge.Visible {
			return refuseAcceptance(RefusalEdgeHidden)
		}
		edges[edge.Occurrence] = edge
	}
	if len(edges) != len(decision.Prerequisites) {
		return refuseAcceptance(RefusalEdgeChanged)
	}
	for _, prereq := range decision.Prerequisites {
		edge, ok := edges[prereq.Occurrence]
		if !ok || edge.DependsOn != prereq.DependsOn {
			return refuseAcceptance(RefusalEdgeChanged)
		}
		if prereq.Outcome == factory.PrereqCode {
			if _, err := c.Store.AcceptanceDecision(ctx, prereq.PrereqAcceptance); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return refuseAcceptance(RefusalPrereqUnknown)
				}
				return err
			}
			head, err := c.Store.AcceptanceHead(ctx, prereq.EndpointRepo, prereq.EndpointIssue)
			if err != nil || head != prereq.PrereqAcceptance {
				if err != nil && !errors.Is(err, store.ErrNotFound) {
					return err
				}
				return refuseAcceptance(RefusalPrereqStale)
			}
		}
	}
	return nil
}
