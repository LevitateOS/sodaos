package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// postContext supplies the current human's Fountain reads. Paging, candidate
// selection and durable capture live in the shared application service.
func (s *API) postContext(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.PostContext {
	return factoryservice.PostContext{Posts: s.posts(r, v, admitted), Issues: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, query factoryservice.PostIssueQuery) (collaboration.Page[collaboration.Issue], error) {
		var empty collaboration.Page[collaboration.Issue]
		request := r.WithContext(ctx)
		if actor != admitted || !s.extensionSessionCurrent(ctx, request, v) {
			return empty, auth.ErrNativeSessionChanged
		}
		authority, ok := requestExtensionAuthority(request)
		if !ok {
			return empty, auth.ErrNativeSessionChanged
		}
		native, err := s.FountainIdentities.NativeRepository(ctx, repository)
		if err != nil {
			return empty, err
		}
		number := ""
		if query.Number > 0 {
			number = strconv.FormatInt(query.Number, 10)
		}
		page, err := authority.RepositoryIssues(ctx, native, query.Cursor, number, query.Limit, false, "")
		if err != nil {
			return empty, err
		}
		observation := collaboration.Observation{State: collaboration.ReadAvailable, ObservedAt: time.Now()}
		result := collaboration.Page[collaboration.Issue]{Observation: observation, Items: make([]collaboration.Issue, 0, len(page.Items)), Next: page.NextCursor, Complete: page.NextCursor == ""}
		for _, issue := range page.Items {
			index, err := strconv.ParseInt(issue.Index, 10, 64)
			if err != nil {
				return empty, err
			}
			state := "open"
			if issue.Closed {
				state = "closed"
			}
			result.Items = append(result.Items, collaboration.Issue{Ref: collaboration.IssueRef{Kind: collaboration.SubjectIssue, Repository: repository, ExternalID: issue.ID, Number: index}, Observation: observation, Title: issue.Title, Body: issue.Body, State: state})
		}
		return result, nil
	}}
}
