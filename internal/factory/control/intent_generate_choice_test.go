package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
)

// refusingDecompositionChooser fails any judgment call. A passing
// choice-adoption run behind it proves the serving path performed no
// second triplet or template evaluation.
type refusingDecompositionChooser struct {
	calls int
}

func (f *refusingDecompositionChooser) Select(_ context.Context, _ any, _ string, _ map[string]any) (factory.PriorityAnswer, error) {
	f.calls++
	return factory.PriorityAnswer{}, errors.New("serving judgment must not evaluate a pre-chosen plan")
}

func choiceTaskKey(n int) string { return fmt.Sprintf("d%024x", 0xc000+n) }

// broadJoinAcceptedRevision is a reviewed draft turned accepted with no
// preexisting graph: three independent roots joined by one consumer. Human
// provenance marks the reviewed product choices.
func broadJoinAcceptedRevision() factory.IntentRevision {
	idea := func(id, desc, consequence string) factory.ProductIdea {
		return factory.ProductIdea{ID: id, Description: desc, Consequence: consequence, Decision: "accept", Source: "requested", Provenance: "human", Actor: 7}
	}
	return factory.IntentRevision{
		ID: "d333333333333333333333333", Intent: "d444444444444444444444444",
		Owner: 7, Repository: 42, Objective: "Build a tracker", State: "accepted",
		MaxIssues: 10, PlanningCalls: 12, CreatedUnix: 1,
		Ideas: []factory.ProductIdea{
			idea("d555555555555555555555555", "Capture leave requests", "Requests can be recorded"),
			idea("d666666666666666666666666", "Managers review requests", "Decisions are visible"),
		},
		Tasks: []factory.IntentTask{
			{Key: choiceTaskKey(1), Objective: "Capture leave requests", Outcomes: []string{"Requests can be recorded"}, IdeaIDs: []string{"d555555555555555555555555"}},
			{Key: choiceTaskKey(2), Objective: "Validate leave requests", Outcomes: []string{"Invalid requests are rejected"}, IdeaIDs: []string{"d555555555555555555555555"}},
			{Key: choiceTaskKey(3), Objective: "Notify managers", Outcomes: []string{"Managers see pending reviews"}, IdeaIDs: []string{"d666666666666666666666666"}},
			{Key: choiceTaskKey(4), Objective: "Manager decision", Outcomes: []string{"Decisions are visible"}, IdeaIDs: []string{"d555555555555555555555555", "d666666666666666666666666"}, DependsOn: []string{choiceTaskKey(1), choiceTaskKey(2), choiceTaskKey(3)}},
		},
	}
}

func choiceFixture(t *testing.T, r factory.IntentRevision) (*Coordinator, *generatedGraphFixture, *refusingDecompositionChooser) {
	t.Helper()
	return choiceFixtureChain(t, r)
}

// choiceFixtureChain saves a predecessor-ordered revision chain (draft
// first) so successor revisions bind their reviewed predecessor.
func choiceFixtureChain(t *testing.T, chain ...factory.IntentRevision) (*Coordinator, *generatedGraphFixture, *refusingDecompositionChooser) {
	t.Helper()
	if len(chain) == 0 {
		t.Fatal("choice fixture needs its revision chain")
	}
	c := coordinatorFixture(t, nil, nil)
	for _, r := range chain {
		if err := c.Store.SaveIntentRevision(context.Background(), r); err != nil {
			t.Fatal(err)
		}
	}
	head := chain[len(chain)-1]
	if err := c.Store.SavePlanningAllowance(context.Background(), factory.PlanningAllowance{ID: "d999999999999999999999999", Intent: head.Intent, Owner: head.Owner, Amount: 32, Unit: factory.PlanningAllowanceUnit, Ceiling: 32, CreatedUnix: 1}); err != nil {
		t.Fatal(err)
	}
	f := &generatedGraphFixture{revision: 10, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
	c.Graph, c.GraphActorID, c.AcceptanceReads = f, 91, f
	never := &refusingDecompositionChooser{}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: never}
	return c, f, never
}

func chosenFor(r factory.IntentRevision) ChosenIntentPlan {
	return ChosenIntentPlan{
		Intent: r.Intent, Revision: r.ID, Operation: factory.IntentPlanningOptions,
		Key: "candidate-one", Tasks: append([]factory.IntentTask(nil), r.Tasks...),
	}
}

func submittedKeys(tasks []factory.IntentTask) []string {
	keys := make([]string, 0, len(tasks))
	for _, task := range tasks {
		keys = append(keys, task.Key)
	}
	return keys
}

