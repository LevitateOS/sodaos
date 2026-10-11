package api

import (
	"context"
	"errors"
	"strconv"

	extensions "forgejo.org/extension-sdk"

	"github.com/levitateos/sodaos/internal/factory"
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
	occurrences, err := issueOccurrences(snapshot.Issue)
	if err != nil {
		return control.AcceptanceEvidence{}, err
	}
	evidence := control.AcceptanceEvidence{
		Revision: snapshot.Revision,
		Issue: control.AcceptanceIssueView{
			ID:                  snapshot.Issue.ID,
			Index:               snapshot.Issue.Index,
			TitleDigest:         snapshot.Issue.TitleDigest,
			ContentDigest:       snapshot.Issue.ContentDigest,
			PosterID:            snapshot.Issue.Provenance.PosterID,
			ContentVer:          snapshot.Issue.ContentVersion,
			TitleOccurrence:     occurrences.title,
			ClosingOccurrence:   occurrences.closing,
			LifecycleOccurrence: occurrences.latest,
			ClosedUnix:          snapshot.Issue.ClosedUnix,
			CreatedUnix:         snapshot.Issue.CreatedUnix,
			Verified:            snapshot.Issue.Provenance.Verified,
			FirstCreated:        snapshot.Issue.Provenance.FirstCreated,
			Visible:             snapshot.Issue.Visible,
			Closed:              snapshot.Issue.IsClosed,
			IsPull:              snapshot.Issue.IsPull,
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

type nativeIssueOccurrences struct {
	title, closing, latest string
}

// issueOccurrences retains native comment identities. A complete visible
// history with no title change explicitly identifies the original title;
// missing or redacted history never supplies that assertion.
func issueOccurrences(issue *forgejo.IssueEvidence) (nativeIssueOccurrences, error) {
	if !issue.Visible {
		return nativeIssueOccurrences{}, nil
	}
	if !issue.Complete {
		return nativeIssueOccurrences{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	}
	result := nativeIssueOccurrences{title: factory.OriginalOccurrence, latest: factory.OriginalOccurrence}
	var previous int64
	closed := false
	for _, event := range issue.Lifecycle {
		id, err := strconv.ParseInt(event.OccurrenceID, 10, 64)
		if err != nil || id <= previous || strconv.FormatInt(id, 10) != event.OccurrenceID {
			return nativeIssueOccurrences{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
		}
		previous = id
		result.latest = event.OccurrenceID
		switch event.Kind {
		case "retitled":
			result.title = event.OccurrenceID
		case "closed":
			result.closing, closed = event.OccurrenceID, true
		case "reopened":
			closed = false
		default:
			return nativeIssueOccurrences{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
		}
	}
	if issue.IsClosed != closed {
		return nativeIssueOccurrences{}, &control.AcceptanceRefusal{Reason: control.RefusalIncompleteEvidence}
	}
	return result, nil
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
