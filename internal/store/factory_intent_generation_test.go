package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestIntentGenerationPinsParentBeforeEffects(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Repository: 42, Objective: "Build a request form", State: "accepted", MaxIssues: 4, PlanningCalls: 2, CreatedUnix: 1, Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Request form", Consequence: "Requests are recorded", Decision: "accept"}}, Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Build request form", Outcomes: []string{"Submit a request"}, IdeaIDs: []string{"d333333333333333333333333"}}}}
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	g := factory.IntentGeneration{RevisionID: r.ID, IntentID: r.Intent, OperationID: "graph-op-1", ActorID: 9, NativeRev: 1, AuthRevision: r.ID, NotAfter: 10, Payload: json.RawMessage(`{"creates":[]}`), IntentDigest: r.Digest(), State: "pending"}
	if err := s.BeginIntentGeneration(ctx, g); err != nil {
		t.Fatal(err)
	}
	if err := s.BeginIntentGeneration(ctx, g); err != nil {
		t.Fatal(err)
	}
	g.ActorID = 10
	if !errors.Is(s.BeginIntentGeneration(ctx, g), ErrCommandConflict) {
		t.Fatal("changed operation accepted")
	}
	g.ActorID = 9
	g.State = "committed"
	g.Receipt = json.RawMessage(`{"issues":[1]}`)
	if err := s.RecordIntentGeneration(ctx, g); err != nil {
		t.Fatal(err)
	}
	b := factory.IntentTaskBinding{RevisionID: r.ID, TaskKey: r.Tasks[0].Key, IssueID: 1, IssueIndex: 1, AcceptanceID: "d555555555555555555555555"}
	if err := s.BindIntentTaskAcceptance(ctx, b); err != nil {
		t.Fatal(err)
	}
	current, found, err := s.GeneratedIntentCurrent(ctx, b.AcceptanceID)
	if err != nil || !found || !current {
		t.Fatal(current, found, err)
	}
	r.Predecessor = r.ID
	r.ID = "d666666666666666666666666"
	r.State = "draft"
	if err = s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	current, found, err = s.GeneratedIntentCurrent(ctx, b.AcceptanceID)
	if err != nil || !found || current {
		t.Fatal("superseded scope stayed authoritative")
	}
	g.State = "pending"
	g.Receipt = nil
	if !errors.Is(s.BeginIntentGeneration(ctx, g), ErrStaleRevision) {
		t.Fatal("stale generation accepted")
	}
}

func TestIntentPlanningHasTwoIndependentAllowances(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	s.PriorityCallBudget = 3
	ctx := context.Background()
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Objective: "Track requests", State: "draft", MaxIssues: 4, PlanningCalls: 1, CreatedUnix: 1}
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	a := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	b := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	ok, err := s.ReserveIntentJudgment(ctx, r, a)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	ok, err = s.ReserveIntentJudgment(ctx, r, b)
	if err != nil || ok {
		t.Fatal("intent allowance exceeded")
	}
}
