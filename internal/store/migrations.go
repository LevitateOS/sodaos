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
}

func migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var hasVersion int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='schema_version'`).Scan(&hasVersion); err != nil {
		return err
	}
	version := 0
	if hasVersion == 0 {
		var objects int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE name NOT LIKE 'sqlite_%'`).Scan(&objects); err != nil {
			return err
		}
		if objects != 0 {
			return errors.New("refusing an unversioned nonempty database")
		}
	} else {
		var count int
		var minimum, maximum sql.NullInt64
		if err = tx.QueryRowContext(ctx, `SELECT count(*),min(version),max(version) FROM schema_version`).Scan(&count, &minimum, &maximum); err != nil {
			return err
		}
		if count != 1 || !minimum.Valid || minimum.Int64 < 1 {
			return errors.New("invalid database schema version record")
		}
		if maximum.Int64 > int64(len(migrations)) {
			return errors.New("database schema is newer than this application")
		}
		version = int(minimum.Int64)
	}
	for version < len(migrations) {
		if _, err = tx.ExecContext(ctx, migrations[version]); err != nil {
			return fmt.Errorf("database migration %d failed: %w", version+1, err)
		}
		version++
		if _, err = tx.ExecContext(ctx, `DELETE FROM schema_version`); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO schema_version(version) VALUES(?)`, version); err != nil {
			return err
		}
	}
	// A version marker alone must not make an incomplete database usable. Check
	// the required columns without reading product rows or recreating tables.
	for _, query := range []string{
		`SELECT id,login,name FROM users LIMIT 0`,
		`SELECT id,user_id,public,fingerprint FROM keys LIMIT 0`,
		`SELECT id,name,repository_id,owner_id,repository,ip,ready FROM projects LIMIT 0`,
		`SELECT project_id,user_id,login FROM memberships LIMIT 0`,
		`SELECT token,user_id,csrf,expires FROM sessions LIMIT 0`,
		`SELECT state,verifier,expires,return_path FROM oauth LIMIT 0`,
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
	return tx.Commit()
}
