package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// This file owns the correctness and inventory queries shared by factory
// reconciliation, dispatch traversal and the Spaces collection. Bounded
// display readers stay on FactoryRuns; every correctness path below finds
// outstanding work independently of how much settled history exists.

// FactoryUnsettledRuns returns recorded runs that reconcile has not
// settled, oldest first. Callers page until a short batch: settled history
// never displaces live work no matter how many settled rows exist.
func (s *Store) FactoryUnsettledRuns(ctx context.Context, limit int) ([]factory.Run, error) {
	if limit < 1 || limit > 1000 {
		return nil, errors.New("invalid unsettled run list bound")
	}
	rows, err := s.query(ctx, `SELECT data FROM factory_runs WHERE NOT settled ORDER BY seq ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var runs []factory.Run
	for rows.Next() {
		var data []byte
		var r factory.Run
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// ProjectFactoryRuns returns one project's recorded runs, newest first.
// It bounds display history; correctness paths union it with
// ProjectUnsettledRuns so settled history never hides live work.
func (s *Store) ProjectFactoryRuns(ctx context.Context, projectID string, limit int) ([]factory.Run, error) {
	if projectID == "" || limit < 1 || limit > 1000 {
		return nil, errors.New("invalid project run list bound")
	}
	rows, err := s.query(ctx, `SELECT data FROM factory_runs
		WHERE data->>'project_id'=? ORDER BY seq DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var runs []factory.Run
	for rows.Next() {
		var data []byte
		var r factory.Run
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// ProjectUnsettledRuns returns one project's recorded runs that reconcile
// has not settled, oldest first. Stop, resume and verification decisions
// read this instead of filtering a bounded newest-first listing.
func (s *Store) ProjectUnsettledRuns(ctx context.Context, projectID string, limit int) ([]factory.Run, error) {
	if projectID == "" || limit < 1 || limit > 1000 {
		return nil, errors.New("invalid project unsettled run list bound")
	}
	rows, err := s.query(ctx, `SELECT data FROM factory_runs
		WHERE NOT settled AND data->>'project_id'=? ORDER BY seq ASC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var runs []factory.Run
	for rows.Next() {
		var data []byte
		var r factory.Run
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// SpaceProjectsAfter continues the bounded Soda-association scan after one
// exclusive project-ID cursor. An empty cursor starts from the beginning.
// Like SpaceProjects this is not an authorized catalog: callers authorize
// every returned row. Labels keep the same oversized rejection.
func (s *Store) SpaceProjectsAfter(ctx context.Context, after string, limit int) ([]Project, error) {
	if limit < 1 || limit > 129 {
		return nil, errors.New("invalid association page bound")
	}
	columns := `CASE WHEN octet_length(id)<=128 THEN id END,
		CASE WHEN octet_length(name)<=1024 THEN name END,repository_id,owner_id,
		CASE WHEN octet_length(repository)<=2048 THEN repository END,
		CASE WHEN octet_length(ip)<=128 THEN ip END,ready,creation_profile`
	var rows *sql.Rows
	var err error
	if after == "" {
		rows, err = s.query(ctx, `SELECT `+columns+` FROM projects ORDER BY id LIMIT ?`, limit)
	} else {
		rows, err = s.query(ctx, `SELECT `+columns+` FROM projects WHERE id>? ORDER BY id LIMIT ?`, after, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// QueuedControlsAfter continues the deterministic oldest-first queued
// listing after one exclusive (first_seen_unix, repository, issue)
// cursor. Without a cursor it matches QueuedControls from the beginning.
// Dispatch rotation advances the cursor so a waiting prefix cannot starve
// later runnable work across repeated passes.
func (s *Store) QueuedControlsAfter(ctx context.Context, limit int, firstSeen, repository, issue int64, hasCursor bool) ([]factory.IssueControl, error) {
	if limit <= 0 || limit > MaxQueuedDispatch {
		return nil, errors.New("invalid queued control listing limit")
	}
	query := `SELECT data FROM issue_controls
		WHERE data->>'readiness'='queued'
		ORDER BY (data->>'first_seen_unix')::bigint ASC, repository ASC, issue ASC LIMIT ?`
	var rows *sql.Rows
	var err error
	if !hasCursor {
		rows, err = s.query(ctx, query, limit)
	} else {
		rows, err = s.query(ctx, `SELECT data FROM issue_controls
			WHERE data->>'readiness'='queued'
			AND ((data->>'first_seen_unix')::bigint, repository, issue) > (?,?,?)
			ORDER BY (data->>'first_seen_unix')::bigint ASC, repository ASC, issue ASC LIMIT ?`,
			firstSeen, repository, issue, limit)
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var controls []factory.IssueControl
	for rows.Next() {
		var data []byte
		var control factory.IssueControl
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(data, &control); err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	return controls, rows.Err()
}
