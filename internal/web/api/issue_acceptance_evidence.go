package api

import (
	"context"
	"errors"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// AcceptanceSnapshotSource adapts the thin F-read transport to coordinator
// acceptance verification. It brackets the objective, every requested
// selected comment and the complete direct edge set through BracketedRead;
// transport failures arrive as acceptance refusals, never guessed evidence.
type AcceptanceSnapshotSource struct {
	Reader     forgejo.SnapshotReader
	Credential extensions.CredentialFile
}

// ReadAcceptanceEvidence brackets one native evidence read at a single idle
// revision. Reads are single-page: over-limit families refuse as incomplete
// instead of verifying decisions from a partial snapshot. The comments page
// covers the issue and is filtered to the requested IDs; unselected
// discussion is ignored and cannot grant scope.
func (s AcceptanceSnapshotSource) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, commentIDs []string) (control.AcceptanceEvidence, error) {
	if s.Reader == nil {
		return control.AcceptanceEvidence{}, &control.AcceptanceRefusal{Reason: control.RefusalSnapshotUnavailable}
	}
	req := forgejo.SnapshotRequest{
		RepositoryID: repository,
		IssueIndex:   issue,
		Families:     []forgejo.SnapshotFamily{forgejo.FamilyIssue, forgejo.FamilyDependencies},
		Limit:        forgejo.SnapshotPageLimit,
	}
	if len(commentIDs) != 0 {
		req.Families = append(req.Families, forgejo.FamilyComments)
	}
	snapshot, err := forgejo.BracketedRead(ctx, s.Reader, s.Credential, req)
	if err != nil {
		return control.AcceptanceEvidence{}, mapAcceptanceSnapshotError(err)
	}
	if snapshot.Issue == nil || snapshot.Dependencies == nil {
		return control.AcceptanceEvidence{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	}
	if len(commentIDs) != 0 && snapshot.Comments == nil {
		return control.AcceptanceEvidence{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	}
	evidence := control.AcceptanceEvidence{
		Revision: snapshot.Revision,
		Issue: control.AcceptanceIssueView{
			Index:         snapshot.Issue.Index,
			TitleDigest:   snapshot.Issue.TitleDigest,
			ContentDigest: snapshot.Issue.ContentDigest,
			PosterID:      snapshot.Issue.Provenance.PosterID,
			ContentVer:    snapshot.Issue.ContentVersion,
			Lifecycle:     len(snapshot.Issue.Lifecycle),
			ClosedUnix:    snapshot.Issue.ClosedUnix,
			Verified:      snapshot.Issue.Provenance.Verified,
			FirstCreated:  snapshot.Issue.Provenance.FirstCreated,
			Visible:       snapshot.Issue.Visible,
			Closed:        snapshot.Issue.IsClosed,
			IsPull:        snapshot.Issue.IsPull,
		},
	}
	if snapshot.Comments != nil {
		wanted := make(map[string]bool, len(commentIDs))
		for _, id := range commentIDs {
			wanted[id] = true
		}
		for _, item := range snapshot.Comments.Items {
			if !wanted[item.ID] {
				continue
			}
			evidence.Comments = append(evidence.Comments, control.AcceptanceComment{
				ID: item.ID, Digest: item.ContentDigest, ContentVer: item.ContentVersion, Visible: item.Visible,
			})
		}
	}
	for _, item := range snapshot.Dependencies.Items {
		evidence.Dependencies = append(evidence.Dependencies, control.AcceptanceEdge{
			Occurrence: item.OccurrenceID, DependsOn: item.DependencyID, Visible: item.Visible,
		})
	}
	return evidence, nil
}

// mapAcceptanceSnapshotError converts bracket failures to acceptance
// refusals. Stale, incomplete and busy brackets refuse without recording;
// malformed answers and transport failures report the source unavailable.
func mapAcceptanceSnapshotError(err error) error {
	switch {
	case errors.Is(err, forgejo.ErrStaleSnapshot):
		return &control.AcceptanceRefusal{Reason: control.RefusalStaleEvidence}
	case errors.Is(err, forgejo.ErrIncompleteSnapshot):
		return &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	case errors.Is(err, forgejo.ErrNativeBusy):
		return &control.AcceptanceRefusal{Reason: control.RefusalNativeBusy}
	case errors.Is(err, forgejo.ErrInvalidSnapshot):
		return &control.AcceptanceRefusal{Reason: control.RefusalSnapshotUnavailable}
	default:
		return err
	}
}
