package factory

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"

	"github.com/levitateos/sodaos/internal/project"
)

// DecompositionCandidate is one scope-preserving technical restructuring of
// accepted tasks. Every candidate keeps the accepted outcomes, idea bindings
// and bounds; only task boundaries and prerequisite edges may change.
type DecompositionCandidate struct {
	Key       string
	Rationale string
	Tasks     []IntentTask
}

// ScopedDecomposition is the selected candidate for native graph writes.
// I consumes it for initial submission and later deltas; S never writes
// native state from it.
type ScopedDecomposition struct {
	Intent   string       `json:"intent"`
	Revision string       `json:"revision"`
	Selected string       `json:"selected"`
	Tasks    []IntentTask `json:"tasks"`
	Advice   []string     `json:"advice"`
}

func derivedTaskKey(parts ...string) string {
	sum := sha256.Sum256([]byte("s-decomp:" + strings.Join(parts, ":")))
	return "d" + hex.EncodeToString(sum[:])[:24]
}

func shortLabel(text string) string {
	runes := []rune(text)
	if len(runes) > 120 {
		return string(runes[:120]) + "..."
	}
	return text
}

func outcomeMultiset(tasks []IntentTask) []string {
	var outcomes []string
	for _, task := range tasks {
		outcomes = append(outcomes, task.Outcomes...)
	}
	sort.Strings(outcomes)
	return outcomes
}

func sameOutcomes(a, b []IntentTask) bool {
	first, second := outcomeMultiset(a), outcomeMultiset(b)
	if len(first) != len(second) {
		return false
	}
	for i := range first {
		if first[i] != second[i] {
			return false
		}
	}
	return true
}

func validCandidateGraph(r IntentRevision, tasks []IntentTask) bool {
	if len(tasks) == 0 || len(tasks) > r.MaxIssues || !sameOutcomes(r.Tasks, tasks) {
		return false
	}
	for _, task := range tasks {
		if !project.ValidDecisionID(task.Key) {
			return false
		}
	}
	candidate := r
	candidate.Tasks = tasks
	return candidate.Validate() == nil
}

// DecompositionCandidates derives the keep option plus bounded structural
// variants: the first splittable multi-outcome task and the first combinable
// task pair. Only candidates passing full accepted validation are offered.
func DecompositionCandidates(r IntentRevision) []DecompositionCandidate {
	keep := DecompositionCandidate{Key: "keep", Rationale: "Retain the accepted task boundaries and prerequisite edges.", Tasks: append([]IntentTask(nil), r.Tasks...)}
	candidates := []DecompositionCandidate{keep}
	if split, ok := splitCandidate(r); ok {
		candidates = append(candidates, split)
	}
	if combine, ok := combineCandidate(r); ok {
		candidates = append(candidates, combine)
	}
	return candidates
}

func splitCandidate(r IntentRevision) (DecompositionCandidate, bool) {
	for index, task := range r.Tasks {
		if len(task.Outcomes) < 2 {
			continue
		}
		var tasks []IntentTask
		parts := make([]string, len(task.Outcomes))
		for i, outcome := range task.Outcomes {
			key := derivedTaskKey(task.Key, "split", outcome)
			parts[i] = key
			tasks = append(tasks, IntentTask{Key: key, Objective: task.Objective, Outcomes: []string{outcome}, IdeaIDs: append([]string(nil), task.IdeaIDs...), DependsOn: append([]string(nil), task.DependsOn...)})
		}
		for j, other := range r.Tasks {
			if j == index {
				continue
			}
			repointed := other
			repointed.DependsOn = nil
			for _, dep := range other.DependsOn {
				if dep == task.Key {
					repointed.DependsOn = append(repointed.DependsOn, parts...)
					continue
				}
				repointed.DependsOn = append(repointed.DependsOn, dep)
			}
			tasks = append(tasks, repointed)
		}
		seen := map[string]bool{}
		for _, t := range tasks {
			if seen[t.Key] {
				return DecompositionCandidate{}, false
			}
			seen[t.Key] = true
		}
		if !validCandidateGraph(r, tasks) {
			return DecompositionCandidate{}, false
		}
		return DecompositionCandidate{Key: "split:" + task.Key, Rationale: "Split '" + shortLabel(task.Objective) + "' into per-outcome tasks to remove overload.", Tasks: tasks}, true
	}
	return DecompositionCandidate{}, false
}

