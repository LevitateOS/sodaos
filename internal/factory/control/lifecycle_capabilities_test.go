package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

type scriptHandshake struct {
	answer factory.LifecycleHandshake
	err    error
	calls  int
}

func (s *scriptHandshake) Handshake(context.Context) (factory.LifecycleHandshake, error) {
	s.calls++
	return s.answer, s.err
}

func lifecycleTestConfig(event string, revision int64, enabled bool) factory.LifecycleConfig {
	c := factory.LifecycleConfig{Event: event, Revision: revision, Enabled: enabled, ActorID: 7, RepositoryID: 9}
	switch event {
	case factory.LifecycleEventIssueCreated:
		c.LabelID = 3
	case factory.LifecycleEventPRCreated:
		c.ReviewerIDs = []int64{15}
	case factory.LifecycleEventVerifiedMerge:
		c.Note = "done"
	}
	return c
}

// TestReconcileLifecycleCapabilities proves production admission through
// the registrar: an empty database plus an authenticated handshake
// enables exactly its event, with no direct marker seeding. A failed or
// unattributable handshake closes new admission until a verified refresh
// recovers it; saved configuration and journal history survive.
func TestReconcileLifecycleCapabilities(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	ctx := context.Background()
	issue := factory.LifecycleEventKind(factory.LifecycleEventIssueCreated)

	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Installation: "inst-1", Generation: "gen-7", Kinds: []string{issue, "unknown.future.kind"}}}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if _, err := db.SaveConfig(ctx, lifecycleTestConfig(factory.LifecycleEventIssueCreated, 0, true)); err != nil {
		t.Fatalf("admitted enable blocked: %v", err)
	}
	if err := RequireCurrentCapability(ctx, db, factory.LifecycleEventIssueCreated); err != nil {
		t.Fatalf("admitted helper blocked: %v", err)
	}
	for _, event := range []string{factory.LifecycleEventPRCreated, factory.LifecycleEventVerifiedMerge} {
		if _, err := db.SaveConfig(ctx, lifecycleTestConfig(event, 0, true)); !errors.Is(err, store.ErrCapabilityUnproved) {
			t.Fatalf("%s enabled without proof: %v", event, err)
		}
		if err := RequireCurrentCapability(ctx, db, event); !errors.Is(err, store.ErrCapabilityUnproved) {
			t.Fatalf("%s helper admitted: %v", event, err)
		}
	}

	failed := &scriptHandshake{err: errors.New("handshake lost")}
	if err := ReconcileLifecycleCapabilities(ctx, db, failed); err == nil {
		t.Fatal("failed handshake accepted")
	}
	if ok, err := db.CapabilityProved(ctx, factory.LifecycleEventIssueCreated); err != nil || ok {
		t.Fatalf("stale proof usable after failure: %v %v", ok, err)
	}
	if _, err := db.SaveConfig(ctx, lifecycleTestConfig(factory.LifecycleEventIssueCreated, 1, true)); !errors.Is(err, store.ErrCapabilityUnproved) {
		t.Fatalf("new enable accepted after failure: %v", err)
	}
	if err := RequireCurrentCapability(ctx, db, factory.LifecycleEventIssueCreated); !errors.Is(err, store.ErrCapabilityUnproved) {
		t.Fatalf("helper admitted after failure: %v", err)
	}
	kept, err := db.LifecycleConfig(ctx, factory.LifecycleEventIssueCreated)
	if err != nil || !kept.Enabled || kept.Revision != 0 {
		t.Fatalf("saved config disturbed: %+v %v", kept, err)
	}

	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Generation: "gen-7", Kinds: []string{issue}}}); !errors.Is(err, factory.ErrLifecycleAction) {
		t.Fatalf("invalid handshake accepted: %v", err)
	}
	if ok, err := db.CapabilityProved(ctx, factory.LifecycleEventIssueCreated); err != nil || ok {
		t.Fatalf("stale proof usable after invalid answer: %v %v", ok, err)
	}

	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Installation: "inst-1", Generation: "gen-7", Kinds: []string{issue}}}); err != nil {
		t.Fatalf("recovery reconcile: %v", err)
	}
	if _, err := db.SaveConfig(ctx, lifecycleTestConfig(factory.LifecycleEventIssueCreated, 1, true)); err != nil {
		t.Fatalf("recovery enable blocked: %v", err)
	}
	if err := RequireCurrentCapability(ctx, db, factory.LifecycleEventIssueCreated); err != nil {
		t.Fatalf("recovery helper blocked: %v", err)
	}
	if ok, err := db.CapabilityProved(ctx, factory.LifecycleEventPRCreated); err != nil || ok {
		t.Fatalf("unsupported event admitted on recovery: %v %v", ok, err)
	}

	a, err := db.RecordAction(ctx, factory.LifecycleAction{ID: "act-stale-1", Event: factory.LifecycleEventIssueCreated, Action: factory.LifecycleActionIssueLabelAssignee, ActorID: 7, RepositoryID: 9, IssueID: 42, TargetID: 420, LabelID: 3, AuthRevision: "rev-1"})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	a = a.RecordAttempt("label", "op-stale-1")
	if _, err := db.UpdateAction(ctx, a, 0); err != nil {
		t.Fatalf("attempt: %v", err)
	}
	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{err: errors.New("handshake lost")}); err == nil {
		t.Fatal("second failure accepted")
	}
	settled := a.RecordResult("label", factory.LifecycleResultDelivered)
	if _, err := db.UpdateAction(ctx, settled, 1); err != nil {
		t.Fatalf("in-flight reconcile blocked by refresh failure: %v", err)
	}
	if _, err := db.SaveConfig(ctx, lifecycleTestConfig(factory.LifecycleEventIssueCreated, 2, true)); !errors.Is(err, store.ErrCapabilityUnproved) {
		t.Fatalf("new enable accepted while unavailable: %v", err)
	}
}

// TestReconcileReplacements covers successful refreshes that change the
// verified answer: lost kinds, new generations and new installations.
func TestReconcileReplacements(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	ctx := context.Background()
	issue := factory.LifecycleEventKind(factory.LifecycleEventIssueCreated)

	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Installation: "inst-1", Generation: "gen-7", Kinds: []string{issue}}}); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Installation: "inst-1", Generation: "gen-7"}}); err != nil {
		t.Fatalf("empty reconcile: %v", err)
	}
	if ok, err := db.CapabilityProved(ctx, factory.LifecycleEventIssueCreated); err != nil || ok {
		t.Fatalf("lost capability retained: %v %v", ok, err)
	}
	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Installation: "inst-1", Generation: "gen-8", Kinds: []string{issue}}}); err != nil {
		t.Fatalf("generation reconcile: %v", err)
	}
	if _, err := db.SaveConfig(ctx, lifecycleTestConfig(factory.LifecycleEventIssueCreated, 0, true)); err != nil {
		t.Fatalf("new generation enable blocked: %v", err)
	}
	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Installation: "inst-2", Generation: "gen-8", Kinds: []string{issue}}}); err != nil {
		t.Fatalf("installation reconcile: %v", err)
	}
	if _, err := db.SaveConfig(ctx, lifecycleTestConfig(factory.LifecycleEventIssueCreated, 1, true)); err != nil {
		t.Fatalf("new installation enable blocked: %v", err)
	}
}
