package store

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestOpenCreatesOnlyTheCurrentSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "current.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var version, count int
	if err = s.db.QueryRow(`SELECT count(*),max(version) FROM schema_version`).Scan(&count, &version); err != nil || count != 1 || version != SchemaVersion() {
		t.Fatalf("schema version: count=%d version=%d err=%v", count, version, err)
	}
	var retired int
	if err = s.db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('sessions','oauth','session_grants','login_contexts')`).Scan(&retired); err != nil || retired != 0 {
		t.Fatalf("retired browser auth tables: count=%d err=%v", retired, err)
	}
	if _, err = s.db.Exec(`INSERT INTO keys(user_id,public,fingerprint) VALUES(999,'x','y')`); err == nil {
		t.Fatal("foreign key enforcement lost")
	}
}

func TestOpenRejectsOldSchemaWithoutMutation(t *testing.T) {
	for _, version := range []int{1, 13, 15} {
		t.Run("version_"+strconv.Itoa(version), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "old.db")
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			for _, statement := range []string{
				`CREATE TABLE schema_version(version INTEGER PRIMARY KEY)`,
				`CREATE TABLE sessions(token TEXT PRIMARY KEY)`,
				`CREATE TABLE oauth(state TEXT PRIMARY KEY)`,
				`INSERT INTO sessions(token) VALUES('preserve-this-file')`,
			} {
				if _, err = db.Exec(statement); err != nil {
					db.Close()
					t.Fatal(err)
				}
			}
			if _, err = db.Exec(`INSERT INTO schema_version(version) VALUES(?)`, version); err != nil {
				db.Close()
				t.Fatal(err)
			}
			if err = db.Close(); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if s, err := Open(path); err == nil {
				s.Close()
				t.Fatal("unsupported schema accepted")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("unsupported database was modified: %v", err)
			}
			observed, err := ReadSchemaVersion(context.Background(), path)
			if err != nil || observed != version {
				t.Fatalf("historical version observation: got %d, err %v", observed, err)
			}
		})
	}
}

func TestOpenRejectsUnversionedDataWithoutMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unversioned.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE unrelated(value TEXT); INSERT INTO unrelated VALUES('keep')`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if s, err := Open(path); err == nil {
		s.Close()
		t.Fatal("unversioned data accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("unversioned database was modified: %v", err)
	}
}
