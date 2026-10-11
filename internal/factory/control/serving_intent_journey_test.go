package control

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
)

// The native boundary updates the serving fixture from the exact graph packet,
// not a second manually authored issue graph or acceptance decision.
type journeyGraph struct {
	*generatedGraphFixture
	inputs      *completionInputs
	coordinator *Coordinator
}

func (g *journeyGraph) SubmitGraph(ctx context.Context, operation factory.IntentGeneration, repository int64) (sdk.OperationRecord, error) {
	stored, err := g.coordinator.Store.IntentGeneration(ctx, operation.RevisionID)
	if err != nil || stored.OperationID != operation.OperationID || string(stored.Payload) != string(operation.Payload) {
		return sdk.OperationRecord{}, fmt.Errorf("graph intent was not persisted before native admission: %w", err)
	}
	result, err := g.generatedGraphFixture.SubmitGraph(ctx, operation, repository)
	if err != nil {
		return result, err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.inputs.mu.Lock()
	defer g.inputs.mu.Unlock()
	acceptance := g.inputs.AcceptanceSource.(*fakeEvidenceSource)
	dispatch := g.inputs.DispatchReads.(*fakeDispatchReads)
	g.inputs.revision.Store(g.revision)
	for _, index := range []int64{31, 32} {
		native := g.issues[index]
		issue := native.Issue
		key := fmt.Sprintf("%d/%d", repository, index)
		acceptance.evidence[key] = AcceptanceEvidence{Revision: g.revision, Issue: AcceptanceIssueView{ID: issue.ID, Index: issue.Index, TitleDigest: issue.TitleDigest, ContentDigest: issue.ContentDigest, ContentVer: issue.ContentVersion, TitleOccurrence: generatedTitleOccurrence(issue), LifecycleOccurrence: factory.OriginalOccurrence, PosterID: issue.Provenance.PosterID, CreatedUnix: time.Now().Unix(), Visible: true}, Dependencies: generatedEdges(native.Dependencies)}
		dispatch.inputs[key] = DispatchInputs{Revision: g.revision, Issue: DispatchIssue{Title: issue.Title, Body: issue.Content, TitleDigest: issue.TitleDigest, ContentDigest: issue.ContentDigest, ContentVer: issue.ContentVersion, TitleOccurrence: generatedTitleOccurrence(issue), LifecycleOccurrence: factory.OriginalOccurrence, Visible: true}, Tip: strings.Repeat("c", 40), TipRef: "refs/heads/main"}
	}
	return result, nil
}

type journeyChooser struct {
	planner       servingPlanChooser
	decomposition generatedGraphChooser
}

func (c *journeyChooser) Select(ctx context.Context, state any, question string, choices map[string]any) (factory.PriorityAnswer, error) {
	if _, ok := state.(factory.PlanningInput); ok {
		return c.planner.Select(ctx, state, question, choices)
	}
	return c.decomposition.Select(ctx, state, question, choices)
}

func TestServingAcceptedIntentCreatesPlansExecutesReviewsMergesAndUnblocks(t *testing.T) {
	ctx := context.Background()
	chooser := &journeyChooser{}
	var coordinator *Coordinator
	var graph *journeyGraph
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Repository: 7, Objective: "Build a request tracker", State: "accepted", MaxIssues: 10, PlanningCalls: 12, CreatedUnix: 1, Ideas: []factory.ProductIdea{{ID: "d333333333333333333333333", Description: "Capture requests and show them in a list", Consequence: "Requests can be recorded and found", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7}}, Tasks: []factory.IntentTask{{Key: "d444444444444444444444444", Objective: "objective 31; inspect README.md", Outcomes: []string{"Capture a request"}, IdeaIDs: []string{"d333333333333333333333333"}}, {Key: "d555555555555555555555555", Objective: "objective 32; inspect README.md", Outcomes: []string{"Show captured requests"}, IdeaIDs: []string{"d333333333333333333333333"}, DependsOn: []string{"d444444444444444444444444"}}}}
	servingCorrectionAndDependent(t, "", func(c *Coordinator) {
		coordinator = c
		c.Store.PriorityCallBudget = 32
		if err := c.Store.SaveIntentRevision(ctx, r); err != nil {
			t.Fatal(err)
		}
		if err := c.Store.SavePlanningAllowance(ctx, factory.PlanningAllowance{ID: "d999999999999999999999999", Intent: r.Intent, Owner: r.Owner, Amount: 64, Unit: factory.PlanningAllowanceUnit, Ceiling: 64, CreatedUnix: 1}); err != nil {
			t.Fatal(err)
		}
		fixture := &generatedGraphFixture{revision: 41, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
		for index := int64(1); index <= 30; index++ {
			fixture.issues[index] = sdk.NativeSnapshot{}
		}
		graph = &journeyGraph{generatedGraphFixture: fixture, inputs: c.AcceptanceReads.(*completionInputs), coordinator: c}
		c.Graph, c.GraphActorID = graph, 91
		c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
		c.Priority = &PriorityJudgments{Journal: c.Store, Chooser: &priorityFixture{choose: priorityLast}, MaxCalls: 16}
	})
	generation, err := coordinator.Store.IntentGeneration(ctx, r.ID)
	if err != nil || generation.State != "completed" || graph.submissions != 1 {
		t.Fatalf("generation=%+v err=%v effects=%d", generation, err, graph.submissions)
	}
	bindings, err := coordinator.Store.IntentTaskBindings(ctx, r.ID)
	if err != nil || len(bindings) != 2 {
		t.Fatalf("generated provenance=%+v %v", bindings, err)
	}
	for _, binding := range bindings {
		acceptance, err := coordinator.Store.AcceptanceDecision(ctx, binding.AcceptanceID)
		if err != nil || acceptance.Approver != r.Owner || acceptance.IssueIndex != strconv.FormatInt(binding.IssueIndex, 10) {
			t.Fatalf("derived acceptance=%+v %v", acceptance, err)
		}
	}
	chooser.planner.mu.Lock()
	defer chooser.planner.mu.Unlock()
	provisional, final := 0, 0
	for _, input := range chooser.planner.inputs {
		if input.Candidate.Control.Issue == 32 {
			if len(input.Missing) > 0 {
				provisional++
			} else if input.Candidate.Source == strings.Repeat("3", 40) {
				final++
			}
		}
	}
	if provisional != 1 || final != 1 {
		t.Fatalf("generated dependent plans provisional=%d final=%d", provisional, final)
	}
}
