package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func verificationAllowanceIntentTx(ctx context.Context, tx *sql.Tx, intent string, owner int64, exclusive bool) (factory.IntentRevision, error) {
	var revision factory.IntentRevision
	if !factory.ValidVerificationAllowanceIntent(intent) || owner <= 0 {
		return revision, ErrNotFound
	}
	lock := " FOR SHARE OF h"
	if exclusive {
		lock = " FOR UPDATE OF h"
	}
	var raw []byte
	var actualOwner int64
	err := tx.QueryRowContext(ctx, `SELECT h.owner_id,r.data FROM factory_intent_heads h JOIN factory_intent_revisions r ON r.id=h.revision WHERE h.id=$1`+lock, intent).Scan(&actualOwner, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return revision, ErrNotFound
	}
	if err != nil {
		return revision, err
	}
	if actualOwner != owner {
		return revision, factory.ErrVerificationAllowanceOwner
	}
	if err = json.Unmarshal(raw, &revision); err != nil {
		return revision, err
	}
	if err = revision.Validate(); err != nil {
		return revision, err
	}
	if revision.Intent != intent || revision.Owner != owner {
		return revision, factory.ErrVerificationAllowanceOwner
	}
	return revision, nil
}

// The unique predecessor-chain tip names the allowance revision. Wall-clock
// command timestamps are deliberately irrelevant. The stable intent head lock
// serializes both this chain's append and every delivery packet admission.
func verificationAllowanceLimits(ctx context.Context, q grantCommandSQL, intent string) (string, factory.AppVerificationLimits, error) {
	defaults := factory.AppVerificationLimits{Runs: factory.DefaultAppVerificationRuns, ActiveMinutes: factory.DefaultAppVerificationActiveMinutes}
	commands, err := finishedCommands(ctx, q, factory.CommandVerificationAllowance, factory.AppVerificationAllowanceTarget(intent))
	if err != nil {
		return "", defaults, err
	}
	if len(commands) == 0 {
		return "", defaults, nil
	}
	changes, referenced := map[string]factory.AppVerificationAllowanceChange{}, map[string]bool{}
	for _, cmd := range commands {
		var change factory.AppVerificationAllowanceChange
		if cmd.Validate() != nil || json.Unmarshal([]byte(cmd.Payload), &change) != nil || change.Validate() != nil || change.CommandID != cmd.ID {
			return "", defaults, errors.New("invalid retained verification allowance")
		}
		changes[cmd.ID] = change
		if change.ExpectedRevision != "" {
			referenced[change.ExpectedRevision] = true
		}
	}
	head := ""
	for id := range changes {
		if !referenced[id] {
			if head != "" {
				return "", defaults, errors.New("forked verification allowance")
			}
			head = id
		}
	}
	seen := map[string]bool{}
	for cursor := head; cursor != ""; {
		change, ok := changes[cursor]
		if !ok || seen[cursor] {
			return "", defaults, errors.New("incomplete verification allowance chain")
		}
		seen[cursor] = true
		prior := defaults
		if change.ExpectedRevision != "" {
			old, ok := changes[change.ExpectedRevision]
			if !ok {
				return "", defaults, errors.New("missing verification allowance predecessor")
			}
			prior = old.Limits
		}
		if !verificationLimitsIncrease(prior, change.Limits) {
			return "", defaults, factory.ErrVerificationAllowanceIncrease
		}
		cursor = change.ExpectedRevision
	}
	if head == "" || len(seen) != len(changes) {
		return "", defaults, errors.New("invalid verification allowance lineage")
	}
	return head, changes[head].Limits, nil
}

func verificationLimitsIncrease(old, next factory.AppVerificationLimits) bool {
	return next.Runs >= old.Runs && next.ActiveMinutes >= old.ActiveMinutes && next != old
}

func verificationConsumptionTx(ctx context.Context, tx *sql.Tx, intent string, now time.Time) (factory.AppVerificationUsage, error) {
	var usage factory.AppVerificationUsage
	rows, err := tx.QueryContext(ctx, `SELECT data FROM factory_attempt_allowances WHERE issue=0 AND data->'delivery'->>'intent_id'=$1 ORDER BY root_assignment FOR SHARE`, intent)
	if err != nil {
		return usage, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		var allowance factory.AttemptAllowance
		if err = rows.Scan(&raw); err != nil {
			return usage, err
		}
		if err = json.Unmarshal(raw, &allowance); err != nil {
			return usage, err
		}
		if allowance.Validate() != nil || allowance.Delivery == nil || allowance.Delivery.IntentID != intent {
			return usage, errors.New("invalid retained application verification allowance")
		}
		if now.Unix() < allowance.CheckpointUnix {
			return usage, factory.ErrAllowanceClock
		}
		spent := allowance.ActiveSeconds
		if allowance.Active {
			if now.Unix()-allowance.CheckpointUnix > math.MaxInt64-spent {
				return usage, factory.ErrAllowanceClock
			}
			spent += now.Unix() - allowance.CheckpointUnix
		}
		if spent > math.MaxInt64-usage.ActiveSeconds {
			return usage, errors.New("verification consumption overflow")
		}
		usage.Runs++
		usage.ActiveSeconds += spent
	}
	return usage, rows.Err()
}

