package api

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// MaxRevisionComparisons bounds one scope-revision page. Each entry
// compares two durable revisions and classifies its mapped work, so
// pages stay small and explicit.
const MaxRevisionComparisons = 5

// DefaultRevisionComparisons covers the current revision against its
// predecessor: the usual change-direction review.
const DefaultRevisionComparisons = 2

// revisionTaskChangeView renders one changed task: which structural
// fields differ plus the old and new objective/outcomes text when those
// changed. Added and removed tasks render whole; unchanged tasks count.
type revisionTaskChangeView struct {
	ChangedFields []string `json:"changed_fields"`
	Key           string   `json:"key"`
	OldObjective  string   `json:"old_objective,omitempty"`
	NewObjective  string   `json:"new_objective,omitempty"`
	OldOutcomes   []string `json:"old_outcomes,omitempty"`
	NewOutcomes   []string `json:"new_outcomes,omitempty"`
}

// revisionTaskDiffView renders the structural task difference between
// two durable revisions. Task keys are the stable identity; bodies
// come from the compared revisions, never a speculative graph.
type revisionTaskDiffView struct {
	Added          []factory.IntentTask     `json:"added"`
	Removed        []factory.IntentTask     `json:"removed"`
	Changed        []revisionTaskChangeView `json:"changed"`
	UnchangedCount int                      `json:"unchanged_count"`
}

func revisionTaskChangedFields(old, current factory.IntentTask) []string {
	fields := []string{}
	if old.Objective != current.Objective {
		fields = append(fields, "objective")
	}
	if !slices.Equal(old.Outcomes, current.Outcomes) {
		fields = append(fields, "outcomes")
	}
	if !sameRevisionIDs(old.IdeaIDs, current.IdeaIDs) {
		fields = append(fields, "idea_ids")
	}
	if !sameRevisionIDs(old.DependsOn, current.DependsOn) {
		fields = append(fields, "depends_on")
	}
	return fields
}

func sameRevisionIDs(old, current []string) bool {
	if len(old) != len(current) {
		return false
	}
	seen := make(map[string]int, len(old))
	for _, id := range old {
		seen[id]++
	}
	for _, id := range current {
		seen[id]--
		if seen[id] < 0 {
			return false
		}
	}
	return true
}

func revisionTaskDiff(old, current []factory.IntentTask) revisionTaskDiffView {
	diff := revisionTaskDiffView{Added: []factory.IntentTask{}, Removed: []factory.IntentTask{}, Changed: []revisionTaskChangeView{}}
	previous := make(map[string]factory.IntentTask, len(old))
	for _, task := range old {
		previous[task.Key] = task
	}
	seen := make(map[string]bool, len(current))
	for _, task := range current {
		seen[task.Key] = true
		before, ok := previous[task.Key]
		if !ok {
			diff.Added = append(diff.Added, task)
			continue
		}
		fields := revisionTaskChangedFields(before, task)
		if len(fields) == 0 {
			diff.UnchangedCount++
			continue
		}
		change := revisionTaskChangeView{ChangedFields: fields, Key: task.Key}
		if slices.Contains(fields, "objective") {
			change.OldObjective, change.NewObjective = before.Objective, task.Objective
		}
		if slices.Contains(fields, "outcomes") {
			change.OldOutcomes, change.NewOutcomes = before.Outcomes, task.Outcomes
		}
		diff.Changed = append(diff.Changed, change)
	}
	for _, task := range old {
		if !seen[task.Key] {
			diff.Removed = append(diff.Removed, task)
		}
	}
	return diff
}

// revisionIdeaChangeView renders one idea's decision transition between
// two durable revisions: the recorded decisions plus whether its
// correction or substance changed.
type revisionIdeaChangeView struct {
	ID                string `json:"id"`
	FromDecision      string `json:"from_decision"`
	ToDecision        string `json:"to_decision"`
	CorrectionChanged bool   `json:"correction_changed"`
	ContentChanged    bool   `json:"content_changed"`
}

// revisionIdeaDiffView renders the idea difference between two durable
// revisions. Idea IDs are the stable identity.
type revisionIdeaDiffView struct {
	Added   []factory.ProductIdea    `json:"added"`
	Removed []factory.ProductIdea    `json:"removed"`
	Changed []revisionIdeaChangeView `json:"changed"`
}

