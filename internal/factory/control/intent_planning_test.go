package control

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func planningTaskKey(n int) string { return fmt.Sprintf("d%024x", 0xb000+n) }

// FW08 admission/handler tests: pure, no Store/Host/Graph. The handler under
// test takes none of those dependencies, which is itself the proof that
// pre-issue planning launches no repository session.

type stubPlanner struct {
	out   factory.SemanticPlannerOutput
	err   error
	calls int
}

func (s *stubPlanner) PlanSemantic(_ context.Context, _ factory.SemanticPlannerInput) (factory.SemanticPlannerOutput, error) {
	s.calls++
	return s.out, s.err
}

func planningHead() factory.IntentRevision {
	return factory.IntentRevision{
		ID: "d111111111111111111111111", Intent: "d222222222222222222222222",
		Owner: 1, Repository: 42, Objective: "Build a leave approval workflow so managers can review time-off requests",
		State: "draft", MaxIssues: 10, PlanningCalls: 3, CreatedUnix: 1,
		Ideas: []factory.ProductIdea{
			{ID: "d333333333333333333333333", Description: "Capture leave requests", Consequence: "Requests can be recorded", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
			{ID: "d444444444444444444444444", Description: "Managers review requests", Consequence: "Decisions are visible", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
		},
		EffectiveOutcomes: []string{"d333333333333333333333333", "d444444444444444444444444"},
	}
}

func planningSubject(head factory.IntentRevision) factory.ExecutionSubject {
	return factory.ExecutionSubject{
		Kind:   factory.SubjectIntent,
		Intent: &factory.IntentSubject{IntentID: head.Intent, RevisionID: head.ID, ScopeDigest: head.Digest(), OwnerID: head.Owner},
	}
}

func planningInput(t *testing.T, head factory.IntentRevision, op string) factory.SemanticPlannerInput {
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
			{Path: "internal/factory/intent.go", Commit: strings.Repeat("c", 40), Evidence: "Draft head carries attributed effective scope with no tasks yet"},
		},
	}
}

func planningOutput(in factory.SemanticPlannerInput) factory.SemanticPlannerOutput {
	return factory.SemanticPlannerOutput{
		Intent: in.Head.Intent, Revision: in.Head.ID, Operation: in.Operation,
		Candidates: []factory.SemanticCandidate{{Key: "candidate-one", Rationale: "Cohesive feature issues with independent roots where justified.", Tasks: []factory.IntentTask{
			{Key: planningTaskKey(1), Objective: "Capture leave requests", Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{"d333333333333333333333333"}},
			{Key: planningTaskKey(2), Objective: "Managers review requests", Outcomes: []string{"Decisions are visible"}, IdeaIDs: []string{"d444444444444444444444444"}, DependsOn: []string{planningTaskKey(1)}},
		}}},
	}
}

func TestAdmitIntentPlanningRefusesWrongKindPhaseAndRole(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	issue := factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: 7, Acceptance: "d" + strings.Repeat("a", 24)}}
	if err := AdmitIntentPlanning(issue, project.ExecutionInvestigate, project.RoleCoder, head, true, in); err == nil {
		t.Fatal("issue subject admitted to intent planning")
	}
	subject := planningSubject(head)
	if err := AdmitIntentPlanning(subject, project.ExecutionInvestigate, project.RoleCoder, head, true, in); err == nil {
		t.Fatal("investigate phase admitted intent planning")
	}
	if err := AdmitIntentPlanning(subject, project.ExecutionDecompose, project.RoleReviewer, head, true, in); err == nil {
		t.Fatal("reviewer role admitted intent planning")
	}
}

