package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const schemaFormatVersion = 19

const currentSchema = `CREATE TABLE schema_version(version INTEGER PRIMARY KEY);
INSERT INTO schema_version(version) VALUES(19);
CREATE TABLE users(id INTEGER PRIMARY KEY CHECK(id>0), login TEXT NOT NULL, name TEXT NOT NULL DEFAULT '');
CREATE TABLE keys(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), public TEXT NOT NULL, fingerprint TEXT NOT NULL, UNIQUE(user_id,fingerprint));
CREATE TABLE projects(id TEXT PRIMARY KEY, name TEXT NOT NULL, repository_id INTEGER NOT NULL UNIQUE, owner_id INTEGER NOT NULL REFERENCES users(id), repository TEXT NOT NULL, ip TEXT NOT NULL DEFAULT '', ready INTEGER NOT NULL DEFAULT 0 CHECK(ready IN(0,1)), creation_profile TEXT CHECK(creation_profile IS NULL OR (length(CAST(creation_profile AS BLOB))<=1024 AND json_valid(creation_profile))));
CREATE TRIGGER immutable_creation_profile BEFORE UPDATE OF creation_profile ON projects BEGIN SELECT RAISE(ABORT,'creation profile is immutable'); END;
CREATE TABLE memberships(project_id TEXT NOT NULL REFERENCES projects(id), user_id INTEGER NOT NULL REFERENCES users(id), login TEXT NOT NULL, PRIMARY KEY(project_id,user_id), UNIQUE(project_id,login));
CREATE TABLE grant_key_check(id INTEGER PRIMARY KEY CHECK(id=1), ciphertext BLOB NOT NULL);
CREATE TABLE factory_runs(
id TEXT PRIMARY KEY,
active INTEGER NOT NULL CHECK(active IN(0,1)), settled INTEGER NOT NULL CHECK(settled IN(0,1)),
data TEXT NOT NULL CHECK(json_valid(data)));
CREATE INDEX factory_unsettled_runs ON factory_runs(settled) WHERE settled=0;
CREATE TRIGGER factory_run_binding_immutable BEFORE UPDATE ON factory_runs
WHEN NEW.id!=OLD.id
OR json_extract(NEW.data,'$.project_id')!=json_extract(OLD.data,'$.project_id')
OR json_extract(NEW.data,'$.role')!=json_extract(OLD.data,'$.role')
OR json_extract(NEW.data,'$.input_sha')!=json_extract(OLD.data,'$.input_sha')
OR json_extract(NEW.data,'$.started')!=json_extract(OLD.data,'$.started')
OR json_extract(NEW.data,'$.deadline')!=json_extract(OLD.data,'$.deadline')
OR json_extract(NEW.data,'$.image')!=json_extract(OLD.data,'$.image')
OR json_extract(NEW.data,'$.harness')!=json_extract(OLD.data,'$.harness')
OR json_extract(NEW.data,'$.model')!=json_extract(OLD.data,'$.model')
OR (coalesce(json_extract(OLD.data,'$.outcome'),'')!='' AND coalesce(json_extract(NEW.data,'$.outcome'),'')!=json_extract(OLD.data,'$.outcome'))
OR (json_extract(OLD.data,'$.reconciled')=1 AND json_extract(NEW.data,'$.reconciled')!=1)
BEGIN SELECT RAISE(ABORT,'factory run binding is immutable'); END;
CREATE TABLE factory_commands(
id TEXT PRIMARY KEY, type TEXT NOT NULL, target TEXT NOT NULL DEFAULT '',
principal TEXT NOT NULL, digest TEXT NOT NULL, payload TEXT NOT NULL DEFAULT '' CHECK(length(payload)<=8192),
outcome TEXT NOT NULL DEFAULT '' CHECK(length(outcome)<=65536),
created TEXT NOT NULL, finished TEXT NOT NULL DEFAULT '');
CREATE TRIGGER factory_command_immutable BEFORE UPDATE ON factory_commands
WHEN NEW.id!=OLD.id OR NEW.type!=OLD.type OR NEW.target!=OLD.target
OR NEW.principal!=OLD.principal OR NEW.digest!=OLD.digest OR NEW.payload!=OLD.payload OR NEW.created!=OLD.created
OR (OLD.finished!='' AND (NEW.finished!=OLD.finished OR NEW.outcome!=OLD.outcome))
BEGIN SELECT RAISE(ABORT,'factory command is immutable'); END;
CREATE TABLE factory_policies(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE factory_capacity(id INTEGER PRIMARY KEY CHECK(id=1), revision INTEGER NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE factory_operator_grants(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE factory_sponsorships(repository INTEGER NOT NULL CHECK(repository>0), connection TEXT NOT NULL, revision INTEGER NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)), PRIMARY KEY(repository,connection));
CREATE TABLE factory_dispatch(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, open INTEGER NOT NULL CHECK(open IN(0,1)), data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE factory_dispatch_regs(id TEXT PRIMARY KEY, repository INTEGER NOT NULL CHECK(repository>0), revision INTEGER NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)));
CREATE INDEX factory_dispatch_regs_repository ON factory_dispatch_regs(repository);
CREATE TRIGGER factory_dispatch_reg_immutable_update BEFORE UPDATE ON factory_dispatch_regs BEGIN SELECT RAISE(ABORT,'dispatch registration is immutable'); END;
CREATE TRIGGER factory_dispatch_reg_immutable_delete BEFORE DELETE ON factory_dispatch_regs BEGIN SELECT RAISE(ABORT,'dispatch registration is immutable'); END;
CREATE TABLE factory_takeovers(run TEXT NOT NULL, member TEXT NOT NULL, project TEXT NOT NULL REFERENCES projects(id), data TEXT NOT NULL CHECK(json_valid(data)), PRIMARY KEY(run,member));
CREATE TRIGGER factory_takeover_immutable_update BEFORE UPDATE ON factory_takeovers BEGIN SELECT RAISE(ABORT,'takeover record is immutable'); END;
CREATE TRIGGER factory_takeover_immutable_delete BEFORE DELETE ON factory_takeovers BEGIN SELECT RAISE(ABORT,'takeover record is immutable'); END;
CREATE TABLE project_environment_grants(repository INTEGER PRIMARY KEY CHECK(repository>0), revision INTEGER NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE project_requirement_decisions(id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), predecessor TEXT NOT NULL DEFAULT '', data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE project_requirement_heads(project_id TEXT PRIMARY KEY REFERENCES projects(id), decision TEXT NOT NULL);
CREATE TRIGGER project_requirement_decision_immutable_update BEFORE UPDATE ON project_requirement_decisions BEGIN SELECT RAISE(ABORT,'requirement decision is immutable'); END;
CREATE TRIGGER project_requirement_decision_immutable_delete BEFORE DELETE ON project_requirement_decisions BEGIN SELECT RAISE(ABORT,'requirement decision is immutable'); END;
CREATE TABLE project_approval_decisions(id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), predecessor TEXT NOT NULL DEFAULT '', data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE project_approval_heads(project_id TEXT PRIMARY KEY REFERENCES projects(id), decision TEXT NOT NULL);
CREATE TRIGGER project_approval_decision_immutable_update BEFORE UPDATE ON project_approval_decisions BEGIN SELECT RAISE(ABORT,'approval decision is immutable'); END;
CREATE TRIGGER project_approval_decision_immutable_delete BEFORE DELETE ON project_approval_decisions BEGIN SELECT RAISE(ABORT,'approval decision is immutable'); END;
CREATE TABLE identity_connections(id TEXT PRIMARY KEY, owner_id INTEGER NOT NULL, generation INTEGER NOT NULL, state TEXT NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)), credential BLOB NOT NULL);
CREATE TABLE identity_grants(id TEXT PRIMARY KEY, connection_id TEXT NOT NULL REFERENCES identity_connections(id), user_id INTEGER NOT NULL, project_id TEXT NOT NULL, revision INTEGER NOT NULL, revoked INTEGER NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)));
CREATE UNIQUE INDEX identity_grant_recipient ON identity_grants(connection_id,user_id,project_id) WHERE revoked=0;
CREATE TABLE identity_leases(id TEXT PRIMARY KEY, connection_id TEXT NOT NULL REFERENCES identity_connections(id), data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE identity_events(id INTEGER PRIMARY KEY AUTOINCREMENT, owner_id INTEGER NOT NULL, connection_id TEXT NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TRIGGER identity_events_immutable_update BEFORE UPDATE ON identity_events BEGIN SELECT RAISE(ABORT,'identity audit is immutable'); END;
CREATE TRIGGER identity_events_immutable_delete BEFORE DELETE ON identity_events BEGIN SELECT RAISE(ABORT,'identity audit is immutable'); END;
CREATE TABLE project_lifecycle_grants(project_id TEXT PRIMARY KEY REFERENCES projects(id), revision INTEGER NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE project_maintenance(project_id TEXT PRIMARY KEY REFERENCES projects(id), revision INTEGER NOT NULL, hold INTEGER NOT NULL CHECK(hold IN(0,1)), data TEXT NOT NULL CHECK(json_valid(data)));
CREATE TABLE project_preparations(id TEXT PRIMARY KEY, project_id TEXT NOT NULL REFERENCES projects(id), role TEXT NOT NULL, revision INTEGER NOT NULL, requirements TEXT NOT NULL, approval TEXT NOT NULL, data TEXT NOT NULL CHECK(json_valid(data)));
CREATE INDEX project_preparations_project ON project_preparations(project_id);
CREATE TRIGGER preparation_refs_immutable BEFORE UPDATE ON project_preparations
WHEN NEW.id!=OLD.id OR NEW.project_id!=OLD.project_id OR NEW.role!=OLD.role OR NEW.requirements!=OLD.requirements OR NEW.approval!=OLD.approval
BEGIN SELECT RAISE(ABORT,'preparation references are immutable'); END;
CREATE TABLE identity_executions(kind TEXT NOT NULL, execution_id TEXT NOT NULL, state TEXT NOT NULL, lease_id TEXT NOT NULL DEFAULT '', data TEXT NOT NULL CHECK(json_valid(data)), PRIMARY KEY(kind,execution_id));
CREATE TRIGGER identity_execution_immutable BEFORE UPDATE ON identity_executions
WHEN NEW.kind!=OLD.kind OR NEW.execution_id!=OLD.execution_id
OR json_extract(NEW.data,'$.digest')!=json_extract(OLD.data,'$.digest')
OR (OLD.state='terminal' AND NEW.state!='terminal')
BEGIN SELECT RAISE(ABORT,'execution identity is immutable'); END;`

