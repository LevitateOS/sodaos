package forgejo

import (
	"strconv"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
)

// GeneratedIssueObservationFromNative projects a validated provider snapshot
// into the facts graph adoption actually consumes. It performs no read or
// permission check itself. Production calls it only after BracketedRead holds.
func GeneratedIssueObservationFromNative(snapshot sdk.NativeSnapshot) (factory.GeneratedIssueObservation, error) {
	var observation factory.GeneratedIssueObservation
	if source := snapshot.Issue; source != nil {
		issue := &factory.GeneratedIssueFacts{ID: source.ID, Index: source.Index, Visible: source.Visible, Complete: source.Complete}
		observation.Issue = issue
		if source.Visible {
			issue.Title, issue.Content = source.Title, source.Content
			issue.TitleDigest, issue.ContentDigest = source.TitleDigest, source.ContentDigest
			issue.ContentVersion = source.ContentVersion
			issue.IsClosed, issue.IsLocked, issue.IsPull = source.IsClosed, source.IsLocked, source.IsPull
			issue.CreatedUnix = source.CreatedUnix
			if issue.CreatedUnix <= 0 {
				issue.CreatedUnix = source.Provenance.CreatedUnix
			}
			issue.CreationVerified, issue.FirstCreated = source.Provenance.Verified, source.Provenance.FirstCreated
			if source.Provenance.PosterID != "" {
				id, err := originalCreationID(source.Provenance.PosterID)
				if err != nil {
					return factory.GeneratedIssueObservation{}, err
				}
				actor := collaboration.ActorID(id)
				issue.Creator = &actor
			}
			issue.TitleOccurrence = factory.OriginalOccurrence
			for _, event := range source.Lifecycle {
				if event.Kind == "retitled" {
					issue.TitleOccurrence = event.OccurrenceID
				}
			}
		}
	}
	if source := snapshot.Dependencies; source != nil {
		page := &factory.GeneratedDependencies{IssueID: source.IssueID, Complete: source.Complete}
		if source.Items != nil {
			page.Items = make([]factory.GeneratedDependency, 0, len(source.Items))
		}
		for _, edge := range source.Items {
			fact := factory.GeneratedDependency{OccurrenceID: edge.OccurrenceID, Visible: edge.Visible, Complete: edge.Complete}
			if edge.Visible {
				fact.DependencyID = edge.DependencyID
			}
			page.Items = append(page.Items, fact)
		}
		observation.Dependencies = page
	}
	return observation, nil
}

func originalCreationID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != value {
		return 0, ErrInvalidResponse
	}
	return id, nil
}

func creationRecoveryBatch(page sdk.CreationRecoveryPage, repository, after int64, limit int) (factory.CreationRecoveryBatch, error) {
	var batch factory.CreationRecoveryBatch
	if repository <= 0 || after < 0 || limit < 1 || limit > sdk.CreationRecoveryPageLimit || page.RepositoryID != strconv.FormatInt(repository, 10) || len(page.Creations) > limit || !page.RetentionLimited {
		return batch, ErrInvalidResponse
	}
	batch = factory.CreationRecoveryBatch{Repository: repository, NextPosition: after, ReachedRetainedEnd: page.Complete, ProvenanceComplete: page.ProvenanceComplete, RetentionLimited: page.RetentionLimited}
	var err error
	if page.NextCursor != "" {
		batch.NextPosition, err = originalCreationID(page.NextCursor)
		if err != nil || batch.NextPosition < after {
			return factory.CreationRecoveryBatch{}, ErrInvalidResponse
		}
	}
	if page.OldestRetainedTaskID != "" {
		batch.OldestRetainedPosition, err = originalCreationID(page.OldestRetainedTaskID)
		if err != nil {
			return factory.CreationRecoveryBatch{}, err
		}
	}
	last := after
	for _, source := range page.Creations {
		position, e := originalCreationID(source.TaskID)
		if e != nil || position <= last || position > batch.NextPosition || source.RepositoryID != page.RepositoryID {
			return factory.CreationRecoveryBatch{}, ErrInvalidResponse
		}
		issue, e := originalCreationID(source.IssueIndex)
		if e != nil {
			return factory.CreationRecoveryBatch{}, e
		}
		id, e := originalCreationID(source.IssueID)
		if e != nil {
			return factory.CreationRecoveryBatch{}, e
		}
		creator, e := originalCreationID(source.CreatorID)
		if e != nil {
			return factory.CreationRecoveryBatch{}, e
		}
		batch.Events = append(batch.Events, factory.OriginalCreationEvent{Position: position, Delivery: source.DeliveryID, Repository: repository, Issue: issue, OriginalIssueID: id, Creator: creator, CreatorWrite: source.CreatorWrite, CreatedUnix: source.CreatedUnix, TitleDigest: source.TitleDigest, ContentDigest: source.ContentDigest, Verified: source.Verified})
		last = position
	}
	return batch, nil
}
