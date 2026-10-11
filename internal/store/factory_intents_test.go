package store

import (
	"context"
	"errors"
	"github.com/levitateos/sodaos/internal/factory"
	"testing"
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
