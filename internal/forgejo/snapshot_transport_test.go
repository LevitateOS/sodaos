package forgejo

import (
	"context"
	"errors"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

type fakeBackgroundClient struct {
	revision    extensions.NativeRevisionObservation
	revisionErr error
	snapshot    extensions.NativeSnapshot
	snapshotErr error
	lastRequest extensions.SnapshotRequest
}

func (f *fakeBackgroundClient) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	if f.revisionErr != nil {
		return extensions.NativeRevisionObservation{}, f.revisionErr
	}
	return f.revision, nil
}

func (f *fakeBackgroundClient) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, req extensions.SnapshotRequest) (extensions.NativeSnapshot, error) {
	f.lastRequest = req
	if f.snapshotErr != nil {
		return extensions.NativeSnapshot{}, f.snapshotErr
	}
	return f.snapshot, nil
}

func (f *fakeBackgroundClient) SubmitOperation(context.Context, extensions.CredentialFile, extensions.OperationIntent) (extensions.OperationRecord, error) {
	return extensions.OperationRecord{}, errors.New("not implemented")
}

func (f *fakeBackgroundClient) GetOperation(context.Context, string) (extensions.OperationLookup, error) {
	return extensions.OperationLookup{}, errors.New("not implemented")
}

func (f *fakeBackgroundClient) CancelOperation(context.Context, string) (extensions.OperationRecord, error) {
	return extensions.OperationRecord{}, errors.New("not implemented")
}

func wireSnapshotAnswer() extensions.NativeSnapshot {
	return extensions.NativeSnapshot{
		RepositoryID: "3",
		Issue: &extensions.SnapshotIssue{
			ID: "11", Index: "5", Title: "Exact objective", Content: "Selected body",
			TitleDigest: ContentDigest("Exact objective"), ContentDigest: ContentDigest("Selected body"),
			ContentVersion: 4, NumComments: 2,
			Provenance: extensions.SnapshotCreationProvenance{
				PosterID: "7", CreatedUnix: 1700000000, FirstCreated: true, Verified: true,
			},
			Lifecycle: []extensions.SnapshotLifecycleEvent{
				{Kind: "retitled", AtUnix: 1700000050, ActorID: "7", OldTitle: "Old", NewTitle: "Exact objective"},
			},
			CreatedUnix: 1700000000, UpdatedUnix: 1700000100,
			Visible: true, Complete: true,
		},
		Dependencies: &extensions.SnapshotDependencyPage{
			IssueID: "11",
			Items: []extensions.SnapshotDependency{
				{OccurrenceID: "21", IssueID: "11", Visible: false, HiddenReason: "no_access", Complete: true},
				{OccurrenceID: "22", IssueID: "11", DependencyID: "12", CreatedUnix: 1700000000, UpdatedUnix: 1700000000, Visible: true, Complete: true},
			},
			Total: 2, Complete: true,
		},
		Checks: &extensions.SnapshotCheckSet{
			SHA: snapshotTestSHA,
			Items: []extensions.SnapshotCheck{
				{ID: "61", Index: 2, SHA: snapshotTestSHA, Context: "verify.yaml", State: "success", CreatorID: "7", CreatedUnix: 1700000000, UpdatedUnix: 1700000100, Visible: true, Complete: true},
			},
			Total: 1, Complete: true,
		},
		Refs: []extensions.SnapshotRef{
			{Ref: snapshotTestRef, OID: snapshotTestSHA, Exists: true, Visible: true, Complete: true},
		},
	}
}

