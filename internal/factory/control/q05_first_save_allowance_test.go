package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// TestFirstSaveAllowanceProducerExploreBoundary proves the Q05 funding
// boundary at the Store/control seam: a first-saved draft whose planning
// control only sets metadata cannot fund exploration until an independent
// allowance head is authorized; after authorization one judgment spends
// exactly one reservation while the intent head stays put. Declared
// external judgment: assumptionChooser{consistent, 0.9}.
func TestFirstSaveAllowanceProducerExploreBoundary(t *testing.T) {
	c := coordinatorFixture(t, nil, nil)
	ctx := context.Background()
	ideas := []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept"},
	}
	r := factory.IntentRevision{ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 7, Objective: "Time-off requests my manager can approve", State: "draft", MaxIssues: 10, PlanningCalls: 8, CreatedUnix: 1, Ideas: ideas}
	if err := c.Store.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	snapshot := strings.Repeat("a", 64)
	funded, err := c.Store.ReserveIntentJudgment(ctx, r, snapshot)
	if funded || !errors.Is(err, store.ErrPlanningAllowanceExhausted) {
		t.Fatal("first exploration funded without an allowance head")
	}
	if err := c.Store.SavePlanningAllowance(ctx, factory.PlanningAllowance{ID: "d999999999999999999999999", Intent: r.Intent, Owner: r.Owner, Amount: 8, Unit: factory.PlanningAllowanceUnit, Ceiling: 8, CreatedUnix: 1}); err != nil {
		t.Fatal(err)
	}
	funded, err = c.Store.ReserveIntentJudgment(ctx, r, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !funded {
		t.Fatal("authorized allowance head refused its first reservation")
	}
	chooser := &assumptionChooser{selected: "consistent", confidence: 0.9}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := factory.ConsolidateContent(content, "shared_review", strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	report, err := c.JudgeConsistency(ctx, r, content, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Mismatches) != 0 || chooser.calls != 1 {
		t.Fatalf("report=%+v calls=%d", report, chooser.calls)
	}
	head, err := c.Store.IntentRevision(ctx, r.Intent, r.Owner)
	if err != nil {
		t.Fatal(err)
	}
	if head.ID != r.ID {
		t.Fatalf("allowance operations moved the intent head to %s", head.ID)
	}
}
