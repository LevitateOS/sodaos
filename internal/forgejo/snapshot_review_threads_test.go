package forgejo

import (
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
)

func threadTestPage() *ThreadPage {
	return &ThreadPage{
		IssueID:    "11",
		PullNumber: "9",
		Items: []ThreadEvidence{
			{
				RootID: "21", IssueID: "11", ReviewID: "31",
				Path: "a.go", Side: "new", Line: 4,
				OriginalCommit: snapshotTestSHA, HeadCommit: snapshotTestSHA,
				ResolverID: "", Resolved: false,
				Digest: strings.Repeat("a", 64),
				Members: []ThreadMemberEvidence{
					{ID: "21", IssueID: "11", ContentDigest: ContentDigest("needs a guard"), ContentVersion: 1, CreatedUnix: 1700000000, UpdatedUnix: 1700000000, Visible: true, Complete: true},
					{ID: "22", IssueID: "11", ContentDigest: ContentDigest("guard added"), ContentVersion: 2, CreatedUnix: 1700000100, UpdatedUnix: 1700000100, Visible: true, Complete: true},
				},
				MemberTotal: 2, Visible: true, Complete: true,
			},
		},
		Total: 1, Complete: true,
	}
}

func TestValidateRequestReviewThreadsAndEligibility(t *testing.T) {
	byPull := SnapshotRequest{
		RepositoryID: "3",
		Families:     []SnapshotFamily{FamilyReviewThreads, FamilyMergeEligibility},
		PullNumber:   "9",
	}
	if err := ValidateRequest(byPull); err != nil {
		t.Fatalf("threads+eligibility by pull refused: %v", err)
	}
	byIssue := SnapshotRequest{
		RepositoryID: "3",
		Families:     []SnapshotFamily{FamilyReviewThreads},
		IssueIndex:   "5",
	}
	if err := ValidateRequest(byIssue); err != nil {
		t.Fatalf("threads by issue refused: %v", err)
	}
	bad := []SnapshotRequest{
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyReviewThreads}},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyReviewThreads}, IssueIndex: "5", PullNumber: "9"},
		{RepositoryID: "3", Families: []SnapshotFamily{FamilyMergeEligibility}},
	}
	for i, req := range bad {
		if err := ValidateRequest(req); err == nil {
			t.Fatalf("case %d accepted", i)
		}
	}
}

func TestValidThreadPageAcceptsCompleteEvidence(t *testing.T) {
	req := SnapshotRequest{RepositoryID: "3", Families: []SnapshotFamily{FamilyReviewThreads}, PullNumber: "9"}
	if err := validThreadPage(threadTestPage(), req); err != nil {
		t.Fatalf("complete thread page refused: %v", err)
	}
}

func TestValidThreadPageRefusesShapes(t *testing.T) {
	req := SnapshotRequest{RepositoryID: "3", Families: []SnapshotFamily{FamilyReviewThreads}, PullNumber: "9"}
	good := threadTestPage()
	good.NextCursor = ""
	good.Complete = false
	if err := validThreadPage(good, req); err != nil {
		t.Fatalf("terminal-incomplete page refused: %v", err)
	}
	next := threadTestPage()
	next.NextCursor = "21"
	next.Complete = false
	cursorReq := SnapshotRequest{RepositoryID: "3", Families: []SnapshotFamily{FamilyReviewThreads}, PullNumber: "9", Cursor: "20"}
	if err := validThreadPage(next, cursorReq); err != nil {
		t.Fatalf("nonterminal page refused: %v", err)
	}
	badCursor := threadTestPage()
	badCursor.NextCursor = "22"
	badCursor.Complete = false
	if err := validThreadPage(badCursor, cursorReq); err == nil {
		t.Fatal("cursor off last root accepted")
	}
	empty := threadTestPage()
	empty.Items = nil
	empty.Complete = false
	if err := validThreadPage(empty, req); err == nil {
		t.Fatal("empty incomplete page accepted")
	}
	hidden := threadTestPage()
	hidden.Items[0].Visible = false
	hidden.Items[0].Digest = strings.Repeat("b", 64)
	if err := validThreadPage(hidden, req); err == nil {
		t.Fatal("hidden thread with digest accepted")
	}
	redacted := threadTestPage()
	redacted.Items[0] = ThreadEvidence{RootID: "21", IssueID: "11", ReviewID: "31", Visible: false, Complete: true}
	redacted.Complete = false
	if err := validThreadPage(redacted, req); err != nil {
		t.Fatalf("redacted thread refused: %v", err)
	}
	mismatch := threadTestPage()
	mismatch.Items[0].Resolved = true
	if err := validThreadPage(mismatch, req); err == nil {
		t.Fatal("resolver mismatch accepted")
	}
}

