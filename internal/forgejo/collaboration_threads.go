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

var _ factoryservice.CollaborationThreadSource = HumanReads{}

func threadAnchor(a *extensions.RepositoryThreadAnchor, pull collaboration.PullRef) *collaboration.ReviewAnchor {
	if a == nil {
		return nil
	}
	r := &collaboration.ReviewAnchor{Diff: collaboration.DiffRef{Pull: pull, Base: collaboration.CommitID(a.Base), Head: collaboration.CommitID(a.Head)}, OldPath: a.OldPath, NewPath: a.NewPath, Side: a.Side, Scope: a.Scope}
	if a.Scope == "lines" {
		r.Lines = &collaboration.LineRange{Start: int(a.StartLine), End: int(a.EndLine)}
	}
	return r
}
func (s HumanReads) ReadThreads(ctx context.Context, actor collaboration.ActorID, repo collaboration.RepositoryID, number int64, q factoryservice.ThreadQuery) (collaboration.Page[collaboration.ReviewThread], error) {
	var empty collaboration.Page[collaboration.ReviewThread]
	if actor < 1 || repo < 1 || number < 1 || q.Validate() != nil {
		return empty, factoryservice.ErrThreadQuery
	}
	for _, id := range []string{q.Cursor, q.ThreadID} {
		if id != "" {
			if _, err := originalCreationID(id); err != nil {
				return empty, factoryservice.ErrThreadQuery
			}
		}
	}
	if s.Identities == nil {
		return empty, factoryservice.ErrThreadUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeRepo, nativeActor, err := s.scope(ctx, actor, repo)
	if err != nil {
		return empty, humanThreadError(err)
	}
	native, err := s.Authority.RepositoryThreads(ctx, nativeRepo, extensions.RepositoryThreadsRequest{Index: strconv.FormatInt(number, 10), ThreadID: q.ThreadID, Cursor: q.Cursor, Limit: q.Limit})
	if err != nil {
		return empty, humanThreadError(err)
	}
	observed := time.Now().UTC()
	observation := func(v any) collaboration.Observation {
		return collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(v), ObservedAt: observed}
	}
	p := collaboration.Page[collaboration.ReviewThread]{Observation: observation(native), Items: []collaboration.ReviewThread{}, Next: native.NextCursor, Complete: native.NextCursor == ""}
	pull := collaboration.PullRef{Kind: collaboration.SubjectPull, Repository: repo, Number: number, ExternalID: native.PullID}
	for _, t := range native.Items {
		r := collaboration.ReviewThread{ExternalID: t.ID, Pull: pull, Observation: observation(t), Anchor: threadAnchor(t.Anchor, pull), CurrentAnchor: threadAnchor(t.CurrentAnchor, pull), Outdated: t.Outdated, Resolution: "unresolved", Comments: collaboration.Page[collaboration.ReviewComment]{Observation: observation(t.Comments), Items: []collaboration.ReviewComment{}, Next: t.CommentsNext, Complete: t.CommentsNext == ""}}
		if t.Resolved {
			r.Resolution = "resolved"
		}
		if l := t.RecordedLocation; l != nil {
			r.RecordedLocation = &collaboration.ReviewLocation{Path: l.Path, Side: l.Side, Line: l.Line}
			if l.Commit != "" {
				commit := collaboration.CommitID(l.Commit)
				r.RecordedLocation.Commit = &commit
			}
		}
		for _, c := range t.Comments {
			body := collaboration.Comment{ExternalID: c.Comment.ID, Observation: observation(c), Body: c.Comment.Body, CreatedAt: c.Comment.CreatedAt, UpdatedAt: c.Comment.UpdatedAt}
			if c.Comment.Author != nil {
				author, err := s.actor(ctx, *c.Comment.Author)
				if err != nil {
					return empty, err
				}
				body.Author = &author
			}
			comment := collaboration.ReviewComment{Comment: body, Anchor: threadAnchor(c.Anchor, pull)}
			if t.ReviewID != "" {
				comment.ReviewID = &t.ReviewID
			}
			r.Comments.Items = append(r.Comments.Items, comment)
		}
		p.Items = append(p.Items, r)
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, humanThreadError(err)
	}
	if after.ID != nativeActor {
		return empty, factoryservice.ErrThreadMissing
	}
	return p, nil
}
func humanThreadError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, extensions.ErrRepositoryNotVisible):
		return factoryservice.ErrThreadMissing
	case errors.Is(err, extensions.ErrThreadQuery):
		return factoryservice.ErrThreadQuery
	case errors.Is(err, extensions.ErrThreadChanged):
		return factoryservice.ErrThreadChanged
	case errors.Is(err, extensions.ErrThreadsTooLarge):
		return factoryservice.ErrThreadsTooLarge
	default:
		return err
	}
}
