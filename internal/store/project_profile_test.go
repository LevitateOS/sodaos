package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/projectos"
)

func TestV7MigrationKeepsLegacyUnknownAndImmutableCreation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v7.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile("testdata/v7.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(schema)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO users VALUES(1,'alice','Original'); INSERT INTO projects VALUES('legacy','Old',7,1,'alice/Old','10.0.0.2',1); INSERT INTO memberships VALUES('legacy',1,'original-login'); INSERT INTO oauth(state,verifier,expires,settings_return) VALUES('pending','v',9999999999,'runners')`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p, err := s.Project(t.Context(), "legacy")
	if err != nil || p.Profile != nil || !p.Ready || p.IP != "10.0.0.2" {
		t.Fatal(p, err)
	}
	if login, err := s.MemberLogin(t.Context(), "legacy", 1); err != nil || login != "original-login" {
		t.Fatal(login, err)
	}
	var kind string
	var target bool
	if err := s.db.QueryRow(`SELECT settings_return,repository_settings_return FROM oauth WHERE state='pending'`).Scan(&kind, &target); err != nil || kind != "runners" || target {
		t.Fatal("legacy login changed", err)
	}
	profile := projectos.Profile{ID: projectos.RockyHeadless, Distribution: "rocky", Version: "10.2", Interface: "headless", Architecture: "amd64", Image: "sha256:" + strings.Repeat("a", 64), Revision: strings.Repeat("b", 40)}
	p = Project{ID: "new", Name: "New", RepositoryID: 8, OwnerID: 1, Repository: "alice/New", Profile: &profile}
	if err := s.CreateProject(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkReady(t.Context(), "new", "10.0.0.3"); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.ProjectByRepository(t.Context(), 8)
	if err != nil || loaded.Profile == nil || *loaded.Profile != profile || !loaded.Ready {
		t.Fatal(loaded, err)
	}
	if _, err := s.db.Exec(`UPDATE projects SET creation_profile=NULL WHERE id='new'`); err == nil {
		t.Fatal("creation identity was mutable")
	}
	for _, bad := range []OAuthLogin{{RepositorySettingsReturn: true}, {RepositorySettingsReturn: true, RepositoryID: 7, SpacesReturn: true}, {RepositorySettingsReturn: true, RepositoryID: 7, SettingsReturn: "runners"}} {
		if s.BeginOAuth(t.Context(), "bad", bad, "", "") == nil {
			t.Fatal("ambiguous return accepted")
		}
	}
	if err := s.BeginOAuth(t.Context(), "new-settings", OAuthLogin{RepositoryID: 9223372036854775807, ExpectedUserID: 1, RepositorySettingsReturn: true, Verifier: "v"}, "", ""); err != nil {
		t.Fatal(err)
	}
	attempt, err := s.ConsumeOAuth(t.Context(), "new-settings", "")
	if err != nil || !attempt.RepositorySettingsReturn || attempt.RepositoryID != 9223372036854775807 {
		t.Fatal(attempt, err)
	}
	if _, err := s.ConsumeOAuth(t.Context(), "new-settings", ""); err == nil {
		t.Fatal("return replayed")
	}
	if err := s.EndLoginContext(t.Context(), attempt.contextID); err != nil {
		t.Fatal(err)
	}
	if s.FinishOAuth(t.Context(), attempt, User{ID: 1, Login: "alice"}, "new", "csrf", Grant{}) == nil {
		t.Fatal("return bypassed logout")
	}
}
