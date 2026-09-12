package store

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tailnetV9Fixture(t *testing.T) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "v9.db")
	db, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(1)
	if _, e = db.Exec(`PRAGMA foreign_keys=ON`); e != nil {
		t.Fatal(e)
	}
	for _, migration := range migrations[:9] {
		if _, e = db.Exec(migration); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = db.Exec(`INSERT INTO schema_version VALUES(9)`); e != nil {
		t.Fatal(e)
	}
	key := bytes.Repeat([]byte{23}, 32)
	cipher, e := newGrantCipher(key)
	if e != nil {
		t.Fatal(e)
	}
	s := &Store{db: db, grants: cipher}
	if e = s.initializeGrantKey(t.Context()); e != nil {
		t.Fatal(e)
	}
	if e = s.UpsertUser(t.Context(), User{ID: 1, Login: "alice", Name: "Preserved"}); e != nil {
		t.Fatal(e)
	}
	if e = s.CreateProject(t.Context(), Project{ID: "p-original", Name: "Keep", RepositoryID: 7, OwnerID: 1, Repository: "alice/original"}); e != nil {
		t.Fatal(e)
	}
	if e = s.Join(t.Context(), "p-original", 1, "original-linux-login"); e != nil {
		t.Fatal(e)
	}
	g := Grant{Access: "synthetic-access", Refresh: "synthetic-refresh", Scopes: "read:user read:repository read:organization", Expires: time.Now().Add(time.Hour).Unix()}
	if e = s.CreateGrantedSession(t.Context(), "existing", 1, "keep-csrf", g); e != nil {
		t.Fatal(e)
	}
	for _, entry := range []struct {
		state, session string
		login          OAuthLogin
	}{
		{"runners", "existing", OAuthLogin{SettingsReturn: "runners", Verifier: "runner-verifier", ExpectedUserID: 1}},
		{"spaces", "", OAuthLogin{SpacesReturn: true, Verifier: "spaces-verifier"}},
		{"repository", "", OAuthLogin{RepositorySettingsReturn: true, RepositoryID: 7, Verifier: "repo-verifier"}},
	} {
		if e = s.BeginOAuth(t.Context(), entry.state, entry.login, entry.session, ""); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = db.Exec(`INSERT INTO oauth(state,verifier,expires,settings_return) VALUES('not-yet','v',9999999999,'tailnet')`); e == nil {
		t.Fatal("fixture was not v9")
	}
	s.Close()
	if e = os.Chmod(path, 0600); e != nil {
		t.Fatal(e)
	}
	return path, key
}
func TestTailnetV10PreservesV9ContextsAndEncryptedGrants(t *testing.T) {
	path, key := tailnetV9Fixture(t)
	db, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	var before []byte
	if e = db.QueryRow(`SELECT ciphertext FROM session_grants WHERE session_token=?`, hash("existing")).Scan(&before); e != nil {
		t.Fatal(e)
	}
	db.Close()
	// Wrong-key refusal precedes migration, not just grant reads after upgrade.
	if s, e := OpenEncrypted(path, bytes.Repeat([]byte{99}, 32)); e == nil {
		s.Close()
		t.Fatal("wrong key accepted")
	}
	db, e = sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	var version int
	if e = db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); e != nil || version != 9 {
		t.Fatal("wrong key migrated state", version, e)
	}
	db.Close()
	s, e := OpenEncrypted(path, key)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); e != nil || version != 10 {
		t.Fatal(version, e)
	}
	var pending int
	if e = s.db.QueryRow(`SELECT count(*) FROM oauth`).Scan(&pending); e != nil || pending != 3 {
		t.Fatal("contract probe was committed", pending, e)
	}
	var after []byte
	if e = s.db.QueryRow(`SELECT ciphertext FROM session_grants WHERE session_token=?`, hash("existing")).Scan(&after); e != nil || !bytes.Equal(before, after) {
		t.Fatal("encrypted grant rewritten", e)
	}
	v, e := s.Session(t.Context(), "existing")
	if e != nil || v.CSRF != "keep-csrf" || v.User.Name != "Preserved" {
		t.Fatal(v, e)
	}
	if login, e := s.MemberLogin(t.Context(), "p-original", 1); e != nil || login != "original-linux-login" {
		t.Fatal(login, e)
	}
	p, e := s.Project(t.Context(), "p-original")
	if e != nil || p.Repository != "alice/original" || p.Ready {
		t.Fatal(p, e)
	}
	for _, entry := range []struct{ state, session string }{{"runners", "existing"}, {"spaces", ""}, {"repository", ""}} {
		a, e := s.ConsumeOAuth(t.Context(), entry.state, entry.session)
		if e != nil {
			t.Fatal(entry.state, e)
		}
		if entry.state == "runners" && (a.SettingsReturn != "runners" || a.ExpectedUserID != 1) {
			t.Fatal("changed return")
		}
		if entry.state == "spaces" && !a.SpacesReturn {
			t.Fatal("lost Spaces intent")
		}
		if entry.state == "repository" && (!a.RepositorySettingsReturn || a.RepositoryID != 7) {
			t.Fatal("lost repository intent")
		}
		if _, e = s.ConsumeOAuth(t.Context(), entry.state, entry.session); e == nil {
			t.Fatal("replayed migrated OAuth")
		}
	}
	if e = s.BeginOAuth(t.Context(), "tailnet", OAuthLogin{SettingsReturn: "tailnet", ExpectedUserID: 1, Verifier: "tailnet-verifier"}, "existing", ""); e != nil {
		t.Fatal(e)
	}
	a, e := s.ConsumeOAuth(t.Context(), "tailnet", "existing")
	if e != nil || a.SettingsReturn != "tailnet" {
		t.Fatal(a, e)
	}
	if e = s.EndLoginContext(t.Context(), v.ContextID); e != nil {
		t.Fatal(e)
	}
	if s.FinishOAuth(t.Context(), a, User{ID: 1, Login: "alice"}, "late", "csrf", Grant{}) == nil {
		t.Fatal("Tailnet return bypassed logout")
	}
}
func TestTailnetV10RejectsMixedAndMalformedState(t *testing.T) {
	path, key := tailnetV9Fixture(t)
	s, e := OpenEncrypted(path, key)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for _, bad := range []OAuthLogin{{SettingsReturn: "tailnet", SpacesReturn: true}, {SettingsReturn: "tailnet", RepositoryID: 7}, {SettingsReturn: "tailnet", RepositorySettingsReturn: true, RepositoryID: 7}, {SettingsReturn: "other"}} {
		if s.BeginOAuth(t.Context(), "bad", bad, "", "") == nil {
			t.Fatal("mixed destination accepted")
		}
	}
	for _, values := range []string{`'tailnet',1,0,0`, `'tailnet',0,7,0`, `'tailnet',0,7,1`, `'other',0,0,0`} {
		if _, e = s.db.Exec(`INSERT INTO oauth(state,verifier,expires,settings_return,spaces_return,repository_id,repository_settings_return) VALUES('bad','v',9999999999,` + values + `)`); e == nil {
			t.Fatal("database accepted mixed destination")
		}
	}
}
func TestTailnetV10RejectsStaleVersionMarker(t *testing.T) {
	path, key := tailnetV9Fixture(t)
	db, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`UPDATE schema_version SET version=10`); e != nil {
		t.Fatal(e)
	}
	db.Close()
	if s, e := OpenEncrypted(path, key); e == nil {
		s.Close()
		t.Fatal("v9 constraint accepted under v10 marker")
	}
	db, e = sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var count int
	if e = db.QueryRow(`SELECT count(*) FROM oauth`).Scan(&count); e != nil || count != 3 {
		t.Fatal("probe changed pending rows", count, e)
	}
}

func TestTailnetV10IncompleteUpgradeRollsBack(t *testing.T) {
	path, key := tailnetV9Fixture(t)
	db, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`ALTER TABLE oauth DROP COLUMN expected_user_id`); e != nil {
		t.Fatal(e)
	}
	db.Close()
	if s, e := OpenEncrypted(path, key); e == nil {
		s.Close()
		t.Fatal("incomplete schema accepted")
	}
	db, e = sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var version, count int
	if e = db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); e != nil || version != 9 {
		t.Fatal(version, e)
	}
	if e = db.QueryRow(`SELECT count(*) FROM oauth WHERE settings_return='runners'`).Scan(&count); e != nil || count != 1 {
		t.Fatal("pending row lost", count, e)
	}
	if e = db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='oauth_tailnet'`).Scan(&count); e != nil || count != 0 {
		t.Fatal("partial migration committed", count, e)
	}
}
