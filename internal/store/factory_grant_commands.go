package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

type grantCommandSQL interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// FactoryAuthorityInput returns the current grant and preparation inputs used
// by both visible authority reads and atomic grant-command receipts.
func (s *Store) FactoryAuthorityInput(ctx context.Context, repository int64) (factory.AuthorityInput, error) {
	return factoryAuthorityInput(ctx, s.db, repository)
}

func factoryAuthorityInput(ctx context.Context, q grantCommandSQL, repository int64) (factory.AuthorityInput, error) {
	var in factory.AuthorityInput
	var policy factory.RepositoryPolicy
	if err := loadGrantQuery(ctx, q, "factory_policies", "repository", repository, &policy); err == nil {
		in.Policy = &policy
	} else if !errors.Is(err, ErrNotFound) {
		return in, err
	}
	var operator factory.OperatorGrant
	if err := loadGrantQuery(ctx, q, "factory_operator_grants", "repository", repository, &operator); err == nil {
		in.Operator = &operator
	} else if !errors.Is(err, ErrNotFound) {
		return in, err
	}
	var capacity factory.Capacity
	if err := loadGrantQuery(ctx, q, "factory_capacity", "id", 1, &capacity); err == nil {
		in.Appliance = &capacity
	} else if !errors.Is(err, ErrNotFound) {
		return in, err
	}
	sponsorships, err := sponsorshipsQuery(ctx, q, repository)
	if err != nil {
		return in, err
	}
	for i := range sponsorships {
		if sponsorships[i].Active {
			in.Sponsorship = &sponsorships[i]
			break
		}
	}
	if in.Sponsorship == nil && len(sponsorships) > 0 {
		in.Sponsorship = &sponsorships[0]
	}
	var environment project.EnvironmentGrant
	if err = loadGrantQuery(ctx, q, "project_environment_grants", "repository", repository, &environment); err == nil {
		in.Environment = &environment
	} else if !errors.Is(err, ErrNotFound) {
		return in, err
	}
	projectRecord, err := scanProject(q.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE repository_id=$1`, repository))
	if err != nil && !errors.Is(err, ErrNotFound) {
		return in, err
	}
	if err == nil {
		in.ProjectExists = true
		maintenanceHeld := false
		var holdData []byte
		err = q.QueryRowContext(ctx, `SELECT data FROM project_maintenance WHERE project_id=$1`, projectRecord.ID).Scan(&holdData)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return in, err
		}
		if err == nil {
			var hold project.MaintenanceHold
			if err = json.Unmarshal(holdData, &hold); err != nil {
				return in, err
			}
			if hold.Hold {
				maintenanceHeld = true
			}
		}
		if !maintenanceHeld {
			rows, queryErr := q.QueryContext(ctx, `SELECT data FROM project_preparations WHERE project_id=$1 ORDER BY id LIMIT 129`, projectRecord.ID)
			if queryErr != nil {
				return in, queryErr
			}
			ready := map[string]bool{}
			for rows.Next() {
				var data []byte
				var preparation project.StoredPreparation
				if err = rows.Scan(&data); err != nil {
					_ = rows.Close()
					return in, err
				}
				if err = json.Unmarshal(data, &preparation); err != nil {
					_ = rows.Close()
					return in, err
				}
				if preparation.State.Ready {
					ready[preparation.Preparation.Role] = true
				}
			}
			if err = rows.Err(); err != nil {
				_ = rows.Close()
				return in, err
			}
			if err = rows.Close(); err != nil {
				return in, err
			}
			in.PreparationReady = ready[project.RoleCoder] && ready[project.RoleReviewer]
		}
	}

	open, _, _, err := dispatchStateQuery(ctx, q, repository)
	if err != nil {
		return in, err
	}
	in.DispatchOpen = open
	return in, nil
}

func (s *Store) ApplyRepositoryPolicyCommand(ctx context.Context, cmd factory.Command, change factory.RepositoryPolicy, now time.Time) (factory.Command, bool, error) {
	return s.applyFactoryGrantCommand(ctx, cmd, change, now)
}

func (s *Store) ApplyCapacityCommand(ctx context.Context, cmd factory.Command, change factory.Capacity, now time.Time) (factory.Command, bool, error) {
	return s.applyFactoryGrantCommand(ctx, cmd, change, now)
}

func (s *Store) ApplyOperatorGrantCommand(ctx context.Context, cmd factory.Command, change factory.OperatorGrant, now time.Time) (factory.Command, bool, error) {
	return s.applyFactoryGrantCommand(ctx, cmd, change, now)
}

func (s *Store) ApplySponsorshipCommand(ctx context.Context, cmd factory.Command, change factory.Sponsorship, now time.Time) (factory.Command, bool, error) {
	return s.applyFactoryGrantCommand(ctx, cmd, change, now)
}

func (s *Store) ApplyEnvironmentGrantCommand(ctx context.Context, cmd factory.Command, change project.EnvironmentGrant, now time.Time) (factory.Command, bool, error) {
	return s.applyFactoryGrantCommand(ctx, cmd, change, now)
}

func (s *Store) applyFactoryGrantCommand(ctx context.Context, cmd factory.Command, change any, now time.Time) (factory.Command, bool, error) {
	if err := cmd.Validate(); err != nil {
		return factory.Command{}, false, err
	}
	payload, err := json.Marshal(change)
	if err != nil {
		return factory.Command{}, false, err
	}
	if string(payload) != cmd.Payload || cmd.Digest != factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload) {
		return factory.Command{}, false, ErrCommandConflict
	}
	target := ""
	repository := int64(0)
	revision := int64(0)
	withdraw := false
	cause := ""
	var validate func() error
	switch g := change.(type) {
	case factory.RepositoryPolicy:
		target, repository, revision = grantCommandTarget("policy", g.Repository), g.Repository, g.Revision
		withdraw, cause = !g.Enabled || g.Paused, "policy_disabled"
		if g.Paused {
			cause = "policy_paused"
		}
		validate = g.Validate
	case factory.Capacity:
		target, revision, validate = "capacity", g.Revision, g.Validate
	case factory.OperatorGrant:
		target, repository, revision = grantCommandTarget("operator-grant", g.Repository), g.Repository, g.Revision
		withdraw, cause, validate = !g.Active, "operator_grant_withdrawn", g.Validate
	case factory.Sponsorship:
		target, repository, revision = grantCommandTarget("sponsorship/"+g.Connection, g.Repository), g.Repository, g.Revision
		cause, validate = "sponsorship_withdrawn", g.Validate
	case project.EnvironmentGrant:
		target, repository, revision = grantCommandTarget("environment-grant", g.Repository), g.Repository, g.Revision
		withdraw, cause, validate = !g.Active, "environment_grant_withdrawn", g.Validate
	default:
		return factory.Command{}, false, errors.New("unsupported factory grant command")
	}
	if err = validate(); err != nil {
		return factory.Command{}, false, err
	}
	expectedType := ""
	switch change.(type) {
	case factory.RepositoryPolicy:
		expectedType = factory.CommandPolicy
	case factory.Capacity:
		expectedType = factory.CommandCapacity
	case factory.OperatorGrant:
		expectedType = factory.CommandOperatorGrant
	case factory.Sponsorship:
		expectedType = factory.CommandSponsorship
	case project.EnvironmentGrant:
		expectedType = factory.CommandEnvironmentGrant
	}
	if cmd.Type != expectedType || cmd.Target != target || revision < 0 {
		return factory.Command{}, false, errors.New("grant command does not match its target")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.Command{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	if repository > 0 {
		if _, _, _, err = ensureDispatchGateTx(ctx, tx, repository); err != nil {
			return factory.Command{}, false, err
		}
	} else {
		if _, err = tx.ExecContext(ctx, `INSERT INTO factory_capacity(id,revision,data) VALUES(1,0,'{}'::jsonb) ON CONFLICT(id) DO NOTHING`); err != nil {
			return factory.Command{}, false, err
		}
		var locked int64
		if err = tx.QueryRowContext(ctx, `SELECT revision FROM factory_capacity WHERE id=1 FOR UPDATE`).Scan(&locked); err != nil {
			return factory.Command{}, false, err
		}
	}
	stored, created, err := recordFactoryCommandTx(ctx, tx, cmd, now)
	if err != nil || !created {
		return stored, false, err
	}
	if err = saveFactoryGrantChangeTx(ctx, tx, change); err != nil {
		return factory.Command{}, false, err
	}
	if sp, ok := change.(factory.Sponsorship); ok && !sp.Active {
		var active bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_sponsorships
			WHERE repository=$1 AND connection<>$2 AND COALESCE((data->>'active')::boolean,FALSE))`, sp.Repository, sp.Connection).Scan(&active); err != nil {
			return factory.Command{}, false, err
		}
		withdraw = !active
	}
	receipt := factory.GrantReceipt{CommandID: cmd.ID, Revision: revision + 1, Withdrawn: withdraw}
	if repository > 0 {
		if withdraw {
			withdrawal, withdrawErr := withdrawDispatchTx(ctx, tx, repository, cause, cmd.Principal)
			if withdrawErr != nil {
				return factory.Command{}, false, withdrawErr
			}
			receipt.Captured = withdrawal.Captured
			if receipt.Captured == nil {
				receipt.Captured = []string{}
			}
		}
		if receipt.PublicationsPending, err = requestPublicationWithdrawalsTx(ctx, tx, repository); err != nil {
			return factory.Command{}, false, err
		}
		if receipt.MergesPending, err = requestMergeWithdrawalsTx(ctx, tx, repository); err != nil {
			return factory.Command{}, false, err
		}
	}
	input, err := factoryAuthorityInput(ctx, tx, repository)
	if err != nil {
		return factory.Command{}, false, err
	}
	receipt.Effective = factory.EvaluateAuthority(input)
	outcome, err := json.Marshal(receipt)
	if err != nil {
		return factory.Command{}, false, err
	}
	if err = finishFactoryCommandTx(ctx, tx, cmd.ID, string(outcome), now); err != nil {
		return factory.Command{}, false, err
	}
	stored.Outcome = string(outcome)
	stored.Finished = now.UTC().Format(time.RFC3339Nano)
	if err = tx.Commit(); err != nil {
		return factory.Command{}, false, err
	}
	return stored, true, nil
}

