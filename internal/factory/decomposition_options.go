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

// IntentPlanningOptions names the pre-issue planning operation that derives
// candidate issue graphs from planner outputs. It enters only through the
// decompose phase with an intent subject and the coder role (FW11
// admission); this vocabulary states the operation, it does not admit it.
const IntentPlanningOptions = "decomposition_options"

// SemanticCandidate is one planner-produced candidate issue graph: bounded
// tasks with a rationale. Candidates carry decision-ID task keys only;
// native issue IDs are never chosen by the planner.
type SemanticCandidate struct {
	Key       string       `json:"key"`
	Rationale string       `json:"rationale"`
	Tasks     []IntentTask `json:"tasks"`
}

// SemanticPlannerInput binds one semantic planning run to its exact inputs:
// the admitted operation, the authoritative draft head, the planner outputs
// it consumes (authored proposal content and alternatives), and the source
// investigation the plan was built from. Accepted graphs and history are
// never inputs: planning consumes planner outputs only.
type SemanticPlannerInput struct {
	Operation    string
	Head         IntentRevision
	Content      ProposalContent
	Alternatives AlternativeSet
	Inspected    []InspectedSource
}

// SemanticPlannerOutput is one bounded semantic planning answer: one to
// three assessed candidates, or a decomposition_needs_attention reason with
// no candidates. Attention records the product decision the planner needs;
// it never triggers another batch automatically.
type SemanticPlannerOutput struct {
	Intent     string              `json:"intent"`
	Revision   string              `json:"revision"`
	Operation  string              `json:"operation"`
	Candidates []SemanticCandidate `json:"candidates,omitempty"`
	Attention  string              `json:"attention,omitempty"`
}

// Validate checks the planner input shape and bindings without running any
// provider: operation vocabulary, draft planning scope, proposal content
// bound to the head, supported alternatives for the options operation, and
// the investigation evidence planning was built from.
func (in SemanticPlannerInput) Validate() error {
	if !ValidIntentPlanningOperation(in.Operation) {
		return errors.New("unknown intent planning operation")
	}
	if err := ValidateDraftPlanningScope(in.Head); err != nil {
		return err
	}
	if err := in.Content.Validate(); err != nil {
		return err
	}
	if in.Content.Intent != in.Head.Intent || in.Content.Revision != in.Head.ID {
		return errors.New("proposal content does not bind the planning head")
	}
	if err := in.Alternatives.Validate(); err != nil {
		return err
	}
	if in.Operation == IntentPlanningOptions && !in.Alternatives.Supported {
		return errors.New("decomposition options need supported alternatives")
	}
	if len(in.Inspected) == 0 || len(in.Inspected) > MaxInspectedRefs {
		return errors.New("planning needs its source investigation first")
	}
	for _, ref := range in.Inspected {
		if !validOwnershipEntry(ref.Path) || strings.HasSuffix(ref.Path, "/") {
			return errors.New("inspected source is not an exact file")
		}
		if !ValidCommit(ref.Commit) {
			return errors.New("inspected source lacks its commit")
		}
		if ref.Evidence == "" || len(ref.Evidence) > MaxPlanTextBytes {
			return errors.New("inspected evidence outside its bound")
		}
	}
	return nil
}

// Validate checks the planner output shape: revision binding, operation
// match, and attention XOR candidates. Candidate task graphs are assessed
// by AssessSemanticOutput against the planning input.
func (out SemanticPlannerOutput) Validate() error {
	if !project.ValidDecisionID(out.Intent) || !project.ValidDecisionID(out.Revision) {
		return errors.New("invalid semantic output identity")
	}
	if !ValidIntentPlanningOperation(out.Operation) {
		return errors.New("unknown intent planning operation")
	}
	if out.Attention != "" {
		if len(out.Candidates) != 0 {
			return errors.New("attention carries no new candidates")
		}
		if strings.TrimSpace(out.Attention) == "" || len(out.Attention) > 4000 {
			return errors.New("invalid attention reason")
		}
		return nil
	}
	if len(out.Candidates) == 0 || len(out.Candidates) > 3 {
		return errors.New("semantic output needs one to three candidates")
	}
	seen := map[string]bool{}
	for _, candidate := range out.Candidates {
		if strings.TrimSpace(candidate.Key) == "" || len(candidate.Key) > 200 || seen[candidate.Key] {
			return errors.New("invalid semantic candidate identity")
		}
		seen[candidate.Key] = true
		if !intentText(candidate.Rationale, 4000) {
			return errors.New("invalid semantic candidate rationale")
		}
	}
	return nil
}

// AssessSemanticOutput validates one planner answer against the exact input
// that produced it: revision/operation binding, then every candidate task
// graph against the draft head. Broad multi-task graphs and single-task
// graphs both assess; uncovered authorized outcomes, false dependencies,
// revision changes and over-broad answers refuse. Attention outputs assess
// without candidates. Assessment is pure: replaying the same input and
// output reaches the same verdict and spends nothing.
func AssessSemanticOutput(in SemanticPlannerInput, out SemanticPlannerOutput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	if err := out.Validate(); err != nil {
		return err
	}
	if out.Intent != in.Head.Intent || out.Revision != in.Head.ID || out.Operation != in.Operation {
		return errors.New("semantic output does not bind its planning input")
	}
	if out.Attention != "" {
		return nil
	}
	for _, candidate := range out.Candidates {
		if err := AssessProposedTasks(in.Head, candidate.Tasks, in.Head.MaxIssues); err != nil {
			return err
		}
	}
	return nil
}
