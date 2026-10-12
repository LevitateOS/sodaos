package control

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

type consumeFakeExecutor struct {
	actor    int64
	revision int64
	target   int64
	submits  []factory.LifecycleWork
	submit   func(factory.LifecycleWork) (factory.OperationOutcome, error)
	lookup   func(string) (factory.OperationOutcome, error)
}

func (f *consumeFakeExecutor) EnrolledActor(context.Context) (int64, error) {
	return f.actor, nil
}

func (f *consumeFakeExecutor) ObserveLifecycleRevision(_ context.Context, _ int64) (int64, error) {
	return f.revision, nil
}

func (f *consumeFakeExecutor) ResolveNoteTarget(_ context.Context, _, _ int64) (int64, error) {
	return f.target, nil
}

func (f *consumeFakeExecutor) SubmitLifecycle(_ context.Context, work factory.LifecycleWork) (factory.OperationOutcome, error) {
	f.submits = append(f.submits, work)
	return f.submit(work)
}

func (f *consumeFakeExecutor) LookupLifecycle(_ context.Context, id string) (factory.OperationOutcome, error) {
	return f.lookup(id)
}

func consumeOutcome(effect, reason string) factory.OperationOutcome {
	return factory.OperationOutcome{Effect: effect, Reason: reason}
}

func enableConsumeConfig(t *testing.T, s *store.Store, event string, mutate func(*factory.LifecycleConfig)) {
	t.Helper()
	ctx := context.Background()
	if _, err := s.RegisterCapability(ctx, event, "gen-1", "install-1"); err != nil {
		t.Fatalf("capability: %v", err)
	}
	cfg := factory.LifecycleConfig{Event: event, Enabled: true, ActorID: 99, RepositoryID: 9}
	switch event {
	case factory.LifecycleEventIssueCreated:
		cfg.LabelID = 3
	case factory.LifecycleEventPRCreated:
		cfg.ReviewerIDs = []int64{15}
	case factory.LifecycleEventVerifiedMerge:
		cfg.Note = "done"
	}
	if mutate != nil {
		mutate(&cfg)
	}
	if _, err := s.SaveConfig(ctx, cfg); err != nil {
		t.Fatalf("config: %v", err)
	}
}

func consumeIssueInput() LifecycleEventInput {
	return LifecycleEventInput{Event: factory.LifecycleEventIssueCreated, SourceID: "rev-1:task-a", RepositoryID: 9, IssueIndex: 7, TargetID: 42, AuthRevision: "rev-1:digest"}
}

// TestLifecycleNewIssueKeys proves reused and delivered bindings never
// become creation events.
func TestLifecycleNewIssueKeys(t *testing.T) {
	g := factory.IntentGeneration{
		TaskIssueIDs: map[string]int64{"reused": 10},
		Delivered:    map[string]factory.DeliveredIssue{"old": {IssueID: 11}},
	}
	got := LifecycleNewIssueKeys([]string{"new-a", "reused", "old", "new-b"}, g)
	if len(got) != 2 || got[0] != "new-a" || got[1] != "new-b" {
		t.Fatalf("filter: %v", got)
	}
}

