package store

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
)

func factoryFixture(t *testing.T) (*Store, time.Time) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "factory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
}

func factoryRun(now time.Time) factory.Run {
	return factory.Run{ID: factory.NewID(), ProjectID: "p123456789012345678901234", Role: "coder", InputSHA: strings.Repeat("a", 40), Started: now, Deadline: now.Add(time.Hour), Image: "sha256:" + strings.Repeat("d", 64), Harness: "codex-0.157.1", Model: "test-model"}
}

func TestFactoryRunIdentityOnlyAdvances(t *testing.T) {
	s, now := factoryFixture(t)
	ctx := context.Background()
	r := factoryRun(now)
	if err := s.RecordFactoryRun(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordFactoryRun(ctx, r); err == nil {
		t.Fatal("consumed run identity reused")
	}
	r.IdentityLeaseID, r.IdentityGeneration, r.CredentialDelegated = "lease", 3, true
	id := r.ID
	r.IdentityBinding = &identity.Binding{Kind: identity.Factory, ID: id, Generation: 3}
	if err := s.SaveFactoryRun(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.IdentityGeneration = 4
	if err := s.SaveFactoryRun(ctx, r); err == nil {
		t.Fatal("reserved credential generation changed")
	}
	r.IdentityGeneration = 3
	r.Outcome, r.Summary, r.CredentialReturned, r.Reconciled = factory.Cancelled, "stopped by operator", true, true
	if err := s.SaveFactoryRun(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.Outcome = factory.Failed
	if err := s.SaveFactoryRun(ctx, r); err == nil {
		t.Fatal("terminal outcome changed")
	}
	r.Outcome = factory.Cancelled
	r.Reconciled = false
	if err := s.SaveFactoryRun(ctx, r); err == nil {
		t.Fatal("settled run reopened")
	}
}

func TestFactoryRunsListNewestFirst(t *testing.T) {
	s, now := factoryFixture(t)
	ctx := context.Background()
	var ids []string
	for range 3 {
		r := factoryRun(now)
		ids = append(ids, r.ID)
		if err := s.RecordFactoryRun(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.FactoryRuns(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].ID != ids[2] || got[2].ID != ids[0] {
		t.Fatal("recorded runs out of order")
	}
	got, err = s.FactoryRuns(ctx, 2)
	if err != nil || len(got) != 2 {
		t.Fatal("run list bound ignored")
	}
}

func TestFactoryCommandIdentityConflictsOnChangedPayload(t *testing.T) {
	s, now := factoryFixture(t)
	ctx := context.Background()
	run := factory.NewID()
	c := factory.Command{ID: factory.NewID(), Type: factory.CommandStop, Target: run, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandStop, run)}
	stored, created, err := s.RecordFactoryCommand(ctx, c, now)
	if err != nil || !created || stored.Finished != "" {
		t.Fatal(stored, created, err)
	}
	replay, created, err := s.RecordFactoryCommand(ctx, c, now)
	if err != nil || created || replay.ID != c.ID {
		t.Fatal(replay, created, err)
	}
	c.Target = factory.NewID()
	if _, _, err = s.RecordFactoryCommand(ctx, c, now); err == nil {
		t.Fatal("changed command payload reused its identity")
	}
	if err = s.FinishFactoryCommand(ctx, stored.ID, `{"stopped":true}`, now); err != nil {
		t.Fatal(err)
	}
	finished, err := s.FactoryCommand(ctx, stored.ID)
	if err != nil || finished.Outcome != `{"stopped":true}` || finished.Finished == "" {
		t.Fatal(finished, err)
	}
	if err = s.FinishFactoryCommand(ctx, stored.ID, `{"stopped":false}`, now); !errors.Is(err, ErrNotFound) {
		t.Fatal("durable outcome rewritten")
	}
}
