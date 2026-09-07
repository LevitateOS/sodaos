package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPersistenceAndMembership(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "soda.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.UpsertUser(ctx, User{ID: 1, Login: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err = s.CreateProject(ctx, Project{ID: "p123", Name: "Demo", RepositoryID: 42, OwnerID: 1, Repository: "alice/demo"}); err != nil {
		t.Fatal(err)
	}
	p, err := s.Project(ctx, "p123")
	if err != nil || p.Ready {
		t.Fatalf("project: %+v %v", p, err)
	}
	if err = s.Join(ctx, "p123", 1, "alice"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if login, err := s.MemberLogin(ctx, "p123", 1); err != nil || login != "alice" {
		t.Fatalf("%q %v", login, err)
	}
}
func TestOAuthSingleUse(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "soda.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err = s.BeginOAuth(ctx, "state", "verifier"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConsumeOAuth(ctx, "state"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ConsumeOAuth(ctx, "state"); err == nil {
		t.Fatal("state reused")
	}
}
