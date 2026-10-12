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
	if len(mismatches) != 1 || mismatches[0].Class != MismatchEffectAdded || !strings.Contains(mismatches[0].Detail, "email") {
		t.Fatalf("mismatches=%+v", mismatches)
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
