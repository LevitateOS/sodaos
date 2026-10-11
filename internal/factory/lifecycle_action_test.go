package factory

import (
	"strings"
	"testing"
)

func testIssueAction() LifecycleAction {
	return LifecycleAction{ID: "act-1", Event: LifecycleEventIssueCreated, Action: LifecycleActionIssueLabelAssignee, ActorID: 7, RepositoryID: 9, IssueID: 42, LabelID: 3, AssigneeID: 15, AuthRevision: "rev-1"}
}

func TestLifecycleActionValidate(t *testing.T) {
	if err := testIssueAction().Validate(); err != nil {
		t.Fatalf("valid issue action rejected: %v", err)
	}
	review := LifecycleAction{ID: "act-2", Event: LifecycleEventPRCreated, Action: LifecycleActionPRReviewerRequest, ActorID: 7, RepositoryID: 9, IssueID: 88, ReviewerIDs: []int64{15, 23, 29}, AuthRevision: "rev-1"}
	if err := review.Validate(); err != nil {
		t.Fatalf("valid review action rejected: %v", err)
	}
	note := LifecycleAction{ID: "act-3", Event: LifecycleEventVerifiedMerge, Action: LifecycleActionCompletionNote, ActorID: 7, RepositoryID: 9, IssueID: 12, Note: "Factory completed the work.", AuthRevision: "rev-1", MergeRef: "pr/88@abc", Closed: true}
	if err := note.Validate(); err != nil {
		t.Fatalf("valid completion action rejected: %v", err)
	}
	emptyLabel := testIssueAction()
	emptyLabel.LabelID, emptyLabel.AssigneeID = 0, 0
	manyReviewers := review
	manyReviewers.ReviewerIDs = []int64{1, 2, 3, 4}
	noteOpen := note
	noteOpen.Closed = false
	noteLong := note
	noteLong.Note = strings.Repeat("x", MaxCompletionNoteChars+1)
	for i, a := range []LifecycleAction{{}, emptyLabel, manyReviewers, noteOpen, noteLong} {
		if err := a.Validate(); err == nil {
			t.Fatalf("case %d accepted", i)
		}
	}
}

func TestLifecycleConfigValidate(t *testing.T) {
	ok := LifecycleConfig{Event: LifecycleEventPRCreated, Revision: 1, Enabled: true, ActorID: 7, RepositoryID: 9, ReviewerIDs: []int64{15}}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	disabled := LifecycleConfig{Event: LifecycleEventIssueCreated, Revision: 0, ActorID: 7, RepositoryID: 9}
	if err := disabled.Validate(); err != nil {
		t.Fatalf("disabled config rejected: %v", err)
	}
	empty := ok
	empty.ReviewerIDs = nil
	mixed := ok
	mixed.Note = "x"
	if err := empty.Validate(); err == nil {
		t.Fatal("enabled review config without reviewers accepted")
	}
	if err := mixed.Validate(); err == nil {
		t.Fatal("review config with note accepted")
	}
}

func TestLifecycleActionAttempts(t *testing.T) {
	a := testIssueAction()
	if !a.Applicable() {
		t.Fatal("valid action not applicable")
	}
	if !a.CanAttempt("label", 1) {
		t.Fatal("first attempt refused")
	}
	if a.CanAttempt("label", 2) {
		t.Fatal("retry admitted before first attempt")
	}
	a = a.RecordAttempt("label", "op-1")
	if a.Attempts != 1 || a.Results["label"] != LifecycleResultUncertain {
		t.Fatalf("attempt not recorded: %+v", a)
	}
	if a.CanAttempt("label", 2) {
		t.Fatal("retry admitted while uncertain")
	}
	a = a.RecordResult("label", LifecycleResultFailed)
	if !a.CanAttempt("label", 2) {
		t.Fatal("retry refused after definitive no-effect")
	}
	a = a.RecordAttempt("label", "op-2")
	a = a.RecordResult("label", LifecycleResultDelivered)
	if a.CanAttempt("label", 3) || a.Attempts != MaxLifecycleAttempts {
		t.Fatal("third attempt admitted")
	}
	if a.Settled() {
		t.Fatal("settled with pending assignee target")
	}
	a = a.RecordResult("assignee", LifecycleResultDenied)
	if !a.Settled() {
		t.Fatal("complete action not settled")
	}
	revoked := testIssueAction()
	revoked.Revoked = true
	if revoked.Applicable() || revoked.CanAttempt("label", 1) {
		t.Fatal("revoked action still applicable")
	}
}
