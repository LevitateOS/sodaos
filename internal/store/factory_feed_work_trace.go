package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// CommentLinkedWork reads the actual native work currently linked to a human
// comment. It deliberately does not infer adoption from issue progress. Only
// the separate recorded post disposition can say the comment was incorporated.
func (s *Store) CommentLinkedWork(ctx context.Context, repository int64, comment string) (map[string]any, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT jsonb_build_object('status','recorded','issue',r.issue,
 'adoption',COALESCE((SELECT jsonb_build_object('state',d.resolution,'actor',d.actor,'assignment',d.assignment,'reason',d.reason) FROM factory_post_resolutions d WHERE d.post=c.id AND d.repository=c.repository),'{}'::jsonb),
 'work',COALESCE((SELECT jsonb_agg(item ORDER BY phase) FROM (
  SELECT DISTINCT ON (a.data->>'phase') a.data->>'phase' AS phase,jsonb_build_object('assignment',a.id,'phase',COALESCE(a.data->>'phase',''),'stage',a.stage,'outcome',COALESCE(a.data->>'outcome',''),'source_revision',COALESCE(a.data->>'source_commit','')) AS item
  FROM factory_assignments a WHERE a.repository=c.repository AND a.issue=r.issue AND r.issue>0 ORDER BY a.data->>'phase',a.seq DESC
 ) phases),'[]'::jsonb),
 'completion',COALESCE((SELECT e.event FROM factory_completions e
 JOIN factory_publications p ON p.data->>'id'=e.publication
 JOIN factory_assignments a ON a.id=p.assignment AND a.repository=e.repository
 WHERE e.repository=c.repository AND a.issue=r.issue AND r.issue>0 ORDER BY e.generation DESC LIMIT 1),''))
 FROM factory_feed_comments c JOIN factory_feed_comment_requests r ON r.comment=c.id
 WHERE c.id=$1 AND c.repository=$2 AND r.issue>0`, comment, repository).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var trace map[string]any
	if err = json.Unmarshal(raw, &trace); err != nil {
		return nil, err
	}
	return trace, nil
}