// SchemaVersion is a format identifier. This unreleased database format has no
// compatibility migrations; older versioned files are refused without mutation.
func SchemaVersion() int { return schemaFormatVersion }

func schemaPresent(ctx context.Context, tx *sql.Tx) (bool, error) {
	var hasVersion int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='schema_version'`).Scan(&hasVersion); err != nil {
		return false, err
	}
	return hasVersion != 0, nil
}

func refuseUnversionedDatabase(ctx context.Context, tx *sql.Tx) error {
	var objects int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE name NOT LIKE 'sqlite_%'`).Scan(&objects); err != nil {
		return err
	}
	if objects != 0 {
		return errors.New("refusing an unversioned nonempty database")
	}
	return nil
}

func storedSchemaVersion(ctx context.Context, tx *sql.Tx) (int, error) {
	var count int
	var minimum, maximum sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT count(*),min(version),max(version) FROM schema_version`).Scan(&count, &minimum, &maximum); err != nil {
		return 0, err
	}
	if count != 1 || !minimum.Valid || minimum.Int64 < 1 || minimum.Int64 != maximum.Int64 {
		return 0, errors.New("invalid database schema version record")
	}
	return int(minimum.Int64), nil
}

func verifyRequiredColumns(ctx context.Context, tx *sql.Tx) error {
	for _, query := range []string{
		`SELECT id,login,name FROM users LIMIT 0`,
		`SELECT id,user_id,public,fingerprint FROM keys LIMIT 0`,
		`SELECT id,name,repository_id,owner_id,repository,ip,ready,creation_profile FROM projects LIMIT 0`,
		`SELECT project_id,user_id,login FROM memberships LIMIT 0`,
		`SELECT id,ciphertext FROM grant_key_check LIMIT 0`,
		`SELECT project_id,revision,data FROM project_lifecycle_grants LIMIT 0`,
		`SELECT project_id,revision,hold,data FROM project_maintenance LIMIT 0`,
		`SELECT id,project_id,role,revision,requirements,approval,data FROM project_preparations LIMIT 0`,
		`SELECT kind,execution_id,state,lease_id,data FROM identity_executions LIMIT 0`,
		`SELECT id,active,settled,data FROM factory_runs LIMIT 0`,
		`SELECT id,type,target,principal,digest,payload,outcome,created,finished FROM factory_commands LIMIT 0`,
		`SELECT repository,revision,data FROM factory_policies LIMIT 0`,
		`SELECT id,revision,data FROM factory_capacity LIMIT 0`,
		`SELECT repository,revision,data FROM factory_operator_grants LIMIT 0`,
		`SELECT repository,connection,revision,data FROM factory_sponsorships LIMIT 0`,
		`SELECT repository,revision,open,data FROM factory_dispatch LIMIT 0`,
		`SELECT id,repository,revision,data FROM factory_dispatch_regs LIMIT 0`,
		`SELECT run,member,project,data FROM factory_takeovers LIMIT 0`,
		`SELECT repository,revision,data FROM project_environment_grants LIMIT 0`,
		`SELECT id,project_id,predecessor,data FROM project_requirement_decisions LIMIT 0`,
		`SELECT project_id,decision FROM project_requirement_heads LIMIT 0`,
		`SELECT id,project_id,predecessor,data FROM project_approval_decisions LIMIT 0`,
		`SELECT project_id,decision FROM project_approval_heads LIMIT 0`,
	} {
		rows, err := tx.QueryContext(ctx, query)
		if err != nil {
			return errors.New("database schema is incomplete")
		}
		if err = rows.Close(); err != nil {
			return err
		}
	}
	return nil
}

func verifyImmutableCreationProfile(ctx context.Context, tx *sql.Tx) error {
	var hasImmutableProfile int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name='immutable_creation_profile' AND tbl_name='projects'`).Scan(&hasImmutableProfile); err != nil {
		return err
	}
	if hasImmutableProfile != 1 {
		return errors.New("database schema is incomplete")
	}
	return nil
}

