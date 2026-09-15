package store

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
)

func openReadOnly(path string) (*sql.DB, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("database path must be a regular file")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dsn := url.URL{Scheme: "file", Path: absolute, RawQuery: url.Values{
		"mode":    {"ro"},
		"_pragma": {"foreign_keys(1)", "busy_timeout(5000)"},
	}.Encode()}
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// ReadSchemaVersion opens an existing database read-only without migrating and
// returns the single schema_version row. Callers that need the current schema
// must use OpenObserve or Open instead.
func ReadSchemaVersion(ctx context.Context, path string) (int, error) {
	db, err := openReadOnly(path)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	return loadSchemaVersion(ctx, tx)
}

// OpenObserve opens an existing database read-only without migrating. It refuses
// unless the stored schema exactly matches SchemaVersion().
func OpenObserve(ctx context.Context, path string) (*Store, error) {
	db, err := openReadOnly(path)
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		db.Close()
		return nil, err
	}
	version, err := loadSchemaVersion(ctx, tx)
	if err != nil {
		tx.Rollback()
		db.Close()
		return nil, err
	}
	if err = tx.Rollback(); err != nil {
		db.Close()
		return nil, err
	}
	if version != SchemaVersion() {
		db.Close()
		return nil, errors.New("database schema differs from this application")
	}
	return &Store{db: db}, nil
}

// IntegrityCheck runs SQLite's integrity_check against an already opened store.
func (s *Store) IntegrityCheck(ctx context.Context) error {
	var integrity string
	if err := s.db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
		return errors.New("soda database integrity failed")
	}
	return nil
}
