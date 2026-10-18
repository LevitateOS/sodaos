package factory

import (
	"errors"
	"sort"
)

// FeedSubscription is one reader's combined-feed preferences: the
// repository allow-list and kind filter applied inside already
// permitted activity. A subscription never grants access; the serving
// layer intersects it with the caller's permitted scope and rechecks
// visibility per entry. Revision guards concurrent preference writes:
// a stale write refuses instead of dropping another update.
type FeedSubscription struct {
	Subscriber   string   `json:"subscriber"`
	Repositories []int64  `json:"repositories"`
	Kinds        []string `json:"kinds,omitempty"`
	Revision     int64    `json:"revision"`
}

// Validate admits one feed subscription with a positive revision and
// scope.
func (s FeedSubscription) Validate() error {
	if s.Subscriber == "" {
		return errors.New("feed subscription needs its subscriber")
	}
	if len(s.Repositories) == 0 {
		return errors.New("feed subscription needs its repository scope")
	}
	for _, repository := range s.Repositories {
		if repository <= 0 {
			return errors.New("feed subscription needs its positive repository scope")
		}
	}
	for _, kind := range s.Kinds {
		if kind != FeedKindRecord && kind != FeedKindPost {
			return errors.New("feed subscription needs its known kinds")
		}
	}
	if s.Revision < 1 {
		return errors.New("feed subscription needs its positive revision")
	}
	return nil
}

// ApplyFeedSubscription narrows one feed query to the intersection of
// its scope with the subscription, preserving kinds, window and
// cursor. An empty intersection refuses honestly at serve time
// through query validation; nothing renders and nothing leaks.
func ApplyFeedSubscription(query FeedQuery, sub FeedSubscription) FeedQuery {
	member := make(map[int64]bool, len(sub.Repositories))
	for _, repository := range sub.Repositories {
		member[repository] = true
	}
	narrowed := query
	narrowed.Repositories = nil
	for _, repository := range query.Repositories {
		if member[repository] {
			narrowed.Repositories = append(narrowed.Repositories, repository)
		}
	}
	sort.Slice(narrowed.Repositories, func(i, j int) bool { return narrowed.Repositories[i] < narrowed.Repositories[j] })
	return narrowed
}
