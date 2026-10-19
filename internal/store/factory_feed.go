package store

import (
	"context"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// SubscribedFeed reads canonical assignments, comments and milestones in
// newest-first recording order. Each milestone receives its own key from
// the assignment/comment sequence when recorded, including JEV selections
// whose issue has not dispatched. The exclusive seq cursor is shared by
// all kinds. Reads do not allocate keys or dispatch work. Scope filtering
// happens here and access is checked again by the serving layer.
func (s *Store) SubscribedFeed(ctx context.Context, query factory.FeedQuery) ([]factory.FeedEntry, string, bool, error) {
	if err := query.Validate(); err != nil {
		return nil, "", false, err
	}
	wantsRecords, wantsPosts := false, false
	if len(query.Kinds) == 0 {
		wantsRecords, wantsPosts = true, true
	}
	for _, kind := range query.Kinds {
		if kind == factory.FeedKindRecord {
			wantsRecords = true
		}
		if kind == factory.FeedKindPost {
			wantsPosts = true
		}
	}
	if !wantsRecords && !wantsPosts {
		return nil, "", false, nil
	}
	bound, err := factory.ParseFeedCursor(query.Cursor)
	if err != nil {
		return nil, "", false, err
	}
	rows, err := s.db.QueryContext(ctx, `WITH members AS (
			SELECT seq,id,repository,'record' AS kind,'' AS activity,'' AS assignment
			FROM factory_assignments
			WHERE $1 AND repository = ANY($2) AND ($3 = 0 OR seq < $3)
			UNION ALL
			SELECT seq,id,repository,'post','','' FROM factory_feed_comments
			WHERE (($4 AND NOT (data ? 'communication')) OR ($1 AND data ? 'communication'))
 AND repository = ANY($2) AND ($3 = 0 OR seq < $3)
			AND COALESCE((data->>'deleted')::boolean,false) = false
			AND COALESCE((data->>'revoked')::boolean,false) = false
			UNION ALL
			SELECT c.feed_seq,c.event,a.repository,'record','completed',a.id
			FROM factory_completions c
			JOIN factory_publications p ON (p.data->>'id')=c.publication
			JOIN factory_assignments a ON a.id=p.assignment
			WHERE $1 AND c.repository=a.repository AND a.repository = ANY($2)
			AND ($3 = 0 OR c.feed_seq < $3)
			UNION ALL
			SELECT m.feed_seq,m.publication,a.repository,'record','merged',a.id
			FROM factory_merges m
			JOIN factory_publications p ON (p.data->>'id')=m.publication
			JOIN factory_assignments a ON a.id=p.assignment
			WHERE $1 AND m.stage='merged' AND a.repository = ANY($2)
			AND ($3 = 0 OR m.feed_seq < $3)
			UNION ALL
			SELECT j.feed_seq,j.snapshot,a.repository,'record','judgment',a.id
			FROM factory_priority_judgments j
			JOIN factory_publications p ON (p.data->>'id')=(j.data->'input'->>'publication_id')
			JOIN factory_assignments a ON a.id=p.assignment
			WHERE $1 AND j.data->>'kind'='delivery-v1' AND a.repository = ANY($2)
			AND ($3 = 0 OR j.feed_seq < $3)
			UNION ALL
			SELECT j.feed_seq,j.snapshot,sel.repo,'record','judgment',COALESCE(a.id,'')
			FROM factory_priority_judgments j
			CROSS JOIN LATERAL (
				SELECT CASE WHEN c->'control'->>'repository' ~ '^[0-9]+$'
					THEN (c->'control'->>'repository')::bigint ELSE 0 END AS repo,
				CASE WHEN c->'control'->>'issue' ~ '^[0-9]+$'
					THEN (c->'control'->>'issue')::bigint ELSE 0 END AS issue
				FROM jsonb_array_elements(COALESCE(j.data->'candidates','[]'::jsonb)) c
				WHERE c->>'id'=j.data->>'selected' LIMIT 1
			) sel
			LEFT JOIN LATERAL (
				SELECT seq,id,repository FROM factory_assignments
				WHERE repository=sel.repo AND issue=sel.issue
				AND data->>'created_unix' ~ '^[0-9]+$' AND j.data->>'created_unix' ~ '^[0-9]+$'
				AND (data->>'created_unix')::bigint >= (j.data->>'created_unix')::bigint
				ORDER BY seq ASC LIMIT 1
			) a ON true
			WHERE $1 AND j.data IS NOT NULL AND COALESCE(j.data->>'kind','')<>'delivery-v1'
			AND sel.repo>0 AND sel.issue>0 AND sel.repo = ANY($2)
			AND ($3 = 0 OR j.feed_seq < $3)
		)
		SELECT seq,id,repository,kind,activity,assignment FROM members
		ORDER BY seq DESC LIMIT $5`, wantsRecords, query.Repositories, bound, wantsPosts, query.Limit+1)
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
		if err = rows.Scan(&entry.Seq, &entry.ID, &entry.Repository, &entry.Kind, &entry.Activity, &entry.Assignment); err != nil {
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
