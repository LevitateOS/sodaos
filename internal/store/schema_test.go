package store

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"testing"
)

func TestOpenCreatesOnlyTheCurrentSchema(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	var version, count int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*),max(version) FROM schema_version`).Scan(&count, &version); err != nil || count != 1 || version != SchemaVersion() {
		t.Fatalf("schema version: count=%d version=%d err=%v", count, version, err)
	}
	var retired int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('sessions','oauth','session_grants','login_contexts')`).Scan(&retired); err != nil || retired != 0 {
		t.Fatalf("retired browser auth tables: count=%d err=%v", retired, err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO keys(user_id,public,fingerprint) VALUES(999,'x','y')`); err == nil {
		t.Fatal("foreign key enforcement lost")
	}
}

// legacyFingerprint captures the staged legacy content a refused Open must
// leave untouched: the version marker, the table inventory and one legacy
// row.
func legacyFingerprint(t *testing.T, db *sql.DB) string {
	t.Helper()
	var version int
	if err := db.QueryRow(`SELECT max(version) FROM schema_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			t.Fatal(err)
		}
		tables = append(tables, name)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	var token string
	if err := db.QueryRow(`SELECT token FROM sessions LIMIT 1`).Scan(&token); err != nil {
		t.Fatal(err)
	}
	return strconv.Itoa(version) + "|" + strings.Join(tables, ",") + "|" + token
}

func TestOpenRejectsOldSchemaWithoutMutation(t *testing.T) {
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	for _, version := range []int{1, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29} {
		t.Run("version_"+strconv.Itoa(version), func(t *testing.T) {
			ctx := context.Background()
			dsn, drop, err := createEphemeralDatabase(ctx, admin)
			if err != nil {
				t.Fatal(err)
			}
			defer drop()
			db, err := sql.Open("pgx", dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			for _, statement := range []string{
				`CREATE TABLE schema_version(version INTEGER PRIMARY KEY)`,
				`CREATE TABLE sessions(token TEXT PRIMARY KEY)`,
				`CREATE TABLE oauth(state TEXT PRIMARY KEY)`,
				`INSERT INTO sessions(token) VALUES('preserve-this-file')`,
			} {
				if _, err = db.Exec(statement); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = db.Exec(`INSERT INTO schema_version(version) VALUES($1)`, version); err != nil {
				t.Fatal(err)
			}
			before := legacyFingerprint(t, db)
			if s, err := Open(dsn); err == nil {
				_ = s.Close()
				t.Fatal("unsupported schema accepted")
			}
			if after := legacyFingerprint(t, db); after != before {
				t.Fatal("unsupported database was modified")
			}
			observed, err := ReadSchemaVersion(ctx, dsn)
			if err != nil || observed != version {
				t.Fatalf("historical version observation: got %d, err %v", observed, err)
			}
		})
	}
}

func TestOpenRejectsUnversionedDataWithoutMutation(t *testing.T) {
	admin, ok := TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx := context.Background()
	dsn, drop, err := createEphemeralDatabase(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer drop()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE unrelated(value TEXT)`,
		`INSERT INTO unrelated VALUES('keep')`,
	} {
		if _, err = db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	var before string
	if err = db.QueryRow(`SELECT value FROM unrelated LIMIT 1`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(dsn); err == nil {
		_ = s.Close()
		t.Fatal("unversioned data accepted")
	}
	var after string
	if err = db.QueryRow(`SELECT value FROM unrelated LIMIT 1`).Scan(&after); err != nil || after != before {
		t.Fatalf("unversioned database was modified: %v", err)
	}
}
