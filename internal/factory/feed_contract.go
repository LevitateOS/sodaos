package factory

import (
	"errors"
	"sort"
	"strconv"
	"strings"
)

// Combined subscribed-feed query/cursor contract (R42 preparation).
//
// The combined feed reads newest-first across projects with a stable
// unique tie-break and bounded keyset paging; ordering never depends
// on whichever page arrived last. Entries come from canonical record
// owners only: existing workflow records (assignments, judgments,
// corrections rendered from the journey readback owners) and, once
// R32 produces them, communication posts. No fixture-only DTO or
// parallel dashboard may serve this contract.
//
// Subscription membership is a filter within already permitted
// activity, never a grant: the serving layer rechecks access per
// entry — including after revocation, account changes or a
// subscription change — and forbidden entries and counts must not
// leak. A cut page names its resume cursor and marks itself
// incomplete instead of rendering a partial window as the whole feed.

// Feed entry kinds admitted to the combined feed.
const (
	// FeedKindRecord is one canonical workflow record rendered from
	// its existing owner (assignment, judgment, correction).
	FeedKindRecord = "record"
	// FeedKindPost is one R32 communication post, comment or revision.
	FeedKindPost = "post"
)

// feedCursorVersion versions the opaque feed cursor.
const feedCursorVersion = "v1"

// FeedPageLimit bounds one combined feed window, mirroring the
// existing 256-row read windows.
const FeedPageLimit = 256

// FeedEntry is one combined feed member: a stable unique identity
// with its global order key and scope. Seq is the table-global
// sequence behind the owning record (as factory_assignments.seq),
// unique across projects, so newest-first by seq is deterministic
// with no second key.
type FeedEntry struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Repository int64  `json:"repository"`
	Seq        int64  `json:"seq"`
}

// Validate admits one feed entry with a known kind and positive scope.
func (e FeedEntry) Validate() error {
	if e.Kind != FeedKindRecord && e.Kind != FeedKindPost {
		return errors.New("feed entry needs its known kind")
	}
	if e.ID == "" {
		return errors.New("feed entry needs its stable identity")
	}
	if e.Repository <= 0 || e.Seq <= 0 {
		return errors.New("feed entry needs its positive scope and order")
	}
	return nil
}

// FeedQuery scopes one combined feed read: an explicit repository
// allow-list (the subscription filter applied inside permitted
// activity), an optional kind filter, a bounded window and the resume
// cursor. Empty cursor reads from the latest; empty kinds read all.
type FeedQuery struct {
	Repositories []int64  `json:"repositories"`
	Kinds        []string `json:"kinds,omitempty"`
	Limit        int      `json:"limit"`
	Cursor       string   `json:"cursor,omitempty"`
}

// Validate admits one feed query with a bounded window and a
// well-formed cursor.
func (q FeedQuery) Validate() error {
	if len(q.Repositories) == 0 {
		return errors.New("feed query needs its repository scope")
	}
	for _, repository := range q.Repositories {
		if repository <= 0 {
			return errors.New("feed query needs its positive repository scope")
		}
	}
	for _, kind := range q.Kinds {
		if kind != FeedKindRecord && kind != FeedKindPost {
			return errors.New("feed query needs its known kinds")
		}
	}
	if q.Limit <= 0 || q.Limit > FeedPageLimit {
		return errors.New("feed query window is outside its bound")
	}
	if _, err := ParseFeedCursor(q.Cursor); err != nil {
		return err
	}
	return nil
}

// FormatFeedCursor renders the resume cursor behind one entry seq:
// the next page reads strictly older entries.
func FormatFeedCursor(seq int64) string {
	return feedCursorVersion + ":" + strconv.FormatInt(seq, 10)
}

// ParseFeedCursor admits one feed cursor, returning the exclusive
// order bound: entries read with seq below it. Empty admits the
// latest-first read.
func ParseFeedCursor(cursor string) (int64, error) {
	if cursor == "" {
		return 0, nil
	}
	head, tail, ok := strings.Cut(cursor, ":")
	if !ok || head != feedCursorVersion {
		return 0, errors.New("feed cursor needs its versioned order bound")
	}
	seq, err := strconv.ParseInt(tail, 10, 64)
	if err != nil || seq <= 0 {
		return 0, errors.New("feed cursor needs its positive order bound")
	}
	return seq, nil
}

// SortFeedEntriesNewestFirst orders one feed window newest-first by
// the global order key. Seq is unique, so the order is deterministic
// regardless of page arrival order; the ID comparison only defends
// against a duplicated order key, which never outranks it.
func SortFeedEntriesNewestFirst(entries []FeedEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Seq != entries[j].Seq {
			return entries[i].Seq > entries[j].Seq
		}
		return entries[i].ID < entries[j].ID
	})
}

// WindowFeedEntries cuts one newest-first read of up to limit+1
// entries to its page: at most limit members, the resume cursor
// behind the page tail, and whether the bound cut the read. A cut
// page stays incomplete; the caller renders the resume cursor instead
// of the window as the whole feed.
func WindowFeedEntries(entries []FeedEntry, limit int) (page []FeedEntry, next string, truncated bool) {
	if len(entries) > limit {
		page = entries[:limit]
		truncated = true
	} else {
		page = entries
	}
	if len(page) > 0 {
		next = FormatFeedCursor(page[len(page)-1].Seq)
	}
	return page, next, truncated
}
