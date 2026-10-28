package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestIntentGenerationDeliveredMustChainPriorBindings(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	task := factory.IntentTask{Key: "d444444444444444444444444", Objective: "Build request form", Outcomes: []string{"Submit a request"}, IdeaIDs: []string{"d333333333333333333333333"}}
	first := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Repository: 42, Objective: "Build a request form", State: "accepted", MaxIssues: 4, CreatedUnix: 1, Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Request form", Consequence: "Requests are recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}}, Tasks: []factory.IntentTask{task}}
	if err := s.SaveIntentRevision(ctx, first); err != nil {
		t.Fatal(err)
	}
	g1 := factory.IntentGeneration{RevisionID: first.ID, IntentID: first.Intent, OperationID: "graph-op-1", ActorID: 9, NativeRev: 1, AuthRevision: first.ID, NotAfter: 10, Payload: json.RawMessage(`{"creates":[]}`), SubmittedTasks: []factory.IntentTask{task}, IntentDigest: first.Digest(), State: "pending"}
	if err := s.BeginIntentGeneration(ctx, g1); err != nil {
		t.Fatal(err)
	}
	if err := s.BindIntentTaskAcceptance(ctx, factory.IntentTaskBinding{RevisionID: first.ID, TaskKey: task.Key, IssueID: 101, IssueIndex: 1, AcceptanceID: "d555555555555555555555555"}); err != nil {
		t.Fatal(err)
	}
	next := first
	next.ID = "d666666666666666666666666"
	next.Predecessor = first.ID
	if err := s.SaveIntentRevision(ctx, next); err != nil {
		t.Fatal(err)
	}
	base := factory.IntentGeneration{RevisionID: next.ID, IntentID: next.Intent, OperationID: "graph-op-2", ActorID: 9, NativeRev: 2, AuthRevision: next.ID, NotAfter: 10, SubmittedTasks: []factory.IntentTask{task}, IntentDigest: next.Digest(), State: "pending"}
	forged := base
	forged.Payload = json.RawMessage(`{"creates":[]}`)
	forged.Delivered = map[string]factory.DeliveredIssue{task.Key: {IssueID: 999, IssueIndex: 9}}
	if err := s.BeginIntentGeneration(ctx, forged); err == nil || !strings.Contains(err.Error(), "invalid delivered reuse") {
		t.Fatalf("fabricated delivered reuse accepted: %v", err)
	}
	edited := base
	edited.Payload = json.RawMessage(`{"edits":[{"issue_id":101,"expected_title":"t","expected_body":"b","expected_content_version":0,"title":"t","body":"b"}]}`)
	edited.Delivered = map[string]factory.DeliveredIssue{task.Key: {IssueID: 101, IssueIndex: 1}}
	if err := s.BeginIntentGeneration(ctx, edited); err == nil || !strings.Contains(err.Error(), "delivered history unexpectedly mutated") {
		t.Fatalf("delivered edit accepted: %v", err)
	}
	chained := base
	chained.Payload = json.RawMessage(`{"creates":[]}`)
	chained.Delivered = map[string]factory.DeliveredIssue{task.Key: {IssueID: 101, IssueIndex: 1}}
	if err := s.BeginIntentGeneration(ctx, chained); err != nil {
		t.Fatal(err)
	}
}

func TestIntentGenerationPinsParentBeforeEffects(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Repository: 42, Objective: "Build a request form", State: "accepted", MaxIssues: 4, PlanningCalls: 2, CreatedUnix: 1, Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Request form", Consequence: "Requests are recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}}, Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Build request form", Outcomes: []string{"Submit a request"}, IdeaIDs: []string{"d333333333333333333333333"}}}}
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	g := factory.IntentGeneration{RevisionID: r.ID, IntentID: r.Intent, OperationID: "graph-op-1", ActorID: 9, NativeRev: 1, AuthRevision: r.ID, NotAfter: 10, Payload: json.RawMessage(`{"creates":[]}`), IntentDigest: r.Digest(), State: "pending"}
	if s.BeginIntentGeneration(ctx, g) == nil {
		t.Fatal("generation without submitted tasks accepted")
	}
	g.SubmittedTasks = append([]factory.IntentTask(nil), r.Tasks...)
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

