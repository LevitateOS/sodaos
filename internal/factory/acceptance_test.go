package factory

import (
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func acceptanceFixture() Acceptance {
	return Acceptance{
		ID: "d0123456789abcdef01234567", Repository: 7, IssueIndex: "3", Approver: 5, NativeRev: 41,
		TitleDigest: strings.Repeat("a", 64), ContentDigest: strings.Repeat("b", 64), ContentVersion: 2,
		Sources:       []SelectedSource{{ID: "11", ContentVersion: 0, Digest: strings.Repeat("c", 64)}},
		Prerequisites: []AcceptedPrerequisite{{Occurrence: "21", DependsOn: "9", EndpointRepo: 7, EndpointIssue: 2, Outcome: PrereqResult}},
		Resolutions:   []SelectedSource{{ID: "12", ContentVersion: 1, Digest: strings.Repeat("d", 64)}},
	}
}

func TestAcceptanceValidateAcceptsFullDecision(t *testing.T) {
	if err := acceptanceFixture().Validate(); err != nil {
		t.Fatal(err)
	}
	bare := acceptanceFixture()
	bare.Sources, bare.Prerequisites, bare.Resolutions = nil, nil, nil
	if err := bare.Validate(); err != nil {
		t.Fatalf("bare adoption: %v", err)
	}
	code := acceptanceFixture()
	code.Prerequisites = []AcceptedPrerequisite{{Occurrence: "21", DependsOn: "9", EndpointRepo: 7, EndpointIssue: 2, Outcome: PrereqCode, PrereqAcceptance: "d123456789abcdef012345678"}}
	if err := code.Validate(); err != nil {
		t.Fatalf("code route: %v", err)
	}
}

func TestAcceptanceValidateRejectsBadShape(t *testing.T) {
	base := acceptanceFixture()
	for name, mutate := range map[string]func(*Acceptance){
		"id":          func(a *Acceptance) { a.ID = "short" },
		"predecessor": func(a *Acceptance) { a.Predecessor = a.ID },
		"repository":  func(a *Acceptance) { a.Repository = 0 },
		"issue":       func(a *Acceptance) { a.IssueIndex = "0" },
		"approver":    func(a *Acceptance) { a.Approver = 0 },
		"revision":    func(a *Acceptance) { a.NativeRev = 0 },
		"title":       func(a *Acceptance) { a.TitleDigest = "short" },
		"content":     func(a *Acceptance) { a.ContentDigest = "" },
		"version":     func(a *Acceptance) { a.ContentVersion = -1 },
		"source":      func(a *Acceptance) { a.Sources[0].ID = "x" },
		"source_dup":  func(a *Acceptance) { a.Sources = append(a.Sources, a.Sources[0]) },
		"edge":        func(a *Acceptance) { a.Prerequisites[0].Occurrence = "" },
		"edge_dup":    func(a *Acceptance) { a.Prerequisites = append(a.Prerequisites, a.Prerequisites[0]) },
		"endpoint":    func(a *Acceptance) { a.Prerequisites[0].EndpointIssue = 0 },
		"outcome":     func(a *Acceptance) { a.Prerequisites[0].Outcome = "waived" },
		"code_open":   func(a *Acceptance) { a.Prerequisites[0].Outcome = PrereqCode },
		"result_ref":  func(a *Acceptance) { a.Prerequisites[0].PrereqAcceptance = "d123456789abcdef012345678" },
		"initial":     func(a *Acceptance) { a.Initial = true },
	} {
		a := base
		a.Sources = append([]SelectedSource(nil), base.Sources...)
		a.Prerequisites = append([]AcceptedPrerequisite(nil), base.Prerequisites...)
		mutate(&a)
		if err := a.Validate(); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	initial := Acceptance{ID: "d0123456789abcdef01234567", Repository: 7, IssueIndex: "3", Approver: 5, NativeRev: 41,
		TitleDigest: strings.Repeat("a", 64), ContentDigest: strings.Repeat("b", 64), Initial: true}
	if err := initial.Validate(); err != nil {
		t.Fatalf("initial: %v", err)
	}
}

func TestInitialAcceptanceIDIsStable(t *testing.T) {
	first, second := InitialAcceptanceID(7, "3"), InitialAcceptanceID(7, "3")
	if first != second || !project.ValidDecisionID(first) {
		t.Fatalf("unstable: %q %q", first, second)
	}
	if other := InitialAcceptanceID(7, "4"); other == first {
		t.Fatalf("collision: %q", other)
	}
}