func TestValidEligibilityVerdicts(t *testing.T) {
	req := SnapshotRequest{RepositoryID: "3", Families: []SnapshotFamily{FamilyMergeEligibility}, PullNumber: "9"}
	eligible := &MergeEligibility{PullNumber: "9", HeadTip: snapshotTestSHA, Eligible: true, Visible: true, Complete: true}
	if err := validEligibility(eligible, req); err != nil {
		t.Fatalf("eligible verdict refused: %v", err)
	}
	blocked := &MergeEligibility{
		PullNumber: "9", HeadTip: snapshotTestSHA, Eligible: false,
		Blocker: extensions.MergeBlockerApprovals, Visible: true, Complete: true,
	}
	if err := validEligibility(blocked, req); err != nil {
		t.Fatalf("blocked verdict refused: %v", err)
	}
	rejecting := &MergeEligibility{
		PullNumber: "9", Eligible: false, Blocker: extensions.MergeBlockerRequestedChanges,
		BlockingReviewIDs: []string{"31", "32"}, Visible: true, Complete: true,
	}
	if err := validEligibility(rejecting, req); err != nil {
		t.Fatalf("requested-changes verdict refused: %v", err)
	}
	bad := []*MergeEligibility{
		{PullNumber: "8", Eligible: true, Visible: true, Complete: true},
		{PullNumber: "9", Eligible: true, Blocker: extensions.MergeBlockerApprovals, Visible: true, Complete: true},
		{PullNumber: "9", Eligible: false, Blocker: "vibes", Visible: true, Complete: true},
		{PullNumber: "9", Eligible: false, Blocker: extensions.MergeBlockerRequestedChanges, BlockingReviewIDs: []string{"31", "31"}, Visible: true, Complete: true},
		{PullNumber: "9", Eligible: true, Visible: false, Complete: true},
	}
	for i, got := range bad {
		if err := validEligibility(got, req); err == nil {
			t.Fatalf("case %d accepted", i)
		}
	}
	hidden := &MergeEligibility{PullNumber: "9", Visible: false, Complete: true}
	if err := validEligibility(hidden, req); err != nil {
		t.Fatalf("hidden verdict refused: %v", err)
	}
}

func TestValidateSnapshotThreadsAndEligibility(t *testing.T) {
	req := SnapshotRequest{
		RepositoryID: "3",
		Families:     []SnapshotFamily{FamilyReviewThreads, FamilyMergeEligibility},
		PullNumber:   "9",
	}
	snapshot := NativeSnapshot{
		Revision: 7, RepositoryID: "3",
		Threads:     threadTestPage(),
		Eligibility: &MergeEligibility{PullNumber: "9", Eligible: true, Visible: true, Complete: true},
	}
	if err := ValidateSnapshot(req, snapshot, 7); err != nil {
		t.Fatalf("thread+eligibility snapshot refused: %v", err)
	}
	incomplete := snapshot
	incomplete.Threads = threadTestPage()
	incomplete.Threads.Complete = false
	if err := ValidateSnapshot(req, incomplete, 7); err == nil {
		t.Fatal("incomplete threads accepted")
	}
	missing := snapshot
	missing.Eligibility = nil
	if err := ValidateSnapshot(req, missing, 7); err == nil {
		t.Fatal("missing eligibility accepted")
	}
	if HasHiddenEvidence(req, snapshot) {
		t.Fatal("visible evidence reports hidden")
	}
	hidden := snapshot
	hidden.Threads = threadTestPage()
	hidden.Threads.Items[0] = ThreadEvidence{RootID: "21", IssueID: "11", ReviewID: "31", Visible: false, Complete: true}
	if !HasHiddenEvidence(req, hidden) {
		t.Fatal("hidden thread not reported")
	}
	hiddenEligibility := snapshot
	hiddenEligibility.Eligibility = &MergeEligibility{PullNumber: "9", Visible: false, Complete: true}
	if !HasHiddenEvidence(req, hiddenEligibility) {
		t.Fatal("hidden eligibility not reported")
	}
}

func TestThreadLocationMatchesHostRendering(t *testing.T) {
	if got := threadLocation("a.go", "new", 4); got != "a.go|new|4" {
		t.Fatalf("thread location diverged: %q", got)
	}
	if got := threadLocation("README.md", "old", 4); got != extensions.RenderThreadLocation("README.md", "old", 4) {
		t.Fatalf("thread location alias diverged: %q", got)
	}
}