func TestBackgroundSnapshotReaderMapsWireSnapshot(t *testing.T) {
	client := &fakeBackgroundClient{
		revision: idleRevision(9),
		snapshot: wireSnapshotAnswer(),
	}
	reader := &BackgroundSnapshotReader{Client: client, ActorID: "7"}

	req := snapshotTestRequest()
	snapshot, err := BracketedRead(context.Background(), reader, "testdata/credential", req)
	if err != nil {
		t.Fatalf("bracketed transport read: %v", err)
	}
	if snapshot.Revision != 9 {
		t.Fatalf("revision %d, want bracket-bound 9", snapshot.Revision)
	}
	if client.lastRequest.ActorID != "7" || client.lastRequest.RepositoryID != "3" {
		t.Fatalf("request not mapped: %+v", client.lastRequest)
	}
	if snapshot.Issue == nil || snapshot.Issue.Title != "Exact objective" {
		t.Fatalf("issue not mapped: %+v", snapshot.Issue)
	}
	if !snapshot.Issue.Provenance.Verified || len(snapshot.Issue.Lifecycle) != 1 {
		t.Fatalf("provenance/lifecycle not mapped: %+v", snapshot.Issue)
	}
	if snapshot.Dependencies == nil || len(snapshot.Dependencies.Items) != 2 {
		t.Fatalf("dependencies not mapped: %+v", snapshot.Dependencies)
	}
	if !HasHiddenEvidence(req, snapshot) {
		t.Fatal("hidden edge not reported")
	}
	if snapshot.Checks == nil || snapshot.Checks.Items[0].State != "success" {
		t.Fatalf("checks not mapped: %+v", snapshot.Checks)
	}
	if len(snapshot.Refs) != 1 || snapshot.Refs[0].OID != snapshotTestSHA {
		t.Fatalf("refs not mapped: %+v", snapshot.Refs)
	}
}

func TestBackgroundSnapshotReaderRefusesUnusableTransport(t *testing.T) {
	req := snapshotTestRequest()
	if _, err := (*BackgroundSnapshotReader)(nil).ReadSnapshot(context.Background(), "testdata/credential", req); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("nil reader accepted: %v", err)
	}
	bad := &BackgroundSnapshotReader{Client: &fakeBackgroundClient{}, ActorID: "abc"}
	if _, err := bad.ReadSnapshot(context.Background(), "testdata/credential", req); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("bad actor accepted: %v", err)
	}
	unbound := &BackgroundSnapshotReader{Client: &fakeBackgroundClient{snapshot: wireSnapshotAnswer()}}
	if _, err := unbound.ReadSnapshot(context.Background(), "testdata/credential", req); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("missing actor accepted: %v", err)
	}
}

func TestBracketedReadBindsRevisionFromBracket(t *testing.T) {
	// A transport-returned revision is replaced by the bracket evidence,
	// never trusted: the fake answers revision 99 while the bracket holds 9.
	answered := snapshotTestSnapshot()
	answered.Revision = 99
	reader := &fakeSnapshotReader{before: idleRevision(9), after: idleRevision(9), snapshot: answered}
	snapshot, err := BracketedRead(context.Background(), reader, "testdata/credential", snapshotTestRequest())
	if err != nil {
		t.Fatalf("bracketed read: %v", err)
	}
	if snapshot.Revision != 9 {
		t.Fatalf("revision %d, want bracket-bound 9", snapshot.Revision)
	}
}

func TestValidateRequestRequiresFamilySelectors(t *testing.T) {
	bad := []SnapshotRequest{
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyIssue}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyDependencies}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyPull}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyComments}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyReviews}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyComments}, IssueIndex: "5", CommentIDs: []string{"11"}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyReviews}, IssueIndex: "5", PullNumber: "9"},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyIssue}, IssueIndex: "5", Cursor: "nope"},
	}
	for i, req := range bad {
		if err := ValidateRequest(req); !errors.Is(err, ErrInvalidSnapshot) {
			t.Fatalf("case %d accepted: %v", i, err)
		}
	}
	good := []SnapshotRequest{
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyComments}, CommentIDs: []string{"11"}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyPull, FamilyReviews}, PullNumber: "9"},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyComments, FamilyReviews}, IssueIndex: "5"},
	}
	for i, req := range good {
		if err := ValidateRequest(req); err != nil {
			t.Fatalf("case %d rejected: %v", i, err)
		}
	}
}
