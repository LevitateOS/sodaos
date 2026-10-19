package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// ErrTakeoverConflict reports a reused takeover identity with changed
// content. The same run and member with identical content replays.
var ErrTakeoverConflict = errors.New("takeover identity reused for different content")

// RecordTakeover stores one member's takeover of a reconciled run. Each
// member holds their own independent record; replaying the same record
// returns it instead of copying twice.
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
	result, err := tx.ExecContext(ctx, `INSERT INTO factory_takeovers(run,member,project,data) VALUES($1,$2,$3,$4) ON CONFLICT(run,member) DO NOTHING`,
		r.Run, r.Member, r.Project, string(data))
	if err != nil {
		return factory.TakeoverRecord{}, fmt.Errorf("takeover record failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	if n == 1 {
		repository, issue, assignment, scopeErr := factoryRunFeedScopeTx(ctx, tx, r.Run, r.Project)
		if scopeErr != nil {
			return factory.TakeoverRecord{}, scopeErr
		}
		at, _ := time.Parse(time.RFC3339, r.Copied)
		if repository > 0 {
			if err = recordFactoryFeedEventTx(ctx, tx, repository, issue, assignment, r.CommandID, "takeover:copied", factory.FeedCommunication{
				ComponentID: "F08", Title: "Work copied for human takeover", Summary: r.Member + " can continue from the retained worker changes.",
				Items: []factory.FeedCommunicationItem{{Label: "Working copy", State: "copied"}},
			}, at.Unix()); err != nil {
				return factory.TakeoverRecord{}, err
			}
		}
		return r, tx.Commit()
	}
	if err = tx.Commit(); err != nil {
		return factory.TakeoverRecord{}, err
	}
	existing, err := s.Takeover(ctx, r.Run, r.Member)
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	if existing != r {
		return factory.TakeoverRecord{}, ErrTakeoverConflict
	}
	return existing, nil
}

// Takeover returns one member's recorded takeover of a run.
func (s *Store) Takeover(ctx context.Context, run, member string) (factory.TakeoverRecord, error) {
	var record factory.TakeoverRecord
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_takeovers WHERE run=$1 AND member=$2`, run, member).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &record)
	}
	return record, err
}
