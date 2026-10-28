package store

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func testLifecycleAction() factory.LifecycleAction {
	return factory.LifecycleAction{ID: "act-store-1", Event: factory.LifecycleEventIssueCreated, Action: factory.LifecycleActionIssueLabelAssignee, ActorID: 7, RepositoryID: 9, IssueID: 42, TargetID: 420, LabelID: 3, AuthRevision: "rev-1"}
}

func TestRecordActionReplaysIdentical(t *testing.T) {
	s, _ := postgresFixture(t)
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

func TestUnsettledLifecycleActions(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	issue := testLifecycleAction()
	pending := issue
	pending.ID = "act-unsettled-pending"
	if _, err := s.RecordAction(ctx, pending); err != nil {
		t.Fatalf("pending record: %v", err)
	}
	uncertain := issue.RecordActionAttempt("op-u1")
	uncertain.ID = "act-unsettled-uncertain"
	if _, err := s.RecordAction(ctx, uncertain); err != nil {
		t.Fatalf("uncertain record: %v", err)
	}
	settled := issue.RecordActionAttempt("op-s1").RecordActionResult(factory.LifecycleResultDelivered)
	settled.ID = "act-unsettled-settled"
	if _, err := s.RecordAction(ctx, settled); err != nil {
		t.Fatalf("settled record: %v", err)
	}
	retryable := issue.RecordActionAttempt("op-r1").RecordActionResult(factory.LifecycleResultFailed)
	retryable.ID = "act-unsettled-retryable"
	if _, err := s.RecordAction(ctx, retryable); err != nil {
		t.Fatalf("retryable record: %v", err)
	}
	exhausted := issue.RecordActionAttempt("op-x1").RecordActionResult(factory.LifecycleResultFailed)
	exhausted = exhausted.RecordActionAttempt("op-x2").RecordActionResult(factory.LifecycleResultFailed)
	exhausted.ID = "act-unsettled-exhausted"
	if _, err := s.RecordAction(ctx, exhausted); err != nil {
		t.Fatalf("exhausted record: %v", err)
	}
	revoked := issue
	revoked.ID = "act-unsettled-revoked"
	if _, err := s.RecordAction(ctx, revoked); err != nil {
		t.Fatalf("revoked record: %v", err)
	}
	if _, err := s.RevokeAction(ctx, revoked.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	got, err := s.UnsettledLifecycleActions(ctx, 64)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	ids := map[string]bool{}
	for _, a := range got {
		ids[a.ID] = true
	}
	if !ids[pending.ID] || !ids[uncertain.ID] || !ids[retryable.ID] || ids[settled.ID] || ids[revoked.ID] || ids[exhausted.ID] {
		t.Fatalf("unsettled: %v", ids)
	}
	one, err := s.UnsettledLifecycleActions(ctx, 1)
	if err != nil || len(one) != 1 {
		t.Fatalf("limit: %v %v", one, err)
	}
}

func TestUpdateActionFencesSameAttemptResults(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	a := testLifecycleAction()
	a.ID = "act-fence-1"
	if _, err := s.RecordAction(ctx, a); err != nil {
		t.Fatalf("record: %v", err)
	}
	claimed, err := s.UpdateAction(ctx, a.RecordActionAttempt("op-f1"), 0)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if _, err := s.UpdateAction(ctx, claimed.RecordActionResult(factory.LifecycleResultDelivered), 1); err != nil {
		t.Fatalf("settle: %v", err)
	}
	// A stale same-attempt write regressing delivered to uncertain is
	// refused and the terminal result stands.
	stale := claimed.RecordActionResult(factory.LifecycleResultUncertain)
	if _, err := s.UpdateAction(ctx, stale, 1); !errors.Is(err, ErrLifecycleActionStale) {
		t.Fatalf("regression err=%v", err)
	}
	flip := claimed.RecordActionResult(factory.LifecycleResultFailed)
	if _, err := s.UpdateAction(ctx, flip, 1); !errors.Is(err, ErrLifecycleActionStale) {
		t.Fatalf("flip err=%v", err)
	}
	got, err := s.LifecycleAction(ctx, a.ID)
	if err != nil || got.Results["label"] != factory.LifecycleResultDelivered {
		t.Fatalf("terminal lost: %+v %v", got, err)
	}
	// Advancing the attempt count may still reset results.
	retry, err := s.UpdateAction(ctx, got.RecordActionAttempt("op-f2"), 1)
	if err != nil || retry.Attempts != 2 || retry.Results["label"] != factory.LifecycleResultUncertain {
		t.Fatalf("retry claim: %+v %v", retry, err)
	}
}

func TestUpdateActionFencesAttempts(t *testing.T) {
	s, _ := postgresFixture(t)
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
	s, _ := postgresFixture(t)
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
	s, _ := postgresFixture(t)
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
