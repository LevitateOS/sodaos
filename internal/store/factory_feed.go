package store

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// SubscribedFeed reads one combined feed window over canonical
// assignment records across the query's repositories, newest-first by
// the table-global order key with the resume cursor as the exclusive
// bound. The scope allow-list applies inside the read; per-entry
// access recheck stays with the serving layer, which must filter the
// returned window before rendering. No totals are reported, so counts
// cannot leak through this read. Communication posts have no producer
// yet: a posts-only query reads empty without truncation until R32
// records them.
func (s *Store) SubscribedFeed(ctx context.Context, query factory.FeedQuery) ([]factory.FeedEntry, string, bool, error) {
	if err := query.Validate(); err != nil {
		return nil, "", false, err
	}
	wantsRecords := len(query.Kinds) == 0
	for _, kind := range query.Kinds {
		if kind == factory.FeedKindRecord {
			wantsRecords = true
		}
	}
	if !wantsRecords {
		return nil, "", false, nil
	}
	bound, err := factory.ParseFeedCursor(query.Cursor)
	if err != nil {
		return nil, "", false, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT seq,id,repository FROM factory_assignments
		WHERE repository = ANY($1) AND ($2 = 0 OR seq < $2)
		ORDER BY seq DESC LIMIT $3`, query.Repositories, bound, query.Limit+1)
	if err != nil {
		return nil, "", false, err
	}
	defer func() { _ = rows.Close() }()
	allowed := make(map[int64]bool, len(query.Repositories))
	for _, repository := range query.Repositories {
		allowed[repository] = true
	}
	entries := make([]factory.FeedEntry, 0, query.Limit+1)
	for rows.Next() {
		var entry factory.FeedEntry
		entry.Kind = factory.FeedKindRecord
		if err = rows.Scan(&entry.Seq, &entry.ID, &entry.Repository); err != nil {
			return nil, "", false, err
		}
		if err = entry.Validate(); err != nil {
			return nil, "", false, err
		}
		if !allowed[entry.Repository] {
			return nil, "", false, errors.New("feed entry scope mismatch")
		}
		entries = append(entries, entry)
	}
	if err = rows.Err(); err != nil {
		return nil, "", false, err
	}
	page, next, truncated := factory.WindowFeedEntries(entries, query.Limit)
	return page, next, truncated, nil
}