// TestConsumeLifecycleEvent proves the production consumer path: gates,
// durable identity before submit, bounded retry, denial stops and
// duplicate reconciliation through the real journal.
func TestConsumeLifecycleEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("disabledSkips", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		ex := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}}
		if err := ConsumeLifecycleEvent(ctx, s, ex, consumeIssueInput()); err != nil {
			t.Fatalf("disabled: %v", err)
		}
		if len(ex.submits) != 0 {
			t.Fatal("disabled event submitted")
		}
	})

	t.Run("unprovedSkips", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		cfg := factory.LifecycleConfig{Event: factory.LifecycleEventIssueCreated, Enabled: false, ActorID: 7, RepositoryID: 9}
		if _, err := s.SaveConfig(ctx, cfg); err != nil {
			t.Fatalf("config: %v", err)
		}
		ex := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}}
		if err := ConsumeLifecycleEvent(ctx, s, ex, consumeIssueInput()); err != nil {
			t.Fatalf("unproved: %v", err)
		}
		if len(ex.submits) != 0 {
			t.Fatal("unproved event submitted")
		}
	})

	t.Run("separateRolesSubmit", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		enableConsumeConfig(t, s, factory.LifecycleEventIssueCreated, nil)
		ex := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}, lookup: func(string) (factory.OperationOutcome, error) {
			return factory.OperationOutcome{}, errors.New("unused")
		}}
		if err := ConsumeLifecycleEvent(ctx, s, ex, consumeIssueInput()); err != nil {
			t.Fatalf("separate roles: %v", err)
		}
		a, err := s.LifecycleAction(ctx, LifecycleActionID(consumeIssueInput()))
		if err != nil {
			t.Fatalf("journal: %v", err)
		}
		if a.ActorID != 7 || a.ConfigOwner != 99 {
			t.Fatalf("roles: %+v", a)
		}
		if a.IssueID != 7 || a.TargetID != 42 {
			t.Fatalf("read key/target: %+v", a)
		}
	})

	t.Run("unavailableExecutorJournalsUncertain", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		enableConsumeConfig(t, s, factory.LifecycleEventIssueCreated, nil)
		in := consumeIssueInput()
		if err := ConsumeLifecycleEvent(ctx, s, nil, in); !errors.Is(err, ErrLifecycleExecutorUnavailable) {
			t.Fatalf("err=%v", err)
		}
		a, err := s.LifecycleAction(ctx, LifecycleActionID(in))
		if err != nil {
			t.Fatalf("journal: %v", err)
		}
		if a.Attempts != 0 || a.ActorID != 0 || a.Results["label"] != factory.LifecycleResultUncertain || a.ConfigOwner != 99 {
			t.Fatalf("unclaimed: %+v", a)
		}
		// The drain claims and attempts the row once an executor
		// appears, without a duplicate event.
		ex := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}, lookup: func(string) (factory.OperationOutcome, error) {
			return factory.OperationOutcome{NotObserved: true}, nil
		}}
		if err := ReconcileUnsettledLifecycle(ctx, s, ex); err != nil {
			t.Fatalf("drain: %v", err)
		}
		claimed, err := s.LifecycleAction(ctx, LifecycleActionID(in))
		if err != nil {
			t.Fatalf("journal: %v", err)
		}
		if !claimed.Settled() || claimed.ActorID != 7 || len(ex.submits) != 1 || ex.submits[0].IssueID != 42 {
			t.Fatalf("claimed: %+v", claimed)
		}
	})

	t.Run("delivered", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		enableConsumeConfig(t, s, factory.LifecycleEventIssueCreated, nil)
		ex := &consumeFakeExecutor{actor: 7, revision: 4, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}, lookup: func(string) (factory.OperationOutcome, error) {
			return factory.OperationOutcome{}, errors.New("unused")
		}}
		in := consumeIssueInput()
		if err := ConsumeLifecycleEvent(ctx, s, ex, in); err != nil {
			t.Fatalf("consume: %v", err)
		}
		if len(ex.submits) != 1 || ex.submits[0].NativeRev != 4 || ex.submits[0].ActorID != 7 {
			t.Fatalf("submit: %+v", ex.submits)
		}
		a, err := s.LifecycleAction(ctx, LifecycleActionID(in))
		if err != nil {
			t.Fatalf("journal: %v", err)
		}
		if !a.Settled() || a.Results["label"] != factory.LifecycleResultDelivered || a.Attempts != 1 {
			t.Fatalf("action: %+v", a)
		}
		// Duplicate delivery reconciles without resubmitting.
		if err := ConsumeLifecycleEvent(ctx, s, ex, in); err != nil {
			t.Fatalf("duplicate: %v", err)
		}
		if len(ex.submits) != 1 {
			t.Fatal("duplicate resubmitted")
		}
	})

	t.Run("failedRetriesOnceOnLaterTick", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		enableConsumeConfig(t, s, factory.LifecycleEventPRCreated, nil)
		calls := 0
		ex := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			calls++
			if calls == 1 {
				return consumeOutcome(factory.OpEffectNotCommitted, "native_refused"), nil
			}
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}, lookup: func(string) (factory.OperationOutcome, error) {
			return factory.OperationOutcome{}, errors.New("unused")
		}}
		in := LifecycleEventInput{Event: factory.LifecycleEventPRCreated, SourceID: "pub-1", RepositoryID: 9, IssueIndex: 8, TargetID: 88, AuthRevision: "pubauth-1"}
		if err := ConsumeLifecycleEvent(ctx, s, ex, in); err != nil {
			t.Fatalf("consume: %v", err)
		}
		if calls != 1 {
			t.Fatalf("inline submits: %d", calls)
		}
		// The single retry runs on a later tick, keeping the inline
		// hook to one attempt for default-path latency.
		if _, err := ReconcileLifecycleAction(ctx, s, ex, LifecycleActionID(in)); err != nil {
			t.Fatalf("reconcile: %v", err)
		}
		if calls != 2 {
			t.Fatalf("submits: %d", calls)
		}
		a, err := s.LifecycleAction(ctx, LifecycleActionID(in))
		if err != nil {
			t.Fatalf("journal: %v", err)
		}
		if !a.Settled() || a.Attempts != 2 || ex.submits[0].OperationID == ex.submits[1].OperationID {
			t.Fatalf("retry: %+v", a)
		}
	})

	t.Run("deniedStops", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		enableConsumeConfig(t, s, factory.LifecycleEventIssueCreated, nil)
		calls := 0
		ex := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			calls++
			return consumeOutcome(factory.OpEffectNotCommitted, factory.LifecycleReasonAuthorityLost), nil
		}, lookup: func(string) (factory.OperationOutcome, error) {
			return factory.OperationOutcome{}, errors.New("unused")
		}}
		if err := ConsumeLifecycleEvent(ctx, s, ex, consumeIssueInput()); err != nil {
			t.Fatalf("consume: %v", err)
		}
		if calls != 1 {
			t.Fatalf("denied retried: %d", calls)
		}
		a, err := s.LifecycleAction(ctx, LifecycleActionID(consumeIssueInput()))
		if err != nil {
			t.Fatalf("journal: %v", err)
		}
		if !a.Settled() || a.Results["label"] != factory.LifecycleResultDenied {
			t.Fatalf("action: %+v", a)
		}
	})

	t.Run("noteResolvesTarget", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		enableConsumeConfig(t, s, factory.LifecycleEventVerifiedMerge, nil)
		ex := &consumeFakeExecutor{actor: 7, revision: 2, target: 77, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}, lookup: func(string) (factory.OperationOutcome, error) {
			return factory.OperationOutcome{}, errors.New("unused")
		}}
		in := LifecycleEventInput{Event: factory.LifecycleEventVerifiedMerge, SourceID: "merge-1", RepositoryID: 9, IssueIndex: 5, AuthRevision: "mergeauth-1", MergeRef: "abc", Closed: true}
		if err := ConsumeLifecycleEvent(ctx, s, ex, in); err != nil {
			t.Fatalf("consume: %v", err)
		}
		if len(ex.submits) != 1 || ex.submits[0].IssueID != 77 || !ex.submits[0].Closed {
			t.Fatalf("submit: %+v", ex.submits)
		}
		a, err := s.LifecycleAction(ctx, LifecycleActionID(in))
		if err != nil || a.IssueID != 5 || a.TargetID != 77 {
			t.Fatalf("read key/target: %+v %v", a, err)
		}
	})
}

