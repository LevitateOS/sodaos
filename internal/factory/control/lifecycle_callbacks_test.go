package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func callbackLifecycleExecutor() *consumeFakeExecutor {
	return &consumeFakeExecutor{actor: 7, revision: 1, target: 77, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
		return consumeOutcome(factory.OpEffectCommitted, ""), nil
	}, lookup: func(string) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("unused")
	}}
}

func enableCallbackConfig(t *testing.T, fx *publishFixture, event string, mutate func(*factory.LifecycleConfig)) {
	t.Helper()
	ctx := context.Background()
	if _, err := fx.db.RegisterCapability(ctx, event, "gen-1", "install-1"); err != nil {
		t.Fatalf("capability: %v", err)
	}
	cfg := factory.LifecycleConfig{Event: event, Enabled: true, ActorID: 99, RepositoryID: 7}
	switch event {
	case factory.LifecycleEventPRCreated:
		cfg.ReviewerIDs = []int64{15}
	case factory.LifecycleEventVerifiedMerge:
		cfg.Note = "done"
	}
	if mutate != nil {
		mutate(&cfg)
	}
	if _, err := fx.db.SaveConfig(ctx, cfg); err != nil {
		t.Fatalf("config: %v", err)
	}
}

// TestLifecycleCallbackPR proves a real PublishPass to Published fires
// the reviewer effect with default delivery intact, and that an
// unwired coordinator changes nothing.
func TestLifecycleCallbackPR(t *testing.T) {
	ctx := context.Background()

	run := func(t *testing.T, wired bool) (*publishFixture, *consumeFakeExecutor, PublishReport) {
		t.Helper()
		fx := publishSeed(t)
		fx.finishReported(t, 3)
		enableCallbackConfig(t, fx, factory.LifecycleEventPRCreated, nil)
		ex := callbackLifecycleExecutor()
		if wired {
			fx.coord.Lifecycle = ex
		}
		fx.wire(happyPublisher())
		return fx, ex, fx.pass(t)
	}

	fx, ex, first := run(t, true)
	if len(first.Published) != 1 {
		t.Fatalf("default report: %+v", first)
	}
	if len(ex.submits) != 1 {
		t.Fatalf("submits: %+v", ex.submits)
	}
	visible, err := fx.db.ListActions(ctx, 7, 3, factory.LifecycleEventPRCreated, 64)
	if err != nil || len(visible) != 1 || !visible[0].Settled() || visible[0].TargetID != ex.submits[0].IssueID || visible[0].PRNumber != 9 {
		t.Fatalf("journal: %+v %v", visible, err)
	}
	fx.pass(t)
	if len(ex.submits) != 1 {
		t.Fatal("second pass resubmitted")
	}

	_, exOff, off := run(t, false)
	if len(off.Published) != 1 {
		t.Fatalf("unwired default report: %+v", off)
	}
	if len(exOff.submits) != 0 {
		t.Fatal("unwired coordinator submitted")
	}
}

// TestLifecycleCallbackMerge proves a real MergePass to Merged fires
// the note effect after verified merge plus original closure. The row
// opens unbound, binds through the union gate, then drives: an
// unauthorized row waits honestly instead of merging ungated.
func TestLifecycleCallbackMerge(t *testing.T) {
	ctx := context.Background()
	fx, p := mergeSeed(t, 3)
	enableCallbackConfig(t, fx.publish, factory.LifecycleEventVerifiedMerge, nil)
	ex := callbackLifecycleExecutor()
	fx.publish.coord.Lifecycle = ex
	fx.wire(pendingMerger())
	setup := MergeReport{Merged: []MergeLink{}}
	fx.publish.coord.mergeOne(context.Background(), p, &setup)
	if len(setup.Errors) != 0 {
		t.Fatalf("setup merge open: %+v", setup)
	}
	fx.wire(happyMerger())
	authorizeMergeFixture(t, context.Background(), fx.publish.db, fx.publish.coord, p)
	first := fx.pass(t)
	if len(first.Merged) != 1 {
		t.Fatalf("default report: %+v", first)
	}
	if len(ex.submits) != 1 || ex.submits[0].IssueID != 77 || !ex.submits[0].Closed {
		t.Fatalf("submits: %+v", ex.submits)
	}
	visible, err := fx.publish.db.ListActions(ctx, 7, 3, factory.LifecycleEventVerifiedMerge, 64)
	if err != nil || len(visible) != 1 || visible[0].TargetID != 77 {
		t.Fatalf("read visibility: %+v %v", visible, err)
	}
	fx.pass(t)
	if len(ex.submits) != 1 {
		t.Fatal("second pass resubmitted")
	}
}

// TestLifecycleCallbackGeneration proves the real accepted-intent
// generation path fires one label effect per newly created issue.
func TestLifecycleCallbackGeneration(t *testing.T) {
	ctx := context.Background()
	c, _, r := acceptedGenerationFixture(t)
	ex := callbackLifecycleExecutor()
	c.Lifecycle = ex
	if _, err := c.Store.RegisterCapability(ctx, factory.LifecycleEventIssueCreated, "gen-1", "install-1"); err != nil {
		t.Fatalf("capability: %v", err)
	}
	cfg := factory.LifecycleConfig{Event: factory.LifecycleEventIssueCreated, Enabled: true, ActorID: 99, RepositoryID: 42, LabelID: 3}
	if _, err := c.Store.SaveConfig(ctx, cfg); err != nil {
		t.Fatalf("config: %v", err)
	}
	g, err := c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner)
	if err != nil {
		t.Fatalf("generation: %v", err)
	}
	if g.State != "completed" {
		t.Fatalf("default state: %s", g.State)
	}
	if len(ex.submits) != 2 {
		t.Fatalf("submits: %+v", ex.submits)
	}
	visible, err := c.Store.ListActions(ctx, 42, 1, factory.LifecycleEventIssueCreated, 64)
	if err != nil || len(visible) != 1 {
		t.Fatalf("read visibility: %+v %v", visible, err)
	}
}
