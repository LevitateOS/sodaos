package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// feedSubscriptionBody carries one subscription write: the scope and
// kinds with the revision the write expects. The subscriber is the
// host-admitted caller, never a body field.
type feedSubscriptionBody struct {
	Repositories []int64  `json:"repositories"`
	Kinds        []string `json:"kinds,omitempty"`
	Revision     int64    `json:"expected_revision"`
}

// parseFeedSubscriptionBody admits one subscription write for its
// caller, binding the subscriber and returning the expected revision.
func parseFeedSubscriptionBody(data []byte, subscriber string) (factory.FeedSubscription, int64, error) {
	var empty factory.FeedSubscription
	var body feedSubscriptionBody
	if err := json.Unmarshal(data, &body); err != nil {
		return empty, 0, errFeedQueryInvalid
	}
	if subscriber == "" {
		return empty, 0, errFeedQueryInvalid
	}
	sub := factory.FeedSubscription{Subscriber: subscriber, Repositories: body.Repositories, Kinds: body.Kinds, Revision: body.Revision + 1}
	if body.Revision < 0 {
		return empty, 0, errFeedQueryInvalid
	}
	if err := sub.Validate(); err != nil {
		return empty, 0, err
	}
	return sub, body.Revision, nil
}

// apiFactoryFeedSubscription reads and writes the caller's own
// combined-feed subscription: the scope and kind filter applied
// inside already permitted activity, never a grant.
func (s *API) apiFactoryFeedSubscription(w http.ResponseWriter, r *http.Request, v store.Session) {
	subscriber := strconv.FormatInt(v.User.ID, 10)
	switch r.Method {
	case http.MethodGet:
		sub, revision, err := s.Store.FeedSubscription(r.Context(), subscriber)
		if err != nil {
			auth.JSONError(w, 503, "store_unavailable", "Could not read feed subscription.")
			return
		}
		auth.JSONResponse(w, 200, map[string]any{"subscriber": subscriber, "repositories": sub.Repositories, "kinds": sub.Kinds, "revision": revision})
	case http.MethodPut:
		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			auth.JSONError(w, 400, "invalid_request", "Feed subscription body is invalid.")
			return
		}
		sub, expected, err := parseFeedSubscriptionBody(raw, subscriber)
		if err != nil {
			auth.JSONError(w, 400, "invalid_request", "Feed subscription body is invalid.")
			return
		}
		if err := s.Store.SetFeedSubscription(r.Context(), subscriber, expected, sub); err != nil {
			if err == store.ErrStaleRevision {
				auth.JSONError(w, 409, "stale_revision", "Feed subscription changed underneath the write.")
				return
			}
			auth.JSONError(w, 503, "store_unavailable", "Could not write feed subscription.")
			return
		}
		auth.JSONResponse(w, 200, map[string]any{"subscriber": subscriber, "revision": expected + 1})
	default:
		auth.JSONError(w, 405, "invalid_request", "Feed subscription reads and writes only.")
	}
}
