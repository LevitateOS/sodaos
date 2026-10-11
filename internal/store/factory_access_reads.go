package store

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/levitateos/sodaos/internal/factory"
)

// MaxRepositoryAllowances bounds one repository planning-allowance
// listing. Settings show the viewer's own request allowances; the
// per-intent allowance route owns single-record reads and writes.
const MaxRepositoryAllowances = 20

// RepositoryIntentAllowances returns current planning-allowance heads
// for the viewer's intents bound to one repository, in stable intent
// order, with a truncation flag. Owner-private: only the viewer's own
// allowances ever match.
func (s *Store) RepositoryIntentAllowances(ctx context.Context, repository, owner int64, limit int) ([]factory.PlanningAllowance, bool, error) {
	if repository <= 0 || owner <= 0 {
		return nil, false, errors.New("invalid allowance listing scope")
	}
	if limit < 1 || limit > MaxRepositoryAllowances {
		return nil, false, errors.New("invalid allowance listing bound")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.data FROM factory_intent_allowance_heads h
		JOIN factory_intent_allowances a ON a.id=h.revision
		JOIN factory_intent_heads ih ON ih.id=h.intent AND ih.owner_id=$2
		JOIN factory_intent_revisions r ON r.id=ih.revision
		WHERE r.data->>'repository'=$1 ORDER BY h.intent LIMIT $3`,
		strconv.FormatInt(repository, 10), owner, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = rows.Close() }()
	allowances := []factory.PlanningAllowance{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, false, err
		}
		var allowance factory.PlanningAllowance
		if err := json.Unmarshal(raw, &allowance); err != nil {
			return nil, false, err
		}
		allowances = append(allowances, allowance)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(allowances) > limit {
		return allowances[:limit], true, nil
	}
	return allowances, false, nil
}
