package forgejo

import (
	"context"
	"errors"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

// P5 narrow native proof (FW29, Soda side): repository_issues read-only
// adapter shape. The family is requested alone with only Limit/Cursor
// selectors; terminal/nonterminal page rules hold; an empty incomplete
// page is invalid. No acceptance, readiness or other factory policy is
// expressed or asserted here (FW32 owns those).

func repositoryIssuesTestRequest() SnapshotRequest {
	return SnapshotRequest{
		RepositoryID: "3",
		Families:     []SnapshotFamily{FamilyRepositoryIssues},
	}
}

func repositoryIssuesTestPage() *RepositoryIssuesPage {
	return &RepositoryIssuesPage{
		Items: []RepositoryIssueItem{
			{IssueID: "11", IssueIndex: "5", Visible: true},
			{IssueID: "12", IssueIndex: "6", Visible: true},
		},
		Complete: true,
	}
}

func TestValidateRequestBoundsRepositoryIssues(t *testing.T) {
	good := repositoryIssuesTestRequest()
	if err := ValidateRequest(good); err != nil {
		t.Fatalf("valid request: %v", err)
	}
	paged := repositoryIssuesTestRequest()
	paged.Limit = 50
	paged.Cursor = "6"
	if err := ValidateRequest(paged); err != nil {
		t.Fatalf("paged request: %v", err)
	}
	bad := []SnapshotRequest{
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRepositoryIssues}, IssueIndex: "5"},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRepositoryIssues}, PullNumber: "9"},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRepositoryIssues}, CommentIDs: []string{"11"}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRepositoryIssues}, SHA: snapshotTestSHA},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRepositoryIssues}, Refs: []string{snapshotTestRef}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRepositoryIssues, FamilyIssue}, IssueIndex: "5"},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRepositoryIssues}, Limit: 51},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRepositoryIssues}, Cursor: "nope"},
	}
	for i, req := range bad {
		if err := ValidateRequest(req); !errors.Is(err, ErrInvalidSnapshot) {
			t.Fatalf("case %d accepted: %v", i, err)
		}
	}
}

func TestBracketedReadAcceptsTerminalRepositoryIssues(t *testing.T) {
	req := repositoryIssuesTestRequest()
	snapshot := NativeSnapshot{Revision: 9, RepositoryID: "3", RepositoryIssues: repositoryIssuesTestPage()}
	reader := &fakeSnapshotReader{before: idleRevision(9), after: idleRevision(9), snapshot: snapshot}
	accepted, err := BracketedRead(context.Background(), reader, "testdata/credential", req)
	if err != nil {
		t.Fatalf("bracketed read: %v", err)
	}
	if len(accepted.RepositoryIssues.Items) != 2 || accepted.RepositoryIssues.NextCursor != "" {
		t.Fatalf("page not preserved: %+v", accepted.RepositoryIssues)
	}
	if HasHiddenEvidence(req, accepted) {
		t.Fatal("visible page reported hidden")
	}
}

func TestBracketedReadRefusesRepositoryIssuesViolations(t *testing.T) {
	req := repositoryIssuesTestRequest()
	cases := map[string]NativeSnapshot{
		"missing page": {Revision: 9, RepositoryID: "3"},
		"nonterminal page": {
			Revision: 9, RepositoryID: "3",
			RepositoryIssues: &RepositoryIssuesPage{
				Items:      []RepositoryIssueItem{{IssueID: "11", IssueIndex: "5", Visible: true}},
				NextCursor: "5",
			},
		},
		"empty incomplete page": {Revision: 9, RepositoryID: "3", RepositoryIssues: &RepositoryIssuesPage{}},
		"terminal page with cursor": {
			Revision: 9, RepositoryID: "3",
			RepositoryIssues: &RepositoryIssuesPage{
				Items:      []RepositoryIssueItem{{IssueID: "11", IssueIndex: "5", Visible: true}},
				NextCursor: "5", Complete: true,
			},
		},
		"cursor mismatch": {
			Revision: 9, RepositoryID: "3",
			RepositoryIssues: &RepositoryIssuesPage{
				Items:      []RepositoryIssueItem{{IssueID: "11", IssueIndex: "5", Visible: true}},
				NextCursor: "6",
			},
		},
		"descending indexes": {
			Revision: 9, RepositoryID: "3",
			RepositoryIssues: &RepositoryIssuesPage{
				Items: []RepositoryIssueItem{
					{IssueID: "12", IssueIndex: "6", Visible: true},
					{IssueID: "11", IssueIndex: "5", Visible: true},
				},
				Complete: true,
			},
		},
	}
	for name, snapshot := range cases {
		reader := &fakeSnapshotReader{before: idleRevision(9), after: idleRevision(9), snapshot: snapshot}
		if _, err := BracketedRead(context.Background(), reader, "testdata/credential", req); !errors.Is(err, ErrIncompleteSnapshot) && !errors.Is(err, ErrInvalidSnapshot) {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestRepositoryIssuesHiddenItemBlocksConsumer(t *testing.T) {
	req := repositoryIssuesTestRequest()
	snapshot := NativeSnapshot{
		Revision: 9, RepositoryID: "3",
		RepositoryIssues: &RepositoryIssuesPage{
			Items:    []RepositoryIssueItem{{IssueID: "11", IssueIndex: "5", Visible: false}},
			Complete: true,
		},
	}
	reader := &fakeSnapshotReader{before: idleRevision(9), after: idleRevision(9), snapshot: snapshot}
	accepted, err := BracketedRead(context.Background(), reader, "testdata/credential", req)
	if err != nil {
		t.Fatalf("hidden locator refused the read: %v", err)
	}
	if !HasHiddenEvidence(req, accepted) {
		t.Fatal("hidden locator not reported")
	}
}

func TestBackgroundSnapshotReaderMapsRepositoryIssues(t *testing.T) {
	client := &fakeBackgroundClient{
		revision: idleRevision(9),
		snapshot: extensions.NativeSnapshot{
			RepositoryID:     "3",
			RepositoryIssues: repositoryIssuesTestPage(),
		},
	}
	reader := &BackgroundSnapshotReader{Client: client, ActorID: "7"}
	req := repositoryIssuesTestRequest()
	req.Limit = 50
	req.Cursor = "4"
	snapshot, err := BracketedRead(context.Background(), reader, "testdata/credential", req)
	if err != nil {
		t.Fatalf("bracketed transport read: %v", err)
	}
	if snapshot.RepositoryIssues == nil || len(snapshot.RepositoryIssues.Items) != 2 {
		t.Fatalf("page not mapped: %+v", snapshot.RepositoryIssues)
	}
	if client.lastRequest.Families[0] != string(FamilyRepositoryIssues) || client.lastRequest.Limit != 50 || client.lastRequest.Cursor != "4" {
		t.Fatalf("request not mapped: %+v", client.lastRequest)
	}
}