// TestConsumeOwnerHelpers proves the three completion-site helpers fire
// one configured effect each from real owner structs, skip reused
// bindings, and refuse incomplete receipts.
func TestConsumeOwnerHelpers(t *testing.T) {
	ctx := context.Background()

	t.Run("generatedIssuesSkipReused", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		enableConsumeConfig(t, s, factory.LifecycleEventIssueCreated, nil)
		ex := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}, lookup: func(string) (factory.OperationOutcome, error) {
			return factory.OperationOutcome{}, errors.New("unused")
		}}
		receipt, err := json.Marshal(sdk.IssueGraphReceipt{ActorID: 21, RepositoryID: 9, Issues: []sdk.IssueGraphIssueReceipt{
			{Key: "task-new", IssueID: 42, IssueIndex: 7, ContentVersion: 1},
			{IssueID: 10, IssueIndex: 3, ContentVersion: 2},
		}})
		if err != nil {
			t.Fatalf("receipt: %v", err)
		}
		r := factory.IntentRevision{ID: "rev-1", Repository: 9, Owner: 99}
		g := factory.IntentGeneration{
			ActorID: 21, AuthRevision: "rev-1:digest", Receipt: receipt,
			SubmittedTasks: []factory.IntentTask{{Key: "task-new"}, {Key: "task-reused"}},
			TaskIssueIDs:   map[string]int64{"task-reused": 10},
		}
		if err := ConsumeGeneratedIssues(ctx, s, ex, r, g); err != nil {
			t.Fatalf("consume: %v", err)
		}
		if len(ex.submits) != 1 || ex.submits[0].IssueID != 42 || ex.submits[0].ActorID != 7 {
			t.Fatalf("submits: %+v", ex.submits)
		}
		if _, err := s.LifecycleAction(ctx, "lifecycle:issue_created:rev-1:task-new"); err != nil {
			t.Fatalf("journal: %v", err)
		}
		visible, err := s.ListActions(ctx, 9, 7, factory.LifecycleEventIssueCreated, 64)
		if err != nil || len(visible) != 1 || visible[0].TargetID != 42 {
			t.Fatalf("read visibility: %+v %v", visible, err)
		}
		if _, err := s.LifecycleAction(ctx, "lifecycle:issue_created:rev-1:task-reused"); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("reused journaled: %v", err)
		}
	})

	t.Run("publishedPR", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		enableConsumeConfig(t, s, factory.LifecycleEventPRCreated, nil)
		ex := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}, lookup: func(string) (factory.OperationOutcome, error) {
			return factory.OperationOutcome{}, errors.New("unused")
		}}
		p := factory.Publication{
			ID: "pub-1", Stage: factory.PublicationPublished, Repository: 9, Issue: 8, PRNumber: 12, PRID: 34,
			PRCreate: factory.PublicationOperation{IssueID: 88, Work: &factory.PublicationIntent{AuthRevision: "pubauth-1", OperationID: "pr-op-1"}},
		}
		if err := ConsumePublishedPR(ctx, s, ex, p); err != nil {
			t.Fatalf("consume: %v", err)
		}
		if len(ex.submits) != 1 || ex.submits[0].IssueID != 88 {
			t.Fatalf("submits: %+v", ex.submits)
		}
		a, err := s.LifecycleAction(ctx, "lifecycle:pr_created:pub-1:pr-12-34")
		if err != nil || !a.Settled() {
			t.Fatalf("journal: %+v %v", a, err)
		}
		visible, err := s.ListActions(ctx, 9, 8, factory.LifecycleEventPRCreated, 64)
		if err != nil || len(visible) != 1 || visible[0].TargetID != 88 {
			t.Fatalf("read visibility: %+v %v", visible, err)
		}
		open := p
		open.Stage = factory.PublicationOpen
		if err := ConsumePublishedPR(ctx, s, ex, open); err == nil {
			t.Fatal("open publication fired")
		}
	})

	t.Run("verifiedMerge", func(t *testing.T) {
		s, _ := postgresFixture(t, nil)
		enableConsumeConfig(t, s, factory.LifecycleEventVerifiedMerge, nil)
		ex := &consumeFakeExecutor{actor: 7, revision: 1, target: 55, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
			return consumeOutcome(factory.OpEffectCommitted, ""), nil
		}, lookup: func(string) (factory.OperationOutcome, error) {
			return factory.OperationOutcome{}, errors.New("unused")
		}}
		m := factory.Merge{
			ID: "merge-1", Stage: factory.MergeMerged, Repository: 9, Issue: 5, PRNumber: 12, MergedCommit: "abc123", OriginalClosingOccurrence: "99",
			Operation: factory.MergeOperation{Work: &factory.MergeIntent{AuthRevision: "mergeauth-1", OperationID: "merge-op-1"}},
		}
		if err := ConsumeVerifiedMerge(ctx, s, ex, m); err != nil {
			t.Fatalf("consume: %v", err)
		}
		if len(ex.submits) != 1 || ex.submits[0].IssueID != 55 || ex.submits[0].MergeRef != "pr/12@abc123" {
			t.Fatalf("submits: %+v", ex.submits)
		}
		visible, err := s.ListActions(ctx, 9, 5, factory.LifecycleEventVerifiedMerge, 64)
		if err != nil || len(visible) != 1 || visible[0].TargetID != 55 {
			t.Fatalf("read visibility: %+v %v", visible, err)
		}
		unclosed := m
		unclosed.OriginalClosingOccurrence = ""
		if err := ConsumeVerifiedMerge(ctx, s, ex, unclosed); err == nil {
			t.Fatal("unclosed merge fired")
		}
	})
}