// The choice path adopts broad roots with a join exactly: every chosen key
// is submitted, every payload create names a chosen key, and no serving
// judgment evaluates.
func TestGenerateFromChoiceAdoptsBroadJoinExactly(t *testing.T) {
	r := broadJoinAcceptedRevision()
	c, f, never := choiceFixture(t, r)
	g, err := generateChoiceMechanics(t, c, context.Background(), r.Intent, r.Owner, chosenFor(r))
	if err != nil {
		t.Fatal(err)
	}
	if g.State != "completed" || never.calls != 0 {
		t.Fatalf("state=%s judgmentCalls=%d", g.State, never.calls)
	}
	if fmt.Sprintf("%v", submittedKeys(g.SubmittedTasks)) != fmt.Sprintf("%v", submittedKeys(r.Tasks)) {
		t.Fatalf("submitted=%v chosen=%v", submittedKeys(g.SubmittedTasks), submittedKeys(r.Tasks))
	}
	var payload sdk.IssueGraphPayload
	if err := json.Unmarshal(g.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Creates) != len(r.Tasks) || len(payload.Dependencies) != 3 {
		t.Fatalf("payload=%+v", payload)
	}
	chosen := map[string]bool{}
	for _, key := range submittedKeys(r.Tasks) {
		chosen[key] = true
	}
	for _, create := range payload.Creates {
		if !chosen[create.Key] {
			t.Fatalf("payload creates unchosen key %s", create.Key)
		}
	}
	if f.submissions != 1 {
		t.Fatalf("submissions=%d", f.submissions)
	}
	bindings, err := c.Store.IntentTaskBindings(context.Background(), r.ID)
	if err != nil || len(bindings) != len(r.Tasks) {
		t.Fatalf("bindings=%v err=%v", bindings, err)
	}
}

// The tiny cohesive case (one task, no edges) takes the same exact path.
func TestGenerateFromChoiceAdoptsTinyGraphExactly(t *testing.T) {
	single := factory.IntentRevision{
		ID: "d777777777777777777777777", Intent: "d888888888888888888888888",
		Owner: 7, Repository: 42, Objective: "Build a tracker", State: "accepted",
		MaxIssues: 10, PlanningCalls: 12, CreatedUnix: 1,
		Ideas: []factory.ProductIdea{{ID: "d999999999999999999999999", Description: "Track requests", Consequence: "Clear ownership", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7}},
		Tasks: []factory.IntentTask{{Key: choiceTaskKey(9), Objective: "Request entry", Outcomes: []string{"Capture requests"}, IdeaIDs: []string{"d999999999999999999999999"}}},
	}
	c, _, never := choiceFixture(t, single)
	chosen := chosenFor(single)
	g, err := generateChoiceMechanics(t, c, context.Background(), single.Intent, single.Owner, chosen)
	if err != nil {
		t.Fatal(err)
	}
	if g.State != "completed" || never.calls != 0 || len(g.SubmittedTasks) != 1 || g.SubmittedTasks[0].Key != single.Tasks[0].Key {
		t.Fatalf("generation=%+v judgmentCalls=%d", g, never.calls)
	}
}

// Parity: the advised serving entry and the choice entry submit the same
// graph for the same accepted direction.
func TestGenerateChoiceMatchesAdvisedServingGraph(t *testing.T) {
	c, _, r := acceptedGenerationFixture(t)
	advised, err := generateGraphMechanics(t, c, context.Background(), r.Intent, r.Owner)
	if err != nil {
		t.Fatal(err)
	}
	other, _, _ := acceptedGenerationFixture(t)
	never := &refusingDecompositionChooser{}
	other.Planning = &PlanningJudgments{Journal: other.Store, Funding: other.Store, Chooser: never}
	chosen, err := generateChoiceMechanics(t, other, context.Background(), r.Intent, r.Owner, chosenFor(r))
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%v", submittedKeys(advised.SubmittedTasks)) != fmt.Sprintf("%v", submittedKeys(chosen.SubmittedTasks)) {
		t.Fatalf("advised=%v chosen=%v", submittedKeys(advised.SubmittedTasks), submittedKeys(chosen.SubmittedTasks))
	}
	var a, b sdk.IssueGraphPayload
	if err := json.Unmarshal(advised.Payload, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(chosen.Payload, &b); err != nil {
		t.Fatal(err)
	}
	if len(a.Creates) != len(b.Creates) || len(a.Dependencies) != len(b.Dependencies) {
		t.Fatalf("payloads diverge: %+v vs %+v", a, b)
	}
}

// candidateChoiceStub is the deterministic stand-in for the funded
// candidate choice: it selects one planner alternative with an explicit
// confidence, counting calls so tests prove reservation caching (replays
// spend nothing further).
type candidateChoiceStub struct {
	selected   string
	confidence float64
	calls      int
	err        error
}

