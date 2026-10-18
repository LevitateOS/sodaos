package factory

import (
	"testing"
)

func TestFeedCursorRoundTrip(t *testing.T) {
	cursor := FormatFeedCursor(42)
	seq, err := ParseFeedCursor(cursor)
	if err != nil {
		t.Fatalf("parse cursor: %v", err)
	}
	if seq != 42 {
		t.Fatalf("cursor bound = %d, want 42", seq)
	}
	if seq, err := ParseFeedCursor(""); err != nil || seq != 0 {
		t.Fatalf("empty cursor = %d, %v, want 0, nil", seq, err)
	}
	for _, bad := range []string{"42", "v2:42", "v1:0", "v1:-3", "v1:abc", "v1:1:2"} {
		if _, err := ParseFeedCursor(bad); err == nil {
			t.Fatalf("cursor %q admitted", bad)
		}
	}
}

func TestFeedQueryValidation(t *testing.T) {
	valid := FeedQuery{Repositories: []int64{7}, Limit: 10}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid query refused: %v", err)
	}
	invalid := []FeedQuery{
		{},
		{Repositories: []int64{0}, Limit: 10},
		{Repositories: []int64{7}, Limit: 0},
		{Repositories: []int64{7}, Limit: FeedPageLimit + 1},
		{Repositories: []int64{7}, Limit: 10, Kinds: []string{" digest"}},
		{Repositories: []int64{7}, Limit: 10, Cursor: "v9:1"},
	}
	for i, query := range invalid {
		if err := query.Validate(); err == nil {
			t.Fatalf("query %d admitted", i)
		}
	}
}

func TestFeedWindowNewestFirst(t *testing.T) {
	entries := []FeedEntry{
		{Kind: FeedKindRecord, ID: "b", Repository: 1, Seq: 3},
		{Kind: FeedKindPost, ID: "a", Repository: 2, Seq: 9},
		{Kind: FeedKindRecord, ID: "c", Repository: 1, Seq: 5},
	}
	SortFeedEntriesNewestFirst(entries)
	if entries[0].Seq != 9 || entries[1].Seq != 5 || entries[2].Seq != 3 {
		t.Fatalf("unordered window: %+v", entries)
	}
	page, next, truncated := WindowFeedEntries(entries, 2)
	if len(page) != 2 || !truncated {
		t.Fatalf("window = %d entries truncated=%v, want 2 true", len(page), truncated)
	}
	seq, err := ParseFeedCursor(next)
	if err != nil || seq != 5 {
		t.Fatalf("resume = %q (%d, %v), want bound 5", next, seq, err)
	}
	page, next, truncated = WindowFeedEntries(entries, 3)
	if len(page) != 3 || truncated || next == "" {
		t.Fatalf("full window = %d entries truncated=%v next=%q", len(page), truncated, next)
	}
}
