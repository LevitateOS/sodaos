package factory

import "testing"

func validIntent() IntentRevision {
	return IntentRevision{
		ID: "d111111111111111111111111", Intent: "d222222222222222222222222", Owner: 1, Repository: 42, Objective: "Build a leave tracker", State: "accepted", MaxIssues: 10, PlanningCalls: 3, CreatedUnix: 1,
		Ideas: []ProductIdea{{ID: "d333333333333333333333333", Description: "Track leave requests", Consequence: "Managers review requests", Decision: "accept"}},
		Tasks: []IntentTask{{Key: "d444444444444444444444444", Objective: "Create request form", Outcomes: []string{"Requests can be submitted"}, IdeaIDs: []string{"d333333333333333333333333"}}, {Key: "d555555555555555555555555", Objective: "Create manager view", Outcomes: []string{"Requests can be reviewed"}, IdeaIDs: []string{"d333333333333333333333333"}}},
	}
}

func TestAcceptedIntentPreservesIndependentRoots(t *testing.T) {
	r := validIntent()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if r.Digest() == "" {
		t.Fatal("missing revision digest")
	}
}

func TestAcceptedIntentRejectsUnresolvedAndDeclinedIdeas(t *testing.T) {
	for _, decision := range []string{"pending", "decline", "edit"} {
		r := validIntent()
		r.Ideas[0].Decision = decision
		if r.Validate() == nil {
			t.Fatalf("accepted %s", decision)
		}
	}
}

func TestAcceptedIntentRejectsCyclesAndUnknownDependencies(t *testing.T) {
	r := validIntent()
	r.Tasks[0].DependsOn = []string{"d555555555555555555555555"}
	r.Tasks[1].DependsOn = []string{"d444444444444444444444444"}
	if r.Validate() == nil {
		t.Fatal("accepted cycle")
	}
	r = validIntent()
	r.Tasks[0].DependsOn = []string{"missing"}
	if r.Validate() == nil {
		t.Fatal("accepted missing prerequisite")
	}
}

func TestIntentEditsAndBounds(t *testing.T) {
	r := validIntent()
	r.Ideas[0].Decision = "edit"
	r.Ideas[0].Correction = "Require two reviewers"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.MaxIssues = 1
	if r.Validate() == nil {
		t.Fatal("accepted issue overrun")
	}
}

func TestAcceptedIntentAllowsUnrelatedPendingBranch(t *testing.T) {
	r := validIntent()
	r.Ideas[0].Source = "requested"
	r.Ideas[0].Provenance = "human"
	r.Ideas[0].Actor = 1
	r.Ideas = append(r.Ideas, ProductIdea{ID: "d666666666666666666666666", Description: "Notify by email", Consequence: "Needs mail access", Assumption: true, Decision: "pending", Source: "assumption", Material: true})
	r.EffectiveOutcomes = []string{"d333333333333333333333333"}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Tasks[0].IdeaIDs = []string{"d666666666666666666666666"}
	if r.Validate() == nil {
		t.Fatal("task entered unresolved consequential scope")
	}
}

func TestIntentRejectsForgedHumanAttribution(t *testing.T) {
	r := validIntent()
	r.Ideas[0].Source = "supported"
	r.Ideas[0].Provenance = "human"
	r.Ideas[0].Actor = 2
	r.Ideas[0].Evidence = "reviewed request text"
	if r.Validate() == nil {
		t.Fatal("forged human actor accepted")
	}
	r.Ideas[0].Actor = 1
	r.Ideas[0].Provenance = "autonomous"
	if r.Validate() == nil {
		t.Fatal("human actor with autonomous provenance accepted")
	}
	r.Ideas[0].Provenance = "autonomous"
	r.Ideas[0].Actor = 0
	r.Ideas[0].Decision = "edit"
	r.Ideas[0].Correction = "change"
	if r.Validate() == nil {
		t.Fatal("autonomous correction accepted")
	}
}

func TestIntentRequiresEvidenceAndEffectiveAuthorization(t *testing.T) {
	r := validIntent()
	r.Ideas[0].Source = "supported"
	r.Ideas[0].Provenance = "autonomous"
	if r.Validate() == nil {
		t.Fatal("unsupported evidence choice accepted")
	}
	r.Ideas[0].Evidence = "prior approved request"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.EffectiveOutcomes = []string{"d999999999999999999999999"}
	if r.Validate() == nil {
		t.Fatal("unknown effective outcome accepted")
	}
	r.EffectiveOutcomes = []string{"d333333333333333333333333"}
	r.Tasks[0].IdeaIDs = []string{"d333333333333333333333333"}
	r.Ideas = append(r.Ideas, ProductIdea{ID: "d777777777777777777777777", Description: "Extra", Consequence: "Extra", Decision: "accept", Source: "autonomous", Provenance: "autonomous"})
	r.Tasks = append(r.Tasks, IntentTask{Key: "d888888888888888888888888", Objective: "Extra work", Outcomes: []string{"done"}, IdeaIDs: []string{"d777777777777777777777777"}})
	if r.Validate() == nil {
		t.Fatal("task outside effective outcomes accepted")
	}
}

func TestIntentTerminalDraftStates(t *testing.T) {
	for _, state := range []string{"rejected", "abandoned"} {
		r := validIntent()
		r.State = state
		r.Repository = 0
		r.Tasks = nil
		if err := r.Validate(); err != nil {
			t.Fatalf("%s: %v", state, err)
		}
	}
	r := validIntent()
	r.State = "draft"
	r.Ideas[0].Decision = "decline"
	r.Tasks[0].IdeaIDs = []string{"d333333333333333333333333"}
	if r.Validate() == nil {
		t.Fatal("draft task entered declined outcome")
	}
}

func TestIntentSummaryAndListFilter(t *testing.T) {
	r := validIntent()
	r.EffectiveOutcomes = []string{"d333333333333333333333333"}
	s := r.Summary()
	if s.Intent != r.Intent || s.Revision != r.ID || s.EffectiveCount != 1 || s.TaskCount != 2 {
		t.Fatal("summary lost identities")
	}
	f := IntentListFilter{Limit: 20}
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []IntentListFilter{{Limit: 0}, {Limit: 51}, {Cursor: "bad", Limit: 10}, {State: "bogus", Limit: 10}} {
		if bad.Validate() == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}
