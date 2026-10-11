package factory

// ProposeIntent assembles a conservative review baseline from the person's
// objective. It makes no claims about researched domain requirements. Choice
// judgments can advise between these concrete alternatives; people decide the
// material product direction before any native issue is created.
func ProposeIntent(r IntentRevision, ids []string) IntentRevision {
	r.State = "draft"
	if len(r.Ideas) == 0 {
		r.Ideas = []ProductIdea{
			{ID: ids[0], Description: r.Objective, Consequence: "Build the smallest complete workflow for this objective. Additional capabilities need a new product decision.", Decision: "pending"},
			{ID: ids[1], Description: "Keep the first version within the existing repository and Project access", Consequence: "No new external integrations, account enrollment, paid services or shared department permissions are included.", Assumption: true, Decision: "pending"},
		}
	}
	r.Tasks = nil
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
		r.Tasks = append(r.Tasks, IntentTask{Key: key, Objective: objective, IdeaIDs: []string{idea.ID}, Outcomes: []string{"The reviewed product behavior is implemented within the accepted objective", "Existing access and finite funding limits remain enforced"}}, IntentTask{Key: check, Objective: "Verify the reviewed behavior: " + objective, IdeaIDs: []string{idea.ID}, Outcomes: []string{"Affected user workflow and authority boundaries pass focused checks"}, DependsOn: []string{key}})
	}
	return r
}