func grantCommandTarget(record string, repository int64) string {
	return "repository/" + strconv.FormatInt(repository, 10) + "/" + record
}

func saveFactoryGrantChangeTx(ctx context.Context, tx *sql.Tx, change any) error {
	switch g := change.(type) {
	case factory.RepositoryPolicy:
		if err := g.Validate(); err != nil {
			return err
		}
		next := g
		next.Revision++
		return saveRevisionedGrantQuery(ctx, tx, "factory_policies", "repository", g.Repository, g.Revision, next, "repository policy")
	case factory.Capacity:
		if err := g.Validate(); err != nil {
			return err
		}
		next := g
		next.Revision++
		return saveRevisionedGrantQuery(ctx, tx, "factory_capacity", "id", 1, g.Revision, next, "appliance capacity")
	case factory.OperatorGrant:
		if err := g.Validate(); err != nil {
			return err
		}
		next := g
		next.Revision++
		return saveRevisionedGrantQuery(ctx, tx, "factory_operator_grants", "repository", g.Repository, g.Revision, next, "operator grant")
	case factory.Sponsorship:
		if err := g.Validate(); err != nil {
			return err
		}
		next := g
		next.Revision++
		return saveSponsorshipQuery(ctx, tx, g, next)
	case project.EnvironmentGrant:
		if err := g.Validate(); err != nil {
			return err
		}
		next := g
		next.Revision++
		return saveRevisionedGrantQuery(ctx, tx, "project_environment_grants", "repository", g.Repository, g.Revision, next, "environment grant")
	default:
		return errors.New("unsupported factory grant command")
	}
}

