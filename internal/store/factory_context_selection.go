package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// ReserveContextBundleSelection claims the capability's single preparation-time
// consultation and its existing priority receipt in one transaction. A changed
// fingerprint, failed request or restart cannot buy another call for this run.
func (s *Store) ReserveContextBundleSelection(ctx context.Context, capability factory.MCPCapability, fingerprint string) (bool, error) {
	if capability.Validate() != nil || !factory.ValidDigest(fingerprint) {
		return false, errors.New("invalid context selection binding")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var spent int
	if err = tx.QueryRowContext(ctx, `SELECT spent FROM factory_judgment_budget WHERE id=1 FOR UPDATE`).Scan(&spent); err != nil {
		return false, err
	}
	var raw []byte
	var state string
	var prior string
	if err = tx.QueryRowContext(ctx, `SELECT state,data,COALESCE(data->>'context_selection_snapshot','') FROM factory_mcp_capabilities WHERE id=$1 FOR UPDATE`, capability.ID).Scan(&state, &raw, &prior); err != nil {
		return false, err
	}
	var recorded factory.MCPCapability
	if err = json.Unmarshal(raw, &recorded); err != nil {
		return false, err
	}
	if recorded.ScopeDigest() != capability.ScopeDigest() {
		return false, ErrMCPCapabilityConflict
	}
	if state != factory.MCPCapabilityStaged || recorded.Expired(time.Now()) {
		return false, ErrMCPCapabilityState
	}
	if prior != "" {
		return false, nil
	}
	if err = checkMCPWorkActiveTx(ctx, tx, capability); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_priority_judgments(snapshot,max_calls) VALUES($1,1)`, fingerprint); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_mcp_capabilities SET data=jsonb_set(data,'{context_selection_snapshot}',to_jsonb($2::text)) WHERE id=$1`, capability.ID, fingerprint); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_judgment_budget SET spent=spent+1 WHERE id=1`); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
