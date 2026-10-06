package forgejo

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	forgejopublish "github.com/levitateos/sodaos/internal/forgejo/publish"
)

func checkTargetForObserve() factory.CheckTarget {
	return factory.CheckTarget{
		Repository: 7, PRNumber: 3, PRID: 11, IssueID: 13,
		HeadRef: "refs/heads/soda/factory/candidate", BaseRef: "refs/heads/main",
		HeadOID: strings.Repeat("a", 40), BaseOID: strings.Repeat("b", 40),
	}
}

func checkSnapshotFixture() NativeSnapshot {
	head, base := strings.Repeat("a", 40), strings.Repeat("b", 40)
	return NativeSnapshot{
		Revision: 41, RepositoryID: "7",
		Pull: &PullEvidence{
			ID: "11", IssueID: "13", Number: "3",
			HeadRepoID: "7", HeadBranch: "soda/factory/candidate", BaseBranch: "main",
			HeadTip: head, Visible: true, Complete: true,
		},
		Checks: &CheckSet{
			SHA: head,
			Items: []CheckEvidence{
				{ID: "21", SHA: head, Context: "st11-build", State: "failure", CreatedUnix: 1, UpdatedUnix: 1, Visible: true, Complete: true},
				{ID: "22", SHA: head, Context: "st11-build", State: "success", CreatedUnix: 2, UpdatedUnix: 2, Visible: true, Complete: true},
				{ID: "23", SHA: head, Context: "st11-review-gate", State: "success", CreatedUnix: 2, UpdatedUnix: 2, Visible: true, Complete: true},
			},
			Total: 3, Complete: true,
		},
		Refs: []RefEvidence{
			{Ref: "refs/heads/soda/factory/candidate", OID: head, Exists: true, Visible: true, Complete: true},
			{Ref: "refs/heads/main", OID: base, Exists: true, Visible: true, Complete: true},
		},
	}
}

func TestMatchCheckTargetLatestWins(t *testing.T) {
	observed, err := matchCheckTarget(checkTargetForObserve(), checkSnapshotFixture())
	if err != nil {
		t.Fatal(err)
	}
	if observed.NativeRev != 41 || !observed.Complete || observed.Hidden {
		t.Fatalf("observation: %+v", observed)
	}
	if observed.HeadTip != strings.Repeat("a", 40) || observed.BaseTip != strings.Repeat("b", 40) {
		t.Fatalf("tips: %+v", observed)
	}
	if len(observed.Checks) != 2 || observed.Checks[0].State != "success" || observed.Checks[1].State != "success" {
		t.Fatalf("latest did not win: %+v", observed.Checks)
	}
	if observed.ObservedContexts != 2 {
		t.Fatalf("contexts: %+v", observed)
	}
}

func TestMatchCheckTargetStaleTipsPassThrough(t *testing.T) {
	snapshot := checkSnapshotFixture()
	snapshot.Refs[0].OID = strings.Repeat("c", 40)
	observed, err := matchCheckTarget(checkTargetForObserve(), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if observed.HeadTip != strings.Repeat("c", 40) {
		t.Fatalf("moved tip was not observed: %+v", observed)
	}
}

func TestMatchCheckTargetHidden(t *testing.T) {
	snapshot := checkSnapshotFixture()
	snapshot.Checks.Items[1].Visible = false
	snapshot.Checks.Items[1].State = ""
	observed, err := matchCheckTarget(checkTargetForObserve(), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !observed.Hidden {
		t.Fatal("hidden check item was not flagged")
	}
	if len(observed.Checks) != 2 || observed.Checks[0].State != "failure" {
		t.Fatalf("hidden winner leaked or vanished: %+v", observed.Checks)
	}
}

func TestMatchCheckTargetRefusals(t *testing.T) {
	target := checkTargetForObserve()
	mismatch := checkSnapshotFixture()
	mismatch.Pull.ID = "99"
	if _, err := matchCheckTarget(target, mismatch); err == nil {
		t.Fatal("wrong PR matched")
	} else if refusal, ok := err.(*factory.PublicationRefusal); !ok || refusal.Reason != "pr_mismatch" {
		t.Fatalf("refusal: %v", err)
	}
	merged := checkSnapshotFixture()
	merged.Pull.HasMerged = true
	if _, err := matchCheckTarget(target, merged); err == nil {
		t.Fatal("merged PR matched")
	} else if refusal, ok := err.(*factory.PublicationRefusal); !ok || refusal.Reason != "pr_merged" {
		t.Fatalf("refusal: %v", err)
	}
	hidden := checkSnapshotFixture()
	hidden.Pull.Visible = false
	if _, err := matchCheckTarget(target, hidden); err == nil {
		t.Fatal("hidden PR matched")
	}
}

func TestCheckReadErrorMapsBusy(t *testing.T) {
	for err, want := range map[error]string{
		ErrNativeBusy:    "publication waits: native_busy",
		ErrStaleSnapshot: "publication waits: revision_moved",
		&forgejopublish.StatusError{Status: http.StatusServiceUnavailable}:  "publication waits: native_busy",
		&forgejopublish.StatusError{Status: http.StatusInternalServerError}: "publication waits: checks_unavailable",
	} {
		if got := checkReadError(err); got.Error() != want {
			t.Fatalf("read error %v: got %v want %s", err, got, want)
		}
	}
}

func TestObserveChecksRejectsInvalidTarget(t *testing.T) {
	assessor := NewCheckAssessor(nil, nil, "")
	target := checkTargetForObserve()
	target.HeadOID = "short"
	if _, err := assessor.ObserveChecks(context.Background(), target, 2); err == nil {
		t.Fatal("invalid target observed")
	} else if refusal, ok := err.(*factory.PublicationRefusal); !ok || refusal.Reason != "invalid_work" {
		t.Fatalf("refusal: %v", err)
	}
	if _, err := assessor.ObserveChecks(context.Background(), checkTargetForObserve(), 0); err == nil {
		t.Fatal("missing actor observed")
	}
}
