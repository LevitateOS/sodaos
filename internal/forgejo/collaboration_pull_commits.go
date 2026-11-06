package forgejo

import (
	"context"
	"errors"
	"strconv"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

var _ factoryservice.CollaborationPullCommitsSource = HumanReads{}

func (s HumanReads) ReadPullCommits(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, number int64, q factoryservice.PullCommitsQuery) (collaboration.PullCommits, error) {
	var empty collaboration.PullCommits
	if actor < 1 || repository < 1 || number < 1 || q.Validate() != nil {
		return empty, factoryservice.ErrPullCommitsQuery
	}
	if s.Identities == nil {
		return empty, factoryservice.ErrPullCommitsUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeRepo, nativeActor, err := s.scope(ctx, actor, repository)
	if err != nil {
		return empty, humanPullCommitsError(err)
	}
	native, err := s.Authority.RepositoryPullCommits(ctx, nativeRepo, extensions.RepositoryPullCommitsRequest{Index: strconv.FormatInt(number, 10), Head: string(q.Head), Cursor: q.Cursor, Limit: q.Limit})
	if err != nil {
		return empty, humanPullCommitsError(err)
	}
	if _, err := originalCreationID(native.PullID); err != nil {
		return empty, ErrInvalidResponse
	}
	result := collaboration.PullCommits{Pull: collaboration.PullRef{Kind: collaboration.SubjectPull, Repository: repository, ExternalID: native.PullID, Number: number}, Head: collaboration.CommitID(native.Head), Commits: collaboration.Page[collaboration.PullCommit]{Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(native), ObservedAt: time.Now().UTC()}, Items: make([]collaboration.PullCommit, 0, len(native.Items)), Next: native.NextCursor, Complete: native.NextCursor == ""}}
	for _, item := range native.Items {
		result.Commits.Items = append(result.Commits.Items, collaboration.PullCommit{Commit: collaboration.CommitID(item.Commit), Summary: item.Summary, Author: item.Author, AuthoredAt: item.AuthoredAt})
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, humanPullCommitsError(err)
	}
	if after.ID != nativeActor {
		return empty, factoryservice.ErrPullCommitsMissing
	}
	return result, nil
}

func humanPullCommitsError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrPullCommitsMissing
	case errors.Is(err, extensions.ErrPullCommitsQuery):
		return factoryservice.ErrPullCommitsQuery
	case errors.Is(err, extensions.ErrPullCommitsChanged):
		return factoryservice.ErrPullCommitsChanged
	default:
		return err
	}
}
