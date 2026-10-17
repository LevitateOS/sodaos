package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// Deterministic orchestration legs: admission, assessment and the single
// reduction behave identically on every replay and spend nothing. Provider
// semantics legs (funded planner answers, absence, cancellation) are
// labeled separately below.

func TestChooseSingleCandidateAdoptsItsOnlyCandidate(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	out := planningOutput(in)
	first, err := ChooseSingleCandidate(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if first.Intent != head.Intent || first.Revision != head.ID || first.Operation != in.Operation {
		t.Fatalf("choice does not bind its planning input: %+v", first)
	}
	if first.Key != "candidate-one" || len(first.Tasks) != len(out.Candidates[0].Tasks) {
		t.Fatalf("choice drops its candidate: %+v", first)
	}
	// Replay reaches the same choice: pure reduction, no spend.
	again, err := ChooseSingleCandidate(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if again.Key != first.Key || len(again.Tasks) != len(first.Tasks) {
		t.Fatalf("replay diverged: %+v vs %+v", first, again)
	}
}

func TestChooseSingleCandidateAttentionStaysVisible(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	attention := factory.SemanticPlannerOutput{Intent: head.Intent, Revision: head.ID, Operation: in.Operation, Attention: "A consequential unspecified access policy needs a product decision before decomposition."}
	if _, err := ChooseSingleCandidate(in, attention); !errors.Is(err, ErrIntentPlanningAttention) {
		t.Fatalf("attention is not visible: %v", err)
	}
}

func TestChooseSingleCandidateRefusesMultipleCandidates(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	out := planningOutput(in)
	second := out.Candidates[0]
	second.Key = "candidate-two"
	out.Candidates = append(out.Candidates, second)
	if _, err := ChooseSingleCandidate(in, out); !errors.Is(err, ErrIntentPlanningAmbiguous) {
		t.Fatalf("multi-candidate choice is not refused: %v", err)
	}
}

func TestChooseSingleCandidateRefusesInvalidAnswers(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	bad := planningOutput(in)
	bad.Candidates[0].Tasks = bad.Candidates[0].Tasks[:1]
	if _, err := ChooseSingleCandidate(in, bad); err == nil {
		t.Fatal("uncovered provider answer passed")
	}
}

// Real-provider semantics legs, labeled separately from the deterministic
// orchestration above: funded absence, attention and cancellation stay
// visible through the serving joint.

func TestPlanSingleIntentDecompositionServesOneChoice(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	stub := &stubPlanner{out: planningOutput(in)}
	chosen, err := PlanSingleIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, stub)
	if err != nil {
		t.Fatal(err)
	}
	if stub.calls != 1 || chosen.Key != "candidate-one" || len(chosen.Tasks) != 2 {
		t.Fatalf("serving choice=%+v calls=%d", chosen, stub.calls)
	}
}

func TestPlanSingleIntentDecompositionProviderSemanticsStayVisible(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	subject := planningSubject(head)
	if _, err := PlanSingleIntentDecomposition(context.Background(), subject, project.ExecutionDecompose, project.RoleCoder, head, true, in, nil); !errors.Is(err, ErrIntentPlanningExternal) {
		t.Fatalf("missing provider is not an external gate: %v", err)
	}
	attention := factory.SemanticPlannerOutput{Intent: head.Intent, Revision: head.ID, Operation: in.Operation, Attention: "A consequential unspecified access policy needs a product decision before decomposition."}
	if _, err := PlanSingleIntentDecomposition(context.Background(), subject, project.ExecutionDecompose, project.RoleCoder, head, true, in, &stubPlanner{out: attention}); !errors.Is(err, ErrIntentPlanningAttention) {
		t.Fatalf("attention is not visible: %v", err)
	}
	multi := planningOutput(in)
	second := multi.Candidates[0]
	second.Key = "candidate-two"
	multi.Candidates = append(multi.Candidates, second)
	if _, err := PlanSingleIntentDecomposition(context.Background(), subject, project.ExecutionDecompose, project.RoleCoder, head, true, in, &stubPlanner{out: multi}); !errors.Is(err, ErrIntentPlanningAmbiguous) {
		t.Fatalf("multi-candidate answer is not refused: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PlanSingleIntentDecomposition(ctx, subject, project.ExecutionDecompose, project.RoleCoder, head, true, in, &stubPlanner{out: planningOutput(in)}); !errors.Is(err, ErrIntentPlanningCancelled) {
		t.Fatalf("cancellation is not visible: %v", err)
	}
}
