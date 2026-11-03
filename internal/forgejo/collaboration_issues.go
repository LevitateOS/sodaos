package forgejo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

// HumanReads uses this request's live human admission, never an automation PAT.
// Canonical keys resolve on the adopted host before any native read.
type HumanReads struct {
	Authority  extensions.Authority
	Identities *FountainIdentities
}

var _ factoryservice.CollaborationIssueSource = HumanReads{}

func (s HumanReads) ReadIssues(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, query factoryservice.CollaborationIssueQuery) (collaboration.Page[collaboration.Issue], error) {
	var empty collaboration.Page[collaboration.Issue]
	if query.Validate() != nil || actor < 1 || repository < 1 {
		return empty, factoryservice.ErrCollaborationIssueQuery
	}
	if query.Cursor != "" {
		if _, err := originalCreationID(query.Cursor); err != nil {
			return empty, factoryservice.ErrCollaborationIssueQuery
		}
	}
	if s.Identities == nil {
		return empty, factoryservice.ErrCollaborationIssueUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeRepository, nativeActor, err := s.scope(ctx, actor, repository)
	if err != nil {
		return empty, humanIssueError(err)
	}
	number := ""
	if query.Number > 0 {
		number = strconv.FormatInt(query.Number, 10)
	}
	page, err := s.Authority.RepositoryIssues(ctx, nativeRepository, query.Cursor, number, query.Limit, true, query.State)
	if err != nil {
		return empty, humanIssueError(err)
	}
	if page.RepositoryID != nativeRepository {
		return empty, ErrInvalidResponse
	}
	observed := time.Now().UTC()
	result := collaboration.Page[collaboration.Issue]{Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(page), ObservedAt: observed}, Items: make([]collaboration.Issue, 0, len(page.Items)), Next: page.NextCursor, Complete: page.NextCursor == ""}
	for _, native := range page.Items {
		index, err := originalCreationID(native.Index)
		if err != nil || !native.BodyComplete {
			return empty, ErrInvalidResponse
		}
		if _, err = originalCreationID(native.ID); err != nil {
			return empty, ErrInvalidResponse
		}
		item := collaboration.Issue{Ref: collaboration.IssueRef{Kind: collaboration.SubjectIssue, Repository: repository, ExternalID: native.ID, Number: index}, Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(native), ObservedAt: observed}, Title: native.Title, Body: native.Body, State: "open", Assignees: []collaboration.ActorRef{}, Labels: []collaboration.Label{}, CreatedAt: native.CreatedAt, UpdatedAt: native.UpdatedAt, ClosedAt: native.ClosedAt}
		if native.Closed {
			item.State = "closed"
		}
		if native.Author != nil {
			author, err := s.actor(ctx, *native.Author)
			if err != nil {
				return empty, err
			}
			item.Author = &author
		}
		for _, nativeActor := range native.Assignees {
			actor, err := s.actor(ctx, nativeActor)
			if err != nil {
				return empty, err
			}
			item.Assignees = append(item.Assignees, actor)
		}
		for _, label := range native.Labels {
			if _, err := originalCreationID(label.ID); err != nil {
				return empty, ErrInvalidResponse
			}
			item.Labels = append(item.Labels, collaboration.Label{ExternalID: label.ID, Name: label.Name, Color: label.Color})
		}
		// Action facts use the separate current actor/resource capability read.
		result.Items = append(result.Items, item)
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, humanIssueError(err)
	}
	if after.ID != nativeActor {
		return empty, factoryservice.ErrCollaborationIssueMissing
	}
	return result, nil
}
func (s HumanReads) actor(ctx context.Context, native extensions.RepositoryIssueActor) (collaboration.ActorRef, error) {
	if _, err := originalCreationID(native.ID); err != nil {
		return collaboration.ActorRef{}, ErrInvalidResponse
	}
	if !collaboration.ValidText(native.Username, 255) {
		return collaboration.ActorRef{}, ErrInvalidResponse
	}
	return s.Identities.registry.ObserveCollaborationActor(ctx, s.Identities.host, native.ID, native.Username)
}
func issueReadRevision(value any) collaboration.Revision {
	// This version describes the returned resource/page, not an atomic snapshot or
	// a native mutation precondition. Native guarded effects use their own proof.
	body, _ := json.Marshal(value)
	sum := sha256.Sum256(body)
	return collaboration.Revision("fountain-read:" + hex.EncodeToString(sum[:]))
}
func humanIssueError(err error) error {
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, extensions.ErrRepositoryNotVisible) {
		return factoryservice.ErrCollaborationIssueMissing
	}
	return err
}
