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

var _ factoryservice.CollaborationReviewSource = HumanReads{}

func (s HumanReads) ReadReviews(ctx context.Context, actor collaboration.ActorID, repo collaboration.RepositoryID, number int64, q factoryservice.ReviewQuery) (collaboration.Page[collaboration.Review], error) {
	var empty collaboration.Page[collaboration.Review]
	if actor < 1 || repo < 1 || number < 1 || q.Validate() != nil {
		return empty, factoryservice.ErrReviewQuery
	}
	for _, id := range []string{q.Cursor, q.ReviewID} {
		if id != "" {
			if _, err := originalCreationID(id); err != nil {
				return empty, factoryservice.ErrReviewQuery
			}
		}
	}
	if s.Identities == nil {
		return empty, factoryservice.ErrReviewUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeRepo, nativeActor, err := s.scope(ctx, actor, repo)
	if err != nil {
		return empty, humanReviewError(err)
	}
	native, err := s.Authority.RepositoryReviews(ctx, nativeRepo, extensions.RepositoryReviewsRequest{Index: strconv.FormatInt(number, 10), ReviewID: q.ReviewID, Cursor: q.Cursor, Limit: q.Limit})
	if err != nil {
		return empty, humanReviewError(err)
	}
	if _, err := originalCreationID(native.PullID); err != nil {
		return empty, ErrInvalidResponse
	}
	observed := time.Now().UTC()
	page := collaboration.Page[collaboration.Review]{Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(native), ObservedAt: observed}, Items: make([]collaboration.Review, 0, len(native.Items)), Next: native.NextCursor, Complete: native.NextCursor == ""}
	ref := collaboration.PullRef{Kind: collaboration.SubjectPull, Repository: repo, ExternalID: native.PullID, Number: number}
	for _, r := range native.Items {
		stale, official := r.Stale, r.Official
		item := collaboration.Review{ExternalID: r.ID, Pull: ref, Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(r), ObservedAt: observed}, Commit: collaboration.CommitID(r.Commit), State: r.State, Body: r.Body, Stale: &stale, Official: &official, SubmittedAt: r.SubmittedAt}
		if r.Author != nil {
			author, err := s.actor(ctx, *r.Author)
			if err != nil {
				return empty, err
			}
			item.Author = &author
		}
		page.Items = append(page.Items, item)
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, humanReviewError(err)
	}
	if after.ID != nativeActor {
		return empty, factoryservice.ErrReviewMissing
	}
	return page, nil
}
func humanReviewError(err error) error {
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, extensions.ErrRepositoryNotVisible) {
		return factoryservice.ErrReviewMissing
	}
	return err
}
