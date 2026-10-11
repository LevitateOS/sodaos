package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/levitateos/sodaos/internal/factory"
)

// ErrLifecycleActionConflict reports a reused action identity with changed
// content. Replaying identical content returns the recorded action.
var ErrLifecycleActionConflict = errors.New("lifecycle action identity reused for different content")

// ErrLifecycleActionStale reports an update against a superseded attempt
// count. Only the exact next attempt may record.
var ErrLifecycleActionStale = errors.New("lifecycle action attempt superseded")

// ErrCapabilityUnproved reports an enablement without that event's own
// recorded native capability proof.
var ErrCapabilityUnproved = errors.New("lifecycle event capability unproved")

// RecordAction stores one durable optional-effect action. Replaying the
// same identity with identical content returns the recorded row instead
// of journaling twice.
func (s *Store) RecordAction(ctx context.Context, a factory.LifecycleAction) (factory.LifecycleAction, error) {
	if err := a.Validate(); err != nil {
		return factory.LifecycleAction{}, err
	}
	data, err := json.Marshal(a)
	if err != nil {
		return factory.LifecycleAction{}, err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO factory_lifecycle_actions(id,event,action,repository,issue,attempts,revoked,data) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(id) DO NOTHING`,
		a.ID, a.Event, a.Action, a.RepositoryID, a.IssueID, a.Attempts, a.Revoked, string(data))
	if err != nil {
		return factory.LifecycleAction{}, fmt.Errorf("lifecycle action record failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.LifecycleAction{}, err
	}
	if n == 1 {
		return a, nil
	}
	existing, err := s.LifecycleAction(ctx, a.ID)
	if err != nil {
		return factory.LifecycleAction{}, err
	}
	if !lifecycleActionEqual(existing, a) {
		return factory.LifecycleAction{}, ErrLifecycleActionConflict
	}
	return existing, nil
}

// LifecycleAction returns one recorded action by identity.
func (s *Store) LifecycleAction(ctx context.Context, id string) (factory.LifecycleAction, error) {
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_lifecycle_actions WHERE id=$1`, id).Scan(&raw); err != nil {
		return factory.LifecycleAction{}, err
	}
	var a factory.LifecycleAction
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		return factory.LifecycleAction{}, err
	}
	return a, nil
}

