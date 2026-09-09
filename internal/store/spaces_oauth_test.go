package store

import (
	"bytes"
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestSpacesOAuthMigrationPreservesPopulatedV5(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v5.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations[:5] {
		if _, err = db.Exec(migration); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(`INSERT INTO schema_version VALUES(5); INSERT INTO users VALUES(1,'alice','Profile'); INSERT INTO projects VALUES('project','Project',7,1,'alice/project','10.0.0.2',1); INSERT INTO memberships VALUES('project',1,'original-alice'); INSERT INTO keys VALUES(1,1,'public','fingerprint');`); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{1}, 32)
	cipher, err := newGrantCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	old := &Store{db: db, grants: cipher}
	ctx := context.Background()
	if err = old.initializeGrantKey(ctx); err != nil {
		t.Fatal(err)
	}
	if err = old.CreateGrantedSession(ctx, "session", 1, "csrf", Grant{Access: "synthetic-access", Refresh: "synthetic-refresh", Scopes: "read:user read:repository", Expires: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO oauth(state,verifier,expires,context_id) VALUES(?,?,?,?)`, hash("pending"), "verifier", time.Now().Add(time.Minute).Unix(), hash("session")); err != nil {
		t.Fatal(err)
	}
	var ciphertext []byte
	if err = db.QueryRow(`SELECT ciphertext FROM session_grants`).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if wrong, err := OpenEncrypted(path, bytes.Repeat([]byte{2}, 32)); err == nil {
		wrong.Close()
		t.Fatal("wrong key accepted")
	}
	probe, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	_ = probe.QueryRow(`SELECT version FROM schema_version`).Scan(&version)
	probe.Close()
	if version != 5 {
		t.Fatal("wrong key migrated database")
	}
	current, err := OpenEncrypted(path, key)
	if err != nil {
		t.Fatal(err)
	}
	defer current.Close()
	var after []byte
	var intent bool
	if err = current.db.QueryRow(`SELECT ciphertext FROM session_grants`).Scan(&after); err != nil || !bytes.Equal(after, ciphertext) {
		t.Fatal("grant bytes changed")
	}
	if err = current.db.QueryRow(`SELECT spaces_return FROM oauth`).Scan(&intent); err != nil || intent {
		t.Fatal("old OAuth gained intent")
	}
	if p, err := current.Project(ctx, "project"); err != nil || !p.Ready || p.IP != "10.0.0.2" {
		t.Fatal("project changed")
	}
	if login, err := current.MemberLogin(ctx, "project", 1); err != nil || login != "original-alice" {
		t.Fatal("account changed")
	}
	if keys, err := current.Keys(ctx, 1); err != nil || len(keys) != 1 {
		t.Fatal("keys changed")
	}
	if v, err := current.Session(ctx, "session"); err != nil || v.CSRF != "csrf" || v.User.Name != "Profile" {
		t.Fatal("session changed")
	}
	if err = current.BeginOAuth(ctx, "spaces", OAuthLogin{Verifier: "verifier", SpacesReturn: true}, "session", ""); err != nil {
		t.Fatal(err)
	}
	attempt, err := current.ConsumeOAuth(ctx, "spaces", "session")
	if err != nil || !attempt.SpacesReturn {
		t.Fatal("intent lost")
	}
	if err = current.EndLoginContext(ctx, attempt.contextID); err != nil {
		t.Fatal(err)
	}
	if err = current.FinishOAuth(ctx, attempt, User{ID: 1, Login: "alice"}, "replacement", "csrf", Grant{}); err == nil {
		t.Fatal("Spaces bypassed logout")
	}
	if err = current.BeginOAuth(ctx, "mixed", OAuthLogin{Verifier: "v", SpacesReturn: true, RepositoryID: 7}, "", ""); err == nil {
		t.Fatal("ambiguous intent")
	}
}