func verificationAllowanceStatusTx(ctx context.Context, tx *sql.Tx, intent string, now time.Time) (factory.AppVerificationAllowance, error) {
	view := factory.AppVerificationAllowance{Intent: intent}
	var err error
	if view.Revision, view.Limits, err = verificationAllowanceLimits(ctx, tx, intent); err != nil {
		return view, err
	}
	if view.Consumption, err = verificationConsumptionTx(ctx, tx, intent, now); err != nil {
		return view, err
	}
	view.UpdateRemaining()
	return view, nil
}

func (s *Store) AppVerificationAllowance(ctx context.Context, intent string, owner int64, now time.Time) (factory.AppVerificationAllowance, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.AppVerificationAllowance{}, err
	}
	defer tx.Rollback()
	head, err := verificationAllowanceIntentTx(ctx, tx, intent, owner, false)
	if err != nil {
		return factory.AppVerificationAllowance{}, err
	}
	view, err := verificationAllowanceStatusTx(ctx, tx, intent, now)
	if err != nil {
		return view, err
	}
	view.Repository = head.Repository
	view.CanIncrease = head.State == "accepted" && head.Repository > 0
	return view, tx.Commit()
}

// Mutation authorization needs the current owner/repository, not a fresh usage
// calculation that could precede an already-finished command's exact replay.
func (s *Store) AppVerificationAllowanceScope(ctx context.Context, intent string, owner int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	head, err := verificationAllowanceIntentTx(ctx, tx, intent, owner, false)
	if err != nil {
		return 0, err
	}
	return head.Repository, tx.Commit()
}

func (s *Store) IncreaseAppVerificationAllowance(ctx context.Context, intent string, owner int64, cmd factory.Command, change factory.AppVerificationAllowanceChange, now time.Time) (factory.AppVerificationAllowance, error) {
	var none factory.AppVerificationAllowance
	raw, err := json.Marshal(change)
	if err != nil {
		return none, err
	}
	if change.Validate() != nil || cmd.Validate() != nil || cmd.Type != factory.CommandVerificationAllowance || cmd.ID != change.CommandID || cmd.Target != factory.AppVerificationAllowanceTarget(intent) || cmd.Principal != "native:"+strconv.FormatInt(owner, 10) || cmd.Payload != string(raw) {
		return none, ErrCommandConflict
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return none, err
	}
	defer tx.Rollback()
	head, err := verificationAllowanceIntentTx(ctx, tx, intent, owner, true)
	if err != nil {
		return none, err
	}
	stored, created, err := recordFactoryCommandTx(ctx, tx, cmd, now)
	if err != nil {
		return none, err
	}
	if !created {
		if stored.Type != cmd.Type || stored.Target != cmd.Target || stored.Principal != cmd.Principal || stored.Payload != cmd.Payload || stored.Finished == "" || json.Unmarshal([]byte(stored.Outcome), &none) != nil || none.Intent != intent || none.Revision != cmd.ID || none.CommandID != cmd.ID {
			return none, ErrCommandConflict
		}
		return none, tx.Commit()
	}
	view, err := verificationAllowanceStatusTx(ctx, tx, intent, now)
	if err != nil {
		return none, err
	}
	if change.ExpectedRevision != view.Revision {
		return none, ErrStaleRevision
	}
	if head.State != "accepted" || head.Repository <= 0 {
		return none, factory.ErrVerificationAllowanceState
	}
	if !verificationLimitsIncrease(view.Limits, change.Limits) {
		return none, factory.ErrVerificationAllowanceIncrease
	}
	view.Revision, view.CommandID, view.Limits, view.CanIncrease = cmd.ID, cmd.ID, change.Limits, true
	view.UpdateRemaining()
	outcome, err := json.Marshal(view)
	if err != nil {
		return none, err
	}
	if err = finishFactoryCommandTx(ctx, tx, cmd.ID, string(outcome), now); err != nil {
		return none, err
	}
	return view, tx.Commit()
}
