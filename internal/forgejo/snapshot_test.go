package forgejo

import (
	"context"
	"errors"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

const (
	snapshotTestSHA = "0123456789abcdef0123456789abcdef01234567"
	snapshotTestRef = "refs/heads/candidate"
)

type fakeSnapshotReader struct {
	before, after extensions.NativeRevisionObservation
	snapshot      NativeSnapshot
	snapshotErr   error
	revisionErr   error
}

func (f *fakeSnapshotReader) ReadNativeRevision(context.Context) (extensions.NativeRevisionObservation, error) {
	if f.revisionErr != nil {
		return extensions.NativeRevisionObservation{}, f.revisionErr
	}
	if f.before.Revision != 0 && f.after.Revision != 0 {
		observation := f.before
		f.before.Revision = 0
		return observation, nil
	}
	return f.after, nil
}

func (f *fakeSnapshotReader) ReadSnapshot(_ context.Context, _ extensions.CredentialFile, _ SnapshotRequest) (NativeSnapshot, error) {
	return f.snapshot, f.snapshotErr
}

func snapshotTestRequest() SnapshotRequest {
	return SnapshotRequest{
		RepositoryID: "3",
		Families:     []SnapshotFamily{FamilyIssue, FamilyDependencies, FamilyChecks, FamilyRefs},
		IssueIndex:   "5",
		SHA:          snapshotTestSHA,
		Refs:         []string{snapshotTestRef},
	}
}

func snapshotTestSnapshot() NativeSnapshot {
	return NativeSnapshot{
		Revision:     9,
		RepositoryID: "3",
		Issue: &IssueEvidence{
			ID: "11", Index: "5", Title: "Exact objective", Content: "Selected body",
			TitleDigest: ContentDigest("Exact objective"), ContentDigest: ContentDigest("Selected body"),
			ContentVersion: 4, Visible: true, Complete: true,
		},
		Dependencies: &DependencyPage{
			IssueID: "11",
			Items: []DependencyEvidence{
				{OccurrenceID: "31", IssueID: "11", DependencyID: "12", Visible: true, Complete: true},
			},
			Total: 1, Complete: true,
		},
		Checks: &CheckSet{
			SHA: snapshotTestSHA,
			Items: []CheckEvidence{
				{ID: "61", SHA: snapshotTestSHA, Context: "verify.yaml", State: "success", Visible: true, Complete: true},
			},
			Total: 1, Complete: true,
		},
		Refs: []RefEvidence{
			{Ref: snapshotTestRef, OID: snapshotTestSHA, Exists: true, Visible: true, Complete: true},
		},
	}
}

func idleRevision(revision int64) extensions.NativeRevisionObservation {
	return extensions.NativeRevisionObservation{Revision: revision, Idle: true}
}

func TestBracketedReadAcceptsEqualIdleBracket(t *testing.T) {
	reader := &fakeSnapshotReader{before: idleRevision(9), after: idleRevision(9), snapshot: snapshotTestSnapshot()}
	snapshot, err := BracketedRead(context.Background(), reader, "testdata/credential", snapshotTestRequest())
	if err != nil {
		t.Fatalf("bracketed read: %v", err)
	}
	if snapshot.Revision != 9 || snapshot.Issue.Title != "Exact objective" {
		t.Fatal("accepted snapshot lost evidence")
	}
	if HasHiddenEvidence(snapshotTestRequest(), snapshot) {
		t.Fatal("visible snapshot reported hidden")
	}
}

func TestBracketedReadRefusesInterveningChange(t *testing.T) {
	cases := map[string]fakeSnapshotReader{
		"changed revision": {before: idleRevision(9), after: idleRevision(10), snapshot: snapshotTestSnapshot()},
		"busy before":      {before: extensions.NativeRevisionObservation{Revision: 9}, after: idleRevision(9), snapshot: snapshotTestSnapshot()},
		"busy after":       {before: idleRevision(9), after: extensions.NativeRevisionObservation{Revision: 9}, snapshot: snapshotTestSnapshot()},
	}
	for name, reader := range cases {
		t.Run(name, func(t *testing.T) {
			reader := reader
			_, err := BracketedRead(context.Background(), &reader, "testdata/credential", snapshotTestRequest())
			if !errors.Is(err, ErrStaleSnapshot) && !errors.Is(err, ErrNativeBusy) {
				t.Fatalf("bracket accepted: %v", err)
			}
		})
	}
}

func TestBracketedReadRefusesMissingPage(t *testing.T) {
	snapshot := snapshotTestSnapshot()
	snapshot.Checks = nil
	reader := &fakeSnapshotReader{before: idleRevision(9), after: idleRevision(9), snapshot: snapshot}
	if _, err := BracketedRead(context.Background(), reader, "testdata/credential", snapshotTestRequest()); !errors.Is(err, ErrIncompleteSnapshot) {
		t.Fatalf("missing page accepted: %v", err)
	}

	incomplete := snapshotTestSnapshot()
	incomplete.Dependencies.Complete = false
	reader = &fakeSnapshotReader{before: idleRevision(9), after: idleRevision(9), snapshot: incomplete}
	if _, err := BracketedRead(context.Background(), reader, "testdata/credential", snapshotTestRequest()); !errors.Is(err, ErrIncompleteSnapshot) {
		t.Fatalf("incomplete page accepted: %v", err)
	}
}

func TestBracketedReadReturnsHiddenRecordsForAuthorization(t *testing.T) {
	snapshot := snapshotTestSnapshot()
	snapshot.Dependencies.Items[0] = DependencyEvidence{
		OccurrenceID: "31", IssueID: "11", Visible: false, HiddenReason: "no_access", Complete: true,
	}
	reader := &fakeSnapshotReader{before: idleRevision(9), after: idleRevision(9), snapshot: snapshot}
	accepted, err := BracketedRead(context.Background(), reader, "testdata/credential", snapshotTestRequest())
	if err != nil {
		t.Fatalf("hidden prerequisite refused the read: %v", err)
	}
	if !HasHiddenEvidence(snapshotTestRequest(), accepted) {
		t.Fatal("hidden prerequisite not reported")
	}
	if accepted.Dependencies.Items[0].DependencyID != "" {
		t.Fatal("hidden prerequisite leaked its target")
	}
}

func TestBracketedReadRefusesDigestMismatch(t *testing.T) {
	snapshot := snapshotTestSnapshot()
	snapshot.Issue.Content = "changed body"
	reader := &fakeSnapshotReader{before: idleRevision(9), after: idleRevision(9), snapshot: snapshot}
	if _, err := BracketedRead(context.Background(), reader, "testdata/credential", snapshotTestRequest()); !errors.Is(err, ErrInvalidSnapshot) {
		t.Fatalf("digest mismatch accepted: %v", err)
	}
}

func TestValidateRequestBoundsFamilies(t *testing.T) {
	good := snapshotTestRequest()
	if err := ValidateRequest(good); err != nil {
		t.Fatalf("valid request: %v", err)
	}
	bad := []SnapshotRequest{
		{RepositoryID: "0", Families: []SnapshotFamily{FamilyIssue}},
		{RepositoryID: "3"},
		{RepositoryID: "3", Families: []SnapshotFamily{"approval"}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRefs}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyChecks}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyIssue}, IssueIndex: "abc"},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyIssue}, Limit: 500},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyChecks}, SHA: "short"},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyRefs}, Refs: []string{"candidate"}},
	}
	for i, req := range bad {
		if err := ValidateRequest(req); !errors.Is(err, ErrInvalidSnapshot) {
			t.Fatalf("case %d accepted: %v", i, err)
		}
	}
}

func TestContentDigestIsStable(t *testing.T) {
	first, second := ContentDigest("Selected body"), ContentDigest("Selected body")
	if first != second || first == "" {
		t.Fatal("digest not stable")
	}
	if ContentDigest("a") == ContentDigest("b") {
		t.Fatal("digest collision on test inputs")
	}
}
