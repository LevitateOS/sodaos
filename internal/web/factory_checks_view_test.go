package web

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

// TestFactoryIssueChecksView proves the work-status route explains the
// latest check verdict: an unassessed PR shows no verdict, and a
// recorded assessment renders its exact head/base bindings, verdict and
// per-check results.
func TestFactoryIssueChecksView(t *testing.T) {
	s, _ := issueWebFixture(t)
	decision := "d0123456789abcdef01234567"
	admit := lifecycleAction(t, s, "POST", "/api/repositories/7/factory/issues/3/acceptances", issueAcceptanceBody(factory.NewID(), decision, ""), "alice")
	if admit.Code != 201 {
		t.Fatal(admit.Code, admit.Body.String())
	}
	plain := lifecycleAction(t, s, "GET", "/api/repositories/7/factory/issues/3", "", "alice")
	if plain.Code != 200 {
		t.Fatal(plain.Code, plain.Body.String())
	}
	var absent struct {
		Checks *struct{} `json:"checks"`
	}
	decodeBody(t, plain, &absent)
	if absent.Checks != nil {
		t.Fatal("unassessed PR shows a check verdict")
	}
	checks := []string{"st11-build", "st11-review-gate"}
	assessment := factory.CheckAssessment{
		Results: []factory.CheckResult{
			{Context: "st11-build", State: "success", Passed: true},
			{Context: "st11-review-gate", State: "success", Passed: true},
		},
		Checks:           checks,
		Repository:       7,
		PRNumber:         3,
		PRID:             11,
		IssueID:          13,
		PolicyRevision:   4,
		NativeRev:        41,
		AssessedUnix:     1700000000,
		ObservedContexts: 2,
		HeadRef:          "refs/heads/soda/factory/candidate",
		BaseRef:          "refs/heads/main",
		HeadOID:          strings.Repeat("a", 40),
		BaseOID:          strings.Repeat("b", 40),
		ChecksDigest:     factory.ChecksDigest(checks),
		Verdict:          factory.CheckPass,
		Reason:           factory.CheckReasonPass,
	}
	if _, err := s.Store.RecordCheckAssessment(t.Context(), assessment); err != nil {
		t.Fatal(err)
	}
	view := lifecycleAction(t, s, "GET", "/api/repositories/7/factory/issues/3", "", "alice")
	if view.Code != 200 {
		t.Fatal(view.Code, view.Body.String())
	}
	var seen struct {
		Checks *struct {
			Results []struct {
				Context string `json:"context"`
				State   string `json:"state"`
				Passed  bool   `json:"passed"`
			} `json:"results"`
			Resolution string `json:"resolution"`
			Verdict    string `json:"verdict"`
			Reason     string `json:"reason"`
			HeadOID    string `json:"head_oid"`
			BaseOID    string `json:"base_oid"`
			PRNumber   string `json:"pr_number"`
		} `json:"checks"`
	}
	decodeBody(t, view, &seen)
	if seen.Checks == nil {
		t.Fatal("recorded check verdict is missing from work status")
	}
	if seen.Checks.Verdict != factory.CheckPass || seen.Checks.Reason != factory.CheckReasonPass {
		t.Fatalf("checks: %+v", seen.Checks)
	}
	if seen.Checks.HeadOID != strings.Repeat("a", 40) || seen.Checks.BaseOID != strings.Repeat("b", 40) || seen.Checks.PRNumber != "3" {
		t.Fatalf("bindings: %+v", seen.Checks)
	}
	if len(seen.Checks.Results) != 2 || !seen.Checks.Results[0].Passed || !seen.Checks.Results[1].Passed {
		t.Fatalf("results: %+v", seen.Checks.Results)
	}
	if seen.Checks.Resolution == "" {
		t.Fatal("check verdict lacks its resolution")
	}
	assertNoSecrets(t, view.Body.String())
}