func combineCandidate(r IntentRevision) (DecompositionCandidate, bool) {
	for i := 0; i < len(r.Tasks); i++ {
		for j := i + 1; j < len(r.Tasks); j++ {
			first, second := r.Tasks[i], r.Tasks[j]
			if !equalStrings(first.IdeaIDs, second.IdeaIDs) || !equalStrings(first.DependsOn, second.DependsOn) {
				continue
			}
			objective := first.Objective + "; " + second.Objective
			outcomes := append(append([]string(nil), first.Outcomes...), second.Outcomes...)
			if len(objective) > 4000 || len(outcomes) > 20 {
				continue
			}
			merged := IntentTask{Key: derivedTaskKey(first.Key, second.Key, "combine"), Objective: objective, Outcomes: outcomes, IdeaIDs: append([]string(nil), first.IdeaIDs...), DependsOn: append([]string(nil), first.DependsOn...)}
			var tasks []IntentTask
			tasks = append(tasks, merged)
			for k, other := range r.Tasks {
				if k == i || k == j {
					continue
				}
				repointed := other
				repointed.DependsOn = nil
				seen := map[string]bool{}
				for _, dep := range other.DependsOn {
					next := dep
					if dep == first.Key || dep == second.Key {
						next = merged.Key
					}
					if seen[next] {
						continue
					}
					seen[next] = true
					repointed.DependsOn = append(repointed.DependsOn, next)
				}
				tasks = append(tasks, repointed)
			}
			seen := map[string]bool{}
			collides := false
			for _, t := range tasks {
				if seen[t.Key] {
					collides = true
				}
				seen[t.Key] = true
			}
			if collides || !validCandidateGraph(r, tasks) {
				continue
			}
			return DecompositionCandidate{Key: "combine:" + first.Key + "+" + second.Key, Rationale: "Combine '" + shortLabel(first.Objective) + "' with '" + shortLabel(second.Objective) + "' to remove duplicated work.", Tasks: tasks}, true
		}
	}
	return DecompositionCandidate{}, false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Validate checks standalone task-graph structure: identities, bounds,
// resolved prerequisites and acyclic edges. Idea bindings reference the
// accepted revision the consumer already holds.
func (d ScopedDecomposition) Validate() error {
	if !project.ValidDecisionID(d.Intent) || !project.ValidDecisionID(d.Revision) || strings.TrimSpace(d.Selected) == "" || len(d.Selected) > 200 {
		return errors.New("invalid scoped decomposition identity")
	}
	if len(d.Tasks) == 0 || len(d.Tasks) > 64 || len(d.Advice) > 8 {
		return errors.New("invalid scoped decomposition bounds")
	}
	for _, fingerprint := range d.Advice {
		if !ValidDigest(fingerprint) {
			return errors.New("invalid scoped decomposition advice")
		}
	}
	tasks := map[string]IntentTask{}
	for _, task := range d.Tasks {
		if !project.ValidDecisionID(task.Key) || !intentText(task.Objective, 4000) || len(task.Outcomes) == 0 || len(task.Outcomes) > 20 || len(task.IdeaIDs) == 0 {
			return errors.New("invalid scoped task")
		}
		if _, exists := tasks[task.Key]; exists {
			return errors.New("duplicate scoped task")
		}
		for _, outcome := range task.Outcomes {
			if !intentText(outcome, 4000) {
				return errors.New("invalid scoped outcome")
			}
		}
		for _, id := range task.IdeaIDs {
			if !project.ValidDecisionID(id) {
				return errors.New("invalid scoped idea binding")
			}
		}
		tasks[task.Key] = task
	}
	visited, active := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(key string) error {
		if active[key] {
			return errors.New("cyclic scoped dependencies")
		}
		if visited[key] {
			return nil
		}
		task, exists := tasks[key]
		if !exists {
			return errors.New("unknown scoped dependency")
		}
		active[key] = true
		seen := map[string]bool{}
		for _, dep := range task.DependsOn {
			if seen[dep] {
				return errors.New("duplicate scoped dependency")
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
