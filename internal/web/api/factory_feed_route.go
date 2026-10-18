package api

import (
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
	visible := func(entry factory.FeedEntry) bool {
		_, err := s.visibleRepository(r, v, entry.Repository)
		return err == nil
	}
	view, err := ServeSubscribedFeed(r.Context(), s.Store, visible, query, sub)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read combined feed.")
		return
	}
	auth.JSONResponse(w, 200, view)
}
