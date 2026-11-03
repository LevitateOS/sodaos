package forgejo

import (
	"context"
	"errors"
	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"strconv"
	"time"
)

var _ factoryservice.CollaborationDiscussionSource = HumanReads{}

func (s HumanReads) ReadIssueComments(ctx context.Context, actor collaboration.ActorID, repo collaboration.RepositoryID, number int64, q factoryservice.DiscussionQuery) (collaboration.Page[collaboration.IssueComment], error) {
	var empty collaboration.Page[collaboration.IssueComment]
	native, comments, observation, err := s.discussion(ctx, actor, repo, number, "issue", q)
	if err != nil {
		return empty, err
	}
	page := collaboration.Page[collaboration.IssueComment]{Observation: observation, Items: []collaboration.IssueComment{}, Next: native.NextCursor, Complete: native.NextCursor == ""}
	ref := collaboration.IssueRef{Kind: collaboration.SubjectIssue, Repository: repo, ExternalID: native.SubjectID, Number: number}
	for _, comment := range comments {
		page.Items = append(page.Items, collaboration.IssueComment{Issue: ref, Comment: comment})
	}
	return page, nil
}
func (s HumanReads) ReadPullComments(ctx context.Context, actor collaboration.ActorID, repo collaboration.RepositoryID, number int64, q factoryservice.DiscussionQuery) (collaboration.Page[collaboration.PullComment], error) {
	var empty collaboration.Page[collaboration.PullComment]
	native, comments, observation, err := s.discussion(ctx, actor, repo, number, "pull", q)
	if err != nil {
		return empty, err
	}
	page := collaboration.Page[collaboration.PullComment]{Observation: observation, Items: []collaboration.PullComment{}, Next: native.NextCursor, Complete: native.NextCursor == ""}
	ref := collaboration.PullRef{Kind: collaboration.SubjectPull, Repository: repo, ExternalID: native.SubjectID, Number: number}
	for _, comment := range comments {
		page.Items = append(page.Items, collaboration.PullComment{Pull: ref, Comment: comment})
	}
	return page, nil
}
func (s HumanReads) discussion(ctx context.Context, actor collaboration.ActorID, repo collaboration.RepositoryID, number int64, kind string, q factoryservice.DiscussionQuery) (extensions.RepositoryCommentPage, []collaboration.Comment, collaboration.Observation, error) {
	var empty extensions.RepositoryCommentPage
	var unknown collaboration.Observation
	fail := func(err error) (extensions.RepositoryCommentPage, []collaboration.Comment, collaboration.Observation, error) {
		return empty, nil, unknown, discussionError(err)
	}
	if actor < 1 || repo < 1 || number < 1 || q.Validate() != nil {
		return fail(factoryservice.ErrDiscussionQuery)
	}
	if q.Cursor != "" {
		if _, err := originalCreationID(q.Cursor); err != nil {
			return fail(factoryservice.ErrDiscussionQuery)
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeRepo, nativeActor, err := s.scope(ctx, actor, repo)
	if err != nil {
		return fail(err)
	}
	native, err := s.Authority.RepositoryComments(ctx, nativeRepo, extensions.RepositoryCommentsRequest{Subject: kind, Index: strconv.FormatInt(number, 10), Cursor: q.Cursor, Limit: q.Limit})
	if err != nil {
		return fail(err)
	}
	observation := collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(native), ObservedAt: time.Now().UTC()}
	comments := make([]collaboration.Comment, 0, len(native.Items))
	for _, item := range native.Items {
		comment := collaboration.Comment{ExternalID: item.ID, Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(item), ObservedAt: observation.ObservedAt}, Body: item.Body, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
		if item.Author != nil {
			author, err := s.actor(ctx, *item.Author)
			if err != nil {
				return fail(err)
			}
			comment.Author = &author
		}
		comments = append(comments, comment)
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return fail(err)
	}
	if after.ID != nativeActor {
		return fail(factoryservice.ErrDiscussionMissing)
	}
	return native, comments, observation, nil
}
func discussionError(err error) error {
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, extensions.ErrRepositoryNotVisible) {
		return factoryservice.ErrDiscussionMissing
	}
	return err
}