func (s *candidateChoiceStub) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	s.calls++
	if s.err != nil {
		return factory.PriorityAnswer{}, s.err
	}
	selected := s.selected
	if _, ok := choices[selected]; !ok {
		selected = "cand-a"
	}
	share := (1 + s.confidence) / 2
	answer := factory.PriorityAnswer{Selected: selected, Model: "synthetic-model", Confidence: s.confidence, Probabilities: map[string]float64{}}
	for id := range choices {
		answer.Probabilities[id] = (1 - share) / float64(len(choices)-1)
	}
	answer.Probabilities[selected] = share
	return answer, nil
}

// choiceDraftHead builds the reviewed draft behind an accepted revision:
// same intent and choices, draft state, no tasks yet.
func choiceDraftHead(t *testing.T, r factory.IntentRevision, draftID string) factory.IntentRevision {
	t.Helper()
	draft := r
	draft.ID = draftID
	draft.Predecessor = ""
	draft.State = "draft"
	draft.Tasks = nil
	draft.EffectiveOutcomes = []string{r.Ideas[0].ID, r.Ideas[1].ID}
	// The decomposition-options operation needs a supported representative
	// objective for its authored alternatives; the candidate tasks bind
	// the draft's ideas, not its objective text.
	draft.Objective = "Build a leave approval workflow so managers can review time-off requests"
	return draft
}

func choicePlannerInput(t *testing.T, draft factory.IntentRevision) factory.SemanticPlannerInput {
	t.Helper()
	content, err := factory.ComposeProposalContent(draft)
	if err != nil {
		t.Fatalf("proposal content: %v", err)
	}
	return factory.SemanticPlannerInput{
		Operation:    factory.IntentPlanningOptions,
		Head:         draft,
		Content:      content,
		Alternatives: factory.DescribeAlternatives(draft.Objective),
		Inspected: []factory.InspectedSource{
			{Path: "internal/factory/intent.go", Commit: "cccccccccccccccccccccccccccccccccccccccc", Evidence: "Draft head carries attributed effective scope with no tasks yet"},
		},
	}
}

func choicePlannerOutput(draft factory.IntentRevision, keys ...string) factory.SemanticPlannerOutput {
	tasks := append([]factory.IntentTask(nil), broadJoinAcceptedRevision().Tasks...)
	out := factory.SemanticPlannerOutput{Intent: draft.Intent, Revision: draft.ID, Operation: factory.IntentPlanningOptions}
	for _, key := range keys {
		out.Candidates = append(out.Candidates, factory.SemanticCandidate{Key: key, Rationale: "Cohesive feature issues with independent roots joined once.", Tasks: tasks})
	}
	return out
}

// A single planner candidate adopts with no comparative call: the funded
// chooser never evaluates.
func TestChoosePlannedCandidateSingleNeedsNoComparativeCall(t *testing.T) {
	r := broadJoinAcceptedRevision()
	c, _, _ := choiceFixture(t, r)
	stub := &candidateChoiceStub{selected: "cand-a", confidence: 1}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: stub}
	draft := choiceDraftHead(t, r, "d555555555555555555555555")
	in := choicePlannerInput(t, draft)
	chosen, err := c.ChoosePlannedCandidate(context.Background(), r, in, choicePlannerOutput(draft, "cand-a"))
	if err != nil {
		t.Fatal(err)
	}
	if chosen.Key != "cand-a" || chosen.Revision != draft.ID || stub.calls != 0 {
		t.Fatalf("choice=%+v fundedCalls=%d", chosen, stub.calls)
	}
}

// Multiple alternatives resolve through exactly one finite funded choice,
// and resumes replay cached without spending further.
func TestChoosePlannedCandidateMultiUsesOneFundedChoice(t *testing.T) {
	r := broadJoinAcceptedRevision()
	c, _, _ := choiceFixture(t, r)
	stub := &candidateChoiceStub{selected: "cand-b", confidence: 0.9}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: stub}
	draft := choiceDraftHead(t, r, "d555555555555555555555555")
	in := choicePlannerInput(t, draft)
	out := choicePlannerOutput(draft, "cand-a", "cand-b")
	chosen, err := c.ChoosePlannedCandidate(context.Background(), r, in, out)
	if err != nil {
		t.Fatal(err)
	}
	if chosen.Key != "cand-b" || stub.calls != 1 {
		t.Fatalf("choice=%+v fundedCalls=%d", chosen, stub.calls)
	}
	again, err := c.ChoosePlannedCandidate(context.Background(), r, in, out)
	if err != nil || again.Key != "cand-b" || stub.calls != 1 {
		t.Fatalf("replay choice=%+v err=%v fundedCalls=%d", again, err, stub.calls)
	}
}