func TestSupersededAcceptanceInvalidatesPlanningWithoutRetarget(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Repository: 42, Objective: "Build a request form", State: "accepted", MaxIssues: 4, CreatedUnix: 1, Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Request form", Consequence: "Requests are recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1}}, Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "Build request form", Outcomes: []string{"Submit a request"}, IdeaIDs: []string{"d333333333333333333333333"}}}}
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	binding := factory.IntentTaskBinding{RevisionID: r.ID, TaskKey: "d444444444444444444444444", IssueID: 101, IssueIndex: 1, ContentVersion: 1, AcceptanceID: "d777777777777777777777777", CreatedUnix: 1}
	if err := s.BindIntentTaskAcceptance(ctx, binding); err != nil {
		t.Fatal(err)
	}
	next := r
	next.ID = "d555555555555555555555555"
	next.Predecessor = r.ID
	next.Tasks[0].Objective = "Build request form with validation"
	if err := s.SaveIntentRevision(ctx, next); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReserveIssuePlanning(ctx, binding.AcceptanceID, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("superseded acceptance planned: %v", err)
	}
	kept, err := s.IntentTaskBindings(ctx, r.ID)
	if err != nil || len(kept) != 1 || kept[0] != binding {
		t.Fatalf("prior binding retargeted: %+v %v", kept, err)
	}
}

func TestIntentPlanningUsesIndependentAllowance(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Objective: "Track requests", State: "draft", MaxIssues: 4, PlanningCalls: 100, CreatedUnix: 1}
	if err := s.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	a := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	b := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if ok, err := s.ReserveIntentJudgment(ctx, r, a); ok || !errors.Is(err, ErrPlanningAllowanceExhausted) {
		t.Fatal("missing allowance funded from the embedded field")
	}
	if err := s.SavePlanningAllowance(ctx, factory.PlanningAllowance{ID: "d333333333333333333333333", Intent: r.Intent, Owner: r.Owner, Amount: 1, Unit: factory.PlanningAllowanceUnit, Ceiling: 4, CreatedUnix: 1}); err != nil {
		t.Fatal(err)
	}
	ok, err := s.ReserveIntentJudgment(ctx, r, a)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	ok, err = s.ReserveIntentJudgment(ctx, r, b)
	if ok || !errors.Is(err, ErrPlanningAllowanceExhausted) {
		t.Fatal("exhausted allowance admitted")
	}
	raised := factory.PlanningAllowance{ID: "d444444444444444444444444", Intent: r.Intent, Predecessor: "d333333333333333333333333", Owner: r.Owner, Amount: 2, Unit: factory.PlanningAllowanceUnit, Ceiling: 4, Effect: "one more consultation for the same accepted scope", CreatedUnix: 1}
	if err := s.SavePlanningAllowance(ctx, raised); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.ReserveIntentJudgment(ctx, r, b); err != nil || !ok {
		t.Fatal("raised allowance did not release exhaustion")
	}
	head, err := s.IntentRevision(ctx, r.Intent, r.Owner)
	if err != nil || head.ID != r.ID {
		t.Fatal("allowance change moved intent scope")
	}
}

func TestIntentPlanningAllowsIndependentIntents(t *testing.T) {
	s, _ := postgresFixture(t)
	ctx := context.Background()
	for i, intent := range []string{"d222222222222222222222222", "d444444444444444444444444"} {
		r := factory.IntentRevision{ID: []string{"d111111111111111111111111", "d333333333333333333333333"}[i], Intent: intent, Owner: 1, Objective: "Track requests", State: "draft", MaxIssues: 4, CreatedUnix: 1}
		if err := s.SaveIntentRevision(ctx, r); err != nil {
			t.Fatal(err)
		}
		if err := s.SavePlanningAllowance(ctx, factory.PlanningAllowance{ID: []string{"d555555555555555555555555", "d666666666666666666666666"}[i], Intent: intent, Owner: 1, Amount: 4, Unit: factory.PlanningAllowanceUnit, Ceiling: 4, CreatedUnix: 1}); err != nil {
			t.Fatal(err)
		}
	}
	first := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1}
	second := factory.IntentRevision{ID: "d333333333333333333333333", Intent: "d444444444444444444444444", Owner: 1}
	if ok, err := s.ReserveIntentJudgment(ctx, first, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, err := s.ReserveIntentJudgment(ctx, second, "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"); err != nil || !ok {
		t.Fatalf("independent intent reservation: %v %v", ok, err)
	}
}
