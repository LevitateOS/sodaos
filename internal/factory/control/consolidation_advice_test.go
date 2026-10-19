package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestJudgeConsolidationRecommendsOneCoherentProposal(t *testing.T) {
	ideas := []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept"},
		{ID: "d666666666666666666666666", Description: "Guess the approver", Consequence: "Wrong routing", Assumption: true, Decision: "edit", Correction: "The direct manager approves"},
	}
	c, r := draftAssumptionFixture(t, ideas, 5)
	chooser := &assumptionChooser{selected: "shared_review", confidence: 0.8}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := c.JudgeConsolidation(context.Background(), r, content)
	if err != nil {
		t.Fatal(err)
	}
	if err := proposal.Validate(); err != nil {
		t.Fatal(err)
	}
	if proposal.Recommended != "shared_review" || len(proposal.Milestones) != 4 || len(proposal.Outcomes) != 2 || len(proposal.Constraints) != 1 {
		t.Fatalf("proposal=%+v", proposal)
	}
	again, err := c.JudgeConsolidation(context.Background(), r, content)
	if err != nil || again.Recommended != "shared_review" || chooser.calls != 1 {
		t.Fatalf("reuse calls=%d err=%v", chooser.calls, err)
	}
	if again.Advice != proposal.Advice {
		t.Fatal("provenance fingerprint changed across reuse")
	}
}

func TestJudgeConsolidationRefusesUnsupportedContent(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Objective: "Build a database storage engine", State: "draft", MaxIssues: 10, PlanningCalls: 5, CreatedUnix: 1}
	if err := c.Store.SaveIntentRevision(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if err := c.Store.SavePlanningAllowance(context.Background(), factory.PlanningAllowance{ID: "d999999999999999999999999", Intent: r.Intent, Owner: r.Owner, Amount: 5, Unit: factory.PlanningAllowanceUnit, Ceiling: 5, CreatedUnix: 1}); err != nil {
		t.Fatal(err)
	}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: &assumptionChooser{selected: "shared_review", confidence: 1}}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.JudgeConsolidation(context.Background(), r, content); err == nil {
		t.Fatal("unsupported content consolidated")
	} else if errors.Is(err, ErrPriorityUnavailable) {
		t.Fatalf("unsupported misreported as unavailable: %v", err)
	}
}