// TestReconcileAfterRestart proves later recovery: an uncertain action
// left by a failed call settles through a fresh invocation that only
// shares the durable journal, simulating restart or a later tick.
func TestReconcileAfterRestart(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	enableConsumeConfig(t, s, factory.LifecycleEventIssueCreated, nil)
	in := consumeIssueInput()

	first := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("transport lost")
	}, lookup: func(string) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("lookup lost")
	}}
	if err := ConsumeLifecycleEvent(ctx, s, first, in); err != nil {
		t.Fatalf("first: %v", err)
	}
	a, err := s.LifecycleAction(ctx, LifecycleActionID(in))
	if err != nil {
		t.Fatalf("journal: %v", err)
	}
	if a.Settled() || a.Results["label"] != factory.LifecycleResultUncertain || a.Attempts != 1 {
		t.Fatalf("uncertain not journaled: %+v", a)
	}

	// Simulated restart: a fresh executor and a later duplicate delivery
	// reconcile the same durable identity without resubmitting.
	second := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("must not resubmit")
	}, lookup: func(id string) (factory.OperationOutcome, error) {
		if id != a.Operations["label"] {
			t.Fatalf("wrong identity: %s", id)
		}
		return consumeOutcome(factory.OpEffectCommitted, ""), nil
	}}
	if err := ConsumeLifecycleEvent(ctx, s, second, in); err != nil {
		t.Fatalf("recovery: %v", err)
	}
	settled, err := s.LifecycleAction(ctx, LifecycleActionID(in))
	if err != nil {
		t.Fatalf("journal: %v", err)
	}
	if !settled.Settled() || settled.Results["label"] != factory.LifecycleResultDelivered || len(second.submits) != 0 {
		t.Fatalf("not reconciled: %+v", settled)
	}
}

