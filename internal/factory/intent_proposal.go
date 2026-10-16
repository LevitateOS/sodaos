package factory

import (
	"errors"

	"github.com/levitateos/sodaos/internal/project"
)

// IntentPlanningPropose names the pre-issue planning operation that derives
// a conservative review baseline from the person's objective. The companion
// operation IntentPlanningOptions (decomposition_options.go) derives
// candidate issue graphs. Both enter only through the decompose phase with
// an intent subject and the coder role (FW11 admission); this vocabulary
// states the operation, it does not admit it.
const IntentPlanningPropose = "propose_intent"

// ValidIntentPlanningOperation reports whether op is an admitted pre-issue
// planning operation.
func ValidIntentPlanningOperation(op string) bool {
	return op == IntentPlanningPropose || op == IntentPlanningOptions
}

// ValidateDraftPlanningScope checks that a draft head may enter pre-issue
// planning. The draft's effective ideas must be accepted or edited choices
// with explicit source and provenance attribution; material undecided ideas
// cannot enter the effective subset (IntentRevision.Validate enforces the
// subset). Tasks need not exist yet: no accepted graph or history is
// consulted, so a first draft plans without fake issues or prerequisites.
func ValidateDraftPlanningScope(r IntentRevision) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.State != "draft" {
		return errors.New("intent planning needs a draft revision")
	}
	ideas := map[string]ProductIdea{}
	for _, idea := range r.Ideas {
		ideas[idea.ID] = idea
	}
	for _, id := range r.EffectiveOutcomes {
		idea := ideas[id]
		if idea.Source == "" || idea.Provenance == "" {
			return errors.New("effective planning scope needs attributed choices")
		}
	}
	return nil
}

// AssessProposedTasks validates one pre-issue candidate task graph against
// its draft head without requiring an accepted graph or history. Every
// task binds known accepted-or-edited choices, prerequisite edges resolve
// inside the candidate without cycles, and every effective authorized
// outcome is covered by at least one task. Broad multi-task graphs and
// single-task graphs both assess: there is no artificial minimum issue
// count. Models never choose native issue IDs: task keys are decision IDs.
func AssessProposedTasks(r IntentRevision, tasks []IntentTask, maxIssues int) error {
	if maxIssues < 1 || maxIssues > 64 || maxIssues > r.MaxIssues {
		return errors.New("invalid candidate issue bound")
	}
	if len(tasks) == 0 || len(tasks) > maxIssues {
		return errors.New("candidate task count outside its bound")
	}
	ideas := map[string]ProductIdea{}
	for _, idea := range r.Ideas {
		ideas[idea.ID] = idea
	}
	seen := map[string]IntentTask{}
	for _, task := range tasks {
		if !project.ValidDecisionID(task.Key) || !intentText(task.Objective, 4000) ||
			len(task.Outcomes) == 0 || len(task.Outcomes) > 20 || len(task.IdeaIDs) == 0 {
			return errors.New("invalid candidate task")
		}
		if _, exists := seen[task.Key]; exists {
			return errors.New("duplicate candidate task")
		}
		for _, outcome := range task.Outcomes {
			if !intentText(outcome, 4000) {
				return errors.New("invalid candidate outcome")
			}
		}
		for _, id := range task.IdeaIDs {
			idea, exists := ideas[id]
			if !exists {
				return errors.New("candidate references unknown product choice")
			}
			if idea.Decision != "accept" && idea.Decision != "edit" {
				return errors.New("candidate references undecided product choice")
			}
		}
		seen[task.Key] = task
	}
	visited, active := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(key string) error {
		if active[key] {
			return errors.New("cyclic candidate dependencies")
		}
		if visited[key] {
			return nil
		}
		task, exists := seen[key]
		if !exists {
			return errors.New("unknown candidate dependency")
		}
		active[key] = true
		edge := map[string]bool{}
		for _, dep := range task.DependsOn {
			if edge[dep] {
				return errors.New("duplicate candidate dependency")
			}
			edge[dep] = true
			if err := visit(dep); err != nil {
				return err
			}
		}
		active[key] = false
		visited[key] = true
		return nil
	}
	for key := range seen {
		if err := visit(key); err != nil {
			return err
		}
	}
	if len(r.EffectiveOutcomes) > 0 {
		covered := map[string]bool{}
		for _, task := range tasks {
			for _, id := range task.IdeaIDs {
				covered[id] = true
			}
		}
		for _, id := range r.EffectiveOutcomes {
			if !covered[id] {
				return errors.New("candidate leaves authorized outcomes uncovered")
			}
		}
	}
	return nil
}

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
