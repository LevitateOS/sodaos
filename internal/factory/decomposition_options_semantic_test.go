package factory

import (
	"strings"
	"testing"
)

// FW08 semantic output assessment: the handler consumes planner outputs
// only (proposal content + alternatives + investigation), never an accepted
// graph or history. Fixture heads are drafts; revision changes refuse.

func semanticPlanningInput(t *testing.T, op string) SemanticPlannerInput {
	t.Helper()
	head := draftPlanningHead()
	content, err := ComposeProposalContent(head)
	if err != nil {
		t.Fatalf("proposal content: %v", err)
	}
	alternatives := DescribeAlternatives(head.Objective)
	if !alternatives.Supported {
		t.Fatal("representative objective left authored coverage")
	}
	return SemanticPlannerInput{
		Operation:    op,
		Head:         head,
		Content:      content,
		Alternatives: alternatives,
		Inspected: []InspectedSource{
			{Path: "internal/factory/intent.go", Commit: strings.Repeat("c", 40), Evidence: "IntentRevision carries draft state with no tasks yet"},
		},
	}
}

func semanticOutputFor(in SemanticPlannerInput, tasks []IntentTask) SemanticPlannerOutput {
	return SemanticPlannerOutput{
		Intent: in.Head.Intent, Revision: in.Head.ID, Operation: in.Operation,
		Candidates: []SemanticCandidate{{Key: "candidate-one", Rationale: "Cohesive feature issues with independent roots where justified.", Tasks: tasks}},
	}
}

func TestSemanticInputBindsHeadContentAndInvestigation(t *testing.T) {
	in := semanticPlanningInput(t, IntentPlanningOptions)
	if err := in.Validate(); err != nil {
		t.Fatalf("planner input refused: %v", err)
	}
	// The propose operation also runs on unsupported objectives through
	// the conservative baseline path.
	propose := in
	propose.Operation = IntentPlanningPropose
	propose.Head.Objective = "Fix a help-text typo on the status page"
	propose.Alternatives = DescribeAlternatives(propose.Head.Objective)
	if propose.Alternatives.Supported {
		t.Fatal("typo objective unexpectedly entered authored coverage")
	}
	propose.Content = ProposalContent{Producer: AuthoredProducer, Intent: propose.Head.Intent, Revision: propose.Head.ID, Objective: propose.Head.Objective, Facts: []string{propose.Head.Objective}, Unknowns: []string{"Nothing else is established."}}
	if err := propose.Validate(); err != nil {
		t.Fatalf("unsupported propose input refused: %v", err)
	}
	options := in
	options.Alternatives = DescribeAlternatives("Fix a help-text typo on the status page")
	if err := options.Validate(); err == nil {
		t.Fatal("unsupported decomposition options admitted")
	}
	uninspected := in
	uninspected.Inspected = nil
	if err := uninspected.Validate(); err == nil {
		t.Fatal("planning without investigation admitted")
	}
	mismatch := in
	mismatch.Content.Revision = candidateKey(1)
	if err := mismatch.Validate(); err == nil {
		t.Fatal("unbound proposal content admitted")
	}
	unknown := in
	unknown.Operation = "template_split"
	if err := unknown.Validate(); err == nil {
		t.Fatal("unknown planning operation admitted")
	}
}

func TestAssessSemanticOutputAcceptsBroadAndSmall(t *testing.T) {
	in := semanticPlanningInput(t, IntentPlanningOptions)
	tasks := broadPlanningTasks()
	tasks[1].DependsOn = nil
	broad := semanticOutputFor(in, tasks)
	if err := AssessSemanticOutput(in, broad); err != nil {
		t.Fatalf("broad output refused: %v", err)
	}
	small := semanticOutputFor(in, []IntentTask{
		{Key: candidateKey(9), Objective: "Capture leave requests", Outcomes: []string{"Requests can be recorded", "Decisions are visible"}, IdeaIDs: []string{"d333333333333333333333333", "d444444444444444444444444"}},
	})
	if err := AssessSemanticOutput(in, small); err != nil {
		t.Fatalf("small output refused: %v", err)
	}
	// Replay reaches the same verdict and spends nothing.
	if err := AssessSemanticOutput(in, broad); err != nil {
		t.Fatalf("replay diverged: %v", err)
	}
}

func TestAssessSemanticOutputRefusesMismatchUncoveredAndFalse(t *testing.T) {
	in := semanticPlanningInput(t, IntentPlanningOptions)
	moved := semanticOutputFor(in, broadPlanningTasks())
	moved.Revision = candidateKey(2)
	if err := AssessSemanticOutput(in, moved); err == nil {
		t.Fatal("revision change passed")
	}
	uncovered := semanticOutputFor(in, []IntentTask{
		{Key: candidateKey(1), Objective: "Capture leave requests", Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{"d333333333333333333333333"}},
	})
	if err := AssessSemanticOutput(in, uncovered); err == nil {
		t.Fatal("uncovered outcome passed")
	}
	falseDep := broadPlanningTasks()
	falseDep[0].DependsOn = []string{candidateKey(77)}
	if err := AssessSemanticOutput(in, semanticOutputFor(in, falseDep)); err == nil {
		t.Fatal("false dependency passed")
	}
	empty := semanticOutputFor(in, nil)
	empty.Candidates = nil
	if err := AssessSemanticOutput(in, empty); err == nil {
		t.Fatal("empty answer passed")
	}
	overflow := semanticOutputFor(in, broadPlanningTasks())
	overflow.Candidates = []SemanticCandidate{
		{Key: "one", Rationale: "First.", Tasks: broadPlanningTasks()},
		{Key: "two", Rationale: "Second.", Tasks: broadPlanningTasks()},
		{Key: "three", Rationale: "Third.", Tasks: broadPlanningTasks()},
		{Key: "four", Rationale: "Fourth.", Tasks: broadPlanningTasks()},
	}
	if err := AssessSemanticOutput(in, overflow); err == nil {
		t.Fatal("four-candidate answer passed")
	}
	dupe := semanticOutputFor(in, broadPlanningTasks())
	dupe.Candidates = []SemanticCandidate{
		{Key: "same", Rationale: "First.", Tasks: broadPlanningTasks()},
		{Key: "same", Rationale: "Second.", Tasks: broadPlanningTasks()},
	}
	if err := AssessSemanticOutput(in, dupe); err == nil {
		t.Fatal("duplicate candidates passed")
	}
}

func TestAssessSemanticOutputAttentionCarriesNoCandidates(t *testing.T) {
	in := semanticPlanningInput(t, IntentPlanningOptions)
	attention := SemanticPlannerOutput{Intent: in.Head.Intent, Revision: in.Head.ID, Operation: in.Operation, Attention: "A consequential unspecified access policy needs a product decision before decomposition."}
	if err := AssessSemanticOutput(in, attention); err != nil {
		t.Fatalf("attention refused: %v", err)
	}
	withBatch := attention
	withBatch.Candidates = []SemanticCandidate{{Key: "one", Rationale: "Extra.", Tasks: broadPlanningTasks()}}
	if err := AssessSemanticOutput(in, withBatch); err == nil {
		t.Fatal("attention with a new batch passed")
	}
}
