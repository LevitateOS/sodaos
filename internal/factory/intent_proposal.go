package factory

// ProposeIntent assembles a conservative review baseline from the person's
// objective. It makes no claims about researched domain requirements. Choice
// judgments can advise between these concrete alternatives; people decide the
// material product direction before any native issue is created.
func ProposeIntent(r IntentRevision, ids []string) IntentRevision {
	r.State = "draft"
	if len(r.Ideas) == 0 {
		r.Ideas = []ProductIdea{
			{ID: ids[0], Description: proposalObjective(r.Objective), Consequence: "Build the smallest complete workflow for this objective. Additional capabilities need a new product decision.", Decision: "pending", Source: "autonomous", Provenance: "autonomous"},
			{ID: ids[1], Description: "Keep the first version within the existing repository and Project access", Consequence: "No new external integrations, account enrollment, paid services or shared department permissions are included.", Assumption: true, Decision: "pending", Source: "assumption", Provenance: "autonomous", Material: true},
		}
	}
	r.Tasks = []IntentTask{}
	for i, idea := range r.Ideas {
		if idea.Decision == "decline" || idea.Assumption {
			continue
		}
		objective := idea.Description
		if idea.Decision == "edit" {
			objective = idea.Correction
		}
		key := ids[2+i*2]
		check := ids[3+i*2]
		verification := "Verify the reviewed behavior: " + objective
		if len(verification) > 4000 {
			verification = "Verify the accepted product behavior and its authority boundaries"
		}
		if r.Approach == "complete_first" {
			r.Tasks = append(r.Tasks, IntentTask{Key: key, Objective: objective, IdeaIDs: []string{idea.ID}, Outcomes: []string{"The reviewed product behavior is implemented within the accepted objective", "Existing access and finite funding limits remain enforced", "Affected user workflow and authority boundaries pass focused checks"}})
			continue
		}
		r.Tasks = append(r.Tasks, IntentTask{Key: key, Objective: objective, IdeaIDs: []string{idea.ID}, Outcomes: []string{"The reviewed product behavior is implemented within the accepted objective", "Existing access and finite funding limits remain enforced"}}, IntentTask{Key: check, Objective: verification, IdeaIDs: []string{idea.ID}, Outcomes: []string{"Affected user workflow and authority boundaries pass focused checks"}, DependsOn: []string{key}})
	}
	return r
}

// A review label may summarize a large objective; the immutable revision
// retains the full person's text, which also enters native requirements.
func proposalObjective(objective string) string {
	if len(objective) > 4000 {
		return "Implement the recorded product objective"
	}
	return objective
}