// TestReconcileUnsettledDrain proves the existing-tick recovery drain
// settles pending and uncertain journal entries left by finished
// completions.
func TestReconcileUnsettledDrain(t *testing.T) {
	ctx := context.Background()
	s, _ := postgresFixture(t, nil)
	enableConsumeConfig(t, s, factory.LifecycleEventIssueCreated, nil)

	pending := factory.LifecycleAction{ID: "lifecycle:issue_created:rev-9:task-z", Event: factory.LifecycleEventIssueCreated, Action: factory.LifecycleActionIssueLabelAssignee, ActorID: 7, RepositoryID: 9, IssueID: 50, TargetID: 500, LabelID: 3, AuthRevision: "rev-9"}
	if _, err := s.RecordAction(ctx, pending); err != nil {
		t.Fatalf("record: %v", err)
	}
	ex := &consumeFakeExecutor{actor: 7, revision: 1, submit: func(factory.LifecycleWork) (factory.OperationOutcome, error) {
		return consumeOutcome(factory.OpEffectCommitted, ""), nil
	}, lookup: func(string) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{}, errors.New("unused")
	}}
	if err := ReconcileUnsettledLifecycle(ctx, s, ex); err != nil {
		t.Fatalf("drain: %v", err)
	}
	a, err := s.LifecycleAction(ctx, pending.ID)
	if err != nil {
		t.Fatalf("journal: %v", err)
	}
	if !a.Settled() || len(ex.submits) != 1 {
		t.Fatalf("pending not drained: %+v", a)
	}
	if err := ReconcileUnsettledLifecycle(ctx, s, ex); err != nil {
		t.Fatalf("redrain: %v", err)
	}
	if len(ex.submits) != 1 {
		t.Fatal("settled action redriven")
	}
	// An inline definitive failure is selected and retried by a later
	// tick with no duplicate event.
	failed := factory.LifecycleAction{ID: "lifecycle:issue_created:rev-9:task-f", Event: factory.LifecycleEventIssueCreated, Action: factory.LifecycleActionIssueLabelAssignee, ActorID: 7, RepositoryID: 9, IssueID: 51, TargetID: 510, LabelID: 3, AuthRevision: "rev-9"}
	if _, err := s.RecordAction(ctx, failed); err != nil {
		t.Fatalf("record: %v", err)
	}
	if _, err := s.UpdateAction(ctx, failed.RecordActionAttempt("op-f1"), 0); err != nil {
		t.Fatalf("claim: %v", err)
	}
	stored, err := s.LifecycleAction(ctx, failed.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if _, err := s.UpdateAction(ctx, stored.RecordActionResult(factory.LifecycleResultFailed), 1); err != nil {
		t.Fatalf("fail: %v", err)
	}
	if err := ReconcileUnsettledLifecycle(ctx, s, ex); err != nil {
		t.Fatalf("retry drain: %v", err)
	}
	retried, err := s.LifecycleAction(ctx, failed.ID)
	if err != nil {
		t.Fatalf("journal: %v", err)
	}
	if !retried.Settled() || retried.Attempts != 2 || len(ex.submits) != 2 {
		t.Fatalf("failed not retried: %+v", retried)
	}
}