// UpdateAction records the next attempt/results for one action. The write
// applies only when the stored attempt count still equals expected, so
// concurrent writers cannot exceed the finite attempt budget or reorder
// results. Revoked actions refuse updates.
func (s *Store) UpdateAction(ctx context.Context, next factory.LifecycleAction, expectedAttempts int) (factory.LifecycleAction, error) {
	if err := next.Validate(); err != nil {
		return factory.LifecycleAction{}, err
	}
	if next.Attempts != expectedAttempts && next.Attempts != expectedAttempts+1 {
		return factory.LifecycleAction{}, ErrLifecycleActionStale
	}
	data, err := json.Marshal(next)
	if err != nil {
		return factory.LifecycleAction{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE factory_lifecycle_actions SET attempts=$2,revoked=$3,data=$4 WHERE id=$1 AND attempts=$5 AND revoked=FALSE`,
		next.ID, next.Attempts, next.Revoked, string(data), expectedAttempts)
	if err != nil {
		return factory.LifecycleAction{}, fmt.Errorf("lifecycle action update failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.LifecycleAction{}, err
	}
	if n == 0 {
		return factory.LifecycleAction{}, ErrLifecycleActionStale
	}
	return next, nil
}

// RevokeAction latches one action revoked so no further effect attempts.
// Revocation is idempotent and never un-records prior attempts.
func (s *Store) RevokeAction(ctx context.Context, id string) (factory.LifecycleAction, error) {
	current, err := s.LifecycleAction(ctx, id)
	if err != nil {
		return factory.LifecycleAction{}, err
	}
	current.Revoked = true
	data, err := json.Marshal(current)
	if err != nil {
		return factory.LifecycleAction{}, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE factory_lifecycle_actions SET revoked=TRUE,data=$2 WHERE id=$1`, id, string(data)); err != nil {
		return factory.LifecycleAction{}, fmt.Errorf("lifecycle action revoke failed: %w", err)
	}
	return current, nil
}

// ListActions returns up to limit recorded actions for one
// repository/issue/event, newest identity first. Limit is clamped to
// 1..64; empty event matches all three events.
func (s *Store) ListActions(ctx context.Context, repository, issue int64, event string, limit int) ([]factory.LifecycleAction, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 64 {
		limit = 64
	}
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM factory_lifecycle_actions WHERE repository=$1 AND issue=$2 AND ($3='' OR event=$3) ORDER BY id DESC LIMIT $4`, repository, issue, event, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []factory.LifecycleAction
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var a factory.LifecycleAction
		if err := json.Unmarshal([]byte(raw), &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// RegisterCapability stores one event's native capability proof marker
// bound to its installation and native generation. Each event registers
// independently; replaying the identical marker is a no-op returning
// replaced=false, while a verified changed marker supersedes the stored
// one and returns replaced=true. Only the capability registrar calls
// this, never tests-as-seed and never owner configuration.
func (s *Store) RegisterCapability(ctx context.Context, event, revision, installation string) (bool, error) {
	if !factory.ValidLifecycleEvent(event) || revision == "" || installation == "" {
		return false, factory.ErrLifecycleAction
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var storedRevision, storedInstallation string
	err = tx.QueryRowContext(ctx, `SELECT revision,installation FROM factory_lifecycle_capabilities WHERE event=$1 FOR UPDATE`, event).Scan(&storedRevision, &storedInstallation)
	switch {
	case err == nil:
		if storedRevision == revision && storedInstallation == installation {
			if err := tx.Commit(); err != nil {
				return false, err
			}
			return false, nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE factory_lifecycle_capabilities SET revision=$2,installation=$3 WHERE event=$1`, event, revision, installation); err != nil {
			return false, fmt.Errorf("lifecycle capability replace failed: %w", err)
		}
	case errors.Is(err, ErrNotFound):
		if _, err := tx.ExecContext(ctx, `INSERT INTO factory_lifecycle_capabilities(event,revision,installation) VALUES($1,$2,$3)`, event, revision, installation); err != nil {
			return false, fmt.Errorf("lifecycle capability record failed: %w", err)
		}
	default:
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteCapability removes one event's marker, returning to unavailable.
// Capability loss and unsupported events invalidate this way; in-flight
// actions keep their own revalidation and fail safely natively.
func (s *Store) DeleteCapability(ctx context.Context, event string) error {
	if !factory.ValidLifecycleEvent(event) {
		return factory.ErrLifecycleAction
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM factory_lifecycle_capabilities WHERE event=$1`, event); err != nil {
		return fmt.Errorf("lifecycle capability delete failed: %w", err)
	}
	return nil
}

// DeleteForeignCapabilities removes every marker bound to another
// installation. A changed installation never retains availability.
func (s *Store) DeleteForeignCapabilities(ctx context.Context, installation string) error {
	if installation == "" {
		return factory.ErrLifecycleAction
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM factory_lifecycle_capabilities WHERE installation<>$1`, installation); err != nil {
		return fmt.Errorf("lifecycle capability invalidate failed: %w", err)
	}
	return nil
}

// CapabilityProved reports whether event has a recorded capability proof.
func (s *Store) CapabilityProved(ctx context.Context, event string) (bool, error) {
	var revision string
	if err := s.db.QueryRowContext(ctx, `SELECT revision FROM factory_lifecycle_capabilities WHERE event=$1`, event).Scan(&revision); err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SaveConfig stores one event's configuration under CAS revision control.
// Enabling requires that event's own recorded capability proof;
// disabling is always allowed. Revision must advance by exactly one.
func (s *Store) SaveConfig(ctx context.Context, c factory.LifecycleConfig) (factory.LifecycleConfig, error) {
	if err := c.Validate(); err != nil {
		return factory.LifecycleConfig{}, err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return factory.LifecycleConfig{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.LifecycleConfig{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var storedRevision int64
	var storedEnabled bool
	err = tx.QueryRowContext(ctx, `SELECT revision,enabled FROM factory_lifecycle_configs WHERE event=$1`, c.Event).Scan(&storedRevision, &storedEnabled)
	switch {
	case err == nil:
		if c.Revision != storedRevision+1 {
			return factory.LifecycleConfig{}, ErrLifecycleActionStale
		}
	case errors.Is(err, ErrNotFound):
		if c.Revision != 0 {
			return factory.LifecycleConfig{}, ErrLifecycleActionStale
		}
	default:
		return factory.LifecycleConfig{}, err
	}
	if c.Enabled {
		var revision string
		if err := tx.QueryRowContext(ctx, `SELECT revision FROM factory_lifecycle_capabilities WHERE event=$1`, c.Event).Scan(&revision); err != nil {
			if errors.Is(err, ErrNotFound) {
				return factory.LifecycleConfig{}, ErrCapabilityUnproved
			}
			return factory.LifecycleConfig{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO factory_lifecycle_configs(event,revision,enabled,data) VALUES($1,$2,$3,$4) ON CONFLICT(event) DO UPDATE SET revision=$2,enabled=$3,data=$4`,
		c.Event, c.Revision, c.Enabled, string(data)); err != nil {
		return factory.LifecycleConfig{}, fmt.Errorf("lifecycle config save failed: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return factory.LifecycleConfig{}, err
	}
	return c, nil
}

// LifecycleConfig returns one event's configuration.
func (s *Store) LifecycleConfig(ctx context.Context, event string) (factory.LifecycleConfig, error) {
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT data FROM factory_lifecycle_configs WHERE event=$1`, event).Scan(&raw); err != nil {
		return factory.LifecycleConfig{}, err
	}
	var c factory.LifecycleConfig
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return factory.LifecycleConfig{}, err
	}
	return c, nil
}

func lifecycleActionEqual(a, b factory.LifecycleAction) bool {
	aj, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bj, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aj) == string(bj)
}
