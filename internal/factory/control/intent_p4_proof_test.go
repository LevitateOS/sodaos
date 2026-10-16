package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// P4/FW05 closure proof: the authorized repository-bound pre-issue planner
// runs end to end on the FW08 path without an accepted-graph prerequisite,
// with a representative broad tracker output (independent roots, join,
// descendant) and a small output (a help-text typo stays one issue).
//
// The funded semantic provider is not enrolled in this checkout, so live
// model calls stop at the precise external gate ErrIntentPlanningExternal
// (exercised below with a nil planner). Representative outputs use a
// deterministic stub as an explicit substitute: it proves admission and
// assessment plumbing plus output shape, not live semantic quality. No
// provider results are invented.
//
// The Repository=0 leg rests on the prior partial that stands: this handler
// takes no Store, Host or Graph dependency, so an unassociated draft cannot
// launch anything here; admission additionally refuses it so only an
// associated draft enters the generative run.

func p4TrackerHead() factory.IntentRevision {
	return factory.IntentRevision{
		ID: "d111111111111111111111112", Intent: "d222222222222222222222223",
		Owner: 1, Repository: 7,
		Objective: "Build a leave approval tracker so managers can review time-off requests, with CSV export, label filter, dashboard and saved reports",
		State:     "draft", MaxIssues: 10, PlanningCalls: 3, CreatedUnix: 1,
		Ideas: []factory.ProductIdea{
			{ID: "d333333333333333333333331", Description: "Export requests to CSV", Consequence: "Requests leave the tracker independently", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
			{ID: "d333333333333333333333332", Description: "Filter requests by label", Consequence: "Requests can be found without export", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
			{ID: "d333333333333333333333333", Description: "Review requests on a dashboard", Consequence: "Decisions are visible in one place", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
			{ID: "d333333333333333333333334", Description: "Save dashboard reports", Consequence: "Saved reports reproduce the dashboard", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
		},
		EffectiveOutcomes: []string{"d333333333333333333333331", "d333333333333333333333332", "d333333333333333333333333", "d333333333333333333333334"},
	}
}

func p4TypoHead() factory.IntentRevision {
	return factory.IntentRevision{
		ID: "d111111111111111111111113", Intent: "d222222222222222222222224",
		Owner: 1, Repository: 7, Objective: "Fix a help-text typo on the status page",
		State: "draft", MaxIssues: 10, PlanningCalls: 3, CreatedUnix: 1,
		Ideas: []factory.ProductIdea{
			{ID: "d333333333333333333333335", Description: "Correct the approval confirmation wording", Consequence: "The recorded behavior matches the reviewed wording", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
		},
		EffectiveOutcomes: []string{"d333333333333333333333335"},
	}
}

func p4PlannerInput(t *testing.T, head factory.IntentRevision, op string) factory.SemanticPlannerInput {
	t.Helper()
	content, err := factory.ComposeProposalContent(head)
	if err != nil {
		t.Fatalf("proposal content: %v", err)
	}
	return factory.SemanticPlannerInput{
		Operation:    op,
		Head:         head,
		Content:      content,
		Alternatives: factory.DescribeAlternatives(head.Objective),
		Inspected: []factory.InspectedSource{
			{Path: "internal/factory/intent.go", Commit: strings.Repeat("a", 40), Evidence: "P4 proof inspects the draft head carrying attributed effective scope with no tasks yet"},
		},
	}
}

// The broad tracker graph: CSV export and label filter are independent
// roots, the dashboard joins both, and saved reports descend from the
// dashboard.
func p4TrackerTasks() []factory.IntentTask {
	csv, filter := planningTaskKey(31), planningTaskKey(32)
	dashboard, reports := planningTaskKey(33), planningTaskKey(34)
	return []factory.IntentTask{
		{Key: csv, Objective: "Export leave requests to CSV", Outcomes: []string{"Requests leave the tracker independently of the dashboard"}, IdeaIDs: []string{"d333333333333333333333331"}},
		{Key: filter, Objective: "Filter leave requests by label", Outcomes: []string{"Requests can be found without exporting"}, IdeaIDs: []string{"d333333333333333333333332"}},
		{Key: dashboard, Objective: "Review leave requests on a dashboard", Outcomes: []string{"Decisions are visible in one place"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{csv, filter}},
		{Key: reports, Objective: "Save dashboard reports", Outcomes: []string{"Saved reports reproduce the reviewed dashboard"}, IdeaIDs: []string{"d333333333333333333333334"}, DependsOn: []string{dashboard}},
	}
}

func p4TrackerOutput(in factory.SemanticPlannerInput) factory.SemanticPlannerOutput {
	return factory.SemanticPlannerOutput{
		Intent: in.Head.Intent, Revision: in.Head.ID, Operation: in.Operation,
		Candidates: []factory.SemanticCandidate{{Key: "tracker-candidate", Rationale: "Cohesive feature issues: independent export and filter roots joined by the dashboard, with saved reports descending from it.", Tasks: p4TrackerTasks()}},
	}
}

func TestP4RepositoryBoundPlannerProof(t *testing.T) {
	t.Run("broad tracker without accepted graph", func(t *testing.T) {
		head := p4TrackerHead()
		if len(head.Tasks) != 0 {
			t.Fatal("proof head must carry no tasks: no accepted graph is a prerequisite")
		}
		in := p4PlannerInput(t, head, factory.IntentPlanningOptions)
		if !in.Alternatives.Supported {
			t.Fatal("representative tracker objective left authored coverage")
		}
		if err := AdmitIntentPlanning(planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in); err != nil {
			t.Fatalf("authorized repository-bound draft refused: %v", err)
		}
		stub := &stubPlanner{out: p4TrackerOutput(in)}
		out, err := PlanIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, stub)
		if err != nil {
			t.Fatalf("broad tracker run refused: %v", err)
		}
		if stub.calls != 1 || len(out.Candidates) != 1 || out.Attention != "" {
			t.Fatalf("unexpected broad answer %+v", out)
		}
		tasks := out.Candidates[0].Tasks
		if len(tasks) != 4 || len(tasks[0].DependsOn) != 0 || len(tasks[1].DependsOn) != 0 {
			t.Fatal("export and filter are not independent roots")
		}
		if len(tasks[2].DependsOn) != 2 || len(tasks[3].DependsOn) != 1 || tasks[3].DependsOn[0] != tasks[2].Key {
			t.Fatal("dashboard does not join both roots with reports descending from it")
		}
		if err := factory.AssessSemanticOutput(in, out); err != nil {
			t.Fatalf("replay diverged: %v", err)
		}
	})

	t.Run("small typo stays one issue", func(t *testing.T) {
		head := p4TypoHead()
		in := p4PlannerInput(t, head, factory.IntentPlanningPropose)
		if in.Alternatives.Supported {
			t.Fatal("typo objective unexpectedly entered authored coverage")
		}
		small := factory.SemanticPlannerOutput{
			Intent: head.Intent, Revision: head.ID, Operation: in.Operation,
			Candidates: []factory.SemanticCandidate{{Key: "typo-candidate", Rationale: "A wording correction is one cohesive issue.", Tasks: []factory.IntentTask{
				{Key: planningTaskKey(41), Objective: "Correct the approval confirmation wording", Outcomes: []string{"The status page shows the reviewed wording"}, IdeaIDs: []string{"d333333333333333333333335"}},
			}}},
		}
		stub := &stubPlanner{out: small}
		out, err := PlanIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, stub)
		if err != nil {
			t.Fatalf("small typo run refused: %v", err)
		}
		if len(out.Candidates) != 1 || len(out.Candidates[0].Tasks) != 1 {
			t.Fatalf("typo did not stay one issue: %+v", out)
		}
	})

	t.Run("attention carries no batch", func(t *testing.T) {
		head := p4TrackerHead()
		in := p4PlannerInput(t, head, factory.IntentPlanningOptions)
		attention := factory.SemanticPlannerOutput{Intent: head.Intent, Revision: head.ID, Operation: in.Operation, Attention: "A consequential unspecified access policy needs a product decision before decomposition."}
		stub := &stubPlanner{out: attention}
		out, err := PlanIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, stub)
		if err != nil || out.Attention == "" || len(out.Candidates) != 0 {
			t.Fatalf("attention mishandled: %+v %v", out, err)
		}
		if stub.calls != 1 {
			t.Fatal("attention triggered another batch")
		}
	})

	t.Run("missing provider is an external gate", func(t *testing.T) {
		head := p4TrackerHead()
		in := p4PlannerInput(t, head, factory.IntentPlanningOptions)
		out, err := PlanIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, nil)
		if err == nil || !errors.Is(err, ErrIntentPlanningExternal) {
			t.Fatalf("missing provider is not an external gate: %v", err)
		}
		if !strings.Contains(err.Error(), head.Intent) {
			t.Fatalf("gate does not name the intent: %v", err)
		}
		if len(out.Candidates) != 0 {
			t.Fatal("external gate substituted output")
		}
	})

	t.Run("unassociated draft cannot enter the run", func(t *testing.T) {
		head := p4TrackerHead()
		head.Repository = 0
		subject := planningSubject(head)
		in := p4PlannerInput(t, head, factory.IntentPlanningOptions)
		if err := AdmitIntentPlanning(subject, project.ExecutionDecompose, project.RoleCoder, head, true, in); err == nil {
			t.Fatal("repository-less draft entered planning")
		} else if !strings.Contains(err.Error(), "repository association") {
			t.Fatalf("refusal does not name the association gate: %v", err)
		}
		stub := &stubPlanner{out: p4TrackerOutput(in)}
		if _, err := PlanIntentDecomposition(context.Background(), subject, project.ExecutionDecompose, project.RoleCoder, head, true, in, stub); err == nil {
			t.Fatal("repository-less draft ran the planner")
		}
		if stub.calls != 0 {
			t.Fatal("refused draft consumed a planning call")
		}
	})

	t.Run("accepted graph never enters the pre-issue path", func(t *testing.T) {
		head := p4TrackerHead()
		head.State = "accepted"
		head.Tasks = p4TrackerTasks()
		in := p4PlannerInput(t, p4TrackerHead(), factory.IntentPlanningOptions)
		in.Head = head
		if err := AdmitIntentPlanning(planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in); err == nil {
			t.Fatal("accepted graph entered the pre-issue path")
		}
	})
}