func TestAdmitIntentPlanningBindsRevisionOwnerScopeFundsAndInvestigation(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	moved := head
	moved.ID = "d999999999999999999999999"
	if err := AdmitIntentPlanning(planningSubject(moved), project.ExecutionDecompose, project.RoleCoder, head, true, in); err == nil {
		t.Fatal("moved revision admitted")
	} else if !errors.Is(err, store.ErrStaleRevision) {
		t.Fatalf("moved revision is not stale: %v", err)
	}
	// Planner inputs built from a different head refuse even when the
	// subject binds the current one.
	if err := AdmitIntentPlanning(planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, planningInput(t, moved, factory.IntentPlanningOptions)); err == nil {
		t.Fatal("stale planner inputs admitted")
	} else if !errors.Is(err, store.ErrStaleRevision) {
		t.Fatalf("stale inputs are not stale: %v", err)
	}
	foreign := planningSubject(head)
	foreign.Intent.OwnerID = 9
	if err := AdmitIntentPlanning(foreign, project.ExecutionDecompose, project.RoleCoder, head, true, in); err == nil {
		t.Fatal("foreign owner admitted")
	}
	if err := AdmitIntentPlanning(planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, false, in); err == nil {
		t.Fatal("exhausted allowance admitted")
	} else if !errors.Is(err, ErrIntentPlanningExhausted) {
		t.Fatalf("exhaustion is not visible: %v", err)
	}
	bare := in
	bare.Inspected = nil
	if err := AdmitIntentPlanning(planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, bare); err == nil {
		t.Fatal("planning without investigation admitted")
	} else if !errors.Is(err, ErrIntentPlanningInvestigate) {
		t.Fatalf("missing investigation is not distinct: %v", err)
	}
	accepted := head
	accepted.State = "accepted"
	accepted.Repository = 42
	accepted.Tasks = []factory.IntentTask{{Key: planningTaskKey(11), Objective: "Capture leave requests", Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{"d333333333333333333333333"}}}
	acceptedIn := planningInput(t, head, factory.IntentPlanningOptions)
	acceptedIn.Head = accepted
	if err := AdmitIntentPlanning(planningSubject(accepted), project.ExecutionDecompose, project.RoleCoder, accepted, true, acceptedIn); err == nil {
		t.Fatal("accepted graph entered the pre-issue path")
	}
}

func TestPlanIntentDecompositionRunsBothOperationsWithoutSessions(t *testing.T) {
	for _, op := range []string{factory.IntentPlanningPropose, factory.IntentPlanningOptions} {
		head := planningHead()
		in := planningInput(t, head, op)
		stub := &stubPlanner{out: planningOutput(in)}
		// No Coordinator, Store, Host or Graph exists here: the call
		// cannot launch a repository session or grant publication.
		out, err := PlanIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, stub)
		if err != nil {
			t.Fatalf("%s: %v", op, err)
		}
		if stub.calls != 1 || len(out.Candidates) != 1 || out.Attention != "" {
			t.Fatalf("%s: unexpected answer %+v", op, out)
		}
	}
}

func TestPlanIntentDecompositionExternalGateIsPrecise(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
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
}

func TestPlanIntentDecompositionRefusesInvalidAnswers(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	bad := planningOutput(in)
	bad.Candidates[0].Tasks = bad.Candidates[0].Tasks[:1]
	stub := &stubPlanner{out: bad}
	if _, err := PlanIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, stub); err == nil {
		t.Fatal("uncovered provider answer passed")
	}
	if stub.calls != 1 {
		t.Fatal("invalid answer skipped provider accounting")
	}
}

func TestPlanIntentDecompositionAttentionAndCancellationStayVisible(t *testing.T) {
	head := planningHead()
	in := planningInput(t, head, factory.IntentPlanningOptions)
	attention := factory.SemanticPlannerOutput{Intent: head.Intent, Revision: head.ID, Operation: in.Operation, Attention: "A consequential unspecified access policy needs a product decision before decomposition."}
	stub := &stubPlanner{out: attention}
	out, err := PlanIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, stub)
	if err != nil || out.Attention == "" || len(out.Candidates) != 0 {
		t.Fatalf("attention mishandled: %+v %v", out, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PlanIntentDecomposition(ctx, planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, stub); !errors.Is(err, ErrIntentPlanningCancelled) {
		t.Fatalf("cancellation is not visible: %v", err)
	}
	providerGone := &stubPlanner{err: context.DeadlineExceeded}
	if _, err := PlanIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, providerGone); !errors.Is(err, ErrIntentPlanningCancelled) {
		t.Fatalf("provider cancellation is not visible: %v", err)
	}
	failed := &stubPlanner{err: errors.New("model 429")}
	if _, err := PlanIntentDecomposition(context.Background(), planningSubject(head), project.ExecutionDecompose, project.RoleCoder, head, true, in, failed); err == nil || errors.Is(err, ErrIntentPlanningCancelled) {
		t.Fatalf("provider failure misclassified: %v", err)
	}
}
