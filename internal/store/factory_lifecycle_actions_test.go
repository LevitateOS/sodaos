package store

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func testLifecycleAction() factory.LifecycleAction {
	return factory.LifecycleAction{ID: "act-store-1", Event: factory.LifecycleEventIssueCreated, Action: factory.LifecycleActionIssueLabelAssignee, ActorID: 7, RepositoryID: 9, IssueID: 42, LabelID: 3, AuthRevision: "rev-1"}
}

func TestRecordActionReplaysIdentical(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	a := testLifecycleAction()
	if _, err := s.RecordAction(ctx, a); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	replayed, err := s.RecordAction(ctx, a)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if replayed.ID != a.ID {
		t.Fatalf("replay returned %+v", replayed)
	}
	changed := a
	changed.LabelID = 99
	if _, err := s.RecordAction(ctx, changed); !errors.Is(err, ErrLifecycleActionConflict) {
		t.Fatalf("changed replay err=%v", err)
	}
	got, err := s.LifecycleAction(ctx, a.ID)
	if err != nil || got.LabelID != 3 {
		t.Fatalf("get returned %+v err=%v", got, err)
	}
}

func TestUpdateActionFencesAttempts(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	a := testLifecycleAction()
	a.ID = "act-store-2"
	if _, err := s.RecordAction(ctx, a); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	next := a.RecordAttempt("label", "op-1")
	if _, err := s.UpdateAction(ctx, next, 0); err != nil {
		t.Fatalf("first update failed: %v", err)
	}
	stale := a.RecordAttempt("label", "op-other")
	if _, err := s.UpdateAction(ctx, stale, 0); !errors.Is(err, ErrLifecycleActionStale) {
		t.Fatalf("stale update err=%v", err)
	}
	settled := next.RecordResult("label", factory.LifecycleResultFailed)
	if _, err := s.UpdateAction(ctx, settled, 1); err != nil {
		t.Fatalf("result update failed: %v", err)
	}
	revoked, err := s.RevokeAction(ctx, a.ID)
	if err != nil || !revoked.Revoked {
		t.Fatalf("revoke failed: %+v %v", revoked, err)
	}
	if _, err := s.UpdateAction(ctx, settled, 2); !errors.Is(err, ErrLifecycleActionStale) {
		t.Fatalf("post-revoke update err=%v", err)
	}
}

func TestListActionsBounded(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	for _, id := range []string{"act-list-1", "act-list-2"} {
		a := testLifecycleAction()
		a.ID = id
		if _, err := s.RecordAction(ctx, a); err != nil {
			t.Fatalf("record %s failed: %v", id, err)
		}
	}
	got, err := s.ListActions(ctx, 9, 42, factory.LifecycleEventIssueCreated, 64)
	if err != nil || len(got) != 2 {
		t.Fatalf("list returned %d err=%v", len(got), err)
	}
	got, err = s.ListActions(ctx, 9, 42, factory.LifecycleEventPRCreated, 64)
	if err != nil || len(got) != 0 {
		t.Fatalf("filtered list returned %d err=%v", len(got), err)
	}
}

func TestSaveConfigGatesOnCapability(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	c := factory.LifecycleConfig{Event: factory.LifecycleEventIssueCreated, Revision: 0, Enabled: true, ActorID: 7, RepositoryID: 9, LabelID: 3}
	if _, err := s.SaveConfig(ctx, c); !errors.Is(err, ErrCapabilityUnproved) {
		t.Fatalf("unproved enable err=%v", err)
	}
	disabled := c
	disabled.Enabled = false
	disabled.LabelID = 0
	if _, err := s.SaveConfig(ctx, disabled); err != nil {
		t.Fatalf("disabled save failed: %v", err)
	}
	if replaced, err := s.RegisterCapability(ctx, c.Event, "cap-rev-1", "inst-1"); err != nil || !replaced {
		t.Fatalf("capability register failed: %v %v", replaced, err)
	}
	if replaced, err := s.RegisterCapability(ctx, c.Event, "cap-rev-1", "inst-1"); err != nil || replaced {
		t.Fatalf("identical replay replaced: %v %v", replaced, err)
	}
	if replaced, err := s.RegisterCapability(ctx, c.Event, "cap-rev-2", "inst-1"); err != nil || !replaced {
		t.Fatalf("generation supersede failed: %v %v", replaced, err)
	}
	proved, err := s.CapabilityProved(ctx, c.Event)
	if err != nil || !proved {
		t.Fatalf("capability proved=%v err=%v", proved, err)
	}
	c.Revision = 1
	if _, err := s.SaveConfig(ctx, c); err != nil {
		t.Fatalf("proved enable failed: %v", err)
	}
	c.Revision = 1
	if _, err := s.SaveConfig(ctx, c); !errors.Is(err, ErrLifecycleActionStale) {
		t.Fatalf("stale revision err=%v", err)
	}
	got, err := s.LifecycleConfig(ctx, c.Event)
	if err != nil || !got.Enabled || got.Revision != 1 {
		t.Fatalf("config get %+v err=%v", got, err)
	}
	if err := s.DeleteCapability(ctx, c.Event); err != nil {
		t.Fatalf("capability delete failed: %v", err)
	}
	if ok, err := s.CapabilityProved(ctx, c.Event); err != nil || ok {
		t.Fatalf("deleted marker proved: %v %v", ok, err)
	}
	if _, err := s.RegisterCapability(ctx, factory.LifecycleEventPRCreated, "gen-1", "inst-9"); err != nil {
		t.Fatalf("foreign register failed: %v", err)
	}
	if err := s.DeleteForeignCapabilities(ctx, "inst-1"); err != nil {
		t.Fatalf("foreign delete failed: %v", err)
	}
	if ok, err := s.CapabilityProved(ctx, factory.LifecycleEventPRCreated); err != nil || ok {
		t.Fatalf("foreign marker retained: %v %v", ok, err)
	}
	if _, err := s.RegisterCapability(ctx, factory.LifecycleEventVerifiedMerge, "gen-1", "inst-1"); err != nil {
		t.Fatalf("clear-setup register failed: %v", err)
	}
	if err := s.ClearLifecycleCapabilities(ctx); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if ok, err := s.CapabilityProved(ctx, factory.LifecycleEventVerifiedMerge); err != nil || ok {
		t.Fatalf("cleared marker proved: %v %v", ok, err)
	}
}