func revisionIdeaDiff(old, current []factory.ProductIdea) revisionIdeaDiffView {
	diff := revisionIdeaDiffView{Added: []factory.ProductIdea{}, Removed: []factory.ProductIdea{}, Changed: []revisionIdeaChangeView{}}
	previous := make(map[string]factory.ProductIdea, len(old))
	for _, idea := range old {
		previous[idea.ID] = idea
	}
	seen := make(map[string]bool, len(current))
	for _, idea := range current {
		seen[idea.ID] = true
		before, ok := previous[idea.ID]
		if !ok {
			diff.Added = append(diff.Added, idea)
			continue
		}
		change := revisionIdeaChangeView{ID: idea.ID, FromDecision: before.Decision, ToDecision: idea.Decision,
			CorrectionChanged: before.Correction != idea.Correction,
			ContentChanged:    before.Description != idea.Description || before.Consequence != idea.Consequence}
		if change.FromDecision != change.ToDecision || change.CorrectionChanged || change.ContentChanged {
			diff.Changed = append(diff.Changed, change)
		}
	}
	for _, idea := range old {
		if !seen[idea.ID] {
			diff.Removed = append(diff.Removed, idea)
		}
	}
	return diff
}

// revisionIDDiffView renders added and removed IDs under set semantics.
type revisionIDDiffView struct {
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
}

func revisionIDDiff(old, current []string) revisionIDDiffView {
	diff := revisionIDDiffView{Added: []string{}, Removed: []string{}}
	before := make(map[string]bool, len(old))
	for _, id := range old {
		before[id] = true
	}
	now := make(map[string]bool, len(current))
	for _, id := range current {
		now[id] = true
		if !before[id] {
			diff.Added = append(diff.Added, id)
		}
	}
	for _, id := range old {
		if !now[id] {
			diff.Removed = append(diff.Removed, id)
		}
	}
	return diff
}

// revisionScalarChangeView renders one changed scalar context value.
type revisionScalarChangeView struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// revisionComparisonView renders the structural difference between one
// revision and its durable predecessor: tasks, ideas, effective
// outcomes and bound context. The first revision renders initial with
// empty diffs instead of comparing against nothing.
type revisionComparisonView struct {
	Tasks      revisionTaskDiffView      `json:"tasks"`
	Ideas      revisionIdeaDiffView      `json:"ideas"`
	Effective  revisionIDDiffView        `json:"effective_outcomes"`
	Repository *revisionScalarChangeView `json:"repository_change,omitempty"`
	State      *revisionScalarChangeView `json:"state_change,omitempty"`
	Initial    bool                      `json:"initial,omitempty"`
}

func revisionComparisonDTO(old, current factory.IntentRevision) revisionComparisonView {
	diff := revisionComparisonView{
		Tasks: revisionTaskDiff(old.Tasks, current.Tasks), Ideas: revisionIdeaDiff(old.Ideas, current.Ideas),
		Effective: revisionIDDiff(old.EffectiveOutcomes, current.EffectiveOutcomes),
	}
	if old.Repository != current.Repository {
		diff.Repository = &revisionScalarChangeView{From: strconv.FormatInt(old.Repository, 10), To: strconv.FormatInt(current.Repository, 10)}
	}
	if old.State != current.State {
		diff.State = &revisionScalarChangeView{From: old.State, To: current.State}
	}
	return diff
}

func initialRevisionComparison() revisionComparisonView {
	return revisionComparisonView{
		Tasks: revisionTaskDiff(nil, nil), Ideas: revisionIdeaDiff(nil, nil),
		Effective: revisionIDDiff(nil, nil), Initial: true,
	}
}

// revisionAuthorityStates name whether one mapping's bound acceptance
// still matches current readiness: current, superseded by a newer
// acceptance, or unknown when no control record exists.
const (
	revisionAuthorityCurrent    = "current"
	revisionAuthoritySuperseded = "superseded"
	revisionAuthorityUnknown    = "unknown"
)

// revisionWorkView renders one revision's mapped work item with its
// current evidenced state and authority freshness. States reuse the
// request-overview vocabulary; canceled or fenced work reports
// attention, never delivered.
type revisionWorkView struct {
	TaskKey   string `json:"task_key"`
	Issue     string `json:"issue_index"`
	State     string `json:"state"`
	Authority string `json:"authority"`
}

// revisionEntryView renders one scope revision: its identity, lineage,
// structural comparison against its predecessor, generation state and
// affected mapped work with current states.
type revisionEntryView struct {
	Comparison    revisionComparisonView `json:"comparison"`
	Generation    requestGenerationView  `json:"generation"`
	Work          []revisionWorkView     `json:"work"`
	Revision      string                 `json:"revision"`
	State         string                 `json:"state"`
	Repository    string                 `json:"repository"`
	GeneratedFrom string                 `json:"generated_from,omitempty"`
	CreatedUnix   int64                  `json:"created_unix"`
}

