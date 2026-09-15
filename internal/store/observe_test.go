package store

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestReadSchemaVersionAndOpenObserve(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "soda.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.UpsertUser(ctx, User{ID: 1, Login: "soda-tester"}); err != nil {
		t.Fatal(err)
	}
	if err = s.CreateProject(ctx, Project{ID: "p000000000000000000000001", Name: "Demo", RepositoryID: 7, OwnerID: 1, Repository: "soda-tester/demo"}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}

	version, err := ReadSchemaVersion(ctx, path)
	if err != nil || version != SchemaVersion() {
		t.Fatalf("ReadSchemaVersion: %d %v", version, err)
	}
	observed, err := OpenObserve(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer observed.Close()
	if err = observed.IntegrityCheck(ctx); err != nil {
		t.Fatal(err)
	}
	user, err := observed.User(ctx, 1)
	if err != nil || user.Login != "soda-tester" {
		t.Fatalf("User: %+v %v", user, err)
	}
	project, err := observed.Project(ctx, "p000000000000000000000001")
	if err != nil || project.Repository != "soda-tester/demo" {
		t.Fatalf("Project: %+v %v", project, err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE schema_version SET version=?`, SchemaVersion()-1); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = OpenObserve(ctx, path); err == nil {
		t.Fatal("OpenObserve accepted a mismatched schema")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("OpenObserve refusal mutated the database")
	}
	old, err := ReadSchemaVersion(ctx, path)
	if err != nil || old != SchemaVersion()-1 {
		t.Fatalf("ReadSchemaVersion after downgrade marker: %d %v", old, err)
	}
}
