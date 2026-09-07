package store

import (
	"bytes"
	"database/sql"
	"errors"
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
	if err = s.BeginOAuth(t.Context(), "pending", login); err != nil {
		t.Fatal(err)
	}
	if err = s.BeginOAuth(t.Context(), "expired", login); err != nil {
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
	got, err := s.ConsumeOAuth(t.Context(), "pending")
	if err != nil || got != login {
		t.Fatal(got, err)
	}
	for _, state := range []string{"pending", "expired", "unknown"} {
		if _, err = s.ConsumeOAuth(t.Context(), state); !errors.Is(err, sql.ErrNoRows) {
			t.Fatal("state accepted", state, err)
		}
	}
	for _, invalid := range []OAuthLogin{{Verifier: "v", RepositoryID: -1}, {Verifier: "v", ExpectedUserID: -1}} {
		if err = s.BeginOAuth(t.Context(), "invalid", invalid); err == nil {
			t.Fatal("negative native context accepted")
		}
	}
}

func TestV3OAuthContextMigrationPreservesEncryptedProductState(t *testing.T) {
	// Use the exact first three append-only migrations. Do not create a v4 DB
	// and merely label it v3: the old schema must really lack the new columns.
	path, db := legacyDatabase(t)
	for _, migration := range migrations[1:3] {
		if _, err := db.Exec(migration); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`UPDATE schema_version SET version=3; PRAGMA foreign_keys=ON;`); err != nil {
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
	if err = old.CreateGrantedSession(t.Context(), "keep-session", 1, "keep-csrf", grant); err != nil {
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
			err = probe.QueryRow(`SELECT count(*) FROM pragma_table_info('oauth') WHERE name IN ('repository_id','expected_user_id')`).Scan(&columns)
		}
		probe.Close()
		if err != nil || version != 3 || columns != 0 {
			t.Fatal("key failure migrated data", version, columns, err)
		}
	}

	s, err := OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var version int
	if err = s.db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil || version != 4 {
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
	pending, err := s.ConsumeOAuth(t.Context(), "old-pending")
	if err != nil || pending != (OAuthLogin{Verifier: "keep-verifier"}) {
		t.Fatal(pending, err)
	}
}
