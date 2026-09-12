package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV8RequiresColumnsAndProjectImmutabilityTriggerWithoutRepair(t *testing.T) {
	// Frozen v7 plus independent v8 additions: a production migration edit must
	// not silently change the malformed fixtures or the valid-current control.
	v7, err := os.ReadFile("testdata/v7.sql")
	if err != nil {
		t.Fatal(err)
	}
	additions := []struct{ name, sql string }{
		{"creation_profile", `ALTER TABLE projects ADD COLUMN creation_profile TEXT CHECK(creation_profile IS NULL OR (length(CAST(creation_profile AS BLOB))<=1024 AND json_valid(creation_profile)))`},
		{"repository_settings_return", `ALTER TABLE oauth ADD COLUMN repository_settings_return INTEGER NOT NULL DEFAULT 0 CHECK(repository_settings_return IN(0,1) AND (repository_settings_return=0 OR (repository_id>0 AND spaces_return=0 AND settings_return='')))`},
		{"immutable_creation_profile", `CREATE TRIGGER immutable_creation_profile BEFORE UPDATE OF creation_profile ON projects BEGIN SELECT RAISE(ABORT,'creation profile is immutable'); END`},
	}
	for _, tc := range []struct{ name, omit, replacement string }{
		{name: "valid current"},
		{name: "missing creation profile", omit: "creation_profile"},
		{name: "missing repository return", omit: "repository_settings_return"},
		{name: "missing trigger", omit: "immutable_creation_profile"},
		{name: "trigger on wrong table", omit: "immutable_creation_profile", replacement: `CREATE TRIGGER immutable_creation_profile BEFORE UPDATE ON users BEGIN SELECT RAISE(ABORT,'wrong table'); END`},
		{name: "index is not a trigger", omit: "immutable_creation_profile", replacement: `CREATE INDEX immutable_creation_profile ON projects(creation_profile)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "v8.db")
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.Close() })
			if _, err := db.Exec(string(v7)); err != nil {
				t.Fatal(err)
			}
			for _, addition := range additions {
				if addition.name == tc.omit {
					continue
				}
				// SQLite permits UPDATE OF an absent column in a trigger. Keep
				// that trigger so the missing-column case tests only the column.
				if _, err := db.Exec(addition.sql); err != nil {
					t.Fatal(err)
				}
			}
			if tc.replacement != "" {
				if _, err := db.Exec(tc.replacement); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.Exec(`UPDATE schema_version SET version=8;
INSERT INTO users VALUES(1,'alice','Preserve name');
INSERT INTO projects(id,name,repository_id,owner_id,repository,ip,ready) VALUES('legacy','Original',7,1,'alice/original','10.0.0.2',1);
INSERT INTO memberships VALUES('legacy',1,'original-login');
INSERT INTO oauth(state,verifier,expires,settings_return) VALUES('pending','fixture-verifier',9999999999,'runners');`); err != nil {
				t.Fatal(err)
			}
			const catalog = `SELECT group_concat(sql, char(10)) FROM (SELECT sql FROM sqlite_master ORDER BY type,name)`
			var before, after string
			if err := db.QueryRow(catalog).Scan(&before); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				s, err := Open(path)
				if tc.omit == "" {
					if err != nil {
						t.Fatal(err)
					}
					if _, err := s.db.Exec(`UPDATE projects SET creation_profile=NULL WHERE id='legacy'`); err == nil {
						t.Fatal("valid-current creation profile was mutable")
					}
					s.Close()
				} else {
					if err == nil {
						s.Close()
						t.Fatal("malformed v8 accepted")
					}
					// The v10 table copy now consumes this missing column before
					// the final completeness check. Both paths must refuse and
					// preserve the original schema/version/rows checked below.
					if !strings.Contains(err.Error(), "database schema is incomplete") && !(tc.omit == "repository_settings_return" && strings.Contains(err.Error(), "database migration 10 failed:")) {
						t.Fatal("unexpected refusal", err)
					}
				}
				if err := db.QueryRow(catalog).Scan(&after); err != nil || (tc.omit != "" && after != before) {
					t.Fatal("schema was repaired or changed", err)
				}
				var version int
				expectedVersion := 8
				if tc.omit == "" {
					expectedVersion = len(migrations)
				}
				var name, ip, login, verifier string
				if err := db.QueryRow(`SELECT version,users.name,projects.ip,memberships.login,oauth.verifier FROM schema_version,users,projects,memberships,oauth`).Scan(&version, &name, &ip, &login, &verifier); err != nil || version != expectedVersion || name != "Preserve name" || ip != "10.0.0.2" || login != "original-login" || verifier != "fixture-verifier" {
					t.Fatal("version or retained records changed", err)
				}
			}
		})
	}
}
