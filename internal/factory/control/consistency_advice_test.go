package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func consistencyFixture(t *testing.T, chooser *assumptionChooser) (*Coordinator, factory.IntentRevision, factory.ProposalContent, factory.ConsolidatedProposal) {
	t.Helper()
	ideas := []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept"},
	}
	c, r := draftAssumptionFixture(t, ideas, 5)
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := factory.ConsolidateContent(content, "shared_review", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	return c, r, content, proposal
}

func TestJudgeConsistencyConfirmsCleanWalkthrough(t *testing.T) {
	chooser := &assumptionChooser{selected: "consistent", confidence: 0.9}
	c, r, content, proposal := consistencyFixture(t, chooser)
	report, err := c.JudgeConsistency(context.Background(), r, content, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(report.Mismatches) != 0 || report.Revision != r.ID || chooser.calls != 1 {
		t.Fatalf("report=%+v calls=%d", report, chooser.calls)
	}
	again, err := c.JudgeConsistency(context.Background(), r, content, proposal)
	if err != nil || chooser.calls != 1 || again.Advice != report.Advice {
		t.Fatalf("reuse calls=%d err=%v", chooser.calls, err)
	}
}

func TestJudgeConsistencyNamesMismatchClasses(t *testing.T) {
	for verdict, class := range map[string]string{"outcome_gap": factory.MismatchOutcomeGap, "summary_drift": factory.MismatchSummaryDrift} {
		chooser := &assumptionChooser{selected: verdict, confidence: 0.8}
		c, r, content, proposal := consistencyFixture(t, chooser)
		report, err := c.JudgeConsistency(context.Background(), r, content, proposal)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Mismatches) != 1 || report.Mismatches[0].Class != class || report.Mismatches[0].Detail == "" {
			t.Fatalf("verdict %s: %+v", verdict, report.Mismatches)
		}
	}
	chooser := &assumptionChooser{selected: "consistent", confidence: 0.2}
	c, r, content, proposal := consistencyFixture(t, chooser)
	report, err := c.JudgeConsistency(context.Background(), r, content, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Mismatches) != 1 || report.Mismatches[0].Class != factory.MismatchUnverified {
		t.Fatalf("uncertain clean: %+v", report.Mismatches)
	}
}

func TestJudgeConsistencyDeclinedOverlapSkipsJudgment(t *testing.T) {
	ideas := []factory.ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept"},
		{ID: "d666666666666666666666666", Description: "The reviewer decides every request", Consequence: "Reviewer time on every submission", Decision: "decline"},
	}
	c, r := draftAssumptionFixture(t, ideas, 5)
	chooser := &assumptionChooser{selected: "consistent", confidence: 1}
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	content, err := factory.ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := factory.ConsolidateContent(content, "shared_review", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	report, err := c.JudgeConsistency(context.Background(), r, content, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Mismatches) != 1 || report.Mismatches[0].Class != factory.MismatchEffectAdded || chooser.calls != 0 {
		t.Fatalf("report=%+v calls=%d", report, chooser.calls)
	}
	if !strings.Contains(report.Mismatches[0].Detail, "d666666666666666666666666") {
		t.Fatalf("detail=%q", report.Mismatches[0].Detail)
	}
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestJudgeConsistencyCodeFindingsSkipJudgment(t *testing.T) {
	chooser := &assumptionChooser{selected: "consistent", confidence: 1}
	c, r, content, proposal := consistencyFixture(t, chooser)
	proposal.Alternative.Steps[0].Action = "describes what they need and gets calendar invites"
	report, err := c.JudgeConsistency(context.Background(), r, content, proposal)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Mismatches) != 1 || report.Mismatches[0].Class != factory.MismatchEffectAdded || chooser.calls != 0 {
		t.Fatalf("report=%+v calls=%d", report, chooser.calls)
	}
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
}
