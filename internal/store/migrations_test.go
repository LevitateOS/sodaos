package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func legacyDatabase(t *testing.T) (string, *sql.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	schema, err := os.ReadFile("testdata/v1.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	return path, db
}

func TestMigrationPreservesLegacyProductState(t *testing.T) {
	path, db := legacyDatabase(t)
	for _, statement := range []string{
		`INSERT INTO users VALUES(1,'alice','My Soda name')`,
		`INSERT INTO keys VALUES(7,1,'fixture-public','fixture-fingerprint')`,
		`INSERT INTO projects VALUES('pfixture','Demo',42,1,'alice/demo','10.89.0.2',1)`,
		`INSERT INTO projects VALUES('pincomplete','Other',43,1,'alice/other','',0)`,
		`INSERT INTO memberships VALUES('pfixture',1,'alice')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	expires := time.Now().Add(time.Hour).Unix()
	if _, err := db.Exec(`INSERT INTO sessions VALUES(?,?,?,?)`, hash("legacy-session"), 1, "fixture-csrf", expires); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO oauth VALUES(?,?,?)`, hash("legacy-state"), "fixture-verifier", expires); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	v, err := s.Session(ctx, "legacy-session")
	if err != nil || v.User.Name != "My Soda name" || v.CSRF != "fixture-csrf" {
		t.Fatal(v, err)
	}
	keys, err := s.Keys(ctx, 1)
	if err != nil || len(keys) != 1 || keys[0].ID != 7 {
		t.Fatal(keys, err)
	}
	p, err := s.Project(ctx, "pfixture")
	if err != nil || !p.Ready || p.RepositoryID != 42 || p.IP != "10.89.0.2" {
		t.Fatal(p, err)
	}
	p, err = s.Project(ctx, "pincomplete")
	if err != nil || p.Ready {
		t.Fatal(p, err)
	}
	login, err := s.MemberLogin(ctx, "pfixture", 1)
	if err != nil || login != "alice" {
		t.Fatal(login, err)
	}
	oauth, err := s.ConsumeOAuth(ctx, "legacy-state")
	if err != nil || oauth != (OAuthLogin{Verifier: "fixture-verifier"}) {
		t.Fatal(oauth, err)
	}
	if _, err = s.ConsumeOAuth(ctx, "legacy-state"); err == nil {
		t.Fatal("OAuth state reused")
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var version, count int
	if err = s.db.QueryRow(`SELECT count(*),max(version) FROM schema_version`).Scan(&count, &version); err != nil || version != len(migrations) || count != 1 {
		t.Fatal(version, count, err)
	}
	if _, err = s.db.Exec(`INSERT INTO keys(user_id,public,fingerprint) VALUES(999,'x','y')`); err == nil {
		t.Fatal("foreign key enforcement lost")
	}
}

func TestMigrationRejectsUnknownVersionAndUnversionedData(t *testing.T) {
	for _, schema := range []string{
		`CREATE TABLE unrelated(value TEXT); INSERT INTO unrelated VALUES('keep')`,
		`CREATE TABLE schema_version(version INTEGER PRIMARY KEY); INSERT INTO schema_version VALUES(999)`,
		`CREATE TABLE schema_version(version INTEGER PRIMARY KEY); INSERT INTO schema_version VALUES(1),(2)`,
		`CREATE TABLE schema_version(version INTEGER PRIMARY KEY)`,
	} {
		path := filepath.Join(t.TempDir(), "db")
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(schema); err != nil {
			t.Fatal(err)
		}
		db.Close()
		if s, err := Open(path); err == nil {
			s.Close()
			t.Fatal("unknown schema accepted")
		}
		db, err = sql.Open("sqlite", path)
		if err != nil {
			t.Fatal(err)
		}
		var count int
		if err = db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='users'`).Scan(&count); err != nil || count != 0 {
			t.Fatal(count, err)
		}
		db.Close()
	}
}

func TestMigrationFailureRollsBackVersionAndSchema(t *testing.T) {
	path, db := legacyDatabase(t)
	if _, err := db.Exec(`DROP TABLE oauth`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if s, err := Open(path); err == nil {
		s.Close()
		t.Fatal("partial schema silently repaired")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version int
	if err = db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil || version != 1 {
		t.Fatal(version, err)
	}
}

func TestIncompleteLegacySchemaDoesNotCommitMigration(t *testing.T) {
	path, db := legacyDatabase(t)
	if _, err := db.Exec(`DROP TABLE projects`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if s, err := Open(path); err == nil {
		s.Close()
		t.Fatal("incomplete schema accepted")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var version, columns int
	if err = db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil || version != 1 {
		t.Fatal(version, err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM pragma_table_info('oauth') WHERE name='return_path'`).Scan(&columns); err != nil || columns != 0 {
		t.Fatal("migration was partially committed", columns, err)
	}
}

func TestLegacyOAuthDestinationIsIgnoredAndStateIsSingleUse(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	// Exact pre-removal state shape, not a newly supported redirect choice.
	if _, err = s.db.ExecContext(ctx, `INSERT INTO oauth(state,verifier,expires,return_path) VALUES(?,?,?,?)`, hash("state"), "verifier", time.Now().Add(time.Minute).Unix(), "/app/"); err != nil {
		t.Fatal(err)
	}
	v, err := s.ConsumeOAuth(ctx, "state")
	if err != nil || v != (OAuthLogin{Verifier: "verifier"}) {
		t.Fatal(v, err)
	}
	if _, err = s.ConsumeOAuth(ctx, "state"); err == nil {
		t.Fatal("replayed state")
	}
}
