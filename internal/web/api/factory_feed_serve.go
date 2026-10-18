package api

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

var errFeedUnavailable = errors.New("combined feed is unavailable")

var errFeedQueryInvalid = errors.New("combined feed query is invalid")

// FeedReader reads one combined feed window over canonical records.
type FeedReader interface {
	SubscribedFeed(ctx context.Context, query factory.FeedQuery) ([]factory.FeedEntry, string, bool, error)
}

// FeedPageView renders one authorized combined feed window. No totals
// are reported, so counts cannot leak through this read. A visibility
// cut can short a page; the resume cursor still advances past filtered
// entries, so resuming loses nothing and reveals nothing.
type FeedPageView struct {
	Entries   []factory.FeedEntry `json:"entries"`
	Next      string              `json:"next,omitempty"`
	Truncated bool                `json:"truncated"`
}

// ServeSubscribedFeed serves one authorized combined feed window: the
// query narrowed to the subscription intersection, then the store
// window filtered through a per-entry visibility recheck. The
// visibility check runs on every serve — including after revocation,
// account changes or a subscription change — and subscription
// membership never grants access on its own.
func ServeSubscribedFeed(ctx context.Context, reader FeedReader, visible func(factory.FeedEntry) bool, query factory.FeedQuery, sub factory.FeedSubscription) (FeedPageView, error) {
	var empty FeedPageView
	if reader == nil || visible == nil {
		return empty, errFeedUnavailable
	}
	if err := sub.Validate(); err != nil {
		return empty, err
	}
	query = factory.ApplyFeedSubscription(query, sub)
	if err := query.Validate(); err != nil {
		return empty, err
	}
	entries, next, truncated, err := reader.SubscribedFeed(ctx, query)
	if err != nil {
		return empty, err
	}
	view := FeedPageView{Entries: []factory.FeedEntry{}, Next: next, Truncated: truncated}
	for _, entry := range entries {
		if err := entry.Validate(); err != nil {
			return empty, err
		}
		if !visible(entry) {
			continue
		}
		view.Entries = append(view.Entries, entry)
	}
	return view, nil
}