func saveRevisionedGrantQuery(ctx context.Context, q grantCommandSQL, table, key string, id any, revision int64, next any, what string) error {
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	result, err := q.ExecContext(ctx, `INSERT INTO `+table+`(`+key+`,revision,data) VALUES($1,$2,$3)
		ON CONFLICT(`+key+`) DO UPDATE SET revision=$4,data=$5 WHERE `+table+`.revision=$6`,
		id, revision+1, string(data), revision+1, string(data), revision)
	return checkGrantRowsAffected(result, err, what+" save failed")
}

func saveSponsorshipQuery(ctx context.Context, q grantCommandSQL, current, next factory.Sponsorship) error {
	data, err := json.Marshal(next)
	if err != nil {
		return err
	}
	result, err := q.ExecContext(ctx, `INSERT INTO factory_sponsorships(repository,connection,revision,data) VALUES($1,$2,$3,$4)
		ON CONFLICT(repository,connection) DO UPDATE SET revision=$5,data=$6 WHERE factory_sponsorships.revision=$7`,
		current.Repository, current.Connection, next.Revision, string(data), next.Revision, string(data), current.Revision)
	return checkGrantRowsAffected(result, err, "sponsorship save failed")
}

func checkGrantRowsAffected(result sql.Result, err error, what string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrStaleRevision
	}
	return nil
}

func recordFactoryCommandTx(ctx context.Context, tx *sql.Tx, cmd factory.Command, now time.Time) (factory.Command, bool, error) {
	if err := cmd.Validate(); err != nil {
		return factory.Command{}, false, err
	}
	if cmd.Outcome != "" {
		return factory.Command{}, false, errors.New("new command must not carry an outcome")
	}
	created := now.UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `INSERT INTO factory_commands(id,type,target,principal,digest,payload,created) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO NOTHING`, cmd.ID, cmd.Type, cmd.Target, cmd.Principal, cmd.Digest, cmd.Payload, created)
	if err != nil {
		return factory.Command{}, false, fmt.Errorf("factory command record failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return factory.Command{}, false, err
	}
	var stored factory.Command
	var storedCreated, finished string
	err = tx.QueryRowContext(ctx, `SELECT id,type,target,principal,digest,payload,outcome,created,finished FROM factory_commands WHERE id=$1 FOR UPDATE`, cmd.ID).Scan(&stored.ID, &stored.Type, &stored.Target, &stored.Principal, &stored.Digest, &stored.Payload, &stored.Outcome, &storedCreated, &finished)
	if err != nil {
		return factory.Command{}, false, err
	}
	stored.Created, stored.Finished = storedCreated, finished
	if stored.Digest != cmd.Digest {
		return factory.Command{}, false, ErrCommandConflict
	}
	return stored, n == 1, nil
}

func finishFactoryCommandTx(ctx context.Context, tx *sql.Tx, id, outcome string, now time.Time) error {
	if len(outcome) > 64<<10 {
		return errors.New("command outcome exceeds retained output limit")
	}
	finished := now.UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE factory_commands SET outcome=$1,finished=$2 WHERE id=$3 AND finished=''`, outcome, finished, id)
	return checkGrantRowsAffected(result, err, "factory command finish failed")
}
