package factory

import (
	"strings"
	"testing"
)

func TestCheckProposalEffectsCatchesSilentAdditions(t *testing.T) {
	content := supportedContent(t)
	proposal, err := ConsolidateContent(content, "shared_review", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if mismatches := CheckProposalEffects(content, proposal); len(mismatches) != 0 {
		t.Fatalf("clean proposal mismatches=%+v", mismatches)
	}
	smuggled := proposal
	smuggled.Alternative.Steps[0].Action = "describes what they need and gets an email receipt"
	mismatches := CheckProposalEffects(content, smuggled)
	if len(mismatches) != 1 || mismatches[0].Class != MismatchEffectAdded {
		t.Fatalf("mismatches=%+v", mismatches)
	}
	if !strings.Contains(mismatches[0].Detail, "d777777777777777777777777") || !strings.Contains(mismatches[0].Detail, "Email everyone") {
		t.Fatalf("decline not cited=%q", mismatches[0].Detail)
	}
	sourced := content
	sourced.Facts = append(append([]string(nil), content.Facts...), "Send an email receipt for every request")
	if mismatches := CheckProposalEffects(sourced, smuggled); len(mismatches) != 0 {
		t.Fatalf("sourced effect flagged=%+v", mismatches)
	}
	gapped, err := ConsolidateContent(content, "shared_review", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	gapped.Outcomes[0] = "Something else entirely"
	mismatches = CheckProposalEffects(content, gapped)
	if len(mismatches) != 1 || mismatches[0].Class != MismatchOutcomeGap {
		t.Fatalf("mismatches=%+v", mismatches)
	}
}

func TestCheckProposalEffectsIgnoresIncidentalVocabulary(t *testing.T) {
	r := draftRevision()
	r.Objective = "Time-off requests my manager can approve"
	r.Ideas = []ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept"},
		{ID: "d666666666666666666666666", Description: "Email the reviewer", Consequence: "Notify the person", Decision: "decline"},
	}
	content, err := ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ConsolidateContent(content, "shared_review", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if mismatches := CheckProposalEffects(content, proposal); len(mismatches) != 0 {
		t.Fatalf("incidental reviewer/person vocabulary flagged=%+v", mismatches)
	}
}

func TestCheckProposalEffectsDefersSemanticOverlapToJudgment(t *testing.T) {
	r := draftRevision()
	r.Objective = "Time-off requests my manager can approve"
	r.Ideas = []ProductIdea{
		{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept"},
		{ID: "d666666666666666666666666", Description: "The reviewer decides every request", Consequence: "Reviewer time on every submission", Decision: "decline"},
	}
	content, err := ComposeProposalContent(r)
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := ConsolidateContent(content, "shared_review", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if mismatches := CheckProposalEffects(content, proposal); len(mismatches) != 0 {
		t.Fatalf("semantic resemblance decided deterministically=%+v", mismatches)
	}
}

func TestConsistencyReportBindsRevisionAndClasses(t *testing.T) {
	report := ConsistencyReport{Producer: AuthoredProducer, Intent: "d222222222222222222222222", Revision: "d111111111111111111111111", Confidence: 0.9, Advice: strings.Repeat("b", 64)}
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
	report.Mismatches = []Mismatch{{Class: "invented", Detail: "No such class"}}
	if report.Validate() == nil {
		t.Fatal("unknown mismatch class accepted")
	}
}
