package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// mcpVisibleDiscussion derives the worker's conversation from its actual
// assignment posts and replies to that assignment. Repository scope alone is
// insufficient: unrelated project discussions stay outside this read.
const mcpVisibleDiscussion = `WITH RECURSIVE visible AS (
 SELECT id,seq,data FROM factory_feed_comments
 WHERE repository=$1 AND NOT COALESCE((data->>'deleted')::boolean,false) AND NOT COALESCE((data->>'revoked')::boolean,false)
 AND (data->>'assignment'=$2 OR (data->'source'->>'kind'='record' AND data->'source'->>'id'=$2))
 UNION
 SELECT c.id,c.seq,c.data FROM factory_feed_comments c JOIN visible p
 ON c.data->>'reply_to'=p.id OR (c.data->'source'->>'kind'='post' AND c.data->'source'->>'id'=p.id)
 WHERE c.repository=$1 AND NOT COALESCE((c.data->>'deleted')::boolean,false) AND NOT COALESCE((c.data->>'revoked')::boolean,false)
)`

func (s *Store) MCPInbox(ctx context.Context, capability factory.MCPCapability, cursor string) (factory.MCPFeedPage, error) {
	return s.mcpDiscussionPage(ctx, capability, "", cursor)
}

func (s *Store) MCPDiscussion(ctx context.Context, capability factory.MCPCapability, postID, cursor string) (factory.MCPFeedPage, error) {
	if !factory.ValidID(postID) {
		return factory.MCPFeedPage{}, errors.New("discussion needs a post_id")
	}
	var visible bool
	if err := s.db.QueryRowContext(ctx, mcpVisibleDiscussion+` SELECT EXISTS(SELECT 1 FROM visible WHERE id=$3)`, capability.Repository, capability.AssignmentID, postID).Scan(&visible); err != nil {
		return factory.MCPFeedPage{}, err
	}
	if !visible {
		return factory.MCPFeedPage{}, ErrNotFound
	}
	return s.mcpDiscussionPage(ctx, capability, postID, cursor)
}

func (s *Store) mcpDiscussionPage(ctx context.Context, capability factory.MCPCapability, postID, cursor string) (factory.MCPFeedPage, error) {
	out := factory.MCPFeedPage{Items: []factory.MCPFeedMessage{}}
	before, err := factory.ParseFeedCursor(cursor)
	if err != nil {
		return out, err
	}
	query := mcpVisibleDiscussion
	if postID == "" {
		query += ` SELECT id,seq FROM visible WHERE ($3::bigint=0 OR seq<$3) ORDER BY seq DESC LIMIT 51`
	} else {
		query += `, thread AS (
   SELECT id,seq,data FROM visible WHERE id=$4
   UNION SELECT c.id,c.seq,c.data FROM visible c JOIN thread p
   ON c.data->>'reply_to'=p.id OR (c.data->'source'->>'kind'='post' AND c.data->'source'->>'id'=p.id)
  ) SELECT id,seq FROM thread WHERE ($3::bigint=0 OR seq<$3) ORDER BY seq DESC LIMIT 51`
	}
	args := []any{capability.Repository, capability.AssignmentID, before}
	if postID != "" {
		args = append(args, postID)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return out, err
	}
	type member struct {
		id  string
		seq int64
	}
	members := []member{}
	for rows.Next() {
		var m member
		if err := rows.Scan(&m.id, &m.seq); err != nil {
			rows.Close()
			return out, err
		}
		members = append(members, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.id)
	}
	details, err := s.FeedPostDetails(ctx, ids)
	if err != nil {
		return out, err
	}
	totalBytes := 128
	for _, m := range members {
		detail, ok := details[m.id]
		if !ok {
			continue
		}
		message := factory.MCPFeedMessage{ID: m.id, ReplyTo: detail.Record.ReplyTo, Actor: detail.Record.Author, Author: detail.AuthorName, At: detail.At, Status: detail.Status, Summary: TruncateFeedSummary(detail.Body)}
		if detail.Actor != "" {
			message.Actor = detail.Actor
		}
		if postID != "" {
			message.Body = detail.Body
			if detail.Payload != nil {
				message.Communication = detail.Payload.Communication
			}
		}
		encoded, err := json.Marshal(message)
		if err != nil {
			return out, err
		}
		if len(out.Items) == 50 || totalBytes+len(encoded) > factory.MCPMaxPageBytes {
			out.Incomplete = true
			break
		}
		out.Items = append(out.Items, message)
		totalBytes += len(encoded) + 1
		out.NextCursor = factory.FormatFeedCursor(m.seq)
	}
	if !out.Incomplete {
		out.NextCursor = ""
	}
	return out, nil
}
