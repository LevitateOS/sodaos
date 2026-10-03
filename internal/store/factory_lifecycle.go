package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
	result, err := s.exec(ctx, `INSERT INTO factory_takeovers(run,member,project,data) VALUES(?,?,?,?) ON CONFLICT(run,member) DO NOTHING`,
		r.Run, r.Member, r.Project, string(data))
	if err != nil {
		return factory.TakeoverRecord{}, fmt.Errorf("takeover record failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.TakeoverRecord{}, err
	}
	if n == 1 {
		return r, nil
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
	err := s.queryRow(ctx, `SELECT data FROM factory_takeovers WHERE run=? AND member=?`, run, member).Scan(&data)
	if err == nil {
		err = json.Unmarshal(data, &record)
	}
	return record, err
}
