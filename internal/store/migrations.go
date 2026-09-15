package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Entries are append-only. Never infer a database version from application data
// or recreate missing tables over an unknown/partially initialized database.
var migrations = []string{
	`CREATE TABLE schema_version(version INTEGER PRIMARY KEY);
CREATE TABLE users(id INTEGER PRIMARY KEY CHECK(id>0), login TEXT NOT NULL, name TEXT NOT NULL DEFAULT '');
CREATE TABLE keys(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), public TEXT NOT NULL, fingerprint TEXT NOT NULL, UNIQUE(user_id,fingerprint));
CREATE TABLE projects(id TEXT PRIMARY KEY, name TEXT NOT NULL, repository_id INTEGER NOT NULL UNIQUE, owner_id INTEGER NOT NULL REFERENCES users(id), repository TEXT NOT NULL, ip TEXT NOT NULL DEFAULT '', ready INTEGER NOT NULL DEFAULT 0 CHECK(ready IN(0,1)));
CREATE TABLE memberships(project_id TEXT NOT NULL REFERENCES projects(id), user_id INTEGER NOT NULL REFERENCES users(id), login TEXT NOT NULL, PRIMARY KEY(project_id,user_id), UNIQUE(project_id,login));
CREATE TABLE sessions(token TEXT PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), csrf TEXT NOT NULL, expires INTEGER NOT NULL);
CREATE TABLE oauth(state TEXT PRIMARY KEY, verifier TEXT NOT NULL, expires INTEGER NOT NULL);`,
	`ALTER TABLE oauth ADD COLUMN return_path TEXT NOT NULL DEFAULT '/projects' CHECK(return_path IN ('/projects','/app/'));`,
	`CREATE TABLE grant_key_check(id INTEGER PRIMARY KEY CHECK(id=1), ciphertext BLOB NOT NULL);
CREATE TABLE session_grants(session_token TEXT PRIMARY KEY REFERENCES sessions(token) ON DELETE CASCADE, ciphertext BLOB NOT NULL);`,
	`ALTER TABLE oauth ADD COLUMN repository_id INTEGER NOT NULL DEFAULT 0 CHECK(repository_id>=0);
ALTER TABLE oauth ADD COLUMN expected_user_id INTEGER NOT NULL DEFAULT 0 CHECK(expected_user_id>=0);`,
	`CREATE TABLE login_contexts(id TEXT PRIMARY KEY, pending TEXT UNIQUE, expires INTEGER NOT NULL);
ALTER TABLE sessions ADD COLUMN context_id TEXT REFERENCES login_contexts(id) ON DELETE CASCADE;
INSERT INTO login_contexts(id,expires) SELECT token,expires FROM sessions;
UPDATE sessions SET context_id=token;
CREATE UNIQUE INDEX sessions_context ON sessions(context_id);
ALTER TABLE oauth ADD COLUMN context_id TEXT REFERENCES login_contexts(id) ON DELETE CASCADE;`,
	`ALTER TABLE oauth ADD COLUMN spaces_return INTEGER NOT NULL DEFAULT 0 CHECK(spaces_return IN(0,1) AND (spaces_return=0 OR repository_id=0));`,
	`ALTER TABLE oauth ADD COLUMN settings_return TEXT NOT NULL DEFAULT '' CHECK(settings_return IN ('','runners') AND (settings_return='' OR (spaces_return=0 AND repository_id=0)));`,
	`ALTER TABLE projects ADD COLUMN creation_profile TEXT CHECK(creation_profile IS NULL OR (length(CAST(creation_profile AS BLOB))<=1024 AND json_valid(creation_profile)));
CREATE TRIGGER immutable_creation_profile BEFORE UPDATE OF creation_profile ON projects BEGIN SELECT RAISE(ABORT,'creation profile is immutable'); END;
ALTER TABLE oauth ADD COLUMN repository_settings_return INTEGER NOT NULL DEFAULT 0 CHECK(repository_settings_return IN(0,1) AND (repository_settings_return=0 OR (repository_id>0 AND spaces_return=0 AND settings_return='')));`,
	`ALTER TABLE login_contexts ADD COLUMN oauth_cookie TEXT;
CREATE UNIQUE INDEX login_context_oauth_cookie ON login_contexts(oauth_cookie);
UPDATE login_contexts SET oauth_cookie=pending;`,
	`CREATE TABLE oauth_tailnet (
state TEXT PRIMARY KEY, verifier TEXT NOT NULL, expires INTEGER NOT NULL,
return_path TEXT NOT NULL DEFAULT '/projects' CHECK(return_path IN ('/projects','/app/')),
repository_id INTEGER NOT NULL DEFAULT 0 CHECK(repository_id>=0),
expected_user_id INTEGER NOT NULL DEFAULT 0 CHECK(expected_user_id>=0),
context_id TEXT REFERENCES login_contexts(id) ON DELETE CASCADE,
spaces_return INTEGER NOT NULL DEFAULT 0 CHECK(spaces_return IN(0,1) AND (spaces_return=0 OR repository_id=0)),
settings_return TEXT NOT NULL DEFAULT '' CHECK(settings_return IN ('','runners','tailnet') AND (settings_return='' OR (spaces_return=0 AND repository_id=0))),
repository_settings_return INTEGER NOT NULL DEFAULT 0 CHECK(repository_settings_return IN(0,1) AND (repository_settings_return=0 OR (repository_id>0 AND spaces_return=0 AND settings_return='')))
);
INSERT INTO oauth_tailnet SELECT state,verifier,expires,return_path,repository_id,expected_user_id,context_id,spaces_return,settings_return,repository_settings_return FROM oauth;
DROP TABLE oauth;
ALTER TABLE oauth_tailnet RENAME TO oauth;`,
}

