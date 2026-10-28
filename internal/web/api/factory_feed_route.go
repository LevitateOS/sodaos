package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// parseFeedQueryParams admits one combined feed query from its URL
// parameters: comma-separated repository scope, optional kinds, a
// bounded window and the resume cursor.
func parseFeedQueryParams(r *http.Request) (factory.FeedQuery, error) {
	var query factory.FeedQuery
	seen := make(map[int64]bool)
	for _, field := range strings.Split(r.URL.Query().Get("repositories"), ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		repository, err := strconv.ParseInt(field, 10, 64)
		if err != nil || repository <= 0 {
			return factory.FeedQuery{}, errFeedQueryInvalid
		}
		if !seen[repository] {
			seen[repository] = true
			query.Repositories = append(query.Repositories, repository)
		}
	}
	if kinds := strings.TrimSpace(r.URL.Query().Get("kinds")); kinds != "" {
		query.Kinds = strings.Split(kinds, ",")
		for i, kind := range query.Kinds {
			query.Kinds[i] = strings.TrimSpace(kind)
		}
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return factory.FeedQuery{}, errFeedQueryInvalid
		}
		limit = parsed
	}
	query.Limit = limit
	query.View = strings.TrimSpace(r.URL.Query().Get("view"))
	query.Search = strings.TrimSpace(r.URL.Query().Get("search"))
	query.Role = strings.TrimSpace(r.URL.Query().Get("role"))
	query.Current = r.URL.Query().Get("current") == "true"
	query.Cursor = strings.TrimSpace(r.URL.Query().Get("cursor"))
	if err := query.Validate(); err != nil {
		return factory.FeedQuery{}, err
	}
	return query, nil
}

// apiFactoryFeed serves the caller's subscribed combined feed window:
// subscription narrowing, then per-entry visibility recheck. The
// subscriber is the host-admitted caller, never a body field.
func (s *API) apiFactoryFeed(w http.ResponseWriter, r *http.Request, v store.Session) {
	query, err := parseFeedQueryParams(r)
	if err != nil {
		auth.JSONError(w, 400, "invalid_request", "Combined feed query is invalid.")
		return
	}
	subscriber := strconv.FormatInt(v.User.ID, 10)
	sub, _, err := s.Store.FeedSubscription(r.Context(), subscriber)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read feed subscription.")
		return
	}
	if r.URL.Query().Get("scope") == "project" {
		if len(query.Repositories) != 1 {
			auth.JSONError(w, 400, "invalid_request", "Select one project.")
			return
		}
		if _, err := s.visibleRepository(r, v, query.Repositories[0]); err != nil {
			auth.JSONError(w, 404, "repository_not_found", "Project is not visible.")
			return
		}
		sub = factory.FeedSubscription{Subscriber: subscriber, Revision: 1, Repositories: query.Repositories}
	}
	projects := make(map[int64]string)
	visible := func(entry factory.FeedEntry) bool {
		access, err := s.visibleRepository(r, v, entry.Repository)
		if err != nil {
			return false
		}
		projects[entry.Repository] = access.repository.FullName
		return true
	}
	view, err := ServeSubscribedFeed(r.Context(), s.Store, visible, query, sub)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read combined feed.")
		return
	}
	enrichFeedPosts(r.Context(), s.Store, projects, view.Entries)
	enrichFeedRecords(r.Context(), s.Store, projects, view.Entries)
	for i, entry := range view.Entries {
		if entry.Kind == factory.FeedKindPost && (entry.Payload == nil || entry.Payload.Communication == nil) {
			if comment, err := s.feedCommentView(r, v, entry.ID); err == nil {
				view.Entries[i].Comment = comment
			}
		}
	}
	auth.JSONResponse(w, 200, view)
}

// enrichFeedRecords fills the server-derived record enrichment on one
// served window: assignment records from the canonical assignment
// owner (phase profile and persona, typed result payload) and
// Factory/Jev milestone entries from their merge, completion and
// judgment owners. Project fills from the visibility recheck even
// when detail lookup is unavailable; every other field stays empty
// rather than guessed.
func enrichFeedRecords(ctx context.Context, st *store.Store, projects map[int64]string, entries []factory.FeedEntry) {
	var ids []string
	var milestones []factory.FeedEntry
	for i, entry := range entries {
		if entry.Kind != factory.FeedKindRecord {
			continue
		}
		entries[i].Project = projects[entry.Repository]
		if entry.Activity == "" {
			ids = append(ids, entry.ID)
		} else {
			milestones = append(milestones, entry)
		}
	}
	if st == nil {
		return
	}
	details := make(map[string]store.FeedRecordDetail)
	if len(ids) > 0 {
		if found, err := st.FeedRecordDetails(ctx, ids); err == nil {
			for id, detail := range found {
				details[id] = detail
			}
		}
	}
	if len(milestones) > 0 {
		if found, err := st.FeedMilestoneDetails(ctx, milestones); err == nil {
			for id, detail := range found {
				details[id] = detail
			}
		}
	}
	for i, entry := range entries {
		if entry.Kind != factory.FeedKindRecord {
			continue
		}
		detail, ok := details[entry.ID]
		if !ok {
			continue
		}
		entries[i].Actor = detail.Actor
		entries[i].Author = detail.Author
		entries[i].At = detail.At
		entries[i].Status = detail.Status
		entries[i].Issue = detail.Issue
		entries[i].PR = detail.PR
		entries[i].Ref = detail.Ref
		entries[i].Summary = detail.Summary
		entries[i].Payload = detail.Payload
	}
}
