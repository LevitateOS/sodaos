package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// ReserveContextBundleSelection binds the capability's preparation-time
// consultation to fingerprint in one transaction. A recorded selection stays
// frozen for the run (false); an earlier call that left no receipt, including
// one for an older fingerprint, may retry (true).
func (s *Store) ReserveContextBundleSelection(ctx context.Context, capability factory.MCPCapability, fingerprint string) (bool, error) {
	if capability.Validate() != nil || !factory.ValidDigest(fingerprint) {
		return false, errors.New("invalid context selection binding")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
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
	if prior != "" && prior != fingerprint {
		var decided bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_priority_judgments WHERE snapshot=$1 AND data IS NOT NULL)`, prior).Scan(&decided); err != nil || decided {
			return false, err
		}
	}
	if err = checkMCPWorkActiveTx(ctx, tx, capability); err != nil {
		return false, err
	}
	reserved, err := reserveJudgmentTx(ctx, tx, fingerprint, 1)
	if err != nil || !reserved {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_mcp_capabilities SET data=jsonb_set(data,'{context_selection_snapshot}',to_jsonb($2::text)) WHERE id=$1`, capability.ID, fingerprint); err != nil {
		return false, err
	}
	return true, tx.Commit()
}
