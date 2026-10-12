package api

import (
	"context"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
)

func TestLifecycleConfigDTOGatesEnabledOnCapability(t *testing.T) {
	c := factory.LifecycleConfig{Event: factory.LifecycleEventIssueCreated, Revision: 2, Enabled: true, ActorID: 7, RepositoryID: 9, LabelID: 3, AssigneeID: 15}
	proved := lifecycleConfigDTO(c, true)
	if !proved.Enabled || !proved.Capable || proved.LabelID != "3" || proved.AssigneeID != "15" || proved.Revision != "2" {
		t.Fatalf("proved: %+v", proved)
	}
	unproved := lifecycleConfigDTO(c, false)
	if unproved.Enabled || unproved.Capable {
		t.Fatalf("unproved still enabled: %+v", unproved)
	}
	review := lifecycleConfigDTO(factory.LifecycleConfig{Event: factory.LifecycleEventPRCreated, Enabled: true, ActorID: 7, RepositoryID: 9, ReviewerIDs: []int64{15, 23}}, true)
	if len(review.ReviewerIDs) != 2 || review.ReviewerIDs[0] != "15" {
		t.Fatalf("reviewers: %+v", review)
	}
}

func TestAbsentConfigViewReportsCapability(t *testing.T) {
	proved := absentLifecycleConfigView(factory.LifecycleEventIssueCreated, 9, true)
	if proved.Enabled || !proved.Capable || proved.Revision != "0" || proved.RepositoryID != "9" {
		t.Fatalf("proved default: %+v", proved)
	}
	unproved := absentLifecycleConfigView(factory.LifecycleEventIssueCreated, 9, false)
	if unproved.Enabled || unproved.Capable {
		t.Fatalf("unproved default: %+v", unproved)
	}
}

func TestLifecycleActionDTORendersBudget(t *testing.T) {
	a := factory.LifecycleAction{ID: "a1", Event: factory.LifecycleEventVerifiedMerge, Action: factory.LifecycleActionCompletionNote, ActorID: 7, RepositoryID: 9, IssueID: 12, Note: "done", AuthRevision: "r", MergeRef: "m", Closed: true, Attempts: 1, Results: map[string]string{"note": factory.LifecycleResultUncertain}}
	view := lifecycleActionDTO(a)
	if view.Attempts != 1 || view.MaxAttempts != factory.MaxLifecycleAttempts || view.Settled || view.Results["note"] != factory.LifecycleResultUncertain {
		t.Fatalf("view: %+v", view)
	}
}

func TestDecodeLifecycleConfigSaveRejects(t *testing.T) {
	save, err := decodeLifecycleConfigSave([]byte(`{"revision":1,"enabled":true,"reviewer_ids":[15]}`))
	if err != nil || save.Revision != 1 || !save.Enabled || len(save.ReviewerIDs) != 1 {
		t.Fatalf("valid rejected: %+v %v", save, err)
	}
	for name, body := range map[string]string{
		"unknown field": `{"revision":1,"enabled":true,"owner_id":9}`,
		"trailing":      `{"revision":1} {}`,
		"malformed":     `{oops`,
		"wrong type":    `{"revision":"1"}`,
	} {
		if _, err := decodeLifecycleConfigSave([]byte(body)); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

type apiTestLifecycleExecutor struct{}

func (apiTestLifecycleExecutor) EnrolledActor(context.Context) (int64, error) {
	return 7, nil
}

func (apiTestLifecycleExecutor) ObserveLifecycleRevision(context.Context, int64) (int64, error) {
	return 1, nil
}

func (apiTestLifecycleExecutor) ResolveNoteTarget(context.Context, int64, int64) (int64, error) {
	return 1, nil
}

func (apiTestLifecycleExecutor) SubmitLifecycle(context.Context, factory.LifecycleWork) (factory.OperationOutcome, error) {
	return factory.OperationOutcome{}, nil
}

func (apiTestLifecycleExecutor) LookupLifecycle(context.Context, string) (factory.OperationOutcome, error) {
	return factory.OperationOutcome{}, nil
}

// TestLifecycleExecutorReady gates runnable claims on a configured
// executor: marker-only setups never read capable.
func TestLifecycleExecutorReady(t *testing.T) {
	var nilAPI *API
	if nilAPI.lifecycleExecutorReady() {
		t.Fatal("nil API ready")
	}
	if (&API{}).lifecycleExecutorReady() {
		t.Fatal("missing coordinator ready")
	}
	if (&API{Coordinator: &control.Coordinator{}}).lifecycleExecutorReady() {
		t.Fatal("missing executor ready")
	}
	wired := &API{Coordinator: &control.Coordinator{Lifecycle: apiTestLifecycleExecutor{}}}
	if !wired.lifecycleExecutorReady() {
		t.Fatal("wired executor not ready")
	}
}
