package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/levitateos/sodaos/internal/factory"
)

func (s *Store) PlanningRecord(ctx context.Context, repository, issue int64) (factory.PlanningRecord, error) {
	var r factory.PlanningRecord
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_planning WHERE repository=$1 AND issue=$2`, repository, issue).Scan(&raw)
	if err == nil {
		err = json.Unmarshal(raw, &r)
	}
	return r, err
}
func (s *Store) SavePlanningRecord(ctx context.Context, r factory.PlanningRecord) error {
	if err := r.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO factory_planning(repository,issue,snapshot,data) VALUES($1,$2,$3,$4) ON CONFLICT(repository,issue) DO UPDATE SET snapshot=excluded.snapshot,data=excluded.data`, r.Input.Candidate.Control.Repository, r.Input.Candidate.Control.Issue, r.Snapshot, string(raw))
	return err
}
func (s *Store) PlanningControlsAfter(ctx context.Context, limit int, seen, repository, issue int64, hasCursor bool) ([]factory.IssueControl, error) {
	if limit < 1 || limit > 2048 {
		return nil, errors.New("invalid planning inventory limit")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM issue_controls WHERE data->>'readiness' IN ('queued','blocked') AND COALESCE(data->>'acceptance','')<>'' AND (NOT $1 OR ((data->>'first_seen_unix')::bigint,repository,issue)>($2,$3,$4)) ORDER BY (data->>'first_seen_unix')::bigint,repository,issue LIMIT $5`, hasCursor, seen, repository, issue, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []factory.IssueControl{}
	for rows.Next() {
		var raw []byte
		var c factory.IssueControl
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
