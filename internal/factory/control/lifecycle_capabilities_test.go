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
// enables exactly its event, with no direct marker seeding. Unsupported
// events, failed handshakes, changed installations and lost generations
// all leave availability closed.
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
	for _, event := range []string{factory.LifecycleEventPRCreated, factory.LifecycleEventVerifiedMerge} {
		if _, err := db.SaveConfig(ctx, lifecycleTestConfig(event, 0, true)); !errors.Is(err, store.ErrCapabilityUnproved) {
			t.Fatalf("%s enabled without proof: %v", event, err)
		}
		if ok, err := db.CapabilityProved(ctx, event); err != nil || ok {
			t.Fatalf("%s proved=%v err=%v", event, ok, err)
		}
	}

	failed := &scriptHandshake{err: errors.New("handshake lost")}
	if err := ReconcileLifecycleCapabilities(ctx, db, failed); err == nil {
		t.Fatal("failed handshake accepted")
	}
	if failed.calls != 1 {
		t.Fatalf("calls=%d", failed.calls)
	}
	if _, err := db.SaveConfig(ctx, lifecycleTestConfig(factory.LifecycleEventIssueCreated, 1, true)); err != nil {
		t.Fatalf("prior marker disturbed by failed handshake: %v", err)
	}

	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Installation: "inst-1", Generation: "gen-7", Kinds: nil}}); err != nil {
		t.Fatalf("empty reconcile: %v", err)
	}
	if ok, err := db.CapabilityProved(ctx, factory.LifecycleEventIssueCreated); err != nil || ok {
		t.Fatalf("lost capability retained: %v %v", ok, err)
	}
	disabled := lifecycleTestConfig(factory.LifecycleEventIssueCreated, 2, false)
	disabled.LabelID = 0
	if _, err := db.SaveConfig(ctx, disabled); err != nil {
		t.Fatalf("disable after loss blocked: %v", err)
	}
	reenable := lifecycleTestConfig(factory.LifecycleEventIssueCreated, 3, true)
	if _, err := db.SaveConfig(ctx, reenable); !errors.Is(err, store.ErrCapabilityUnproved) {
		t.Fatalf("re-enable without proof: %v", err)
	}

	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Installation: "inst-1", Generation: "gen-8", Kinds: []string{issue}}}); err != nil {
		t.Fatalf("generation reconcile: %v", err)
	}
	if _, err := db.SaveConfig(ctx, lifecycleTestConfig(factory.LifecycleEventIssueCreated, 3, true)); err != nil {
		t.Fatalf("new generation enable blocked: %v", err)
	}

	if err := ReconcileLifecycleCapabilities(ctx, db, &scriptHandshake{answer: factory.LifecycleHandshake{Installation: "inst-2", Generation: "gen-8", Kinds: []string{issue}}}); err != nil {
		t.Fatalf("installation reconcile: %v", err)
	}
	fresh, err := db.SaveConfig(ctx, lifecycleTestConfig(factory.LifecycleEventIssueCreated, 4, true))
	if err != nil {
		t.Fatalf("new installation enable blocked: %v", err)
	}
	if !fresh.Enabled {
		t.Fatal("new installation config not enabled")
	}
}
