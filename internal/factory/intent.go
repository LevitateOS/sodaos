package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

// Department is private organizational context owned by its creator. It grants
// no repository, Project, account or spending permission.
type Department struct {
	ID          string `json:"id"`
	Owner       int64  `json:"owner,string"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (d Department) Validate() error {
	if !project.ValidDecisionID(d.ID) || d.Owner <= 0 || !intentText(d.Name, 120) || !intentText(d.Description, 2000) {
		return errors.New("invalid department")
	}
	return nil
}

// ProductIdea describes a material product choice and its consequence.
type ProductIdea struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Consequence string `json:"consequence"`
	Assumption  bool   `json:"assumption"`
	Decision    string `json:"decision"`
	Correction  string `json:"correction,omitempty"`
}

type IntentTask struct {
	Key       string   `json:"key"`
	Objective string   `json:"objective"`
	Outcomes  []string `json:"outcomes"`
	IdeaIDs   []string `json:"idea_ids"`
	DependsOn []string `json:"depends_on,omitempty"`
}

// IntentRevision is immutable review input or accepted product authority. An
// acceptance covers this exact product direction and bounded derived graph;
// execution still requires current independent grants and native evidence.
type IntentRevision struct {
	ID            string        `json:"id"`
	Intent        string        `json:"intent"`
	Predecessor   string        `json:"predecessor,omitempty"`
	Owner         int64         `json:"owner,string"`
	Repository    int64         `json:"repository,string"`
	Department    string        `json:"department,omitempty"`
	Objective     string        `json:"objective"`
	Ideas         []ProductIdea `json:"ideas"`
	Tasks         []IntentTask  `json:"tasks"`
	State         string        `json:"state"`
	MaxIssues     int           `json:"max_issues"`
	PlanningCalls int           `json:"planning_calls"`
	CreatedUnix   int64         `json:"created_unix"`
}

func intentText(s string, limit int) bool {
	return strings.TrimSpace(s) != "" && len(s) <= limit && !strings.ContainsRune(s, 0)
}

func (r IntentRevision) Validate() error {
	if !project.ValidDecisionID(r.ID) || !project.ValidDecisionID(r.Intent) || r.Owner <= 0 || r.CreatedUnix <= 0 || !intentText(r.Objective, 16000) {
		return errors.New("invalid intent identity or objective")
	}
	if r.Predecessor != "" && !project.ValidDecisionID(r.Predecessor) || r.Department != "" && !project.ValidDecisionID(r.Department) {
		return errors.New("invalid intent reference")
	}
	if r.State != "draft" && r.State != "accepted" && r.State != "declined" {
		return errors.New("invalid intent state")
	}
	if r.Repository < 0 || r.MaxIssues < 1 || r.MaxIssues > 64 || r.PlanningCalls < 0 || r.PlanningCalls > 100 || len(r.Ideas) > 50 || len(r.Tasks) > r.MaxIssues {
		return errors.New("invalid intent bounds")
	}
	if r.State == "accepted" && (r.Repository == 0 || len(r.Tasks) == 0) {
		return errors.New("accepted intent needs repository and work")
	}
	ideas := map[string]bool{}
	for _, idea := range r.Ideas {
		if !project.ValidDecisionID(idea.ID) || ideas[idea.ID] || !intentText(idea.Description, 4000) || !intentText(idea.Consequence, 4000) {
			return errors.New("invalid product idea")
		}
		if idea.Decision != "pending" && idea.Decision != "accept" && idea.Decision != "decline" && idea.Decision != "edit" {
			return errors.New("invalid product decision")
		}
		if idea.Decision == "edit" && !intentText(idea.Correction, 4000) {
			return errors.New("edited idea needs correction")
		}
		if r.State == "accepted" && idea.Decision == "pending" {
			return errors.New("material product decision unresolved")
		}
		ideas[idea.ID] = idea.Decision == "accept" || idea.Decision == "edit"
	}
	tasks := map[string]IntentTask{}
	for _, task := range r.Tasks {
		if !project.ValidDecisionID(task.Key) || !intentText(task.Objective, 4000) || len(task.Outcomes) == 0 || len(task.Outcomes) > 20 || len(task.IdeaIDs) == 0 {
			return errors.New("invalid intent task")
		}
		if _, exists := tasks[task.Key]; exists {
			return errors.New("duplicate task")
		}
		for _, outcome := range task.Outcomes {
			if !intentText(outcome, 4000) {
				return errors.New("invalid task outcome")
			}
		}
		for _, id := range task.IdeaIDs {
			if accepted, exists := ideas[id]; !exists || r.State == "accepted" && !accepted {
				return errors.New("task exceeds accepted product choices")
			}
		}
		tasks[task.Key] = task
	}
	visited, active := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(key string) error {
		if active[key] {
			return errors.New("cyclic intent dependencies")
		}
		if visited[key] {
			return nil
		}
		task, exists := tasks[key]
		if !exists {
			return errors.New("unknown intent dependency")
		}
		active[key] = true
		seen := map[string]bool{}
		for _, dep := range task.DependsOn {
			if seen[dep] {
				return errors.New("duplicate intent dependency")
			}
			seen[dep] = true
			if err := visit(dep); err != nil {
				return err
			}
		}
		active[key] = false
		visited[key] = true
		return nil
	}
	for key := range tasks {
		if err := visit(key); err != nil {
			return err
		}
	}
	return nil
}

func (r IntentRevision) Digest() string {
	raw, _ := json.Marshal(r)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
