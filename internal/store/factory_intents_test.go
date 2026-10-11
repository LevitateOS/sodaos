package store

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestIntentRevisionCASAndPrivacy(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	d := factory.Department{ID: "d111111111111111111111111", Owner: 1, Name: "Operations", Description: "Coordinate requests"}
	if err := s.CreateDepartment(ctx, d); err != nil {
		t.Fatal(err)
	}
	r := factory.IntentRevision{ID: "d222222222222222222222222", Intent: "d333333333333333333333333", Owner: 1, Department: d.ID, Objective: "Build request tracker", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	r.Objective = "Changed objective"
	if !errors.Is(s.SaveIntentRevision(ctx, r), ErrCommandConflict) {
		t.Fatal("changed replay accepted")
	}
	r.ID = "d444444444444444444444444"
	if !errors.Is(s.SaveIntentRevision(ctx, r), ErrStaleRevision) {
		t.Fatal("stale predecessor accepted")
	}
	r.Predecessor = "d222222222222222222222222"
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IntentRevision(ctx, r.Intent, 2); !errors.Is(err, ErrNotFound) {
		t.Fatal("private intent visible")
	}
	r.Owner = 2
	r.Intent = "d555555555555555555555555"
	r.Predecessor = ""
	if s.SaveIntentRevision(ctx, r) == nil {
		t.Fatal("other owner department accepted")
	}
}

func TestListIntentRevisionsIsOwnerPrivateAndBounded(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	a := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Objective: "First", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	b := factory.IntentRevision{ID: "d333333333333333333333333", Intent: "d444444444444444444444444", Owner: 1, Objective: "Second", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	other := factory.IntentRevision{ID: "d555555555555555555555555", Intent: "d666666666666666666666666", Owner: 2, Objective: "Other", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	for _, r := range []factory.IntentRevision{a, b, other} {
		if err := s.SaveIntentRevision(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	got, next, err := s.ListIntentRevisions(ctx, 1, factory.IntentListFilter{Limit: 1})
	if err != nil || len(got) != 1 || next == "" {
		t.Fatalf("bounded list %v %v %v", got, next, err)
	}
	got, next, err = s.ListIntentRevisions(ctx, 1, factory.IntentListFilter{Cursor: next, Limit: 10})
	if err != nil || len(got) != 1 || next != "" || got[0].Intent == other.Intent {
		t.Fatal("cursor leaked or missed")
	}
	got, _, err = s.ListIntentRevisions(ctx, 1, factory.IntentListFilter{State: "accepted", Limit: 10})
	if err != nil || len(got) != 0 {
		t.Fatal("empty filter not empty")
	}
}

func TestIntentHistoryAndRepositoryIdentity(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	a := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Objective: "Tracker", State: "draft", MaxIssues: 10, CreatedUnix: 1}
	if err := s.SaveIntentRevision(ctx, a); err != nil {
		t.Fatal(err)
	}
	b := a
	b.ID = "d333333333333333333333333"
	b.Predecessor = a.ID
	b.Repository = 42
	b.State = "accepted"
	b.Ideas = []factory.ProductIdea{{ID: "d444444444444444444444444", Description: "Track", Consequence: "Visibility", Decision: "accept"}}
	b.Tasks = []factory.IntentTask{{Key: "d555555555555555555555555", Objective: "Entry", Outcomes: []string{"Capture"}, IdeaIDs: []string{"d444444444444444444444444"}}}
	if err := s.SaveIntentRevision(ctx, b); err != nil {
		t.Fatal(err)
	}
	history, err := s.IntentHistory(ctx, a.Intent, 1, 10)
	if err != nil || len(history) != 2 || history[0].ID != b.ID {
		t.Fatalf("history %v %v", history, err)
	}
	c := b
	c.ID = "d666666666666666666666666"
	c.Predecessor = b.ID
	c.Repository = 43
	if s.SaveIntentRevision(ctx, c) == nil {
		t.Fatal("repository change kept same request identity")
	}
}

func TestPriorityBudgetReservesBeforeCalls(t *testing.T) {
	s, _ := postgresFixture(t, nil)
	ctx := context.Background()
	s.PriorityCallBudget = 3
	a := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	b := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	ok, err := s.ReservePriorityJudgment(ctx, a, 2)
	if err != nil || !ok {
		t.Fatalf("reserve %v %v", ok, err)
	}
	ok, err = s.ReservePriorityJudgment(ctx, a, 2)
	if err != nil || ok {
		t.Fatal("repeated reservation")
	}
	ok, err = s.ReservePriorityJudgment(ctx, b, 2)
	if err != nil || ok {
		t.Fatal("exhausted reservation")
	}
}
