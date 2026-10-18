package factory

import (
	"testing"
)

func TestApplyFeedSubscriptionNarrowsScope(t *testing.T) {
	sub := FeedSubscription{Subscriber: "reader-1", Repositories: []int64{7, 9}, Revision: 1}
	if err := sub.Validate(); err != nil {
		t.Fatalf("valid subscription refused: %v", err)
	}
	query := FeedQuery{Repositories: []int64{7, 9, 11}, Limit: 10}
	narrowed := ApplyFeedSubscription(query, sub)
	if len(narrowed.Repositories) != 2 || narrowed.Repositories[0] != 7 || narrowed.Repositories[1] != 9 {
		t.Fatalf("narrowed scope: %+v", narrowed.Repositories)
	}
	if err := narrowed.Validate(); err != nil {
		t.Fatalf("narrowed query refused: %v", err)
	}
	empty := ApplyFeedSubscription(FeedQuery{Repositories: []int64{11}, Limit: 10}, sub)
	if err := empty.Validate(); err == nil {
		t.Fatal("empty intersection admitted")
	}
	for _, bad := range []FeedSubscription{
		{Repositories: []int64{7}, Revision: 1},
		{Subscriber: "r", Revision: 1},
		{Subscriber: "r", Repositories: []int64{0}, Revision: 1},
		{Subscriber: "r", Repositories: []int64{7}, Kinds: []string{"digest"}, Revision: 1},
		{Subscriber: "r", Repositories: []int64{7}},
	} {
		if err := bad.Validate(); err == nil {
			t.Fatalf("subscription admitted: %+v", bad)
		}
	}
}
