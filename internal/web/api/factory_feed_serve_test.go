package api

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

type fakeFeedReader struct {
	entries   []factory.FeedEntry
	next      string
	truncated bool
	err       error
}

func (f *fakeFeedReader) SubscribedFeed(ctx context.Context, query factory.FeedQuery) ([]factory.FeedEntry, string, bool, error) {
	return f.entries, f.next, f.truncated, f.err
}

func TestServeSubscribedFeedRechecksVisibility(t *testing.T) {
	ctx := context.Background()
	reader := &fakeFeedReader{
		entries: []factory.FeedEntry{
			{Kind: factory.FeedKindRecord, ID: "a", Repository: 7, Seq: 9},
			{Kind: factory.FeedKindRecord, ID: "b", Repository: 9, Seq: 8},
			{Kind: factory.FeedKindRecord, ID: "c", Repository: 7, Seq: 7},
		},
		next:      factory.FormatFeedCursor(7),
		truncated: true,
	}
	// Revoked access to repository 9 filters its entry without leaking
	// counts: the window carries no totals, only members.
	visible := func(entry factory.FeedEntry) bool { return entry.Repository == 7 }
	view, err := ServeSubscribedFeed(ctx, reader, visible, factory.FeedQuery{Repositories: []int64{7, 9}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Entries) != 2 || view.Entries[0].ID != "a" || view.Entries[1].ID != "c" {
		t.Fatalf("visible window: %+v", view.Entries)
	}
	if !view.Truncated || view.Next != factory.FormatFeedCursor(7) {
		t.Fatalf("resume lost: %+v", view)
	}
	if _, err := ServeSubscribedFeed(ctx, reader, visible, factory.FeedQuery{Limit: 10}); err == nil {
		t.Fatal("unscoped query served")
	}
	if _, err := ServeSubscribedFeed(ctx, nil, visible, factory.FeedQuery{Repositories: []int64{7}, Limit: 10}); err == nil {
		t.Fatal("nil reader served")
	}
	broken := &fakeFeedReader{err: errors.New("store down")}
	if _, err := ServeSubscribedFeed(ctx, broken, visible, factory.FeedQuery{Repositories: []int64{7}, Limit: 10}); err == nil {
		t.Fatal("store failure swallowed")
	}
}
