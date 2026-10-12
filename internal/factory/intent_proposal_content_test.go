package factory

import (
	"strings"
	"testing"
)

func draftRevision() IntentRevision {
	r := validIntent()
	r.State = "draft"
	r.Repository = 0
	r.Tasks = nil
	return r
}

func TestComposeProposalContentBindsFactsChoicesAndUnknowns(t *testing.T) {
	r := draftRevision()
	r.Objective = "I need a time-off request my manager can approve"
	r.Ideas = []ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept"},
		{ID: "d666666666666666666666666", Description: "Guess the approver", Consequence: "Wrong routing", Assumption: true, Decision: "edit", Correction: "The direct manager approves"},
		{ID: "d777777777777777777777777", Description: "Email everyone", Consequence: "Noise", Decision: "decline"},
	}
	first, err := ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Validate(); err != nil {
		t.Fatal(err)
	}
	if !first.Supported || first.Objective != r.Objective || len(first.Alternatives) != 2 || len(first.Unknowns) == 0 {
		t.Fatalf("unexpected content %+v", first)
	}
	if len(first.Facts) != 2 || first.Facts[0] != r.Objective || first.Facts[1] != "The direct manager approves" {
		t.Fatalf("facts=%q", first.Facts)
	}
	if len(first.Excluded) != 1 || first.Excluded[0] != "d777777777777777777777777" {
		t.Fatalf("excluded=%q", first.Excluded)
	}
	if len(first.Exclusions) != 1 || first.Exclusions[0].Idea != "d777777777777777777777777" || first.Exclusions[0].Description != "Email everyone" || first.Exclusions[0].Consequence != "Noise" {
		t.Fatalf("exclusions=%+v", first.Exclusions)
	}
	for _, alternative := range first.Alternatives {
		if len(alternative.Outcomes) != 2 || alternative.Outcomes[0] != r.Objective || alternative.Outcomes[1] != "The direct manager approves" {
			t.Fatalf("outcomes=%q", alternative.Outcomes)
		}
	}
	second, err := ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	if first.Producer != second.Producer || len(first.Alternatives) != len(second.Alternatives) || first.Alternatives[0].Key != second.Alternatives[0].Key {
		t.Fatal("proposal content is not deterministic")
	}
}

func TestComposeProposalContentPreservesDeclineMeanings(t *testing.T) {
	r := draftRevision()
	r.Objective = "I need a time-off request my manager can approve"
	r.Ideas = []ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept", Source: "requested", Provenance: "human", Actor: 1},
		{ID: "d666666666666666666666666", Description: "Guess the approver", Consequence: "Wrong routing", Assumption: true, Decision: "decline", Source: "assumption", Provenance: "autonomous"},
		{ID: "d777777777777777777777777", Description: "Email everyone", Consequence: "Noise", Decision: "decline", Source: "requested", Provenance: "human", Actor: 1},
	}
	content, err := ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := content.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(content.Exclusions) != 2 || content.Exclusions[0].Idea != "d666666666666666666666666" || content.Exclusions[0].Provenance != "autonomous" {
		t.Fatalf("autonomous exclusion=%+v", content.Exclusions)
	}
	if content.Exclusions[1].Idea != "d777777777777777777777777" || content.Exclusions[1].Description != "Email everyone" || content.Exclusions[1].Consequence != "Noise" || content.Exclusions[1].Provenance != "human" {
		t.Fatalf("human exclusion=%+v", content.Exclusions)
	}
	dropped := content
	dropped.Exclusions = dropped.Exclusions[:1]
	if dropped.Validate() == nil {
		t.Fatal("excluded ID without meaning accepted")
	}
	swapped := content
	swapped.Exclusions[0], swapped.Exclusions[1] = swapped.Exclusions[1], swapped.Exclusions[0]
	if swapped.Validate() == nil {
		t.Fatal("misaligned exclusion meanings accepted")
	}
}

func TestComposeProposalContentHonorsUnsupportedObjectives(t *testing.T) {
	r := draftRevision()
	r.Objective = "Build a database storage engine"
	r.Ideas = nil
	content, err := ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := content.Validate(); err != nil {
		t.Fatal(err)
	}
	if content.Supported || len(content.Alternatives) != 0 || content.Objective != r.Objective {
		t.Fatalf("unexpected content %+v", content)
	}
}

func TestComposeProposalContentRefusesNondraftAndInvalidRevisions(t *testing.T) {
	r := validIntent()
	if _, err := ComposeProposalContent(r); err == nil {
		t.Fatal("accepted revision composed")
	}
	r = draftRevision()
	r.ID = "bad"
	if _, err := ComposeProposalContent(r); err == nil {
		t.Fatal("invalid revision composed")
	}
	content, err := ComposeProposalContent(draftRevision())
	if err != nil {
		t.Fatal(err)
	}
	content.Facts[0] = strings.Repeat("x", 10)
	if content.Validate() == nil {
		t.Fatal("rewritten objective fact accepted")
	}
}
