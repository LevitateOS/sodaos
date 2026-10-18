package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

// ErrPlanningAllowanceExhausted reports a reservation refusal for missing or
// fully consumed intent allowance funding. Duplicate snapshots and the
// operator global ceiling still refuse as (false, nil); stale heads and
// storage faults return their own errors.
var ErrPlanningAllowanceExhausted = errors.New("planning allowance exhausted")

// SavePlanningAllowance serializes independent allowance revisions with
// current-head CAS. Intent, owner and unit are immutable; consumption is
// cumulative and never decreases. Raising Amount alone preserves effective
// scope, native mappings and prior consumption by leaving intent and
// generation tables untouched. An actual Amount increase must state its
// effect; unchanged limits reuse existing authorization.
func (s *Store) SavePlanningAllowance(ctx context.Context, a factory.PlanningAllowance) error {
	if err := a.Validate(); err != nil {
		return err
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	head, owner, err := lockPlanningAllowanceHeadTx(ctx, tx, a)
	if err != nil {
		return err
	}
	if owner != a.Owner {
		return ErrCommandConflict
	}
	if head == a.ID {
		duplicate, err := confirmPlanningAllowanceDuplicateTx(ctx, tx, a, raw)
		if err != nil {
			return err
		}
		if !duplicate {
			return ErrCommandConflict
		}
		return tx.Commit()
	}
	if head != a.Predecessor {
		return ErrStaleRevision
	}
	if err := checkPlanningAllowanceTransitionTx(ctx, tx, a, head); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO factory_intent_allowances(id,intent,data) VALUES($1,$2,$3)`, a.ID, a.Intent, string(raw)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE factory_intent_allowance_heads SET revision=$2 WHERE intent=$1`, a.Intent, a.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func lockPlanningAllowanceHeadTx(ctx context.Context, tx *sql.Tx, a factory.PlanningAllowance) (string, int64, error) {
	_, err := tx.ExecContext(ctx, `INSERT INTO factory_intent_allowance_heads(intent,owner_id,revision) VALUES($1,$2,'') ON CONFLICT(intent) DO NOTHING`, a.Intent, a.Owner)
	if err != nil {
		return "", 0, err
	}
	var head string
	var owner int64
	if err = tx.QueryRowContext(ctx, `SELECT revision,owner_id FROM factory_intent_allowance_heads WHERE intent=$1 FOR UPDATE`, a.Intent).Scan(&head, &owner); err != nil {
		return "", 0, err
	}
	return head, owner, nil
}

func confirmPlanningAllowanceDuplicateTx(ctx context.Context, tx *sql.Tx, a factory.PlanningAllowance, raw []byte) (bool, error) {
	var old []byte
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_allowances WHERE id=$1`, a.ID).Scan(&old); err != nil {
		return false, err
	}
	return sameJSONDocument(ctx, tx, old, raw)
}

func checkPlanningAllowanceTransitionTx(ctx context.Context, tx *sql.Tx, a factory.PlanningAllowance, head string) error {
	if head == "" {
		return nil
	}
	var headRaw []byte
	var old factory.PlanningAllowance
	if err := tx.QueryRowContext(ctx, `SELECT data FROM factory_intent_allowances WHERE id=$1`, head).Scan(&headRaw); err != nil {
		return err
	}
	if err := json.Unmarshal(headRaw, &old); err != nil {
		return err
	}
	if old.Intent != a.Intent || old.Owner != a.Owner || old.Unit != a.Unit {
		return errors.New("allowance identity is immutable")
	}
	if a.Consumed < old.Consumed || a.Reserved+a.Consumed < old.Reserved+old.Consumed {
		return errors.New("allowance consumption cannot decrease")
	}
	if a.Amount > old.Amount && a.Effect == "" {
		return errors.New("allowance increase must state its effect")
	}
	return nil
}

func (s *Store) PlanningAllowance(ctx context.Context, intent string, owner int64) (factory.PlanningAllowance, error) {
	var a factory.PlanningAllowance
	var raw []byte
	err := s.db.QueryRowContext(ctx, `SELECT a.data FROM factory_intent_allowance_heads h JOIN factory_intent_allowances a ON a.id=h.revision WHERE h.intent=$1 AND h.owner_id=$2`, intent, owner).Scan(&raw)
	if err != nil {
		return a, err
	}
	if err = json.Unmarshal(raw, &a); err != nil {
		return a, err
	}
	reserved, consumed, err := s.planningUsage(ctx, intent)
	if err != nil {
		return factory.PlanningAllowance{}, err
	}
	a.Reserved, a.Consumed = reserved, consumed
	return a, nil
}

// planningUsage derives live counters from runtime records: a reserved call
// without a recorded receipt stays reserved (charged and unavailable,
// including interruptions and failures until recorded); a call with a
// recorded receipt, completed or failed, is consumed. Saved record counters
// are revision-time values only; admission and reads always derive live.
func (s *Store) planningUsage(ctx context.Context, intent string) (reserved, consumed int, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_intent_planning_calls p WHERE p.intent=$1 AND EXISTS (SELECT 1 FROM factory_priority_judgments j WHERE j.snapshot=p.snapshot AND j.data IS NOT NULL)`, intent).Scan(&consumed)
	if err != nil {
		return 0, 0, err
	}
	var total int
	if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_intent_planning_calls WHERE intent=$1`, intent).Scan(&total); err != nil {
		return 0, 0, err
	}
	return total - consumed, consumed, nil
}

// PlanningAllowanceAvailable reports whether the current independent
// allowance admits another consultation for this owner-private intent. It is
// an advisory gate; ReserveIntentJudgment remains authoritative.
func (s *Store) PlanningAllowanceAvailable(ctx context.Context, intent string, owner int64) (bool, error) {
	a, err := s.PlanningAllowance(ctx, intent, owner)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return a.Available() > 0, nil
}
