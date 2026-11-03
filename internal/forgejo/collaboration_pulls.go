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

var _ factoryservice.CollaborationPullSource = HumanReads{}

func (s HumanReads) ReadPulls(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, query factoryservice.CollaborationPullQuery) (collaboration.Page[collaboration.PullRequest], error) {
	var empty collaboration.Page[collaboration.PullRequest]
	if query.Validate() != nil || actor < 1 || repository < 1 {
		return empty, factoryservice.ErrCollaborationPullQuery
	}
	if query.Cursor != "" {
		if _, err := originalCreationID(query.Cursor); err != nil {
			return empty, factoryservice.ErrCollaborationPullQuery
		}
	}
	if s.Identities == nil {
		return empty, factoryservice.ErrCollaborationPullUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeRepository, nativeActor, err := s.scope(ctx, actor, repository)
	if err != nil {
		return empty, humanPullError(err)
	}
	q := extensions.RepositoryPullsRequest{Cursor: query.Cursor, State: query.State, Limit: query.Limit}
	if query.Number > 0 {
		q.Index = strconv.FormatInt(query.Number, 10)
	}
	page, err := s.Authority.RepositoryPulls(ctx, nativeRepository, q)
	if err != nil {
		return empty, humanPullError(err)
	}
	base, err := s.Identities.registry.CollaborationRepositoryOnHost(ctx, s.Identities.host, repository)
	if err != nil {
		return empty, humanPullError(err)
	}
	observed := time.Now().UTC()
	result := collaboration.Page[collaboration.PullRequest]{Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(page), ObservedAt: observed}, Items: make([]collaboration.PullRequest, 0, len(page.Items)), Next: page.NextCursor, Complete: page.NextCursor == ""}
	for _, native := range page.Items {
		number, err := originalCreationID(native.Index)
		if err != nil {
			return empty, ErrInvalidResponse
		}
		if _, err := originalCreationID(native.ID); err != nil {
			return empty, ErrInvalidResponse
		}
		draft := native.Draft
		item := collaboration.PullRequest{Ref: collaboration.PullRef{Kind: collaboration.SubjectPull, Repository: repository, ExternalID: native.ID, Number: number}, Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(native), ObservedAt: observed}, Title: native.Title, Body: native.Body, State: "open", Draft: &draft, Base: collaboration.GitRef{Repository: &base, Name: native.BaseBranch}, CreatedAt: native.CreatedAt, UpdatedAt: native.UpdatedAt, MergedAt: native.MergedAt}
		if native.Closed {
			item.State = "closed"
		}
		if native.Merged {
			item.State = "merged"
		}
		if native.Author != nil {
			author, err := s.actor(ctx, *native.Author)
			if err != nil {
				return empty, err
			}
			item.Author = &author
		}
		if native.Head != nil {
			head := native.Head
			if _, err := originalCreationID(head.RepositoryID); err != nil || !collaboration.ValidText(head.Owner+"/"+head.Name, 512) {
				return empty, ErrInvalidResponse
			}
			ref, err := s.Identities.registry.ObserveCollaborationRepository(ctx, s.Identities.host, head.RepositoryID, head.Owner+"/"+head.Name)
			if err != nil {
				return empty, err
			}
			item.Head = collaboration.GitRef{Repository: &ref, Name: head.Branch}
		}
		// Commit IDs are unobserved here. The immutable file reader resolves Git;
		// branch labels or a stored merge base must never impersonate live commits.
		result.Items = append(result.Items, item)
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, humanPullError(err)
	}
	if after.ID != nativeActor {
		return empty, factoryservice.ErrCollaborationPullMissing
	}
	return result, nil
}

func humanPullError(err error) error {
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, extensions.ErrRepositoryNotVisible) {
		return factoryservice.ErrCollaborationPullMissing
	}
	return err
}