func TestChoosePlannedCandidateUncertainAndFailedRefuse(t *testing.T) {
	r := broadJoinAcceptedRevision()
	c, _, _ := choiceFixture(t, r)
	draft := choiceDraftHead(t, r, "d555555555555555555555555")
	in := choicePlannerInput(t, draft)
	out := choicePlannerOutput(draft, "cand-a", "cand-b")
	uncertain := &candidateChoiceStub{selected: "cand-a", confidence: 0.2}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: uncertain}
	if _, err := c.ChoosePlannedCandidate(context.Background(), r, in, out); !errors.Is(err, ErrIntentPlanningAmbiguous) {
		t.Fatalf("uncertain choice is not refused: %v", err)
	}
	failed := &candidateChoiceStub{err: errors.New("model 429")}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: failed}
	if _, err := c.ChoosePlannedCandidate(context.Background(), r, in, out); err == nil {
		t.Fatal("failed funded choice admitted")
	}
	attention := factory.SemanticPlannerOutput{Intent: draft.Intent, Revision: draft.ID, Operation: in.Operation, Attention: "A consequential unspecified access policy needs a product decision before decomposition."}
	if _, err := c.ChoosePlannedCandidate(context.Background(), r, in, attention); !errors.Is(err, ErrIntentPlanningAttention) {
		t.Fatalf("attention is not visible: %v", err)
	}
}

// The freeze route, end to end with a deterministic planner stand-in:
// reviewed draft plans one candidate, the accepted successor carries
// different authored tasks, and Generate adopts exactly the planner
// graph — never the authored tasks, never a template rerun.
func TestGenerateDraftPlanAcceptAdoptRoute(t *testing.T) {
	r := broadJoinAcceptedRevision()
	authored := r
	authored.ID = "d999999999999999999999999"
	authored.Predecessor = "d555555555555555555555555"
	rekey := map[string]string{}
	for i := range authored.Tasks {
		rekey[authored.Tasks[i].Key] = choiceTaskKey(11 + i)
		authored.Tasks[i].Key = choiceTaskKey(11 + i)
	}
	for i := range authored.Tasks {
		for j, dep := range authored.Tasks[i].DependsOn {
			authored.Tasks[i].DependsOn[j] = rekey[dep]
		}
	}
	draft := choiceDraftHead(t, r, authored.Predecessor)
	// The accepted successor preserves the reviewed substantive scope;
	// only its authored tasks differ, proving adoption follows the
	// planner graph rather than the authored one.
	authored.Objective = draft.Objective
	c, f, never := choiceFixtureChain(t, draft, authored)
	in := choicePlannerInput(t, draft)
	out := choicePlannerOutput(draft, "cand-a")
	chosen, err := c.ChoosePlannedCandidate(context.Background(), authored, in, out)
	if err != nil {
		t.Fatal(err)
	}
	if chosen.Revision != draft.ID {
		t.Fatalf("choice does not bind the reviewed draft: %+v", chosen)
	}
	g, err := generateChoiceMechanics(t, c, context.Background(), authored.Intent, authored.Owner, chosen)
	if err != nil {
		t.Fatal(err)
	}
	if g.State != "completed" || never.calls != 0 || f.submissions != 1 {
		t.Fatalf("state=%s judgmentCalls=%d submissions=%d", g.State, never.calls, f.submissions)
	}
	for i, task := range g.SubmittedTasks {
		if task.Key != out.Candidates[0].Tasks[i].Key {
			t.Fatalf("submitted[%d]=%s planner=%s", i, task.Key, out.Candidates[0].Tasks[i].Key)
		}
		if task.Key == authored.Tasks[i].Key {
			t.Fatalf("submitted[%d] rides authored tasks", i)
		}
	}
}

func TestGenerateFromChoiceRefusesBeforeObservation(t *testing.T) {
	r := broadJoinAcceptedRevision()
	c, f, _ := choiceFixture(t, r)
	good := chosenFor(r)
	for name, mutate := range map[string]func(*ChosenIntentPlan){
		"foreign intent":     func(ch *ChosenIntentPlan) { ch.Intent = "d999999999999999999999999" },
		"stale revision":     func(ch *ChosenIntentPlan) { ch.Revision = "d999999999999999999999999" },
		"wrong operation":    func(ch *ChosenIntentPlan) { ch.Operation = factory.IntentPlanningPropose },
		"out-of-scope tasks": func(ch *ChosenIntentPlan) { ch.Tasks[0].IdeaIDs = []string{"d999999999999999999999999"} },
		"empty tasks":        func(ch *ChosenIntentPlan) { ch.Tasks = nil },
	} {
		bad := good
		bad.Tasks = append([]factory.IntentTask(nil), good.Tasks...)
		mutate(&bad)
		if _, err := generateChoiceMechanics(t, c, context.Background(), r.Intent, r.Owner, bad); err == nil {
			t.Fatalf("%s admitted", name)
		}
	}
	if f.submissions != 0 {
		t.Fatalf("refused choices submitted %d", f.submissions)
	}
	if _, err := c.Store.IntentGeneration(context.Background(), r.ID); err == nil {
		t.Fatal("refused choice recorded a generation")
	}
}