// factoryRevisionHistoryView is the bounded scope-revision read for one
// owner-private build request: recent revisions newest-first, each with
// its structural delta and affected work states.
type factoryRevisionHistoryView struct {
	Revisions    []revisionEntryView `json:"revisions"`
	Intent       string              `json:"intent"`
	ObservedUnix int64               `json:"observed_unix"`
	Truncated    bool                `json:"truncated,omitempty"`
}

func parseRevisionHistoryQuery(raw string) (int, bool) {
	if len(raw) > 512 {
		return 0, false
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return 0, false
	}
	for key := range values {
		if key != "limit" {
			return 0, false
		}
	}
	limit := DefaultRevisionComparisons
	if raw := values.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > MaxRevisionComparisons {
			return 0, false
		}
		limit = parsed
	}
	return limit, true
}

// apiFactoryRevisionHistory shows recent scope revisions for one
// owner-private build request with structural deltas and affected work
// states. Unknown or foreign requests report 404 without disclosure.
// GET never judges, provisions or dispatches.
func (s *API) apiFactoryRevisionHistory(w http.ResponseWriter, r *http.Request, v store.Session) {
	limit, ok := parseRevisionHistoryQuery(r.URL.RawQuery)
	if r.URL.ForceQuery || !ok {
		auth.JSONError(w, 400, "invalid_request", "Use limit 1..5.")
		return
	}
	intent := r.PathValue("intentID")
	chain, err := s.Store.IntentHistory(r.Context(), intent, v.User.ID, limit+1)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 404, "not_found", "Build request not found.")
		} else {
			auth.JSONError(w, 503, "store_unavailable", "Could not read request history.")
		}
		return
	}
	view := factoryRevisionHistoryView{
		Revisions: []revisionEntryView{},
		Intent:    intent, ObservedUnix: time.Now().Unix(),
	}
	shown := chain
	if len(chain) > limit {
		shown, view.Truncated = chain[:limit], true
	}
	for i, revision := range shown {
		entry := revisionEntryView{
			Generation: requestGenerationView{State: requestGenerationAbsent},
			Work:       []revisionWorkView{},
			Revision:   revision.ID, State: revision.State,
			Repository: strconv.FormatInt(revision.Repository, 10), CreatedUnix: revision.CreatedUnix,
		}
		if i+1 < len(chain) {
			entry.Comparison = revisionComparisonDTO(chain[i+1], revision)
		} else if revision.Predecessor == "" {
			entry.Comparison = initialRevisionComparison()
		} else {
			// Unexpected history shape: the chain names a
			// predecessor the bounded walk did not return. Error
			// rather than render a delta against nothing.
			auth.JSONError(w, 503, "store_unavailable", "Could not read request history.")
			return
		}
		if generated, err := s.Store.PreviousGeneratedIntent(r.Context(), intent, revision.ID); err == nil {
			entry.GeneratedFrom = generated.ID
		} else if !errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Could not read generation lineage.")
			return
		}
		if generation, err := s.Store.IntentGeneration(r.Context(), revision.ID); err == nil {
			entry.Generation.State = generation.State
		} else if !errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Could not read request generation.")
			return
		}
		if ok := s.revisionEntryWork(w, r, &entry, revision); !ok {
			return
		}
		view.Revisions = append(view.Revisions, entry)
	}
	auth.JSONResponse(w, 200, view)
}

// revisionEntryWork fills one revision entry's affected mapped work with
// current evidenced states and authority freshness.
func (s *API) revisionEntryWork(w http.ResponseWriter, r *http.Request, entry *revisionEntryView, revision factory.IntentRevision) bool {
	bindings, err := s.Store.IntentTaskBindings(r.Context(), revision.ID)
	if err != nil {
		auth.JSONError(w, 503, "store_unavailable", "Could not read request mappings.")
		return false
	}
	for _, binding := range bindings {
		state, ok := s.requestWorkItemState(w, r, revision.Repository, binding)
		if !ok {
			return false
		}
		authority := revisionAuthorityUnknown
		if control, err := s.Store.IssueControl(r.Context(), revision.Repository, binding.IssueIndex); err == nil {
			authority = revisionAuthoritySuperseded
			if control.Acceptance != "" && control.Acceptance == binding.AcceptanceID {
				authority = revisionAuthorityCurrent
			}
		} else if !errors.Is(err, store.ErrNotFound) {
			auth.JSONError(w, 503, "store_unavailable", "Could not read work authority.")
			return false
		}
		entry.Work = append(entry.Work, revisionWorkView{
			TaskKey: binding.TaskKey, Issue: strconv.FormatInt(binding.IssueIndex, 10),
			State: state, Authority: authority,
		})
	}
	return true
}
