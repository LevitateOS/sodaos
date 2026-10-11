package factory

import (
	"strings"
	"testing"
)

func supportedContent(t *testing.T) ProposalContent {
	t.Helper()
	r := validIntent()
	r.State = "draft"
	r.Repository = 0
	r.Tasks = nil
	r.Objective = "Time-off requests my manager can approve"
	r.Ideas = []ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept"},
		{ID: "d666666666666666666666666", Description: "Guess the approver", Consequence: "Wrong routing", Assumption: true, Decision: "edit", Correction: "The direct manager approves"},
		{ID: "d777777777777777777777777", Description: "Email everyone", Consequence: "Noise", Decision: "decline"},
	}
	content, err := ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func TestConsolidateContentPreservesDirectionAndMilestones(t *testing.T) {
	content := supportedContent(t)
	proposal, err := ConsolidateContent(content, "shared_review", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if err := proposal.Validate(); err != nil {
		t.Fatal(err)
	}
	if proposal.Recommended != "shared_review" || proposal.Objective != content.Objective {
		t.Fatalf("proposal=%+v", proposal)
	}
	if len(proposal.Outcomes) != 2 || proposal.Outcomes[1] != "The direct manager approves" {
		t.Fatalf("outcomes=%q", proposal.Outcomes)
	}
	if len(proposal.Constraints) != 1 || proposal.Constraints[0] != "The direct manager approves" {
		t.Fatalf("constraints=%q", proposal.Constraints)
	}
	if len(proposal.Excluded) != 1 || proposal.Excluded[0] != "d777777777777777777777777" {
		t.Fatalf("excluded=%q", proposal.Excluded)
	}
	if len(proposal.Milestones) != 4 || !strings.HasPrefix(proposal.Milestones[0].Name, "The person: ") || proposal.Milestones[2].Name != "If Approved" {
		t.Fatalf("milestones=%+v", proposal.Milestones)
	}
}

func TestConsolidateContentRejectsUnknownRecommendation(t *testing.T) {
	content := supportedContent(t)
	if _, err := ConsolidateContent(content, "invented", strings.Repeat("a", 64)); err == nil {
		t.Fatal("unknown recommendation accepted")
	}
	if _, err := ConsolidateContent(content, "shared_review", "not-a-digest"); err == nil {
		t.Fatal("unprovenanced advice accepted")
	}
	plain := draftRevision()
	plain.Objective = "Build a database storage engine"
	plain.Ideas = nil
	unsupported, err := ComposeProposalContent(plain)
	if err != nil {
		t.Fatal(err)
	}
	if unsupported.Supported {
		t.Fatal("expected unsupported content")
	}
	if _, err := ConsolidateContent(unsupported, "shared_review", strings.Repeat("a", 64)); err == nil {
		t.Fatal("unsupported content consolidated")
	}
}
