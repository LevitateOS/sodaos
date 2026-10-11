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
