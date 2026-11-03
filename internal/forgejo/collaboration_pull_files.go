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

var _ factoryservice.CollaborationPullFilesSource = HumanReads{}

func (s HumanReads) ReadPullFiles(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, number int64, q factoryservice.PullFilesQuery) (collaboration.PullFiles, error) {
	var empty collaboration.PullFiles
	if actor < 1 || repository < 1 || number < 1 || q.Validate() != nil {
		return empty, factoryservice.ErrPullFilesQuery
	}
	if s.Identities == nil {
		return empty, factoryservice.ErrPullFilesUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeRepo, nativeActor, err := s.scope(ctx, actor, repository)
	if err != nil {
		return empty, humanPullFilesError(err)
	}
	native, err := s.Authority.RepositoryPullFiles(ctx, nativeRepo, extensions.RepositoryPullFilesRequest{Index: strconv.FormatInt(number, 10), Base: string(q.Base), Head: string(q.Head), Cursor: q.Cursor, Limit: q.Limit})
	if err != nil {
		return empty, humanPullFilesError(err)
	}
	if _, err := originalCreationID(native.PullID); err != nil {
		return empty, ErrInvalidResponse
	}
	diff := collaboration.DiffRef{Pull: collaboration.PullRef{Kind: collaboration.SubjectPull, Repository: repository, ExternalID: native.PullID, Number: number}, Base: collaboration.CommitID(native.Base), Head: collaboration.CommitID(native.Head)}
	result := collaboration.PullFiles{Diff: diff, Files: collaboration.Page[collaboration.ChangedFile]{Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(native), ObservedAt: time.Now().UTC()}, Items: make([]collaboration.ChangedFile, 0, len(native.Items)), Next: native.NextCursor, Complete: native.NextCursor == ""}}
	for _, f := range native.Items {
		result.Files.Items = append(result.Files.Items, collaboration.ChangedFile{Diff: diff, OldPath: f.OldPath, NewPath: f.NewPath, Change: f.Change, PatchState: f.PatchState, Patch: f.Patch, Additions: f.Additions, Deletions: f.Deletions})
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, humanPullFilesError(err)
	}
	if after.ID != nativeActor {
		return empty, factoryservice.ErrPullFilesMissing
	}
	return result, nil
}
func humanPullFilesError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrPullFilesMissing
	case errors.Is(err, extensions.ErrPullFilesQuery):
		return factoryservice.ErrPullFilesQuery
	case errors.Is(err, extensions.ErrPullFilesChanged):
		return factoryservice.ErrPullFilesChanged
	case errors.Is(err, extensions.ErrPullFilesTooLarge):
		return factoryservice.ErrPullFilesTooLarge
	default:
		return err
	}
}