// SchemaVersion identifies the schema produced by this source's migration owner.
// Matching versions alone do not establish an approved appliance upgrade path.
func SchemaVersion() int { return len(migrations) }

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

func currentSchemaVersion(ctx context.Context, tx *sql.Tx) (int, error) {
	var count int
	var minimum, maximum sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT count(*),min(version),max(version) FROM schema_version`).Scan(&count, &minimum, &maximum); err != nil {
		return 0, err
	}
	if count != 1 || !minimum.Valid || minimum.Int64 < 1 {
		return 0, errors.New("invalid database schema version record")
	}
	if maximum.Int64 > int64(len(migrations)) {
		return 0, errors.New("database schema is newer than this application")
	}
	return int(minimum.Int64), nil
}

func applyMigrations(ctx context.Context, tx *sql.Tx, version int) error {
	for version < len(migrations) {
		if _, err := tx.ExecContext(ctx, migrations[version]); err != nil {
			return fmt.Errorf("database migration %d failed: %w", version+1, err)
		}
		version++
		if _, err := tx.ExecContext(ctx, `DELETE FROM schema_version`); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_version(version) VALUES(?)`, version); err != nil {
			return err
		}
	}
	return nil
}

func verifyRequiredColumns(ctx context.Context, tx *sql.Tx) error {
	for _, query := range []string{
		`SELECT id,login,name FROM users LIMIT 0`,
		`SELECT id,user_id,public,fingerprint FROM keys LIMIT 0`,
		`SELECT id,name,repository_id,owner_id,repository,ip,ready,creation_profile FROM projects LIMIT 0`,
		`SELECT project_id,user_id,login FROM memberships LIMIT 0`,
		`SELECT token,user_id,csrf,expires,context_id FROM sessions LIMIT 0`,
		`SELECT state,verifier,expires,return_path,repository_id,expected_user_id,context_id,spaces_return,settings_return,repository_settings_return FROM oauth LIMIT 0`,
		`SELECT id,pending,expires,oauth_cookie FROM login_contexts LIMIT 0`,
		`SELECT id,ciphertext FROM grant_key_check LIMIT 0`,
		`SELECT session_token,ciphertext FROM session_grants LIMIT 0`,
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

func probeTailnetReturnContract(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `SAVEPOINT tailnet_return_contract`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO oauth(state,verifier,expires,settings_return) VALUES(lower(hex(randomblob(32))),'',0,'tailnet')`); err != nil {
		return errors.New("database schema is incomplete")
	}
	if _, err := tx.ExecContext(ctx, `ROLLBACK TO tailnet_return_contract; RELEASE tailnet_return_contract`); err != nil {
		return err
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

func loadSchemaVersion(ctx context.Context, tx *sql.Tx) (int, error) {
	present, err := schemaPresent(ctx, tx)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, refuseUnversionedDatabase(ctx, tx)
	}
	return currentSchemaVersion(ctx, tx)
}

func migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	version, err := loadSchemaVersion(ctx, tx)
	if err != nil {
		return err
	}
	if err = applyMigrations(ctx, tx, version); err != nil {
		return err
	}
	if err = verifyRequiredColumns(ctx, tx); err != nil {
		return err
	}
	if err = probeTailnetReturnContract(ctx, tx); err != nil {
		return err
	}
	if err = verifyImmutableCreationProfile(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
