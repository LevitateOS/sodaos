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

// loadAuthorityGrantRows loads the policy, operator and appliance
// capacity grants behind one authority evaluation, tolerating absent
// rows.
func loadAuthorityGrantRows(ctx context.Context, q grantCommandSQL, repository int64, in *factory.AuthorityInput) error {
	var policy factory.RepositoryPolicy
	if err := loadGrantQuery(ctx, q, "factory_policies", "repository", repository, &policy); err == nil {
		in.Policy = &policy
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	var operator factory.OperatorGrant
	if err := loadGrantQuery(ctx, q, "factory_operator_grants", "repository", repository, &operator); err == nil {
		in.Operator = &operator
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	var capacity factory.Capacity
	if err := loadGrantQuery(ctx, q, "factory_capacity", "id", 1, &capacity); err == nil {
		in.Appliance = &capacity
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	return nil
}

// pickAuthoritySponsorship selects the active sponsorship behind one
// authority evaluation, falling back to the first recorded sponsorship.
func pickAuthoritySponsorship(ctx context.Context, q grantCommandSQL, repository int64, in *factory.AuthorityInput) error {
	sponsorships, err := sponsorshipsQuery(ctx, q, repository)
	if err != nil {
		return err
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
	return nil
}

// loadAuthorityProjectReadiness resolves project existence, the
// maintenance hold and coder/reviewer preparation readiness behind
// one authority evaluation.
func loadAuthorityProjectReadiness(ctx context.Context, q grantCommandSQL, repository int64, in *factory.AuthorityInput) error {
	var environment project.EnvironmentGrant
	if err := loadGrantQuery(ctx, q, "project_environment_grants", "repository", repository, &environment); err == nil {
		in.Environment = &environment
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	projectRecord, err := scanProject(q.QueryRowContext(ctx, `SELECT `+projectColumns+` FROM projects WHERE repository_id=$1`, repository))
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if err == nil {
		in.ProjectExists = true
		maintenanceHeld := false
		var holdData []byte
		err = q.QueryRowContext(ctx, `SELECT data FROM project_maintenance WHERE project_id=$1`, projectRecord.ID).Scan(&holdData)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err == nil {
			var hold project.MaintenanceHold
			if err = json.Unmarshal(holdData, &hold); err != nil {
				return err
			}
			if hold.Hold {
				maintenanceHeld = true
			}
		}
		if !maintenanceHeld {
			if err = scanReadyPreparations(ctx, q, projectRecord.ID, in); err != nil {
				return err
			}
		}
	}
	return nil
}

// scanReadyPreparations marks coder/reviewer preparation readiness
// from recorded preparations.
func scanReadyPreparations(ctx context.Context, q grantCommandSQL, projectID string, in *factory.AuthorityInput) error {
	rows, err := q.QueryContext(ctx, `SELECT data FROM project_preparations WHERE project_id=$1 ORDER BY id LIMIT 129`, projectID)
	if err != nil {
		return err
	}
	ready := map[string]bool{}
	for rows.Next() {
		var data []byte
		var preparation project.StoredPreparation
		if err = rows.Scan(&data); err != nil {
			_ = rows.Close()
			return err
		}
		if err = json.Unmarshal(data, &preparation); err != nil {
			_ = rows.Close()
			return err
		}
		if preparation.State.Ready {
			ready[preparation.Preparation.Role] = true
		}
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	in.PreparationReady = ready[project.RoleCoder] && ready[project.RoleReviewer]
	return nil
}

func factoryAuthorityInput(ctx context.Context, q grantCommandSQL, repository int64) (factory.AuthorityInput, error) {
	var in factory.AuthorityInput
	if err := loadAuthorityGrantRows(ctx, q, repository, &in); err != nil {
		return in, err
	}
	if err := pickAuthoritySponsorship(ctx, q, repository, &in); err != nil {
		return in, err
	}
	if err := loadAuthorityProjectReadiness(ctx, q, repository, &in); err != nil {
		return in, err
	}
	open, _, _, err := dispatchStateQuery(ctx, q, repository)
	if err != nil {
		return in, err
	}
	in.DispatchOpen = open
	return in, nil
}

// grantCommandResolution binds one grant change to its command
// target, scope, revision, withdrawal cause and validator.
type grantCommandResolution struct {
	target     string
	repository int64
	revision   int64
	withdraw   bool
	cause      string
	validate   func() error
}

// resolveGrantCommandTarget maps one grant change to its command
// target and withdrawal terms.
func resolveGrantCommandTarget(change any) (grantCommandResolution, error) {
	var resolved grantCommandResolution
	switch g := change.(type) {
	case factory.RepositoryPolicy:
		resolved.target, resolved.repository, resolved.revision = grantCommandTarget("policy", g.Repository), g.Repository, g.Revision
		resolved.withdraw, resolved.cause = !g.Enabled || g.Paused, "policy_disabled"
		if g.Paused {
			resolved.cause = "policy_paused"
		}
		resolved.validate = g.Validate
	case factory.Capacity:
		resolved.target, resolved.revision, resolved.validate = "capacity", g.Revision, g.Validate
	case factory.OperatorGrant:
		resolved.target, resolved.repository, resolved.revision = grantCommandTarget("operator-grant", g.Repository), g.Repository, g.Revision
		resolved.withdraw, resolved.cause, resolved.validate = !g.Active, "operator_grant_withdrawn", g.Validate
	case factory.Sponsorship:
		resolved.target, resolved.repository, resolved.revision = grantCommandTarget("sponsorship/"+g.Connection, g.Repository), g.Repository, g.Revision
		resolved.cause, resolved.validate = "sponsorship_withdrawn", g.Validate
	case project.EnvironmentGrant:
		resolved.target, resolved.repository, resolved.revision = grantCommandTarget("environment-grant", g.Repository), g.Repository, g.Revision
		resolved.withdraw, resolved.cause, resolved.validate = !g.Active, "environment_grant_withdrawn", g.Validate
	default:
		return resolved, errors.New("unsupported factory grant command")
	}
	return resolved, nil
}

// grantCommandExpectedType names the command type one grant change
// requires.
func grantCommandExpectedType(change any) string {
	switch change.(type) {
	case factory.RepositoryPolicy:
		return factory.CommandPolicy
	case factory.Capacity:
		return factory.CommandCapacity
	case factory.OperatorGrant:
		return factory.CommandOperatorGrant
	case factory.Sponsorship:
		return factory.CommandSponsorship
	case project.EnvironmentGrant:
		return factory.CommandEnvironmentGrant
	}
	return ""
}

// lockGrantScopeTx locks the repository gate or the appliance
// capacity row behind one grant command, matching dispatch lock
// order.
func lockGrantScopeTx(ctx context.Context, tx *sql.Tx, repository int64) error {
	if repository > 0 {
		_, _, _, err := ensureDispatchGateTx(ctx, tx, repository)
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO factory_capacity(id,revision,data) VALUES(1,0,'{}'::jsonb) ON CONFLICT(id) DO NOTHING`); err != nil {
		return err
	}
	var locked int64
	return tx.QueryRowContext(ctx, `SELECT revision FROM factory_capacity WHERE id=1 FOR UPDATE`).Scan(&locked)
}

// refreshSponsorshipWithdrawTx clears a sponsorship withdrawal when
// another active connection remains in the repository.
func refreshSponsorshipWithdrawTx(ctx context.Context, tx *sql.Tx, change any, withdraw bool) (bool, error) {
	sp, ok := change.(factory.Sponsorship)
	if !ok || sp.Active {
		return withdraw, nil
	}
	var active bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM factory_sponsorships
		WHERE repository=$1 AND connection<>$2 AND COALESCE((data->>'active')::boolean,FALSE))`, sp.Repository, sp.Connection).Scan(&active); err != nil {
		return withdraw, err
	}
	return !active, nil
}

// settleGrantWithdrawalTx folds dispatch, publication and merge
// withdrawals into one grant receipt. A save that withdraws nothing
// (a changed number, a re-enabled grant) leaves open work alone.
func settleGrantWithdrawalTx(ctx context.Context, tx *sql.Tx, repository int64, withdraw bool, cause, principal string, receipt *factory.GrantReceipt) error {
	if repository <= 0 || !withdraw {
		return nil
	}
	withdrawal, err := withdrawDispatchTx(ctx, tx, repository, cause, principal)
	if err != nil {
		return err
	}
	receipt.Captured = withdrawal.Captured
	if receipt.Captured == nil {
		receipt.Captured = []string{}
	}
	if receipt.PublicationsPending, err = requestPublicationWithdrawalsTx(ctx, tx, repository); err != nil {
		return err
	}
	if receipt.MergesPending, err = requestMergeWithdrawalsTx(ctx, tx, repository); err != nil {
		return err
	}
	return nil
}

// ApplyFactoryGrantCommand records one settings command and saves its grant
// (a policy, capacity, operator grant, sponsorship or environment grant)
// under the record's expected revision in one transaction.
func (s *Store) ApplyFactoryGrantCommand(ctx context.Context, cmd factory.Command, change any, now time.Time) (factory.Command, bool, error) {
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
	resolved, err := resolveGrantCommandTarget(change)
	if err != nil {
		return factory.Command{}, false, err
	}
	if err = resolved.validate(); err != nil {
		return factory.Command{}, false, err
	}
	if cmd.Type != grantCommandExpectedType(change) || cmd.Target != resolved.target || resolved.revision < 0 {
		return factory.Command{}, false, errors.New("grant command does not match its target")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return factory.Command{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	if err = lockGrantScopeTx(ctx, tx, resolved.repository); err != nil {
		return factory.Command{}, false, err
	}
	stored, created, err := recordFactoryCommandTx(ctx, tx, cmd, now)
	if err != nil || !created {
		return stored, false, err
	}
	if err = saveFactoryGrantChangeTx(ctx, tx, change); err != nil {
		return factory.Command{}, false, err
	}
	withdraw, err := refreshSponsorshipWithdrawTx(ctx, tx, change, resolved.withdraw)
	if err != nil {
		return factory.Command{}, false, err
	}
	receipt := factory.GrantReceipt{CommandID: cmd.ID, Revision: resolved.revision + 1, Withdrawn: withdraw}
	if err = settleGrantWithdrawalTx(ctx, tx, resolved.repository, withdraw, resolved.cause, cmd.Principal, &receipt); err != nil {
		return factory.Command{}, false, err
	}
	return finishGrantReceiptTx(ctx, tx, cmd, now, stored, resolved.repository, &receipt)
}

// finishGrantReceiptTx evaluates authority for one settled grant
// receipt, records its outcome and commits the command.
func finishGrantReceiptTx(ctx context.Context, tx *sql.Tx, cmd factory.Command, now time.Time, stored factory.Command, repository int64, receipt *factory.GrantReceipt) (factory.Command, bool, error) {
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
