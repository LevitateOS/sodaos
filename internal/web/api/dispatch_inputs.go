package api

import (
	"context"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// DispatchSnapshotSource adapts the thin F-read transport to coordinator
// dispatch input reads. It brackets the accepted objective, the issue's
// bounded comment page and the target branch tip through BracketedRead;
// transport failures arrive as acceptance refusals, never guessed inputs.
// Comment selection filters the by-issue page to the accepted revisions;
// unselected discussion is ignored and cannot grant scope.
type DispatchSnapshotSource struct {
	Reader     forgejo.SnapshotReader
	Credential extensions.CredentialFile
}

// ReadDispatchInputs brackets one accepted-input read at a single idle
// revision. Reads are single-page: over-limit evidence refuses as
// incomplete instead of dispatching from a partial snapshot. A missing
// or invisible target tip reads as empty; the coordinator waits for the
// source instead of guessing it.
func (s DispatchSnapshotSource) ReadDispatchInputs(ctx context.Context, repository, issue string, commentIDs []string, targetRef string) (control.DispatchInputs, error) {
	if s.Reader == nil {
		return control.DispatchInputs{}, &control.AcceptanceRefusal{Reason: control.RefusalSnapshotUnavailable}
	}
	req := forgejo.SnapshotRequest{
		RepositoryID: repository,
		IssueIndex:   issue,
		Families:     []forgejo.SnapshotFamily{forgejo.FamilyIssue, forgejo.FamilyRefs},
		Refs:         []string{targetRef},
		Limit:        forgejo.SnapshotPageLimit,
	}
	if len(commentIDs) != 0 {
		req.Families = append(req.Families, forgejo.FamilyComments)
	}
	snapshot, err := forgejo.BracketedRead(ctx, s.Reader, s.Credential, req)
	if err != nil {
		return control.DispatchInputs{}, mapAcceptanceSnapshotError(err)
	}
	if snapshot.Issue == nil {
		return control.DispatchInputs{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	}
	if len(commentIDs) != 0 && snapshot.Comments == nil {
		return control.DispatchInputs{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	}
	inputs := control.DispatchInputs{
		Revision: snapshot.Revision,
		TipRef:   targetRef,
		Issue: control.DispatchIssue{
			Title:         snapshot.Issue.Title,
			Body:          snapshot.Issue.Content,
			TitleDigest:   snapshot.Issue.TitleDigest,
			ContentDigest: snapshot.Issue.ContentDigest,
			ContentVer:    snapshot.Issue.ContentVersion,
			Lifecycle:     len(snapshot.Issue.Lifecycle),
			ClosedUnix:    snapshot.Issue.ClosedUnix,
			Visible:       snapshot.Issue.Visible,
			Closed:        snapshot.Issue.IsClosed,
			Locked:        snapshot.Issue.IsLocked,
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
			inputs.Comments = append(inputs.Comments, control.DispatchComment{
				ID: item.ID, Content: item.Content, Digest: item.ContentDigest,
				ContentVer: item.ContentVersion, Visible: item.Visible,
			})
		}
	}
	for _, ref := range snapshot.Refs {
		if ref.Ref == targetRef && ref.Exists && ref.Visible {
			inputs.Tip = ref.OID
		}
	}
	return inputs, nil
}