func verifyImmutablePreparationRefs(ctx context.Context, tx *sql.Tx) error {
	var hasImmutableRefs int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name='preparation_refs_immutable' AND tbl_name='project_preparations'`).Scan(&hasImmutableRefs); err != nil {
		return err
	}
	if hasImmutableRefs != 1 {
		return errors.New("database schema is incomplete")
	}
	return nil
}

func verifyImmutableExecutionIdentity(ctx context.Context, tx *sql.Tx) error {
	var hasImmutableExecution int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name='identity_execution_immutable' AND tbl_name='identity_executions'`).Scan(&hasImmutableExecution); err != nil {
		return err
	}
	if hasImmutableExecution != 1 {
		return errors.New("database schema is incomplete")
	}
	return nil
}

func loadSchemaVersion(ctx context.Context, tx *sql.Tx) (int, error) {
	present, err := schemaPresent(ctx, tx)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, refuseUnversionedDatabase(ctx, tx)
	}
	return storedSchemaVersion(ctx, tx)
}

func initializeSchema(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	version, err := loadSchemaVersion(ctx, tx)
	if err != nil {
		return err
	}
	if version == 0 {
		if _, err = tx.ExecContext(ctx, currentSchema); err != nil {
			return fmt.Errorf("create current database schema: %w", err)
		}
	} else if version != SchemaVersion() {
		return errors.New("database schema differs from this application")
	}
	if err = verifyRequiredColumns(ctx, tx); err != nil {
		return err
	}
	if err = verifyImmutableCreationProfile(ctx, tx); err != nil {
		return err
	}
	if err = verifyImmutablePreparationRefs(ctx, tx); err != nil {
		return err
	}
	if err = verifyImmutableExecutionIdentity(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
