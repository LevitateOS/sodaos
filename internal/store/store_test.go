package store

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func TestMembersListingIsBounded(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "soda.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.UpsertUser(ctx, User{ID: 1, Login: "member-000"}); err != nil {
		t.Fatal(err)
	}
	if err = s.CreateProject(ctx, Project{ID: "p123", Name: "Demo", RepositoryID: 42, OwnerID: 1, Repository: "alice/demo"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 135; i++ {
		login := fmt.Sprintf("member-%03d", i)
		if err = s.UpsertUser(ctx, User{ID: int64(i + 1), Login: login}); err != nil {
			t.Fatal(err)
		}
		if err = s.Join(ctx, "p123", int64(i+1), login); err != nil {
			t.Fatal(err)
		}
	}
	members, err := s.Members(ctx, "p123")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 129 {
		t.Fatalf("members listing returned %d rows, want the bounded 129", len(members))
	}
}

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
	if err = s.BeginOAuth(ctx, "state", OAuthLogin{Verifier: "verifier", RepositoryID: 42, ExpectedUserID: 1}, "", ""); err != nil {
		t.Fatal(err)
	}
	if login, err := s.ConsumeOAuth(ctx, "state", ""); err != nil || login.OAuthLogin != (OAuthLogin{Verifier: "verifier", RepositoryID: 42, ExpectedUserID: 1}) {
		t.Fatal(login, err)
	}
	if _, err = s.ConsumeOAuth(ctx, "state", ""); err == nil {
		t.Fatal("state reused")
	}
}
