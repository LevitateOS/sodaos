package store

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestOAuthContextSurvivesRestartButNotReplayOrExpiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "soda.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	login := OAuthLogin{Verifier: "verifier", RepositoryID: 9223372036854775807, ExpectedUserID: 42}
	if err = s.BeginOAuth(t.Context(), "pending", login, "", ""); err != nil {
		t.Fatal(err)
	}
	if err = s.BeginOAuth(t.Context(), "expired", login, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`UPDATE oauth SET expires=? WHERE state=?`, time.Now().Add(-time.Minute).Unix(), hash("expired")); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.ConsumeOAuth(t.Context(), "pending", "")
	if err != nil || got.OAuthLogin != login {
		t.Fatal(got, err)
	}
	for _, state := range []string{"pending", "expired", "unknown"} {
		if _, err = s.ConsumeOAuth(t.Context(), state, ""); !errors.Is(err, sql.ErrNoRows) {
			t.Fatal("state accepted", state, err)
		}
	}
	for _, invalid := range []OAuthLogin{{Verifier: "v", RepositoryID: -1}, {Verifier: "v", ExpectedUserID: -1}} {
		if err = s.BeginOAuth(t.Context(), "invalid", invalid, "", ""); err == nil {
			t.Fatal("negative native context accepted")
		}
	}
}

func TestLegacyLoginContextMigrationPreservesEncryptedProductState(t *testing.T) {
	for _, oldVersion := range []int{3, 4} {
		t.Run(fmt.Sprint(oldVersion), func(t *testing.T) {
			// Build the genuine old schema, not a newer DB relabelled as old.
			path, db := legacyDatabase(t)
			for _, migration := range migrations[1:oldVersion] {
				if _, err := db.Exec(migration); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := db.Exec(fmt.Sprintf(`UPDATE schema_version SET version=%d; PRAGMA foreign_keys=ON;`, oldVersion)); err != nil {
				t.Fatal(err)
			}
			key := bytes.Repeat([]byte{42}, 32)
			cipher, err := newGrantCipher(key)
			if err != nil {
				t.Fatal(err)
			}
			old := &Store{db: db, grants: cipher}
			if err = old.initializeGrantKey(t.Context()); err != nil {
				t.Fatal(err)
			}
			if err = old.UpsertUser(t.Context(), User{ID: 1, Login: "alice", Name: "Keep profile"}); err != nil {
				t.Fatal(err)
			}
			if err = old.AddKey(t.Context(), 1, "public-fixture", "fingerprint-fixture"); err != nil {
				t.Fatal(err)
			}
			project := Project{ID: "pfixture", RepositoryID: 42, OwnerID: 1, Name: "Keep", Repository: "alice/keep"}
			if err = old.CreateProject(t.Context(), project); err != nil {
				t.Fatal(err)
			}
			if err = old.MarkReady(t.Context(), project.ID, "10.89.0.2"); err != nil {
				t.Fatal(err)
			}
			if err = old.Join(t.Context(), project.ID, 1, "original-linux-login"); err != nil {
				t.Fatal(err)
			}
			grant := Grant{Access: "fixture-access", Refresh: "fixture-refresh", Scopes: "read:user read:repository", Expires: time.Now().Add(time.Hour).Unix()}
			// Seed the genuine old schema without calling new context-aware insertion.
			if _, err = db.Exec(`INSERT INTO sessions(token,user_id,csrf,expires) VALUES(?,?,?,?)`, hash("keep-session"), 1, "keep-csrf", time.Now().Add(12*time.Hour).Unix()); err != nil {
				t.Fatal(err)
			}
			plain, err := json.Marshal(grant)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec(`INSERT INTO session_grants(session_token,ciphertext) VALUES(?,?)`, hash("keep-session"), cipher.seal(plain, grantBinding("keep-session", 1))); err != nil {
				t.Fatal(err)
			}
			if _, err = db.Exec(`INSERT INTO oauth(state,verifier,expires,return_path) VALUES(?,?,?,?)`, hash("old-pending"), "keep-verifier", time.Now().Add(time.Minute).Unix(), "/app/"); err != nil {
				t.Fatal(err)
			}
			var grantBefore, keyBefore []byte
			if err = db.QueryRow(`SELECT ciphertext FROM session_grants`).Scan(&grantBefore); err != nil {
				t.Fatal(err)
			}
			if err = db.QueryRow(`SELECT ciphertext FROM grant_key_check`).Scan(&keyBefore); err != nil {
				t.Fatal(err)
			}
			old.Close()

			// Missing/wrong keys must reject the genuine populated v3 DB before migration.
			for _, key := range [][]byte{nil, bytes.Repeat([]byte{7}, 32)} {
				var rejected *Store
				if key == nil {
					rejected, err = Open(path)
				} else {
					rejected, err = OpenEncrypted(path, key)
				}
				if rejected != nil {
					rejected.Close()
				}
				if !errors.Is(err, ErrGrantKey) {
					t.Fatal("incorrect key accepted", err)
				}
				probe, err := sql.Open("sqlite", path)
				if err != nil {
					t.Fatal(err)
				}
				var version, columns int
				err = probe.QueryRow(`SELECT version FROM schema_version`).Scan(&version)
				if err == nil {
					err = probe.QueryRow(`SELECT count(*) FROM pragma_table_info('oauth') WHERE name='context_id'`).Scan(&columns)
				}
				probe.Close()
				if err != nil || version != oldVersion || columns != 0 {
					t.Fatal("key failure migrated data", version, columns, err)
				}
			}

			s, err := OpenEncrypted(path, key)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			var version int
			if err = s.db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil || version != len(migrations) {
				t.Fatal(version, err)
			}
			var grantAfter, keyAfter []byte
			if err = s.db.QueryRow(`SELECT ciphertext FROM session_grants`).Scan(&grantAfter); err != nil {
				t.Fatal(err)
			}
			if err = s.db.QueryRow(`SELECT ciphertext FROM grant_key_check`).Scan(&keyAfter); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(grantBefore, grantAfter) || !bytes.Equal(keyBefore, keyAfter) {
				t.Fatal("migration rewrote encrypted bytes")
			}
			gotGrant, err := s.Grant(t.Context(), "keep-session", 1)
			if err != nil || gotGrant != grant {
				t.Fatal("grant changed", err)
			}
			session, err := s.Session(t.Context(), "keep-session")
			if err != nil || session.CSRF != "keep-csrf" || session.User != (User{ID: 1, Login: "alice", Name: "Keep profile"}) {
				t.Fatal(session, err)
			}
			keys, err := s.Keys(t.Context(), 1)
			if err != nil || len(keys) != 1 || keys[0].Public != "public-fixture" || keys[0].Fingerprint != "fingerprint-fixture" {
				t.Fatal(keys, err)
			}
			gotProject, err := s.Project(t.Context(), project.ID)
			project.IP, project.Ready = "10.89.0.2", true
			if err != nil || gotProject != project {
				t.Fatal(gotProject, err)
			}
			login, err := s.MemberLogin(t.Context(), project.ID, 1)
			if err != nil || login != "original-linux-login" {
				t.Fatal(login, err)
			}
			pending, err := s.ConsumeOAuth(t.Context(), "old-pending", "")
			if !errors.Is(err, sql.ErrNoRows) {
				t.Fatal(pending, err)
			}
			if session.ContextID == "" {
				t.Fatal("migration did not bind existing session")
			}
		})
	}
}
