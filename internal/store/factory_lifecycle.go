package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// ErrTakeoverConflict reports a reused takeover identity with changed
// content. The same run and member with identical content replays.
var ErrTakeoverConflict = errors.New("takeover identity reused for different content")

// RecordTakeover saves the native copy result on its attributed Feed post.
// The run/member key identifies both the event and its canonical result.
func (s *Store) RecordTakeover(ctx context.Context, r factory.TakeoverRecord) (factory.TakeoverRecord, error) {
	if err := r.Validate(); err != nil {
		return factory.TakeoverRecord{}, err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	defer tx.Rollback()
	repository, issue, assignment, err := factoryRunFeedScopeTx(ctx, tx, r.Run, r.Project)
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	if repository <= 0 {
		return factory.TakeoverRecord{}, ErrNotFound
	}
	identity := "takeover:" + r.Run + ":" + r.Member
	at, _ := time.Parse(time.RFC3339, r.Copied)
	if err = recordFactoryFeedEventTx(ctx, tx, repository, issue, assignment, identity, "takeover:copied", factory.FeedCommunication{
		ComponentID: "F08", Title: "Work copied for human takeover", Summary: r.Member + " can continue from the retained worker changes.",
		Items: []factory.FeedCommunicationItem{{Label: "Working copy", State: "copied"}},
	}, at.Unix()); err != nil {
		return factory.TakeoverRecord{}, err
	}
	var raw []byte
	if err = tx.QueryRowContext(ctx, `UPDATE factory_feed_posts SET data=jsonb_set(data,'{takeover}',COALESCE(data->'takeover',$1::jsonb)) WHERE repository=$2 AND author='factory' AND data->>'event_identity'=$3 AND data->>'status'='takeover:copied' RETURNING data->'takeover'`, string(data), repository, identity).Scan(&raw); err != nil {
		return factory.TakeoverRecord{}, err
	}
	var stored factory.TakeoverRecord
	if err = json.Unmarshal(raw, &stored); err != nil {
		return factory.TakeoverRecord{}, err
	}
	if stored != r {
		return factory.TakeoverRecord{}, ErrTakeoverConflict
	}
	return stored, tx.Commit()
}

// Takeover reads one member's completed native copy from its Feed post.
func (s *Store) Takeover(ctx context.Context, run, member string) (factory.TakeoverRecord, error) {
	var result factory.TakeoverRecord
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT data->'takeover' FROM factory_feed_posts WHERE author='factory' AND data->>'event_identity'=$1 AND data->>'status'='takeover:copied' AND data ? 'takeover'`, "takeover:"+run+":"+member).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &result)
	}
	return result, err
}
