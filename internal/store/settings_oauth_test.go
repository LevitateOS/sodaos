package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestSettingsReturnPreservesV6AndRemainsTransactionBound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v6.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations[:6] {
		if _, err = db.Exec(migration); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(`INSERT INTO schema_version VALUES(6); INSERT INTO users VALUES(1,'alice','Original'); INSERT INTO projects VALUES('p','P',7,1,'alice/P','10.0.0.2',1); INSERT INTO memberships VALUES('p',1,'original-login'); INSERT INTO oauth(state,verifier,expires,spaces_return) VALUES('legacy','v',9999999999,1);`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var legacy string
	var spaces bool
	if err = s.db.QueryRow(`SELECT settings_return,spaces_return FROM oauth WHERE state='legacy'`).Scan(&legacy, &spaces); err != nil || legacy != "" || !spaces {
		t.Fatal("changed legacy intent", err)
	}
	ctx := context.Background()
	if login, err := s.MemberLogin(ctx, "p", 1); err != nil || login != "original-login" {
		t.Fatal("changed member")
	}
	for _, bad := range []OAuthLogin{{SettingsReturn: "https://evil.test"}, {SettingsReturn: "runners", SpacesReturn: true}, {SettingsReturn: "runners", RepositoryID: 7}} {
		if s.BeginOAuth(ctx, "bad", bad, "", "") == nil {
			t.Fatal("ambiguous destination accepted")
		}
	}
	if err = s.BeginOAuth(ctx, "settings", OAuthLogin{SettingsReturn: "runners", ExpectedUserID: 1, Verifier: "verifier"}, "", ""); err != nil {
		t.Fatal(err)
	}
	attempt, err := s.ConsumeOAuth(ctx, "settings", "")
	if err != nil || attempt.SettingsReturn != "runners" || attempt.ExpectedUserID != 1 {
		t.Fatal(attempt, err)
	}
	if _, err = s.ConsumeOAuth(ctx, "settings", ""); err == nil {
		t.Fatal("replayed OAuth")
	}
	if err = s.EndLoginContext(ctx, attempt.contextID); err != nil {
		t.Fatal(err)
	}
	if s.FinishOAuth(ctx, attempt, User{ID: 1, Login: "alice"}, "new", "csrf", Grant{}) == nil {
		t.Fatal("settings bypassed logout")
	}
}
