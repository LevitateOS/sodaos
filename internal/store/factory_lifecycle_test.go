package store

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func takeoverStore(t *testing.T) *Store {
	t.Helper()
	s, _ := postgresFixture(t, nil)
	if err := s.UpsertUser(context.Background(), User{ID: 1, Login: "alice", Name: "alice"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProject(context.Background(), Project{ID: "p765432109876543210987654", Name: "factory", RepositoryID: 7, OwnerID: 1, Repository: "alice/factory"}); err != nil {
		t.Fatal(err)
	}
	return s
}

func takeoverFixture() factory.TakeoverRecord {
	run := strings.Repeat("a", 32)
	return factory.TakeoverRecord{Run: run, Member: "alice", Project: "p765432109876543210987654", Dest: "/home/alice/factory-takeover/" + run, CommandID: factory.NewID(), Copied: "2026-10-01T00:00:00Z"}
}

func TestTakeoverRecordReplays(t *testing.T) {
	s := takeoverStore(t)
	ctx := context.Background()
	want := takeoverFixture()
	stored, err := s.RecordTakeover(ctx, want)
	if err != nil || stored != want {
		t.Fatalf("record: %+v %v", stored, err)
	}
	replayed, err := s.RecordTakeover(ctx, want)
	if err != nil || replayed != want {
		t.Fatalf("replay: %+v %v", replayed, err)
	}
	got, err := s.Takeover(ctx, want.Run, want.Member)
	if err != nil || got != want {
		t.Fatalf("read: %+v %v", got, err)
	}
	other, err := s.RecordTakeover(ctx, factory.TakeoverRecord{Run: want.Run, Member: "bob", Project: want.Project, Dest: "/home/bob/factory-takeover/" + want.Run, CommandID: factory.NewID(), Copied: want.Copied})
	if err != nil || other.Member != "bob" {
		t.Fatalf("second member: %+v %v", other, err)
	}
	moved := want
	moved.Dest = "/home/alice/factory-takeover/" + strings.Repeat("b", 32)
	if _, err = s.RecordTakeover(ctx, moved); !errors.Is(err, ErrTakeoverConflict) {
		t.Fatalf("changed takeover: %v", err)
	}
}

func TestTakeoverUnknownIsNotFound(t *testing.T) {
	s := takeoverStore(t)
	if _, err := s.Takeover(context.Background(), strings.Repeat("c", 32), "alice"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown takeover: %v", err)
	}
}

func TestTakeoverRecordValidates(t *testing.T) {
	s := takeoverStore(t)
	bad := takeoverFixture()
	bad.Dest = "/tmp/takeover"
	if _, err := s.RecordTakeover(context.Background(), bad); err == nil {
		t.Fatal("takeover outside the member home stored")
	}
}
