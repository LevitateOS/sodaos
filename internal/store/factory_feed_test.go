package store

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestSubscribedFeedWindowsAcrossRepositories(t *testing.T) {
	ctx := context.Background()
	s, dsn := postgresFixture(t, nil)
	raw := journeySQL(t, dsn)
	acceptance := "d" + strings.Repeat("a", 24)
	// Insertion order differs from repository grouping: the feed must
	// still read newest-first across projects by the global order key.
	insertJourneyAssignment(t, raw, journeyAssignment(strings.Repeat("1", 32), strings.Repeat("1", 32), acceptance, 7, 3, true), "assigned", 1)
	insertJourneyAssignment(t, raw, journeyAssignment(strings.Repeat("2", 32), strings.Repeat("2", 32), acceptance, 9, 5, true), "assigned", 1)
	insertJourneyAssignment(t, raw, journeyAssignment(strings.Repeat("3", 32), strings.Repeat("3", 32), acceptance, 7, 4, true), "assigned", 1)

	page, next, truncated, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 3 || truncated || next == "" {
		t.Fatalf("full window = %d entries truncated=%v next=%q", len(page), truncated, next)
	}
	if page[0].ID != strings.Repeat("3", 32) || page[1].ID != strings.Repeat("2", 32) || page[2].ID != strings.Repeat("1", 32) {
		t.Fatalf("unordered window: %+v", page)
	}

	first, resume, truncated, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Limit: 2})
	if err != nil || len(first) != 2 || !truncated {
		t.Fatalf("cut window = %d entries truncated=%v err=%v", len(first), truncated, err)
	}
	rest, _, truncated, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Limit: 2, Cursor: resume})
	if err != nil || len(rest) != 1 || truncated || rest[0].ID != strings.Repeat("1", 32) {
		t.Fatalf("resumed window = %+v truncated=%v err=%v", rest, truncated, err)
	}

	scoped, _, truncated, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{9}, Limit: 10})
	if err != nil || len(scoped) != 1 || truncated || scoped[0].Repository != 9 {
		t.Fatalf("scoped window = %+v truncated=%v err=%v", scoped, truncated, err)
	}

	posts, next, truncated, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7, 9}, Kinds: []string{factory.FeedKindPost}, Limit: 10})
	if err != nil || len(posts) != 0 || truncated || next != "" {
		t.Fatalf("posts-only window = %+v next=%q truncated=%v err=%v", posts, next, truncated, err)
	}

	if _, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Repositories: []int64{7}, Limit: 10, Cursor: "v9:1"}); err == nil {
		t.Fatal("versioned cursor admitted")
	}
	if _, _, _, err := s.SubscribedFeed(ctx, factory.FeedQuery{Limit: 10}); err == nil {
		t.Fatal("unscoped query admitted")
	}
}
