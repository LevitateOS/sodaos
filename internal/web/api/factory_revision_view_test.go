package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestRevisionTaskDiff(t *testing.T) {
	old := []factory.IntentTask{
		{Key: "keep", Objective: "same", Outcomes: []string{"a"}, IdeaIDs: []string{"d1"}, DependsOn: []string{"x"}},
		{Key: "edit", Objective: "before", Outcomes: []string{"a"}, IdeaIDs: []string{"d1", "d2"}, DependsOn: []string{}},
		{Key: "drop", Objective: "gone", Outcomes: []string{"g"}},
	}
	current := []factory.IntentTask{
		{Key: "keep", Objective: "same", Outcomes: []string{"a"}, IdeaIDs: []string{"d1"}, DependsOn: []string{"x"}},
		{Key: "edit", Objective: "after", Outcomes: []string{"a", "b"}, IdeaIDs: []string{"d2", "d1"}, DependsOn: []string{}},
		{Key: "new", Objective: "added", Outcomes: []string{"n"}},
	}
	delta, err := factory.DiffIntentTasks(factory.IntentRevision{Intent: "d1", Tasks: old}, factory.IntentRevision{Intent: "d1", Tasks: current})
	if err != nil {
		t.Fatal(err)
	}
	diff := revisionTaskDiff(delta, old, current)
	if len(diff.Reused) != 1 || diff.Reused[0] != "keep" || diff.UnchangedCount != len(diff.Reused) {
		t.Fatalf("reused: %+v", diff)
	}
	if len(diff.Added) != 1 || diff.Added[0].Key != "new" {
		t.Fatalf("added: %+v", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].Key != "drop" || diff.Removed[0].Objective != "gone" {
		t.Fatalf("removed: %+v", diff.Removed)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].Key != "edit" {
		t.Fatalf("changed: %+v", diff.Changed)
	}
	fields := diff.Changed[0].ChangedFields
	if len(fields) != 2 || fields[0] != "objective" || fields[1] != "outcomes" {
		t.Fatalf("fields: %v", fields)
	}
	if diff.Changed[0].OldObjective != "before" || diff.Changed[0].NewObjective != "after" {
		t.Fatalf("objectives: %+v", diff.Changed[0])
	}
	if len(diff.Changed[0].OldOutcomes) != 1 || len(diff.Changed[0].NewOutcomes) != 2 {
		t.Fatalf("outcomes: %+v", diff.Changed[0])
	}
}

func TestRevisionIdeaDiff(t *testing.T) {
	old := []factory.ProductIdea{
		{ID: "d1", Decision: "pending", Description: "same"},
		{ID: "d2", Decision: "pending", Description: "draft", Correction: "fix"},
		{ID: "d3", Decision: "decline"},
	}
	current := []factory.ProductIdea{
		{ID: "d1", Decision: "accept", Description: "same"},
		{ID: "d2", Decision: "pending", Description: "draft", Correction: "fix"},
		{ID: "d4", Decision: "pending"},
	}
	diff := revisionIdeaDiff(old, current)
	if len(diff.Added) != 1 || diff.Added[0].ID != "d4" {
		t.Fatalf("added: %+v", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].ID != "d3" {
		t.Fatalf("removed: %+v", diff.Removed)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].ID != "d1" {
		t.Fatalf("changed: %+v", diff.Changed)
	}
	change := diff.Changed[0]
	if change.FromDecision != "pending" || change.ToDecision != "accept" || change.CorrectionChanged || change.ContentChanged {
		t.Fatalf("transition: %+v", change)
	}
}

func TestRevisionTaskDiffOrdersChangedByOld(t *testing.T) {
	old := []factory.IntentTask{{Key: "b", Objective: "1"}, {Key: "a", Objective: "1"}}
	current := []factory.IntentTask{{Key: "a", Objective: "2"}, {Key: "b", Objective: "2"}}
	delta, err := factory.DiffIntentTasks(factory.IntentRevision{Intent: "d1", Tasks: old}, factory.IntentRevision{Intent: "d1", Tasks: current})
	if err != nil {
		t.Fatal(err)
	}
	diff := revisionTaskDiff(delta, old, current)
	if len(diff.Changed) != 2 || diff.Changed[0].Key != "b" || diff.Changed[1].Key != "a" {
		t.Fatalf("changed order: %+v", diff.Changed)
	}
}

func TestRevisionTaskDiffRejectsCrossIntent(t *testing.T) {
	if _, err := factory.DiffIntentTasks(factory.IntentRevision{Intent: "d1"}, factory.IntentRevision{Intent: "d2"}); err == nil {
		t.Fatal("cross-intent delta accepted")
	}
}

func TestRevisionIDDiff(t *testing.T) {
	diff := revisionIDDiff([]string{"a", "b"}, []string{"b", "c"})
	if len(diff.Added) != 1 || diff.Added[0] != "c" {
		t.Fatalf("added: %+v", diff)
	}
	if len(diff.Removed) != 1 || diff.Removed[0] != "a" {
		t.Fatalf("removed: %+v", diff)
	}
	empty := revisionIDDiff(nil, nil)
	if empty.Added == nil || empty.Removed == nil {
		t.Fatalf("nil diffs: %+v", empty)
	}
}

func TestRevisionComparisonDTO(t *testing.T) {
	old := factory.IntentRevision{Intent: "d1", Repository: 7, State: "accepted", Tasks: []factory.IntentTask{{Key: "a"}}, EffectiveOutcomes: []string{"d1"}}
	current := factory.IntentRevision{Intent: "d1", Repository: 7, State: "accepted", Tasks: []factory.IntentTask{{Key: "a"}, {Key: "b"}}, EffectiveOutcomes: []string{"d2"}}
	delta, err := factory.DiffIntentTasks(old, current)
	if err != nil {
		t.Fatal(err)
	}
	diff := revisionComparisonDTO(delta, old, current)
	if diff.Initial || diff.Repository != nil || diff.State != nil || diff.EdgesChanged {
		t.Fatalf("context: %+v", diff)
	}
	if len(diff.Tasks.Added) != 1 || len(diff.Tasks.Reused) != 1 || diff.Tasks.Reused[0] != "a" {
		t.Fatalf("tasks: %+v", diff.Tasks)
	}
	if len(diff.Effective.Added) != 1 || len(diff.Effective.Removed) != 1 {
		t.Fatalf("effective: %+v", diff.Effective)
	}
	moved := current
	moved.Repository, moved.State = 8, "draft"
	movedDelta, err := factory.DiffIntentTasks(old, moved)
	if err != nil {
		t.Fatal(err)
	}
	shifted := revisionComparisonDTO(movedDelta, old, moved)
	if shifted.Repository == nil || shifted.Repository.From != "7" || shifted.Repository.To != "8" {
		t.Fatalf("repository: %+v", shifted)
	}
	if shifted.State == nil || shifted.State.From != "accepted" || shifted.State.To != "draft" {
		t.Fatalf("state: %+v", shifted)
	}
	initial := initialRevisionComparison()
	if !initial.Initial || initial.Tasks.Added == nil || initial.Ideas.Removed == nil || initial.Effective.Added == nil {
		t.Fatalf("initial: %+v", initial)
	}
}

func TestRevisionAuthorityState(t *testing.T) {
	if got := revisionAuthorityState(true, true); got != revisionAuthorityCurrent {
		t.Fatalf("current: %s", got)
	}
	if got := revisionAuthorityState(false, true); got != revisionAuthoritySuperseded {
		t.Fatalf("superseded: %s", got)
	}
	if got := revisionAuthorityState(true, false); got != revisionAuthorityUnknown {
		t.Fatalf("unrecorded: %s", got)
	}
	if got := revisionAuthorityState(false, false); got != revisionAuthorityUnknown {
		t.Fatalf("unrecorded stale: %s", got)
	}
}

func TestRevisionGenerationDTO(t *testing.T) {
	revision := factory.IntentRevision{Tasks: []factory.IntentTask{{Key: "a"}, {Key: "b"}}}
	matched := revisionGenerationDTO(factory.IntentGeneration{State: "completed", SubmittedTasks: []factory.IntentTask{{Key: "a"}, {Key: "b"}}}, revision)
	if matched.State != "completed" || matched.SubmittedDiffers || len(matched.SubmittedKeys) != 2 {
		t.Fatalf("matched: %+v", matched)
	}
	diverged := revisionGenerationDTO(factory.IntentGeneration{State: "completed", SubmittedTasks: []factory.IntentTask{{Key: "a"}}}, revision)
	if !diverged.SubmittedDiffers || len(diverged.SubmittedKeys) != 1 {
		t.Fatalf("diverged: %+v", diverged)
	}
	absent := revisionGenerationDTO(factory.IntentGeneration{State: "pending"}, revision)
	if absent.State != "pending" || absent.SubmittedDiffers || absent.SubmittedKeys == nil {
		t.Fatalf("absent: %+v", absent)
	}
}

func TestRevisionGenerationDeliveredRefs(t *testing.T) {
	revision := factory.IntentRevision{Tasks: []factory.IntentTask{{Key: "a"}}}
	pinned := revisionGenerationDTO(factory.IntentGeneration{
		State:          "completed",
		SubmittedTasks: []factory.IntentTask{{Key: "a"}},
		Delivered:      map[string]factory.DeliveredIssue{"k": {IssueID: 101, IssueIndex: 3, ContentVersion: 2}},
	}, revision)
	ref, ok := pinned.DeliveredRefs["k"]
	if !ok || ref.IssueID != 101 || ref.IssueIndex != 3 || ref.ContentVersion != 2 {
		t.Fatalf("delivered refs: %+v", pinned)
	}
	legacy := revisionGenerationDTO(factory.IntentGeneration{State: "completed"}, revision)
	if legacy.DeliveredRefs != nil {
		t.Fatalf("legacy delivered refs: %+v", legacy)
	}
	body, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "delivered_refs") {
		t.Fatalf("empty refs rendered: %s", body)
	}
}

func TestParseRevisionHistoryQuery(t *testing.T) {
	if limit, ok := parseRevisionHistoryQuery(""); !ok || limit != DefaultRevisionComparisons {
		t.Fatalf("defaults: %d %v", limit, ok)
	}
	if limit, ok := parseRevisionHistoryQuery("limit=5"); !ok || limit != 5 {
		t.Fatalf("bound: %d %v", limit, ok)
	}
	for _, raw := range []string{"limit=0", "limit=6", "limit=many", "limit=2&cursor=x", strings.Repeat("x", 513)} {
		if _, ok := parseRevisionHistoryQuery(raw); ok {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestRevisionEntryRendersArrays(t *testing.T) {
	body, err := json.Marshal(factoryRevisionHistoryView{Revisions: []revisionEntryView{{Comparison: initialRevisionComparison(), Work: []revisionWorkView{}}}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	entry := decoded["revisions"].([]any)[0].(map[string]any)
	if work, ok := entry["work"].([]any); !ok || len(work) != 0 {
		t.Fatalf("work: %v", entry["work"])
	}
	tasks := entry["comparison"].(map[string]any)["tasks"].(map[string]any)
	for _, key := range []string{"added", "removed", "changed", "reused"} {
		if items, ok := tasks[key].([]any); !ok || len(items) != 0 {
			t.Fatalf("%s: %v", key, tasks[key])
		}
	}
}
