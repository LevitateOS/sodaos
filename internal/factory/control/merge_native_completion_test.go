package control_test

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// TestNativeMergeFullPass proves the complete ST12 path: a published PR
// with an independent approval and a recorded ST11 pass merges through
// MergePass, confirms its native completion, and links exactly.
func TestNativeMergeFullPass(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeFullPassAttempt(t, c) {
			return
		}
		t.Logf("full pass reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("full pass never observed its scenario outcome")
}

// nativeMergeFullPassAttempt runs one full-pass scenario: false means
// host churn (certain stale refusal, no effect) burned the proof
// submit, and only then may the caller reseed.
func nativeMergeFullPassAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	ctx := context.Background()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeApprove(t, c, p, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p.PRCreate.HeadOID, "verify", "success")
	assessment := nativeMergeAssess(t, c, fx, p)
	if assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	m, report := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage == factory.MergeFailed && m.Reason == factory.MergeReasonRefused && m.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m.Stage != factory.MergeMerged || len(report.Merged) != 1 || m.Operation.Attempts != 1 {
		t.Fatalf("merge unsettled: %+v %+v", m, report)
	}
	if m.MergedCommit != p.PRCreate.HeadOID || m.Operation.MergedCommit != p.PRCreate.HeadOID {
		t.Fatalf("merge realized the wrong commit: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p.PRCreate.HeadOID {
		t.Fatal("base tip does not carry the merged head")
	}
	var issue struct {
		State string `json:"state"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(p.PRNumber, 10), nil, &issue)
	if issue.State != "closed" {
		t.Fatalf("merged PR issue is not closed: %q", issue.State)
	}
	// The confirmation stamps come from native evidence, not the submit:
	// the merger, merge stamp and close stamp must match a fresh read.
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	merger := forgejo.NewMerger(bg, forgejo.New(c.FountainURL), c.TokenFile)
	work := m.Operation.Work.Apply(factory.MergeWork{MergeID: m.ID, PublicationID: m.PublicationID, Issue: m.Issue})
	confirmation := nativeMergeCall(t, "completion observation", func() (factory.MergeConfirmation, error) {
		return merger.ObserveCompletion(ctx, work)
	})
	if m.MergedUnix != confirmation.MergedUnix || m.ClosedUnix != confirmation.ClosedUnix || confirmation.MergerID != c.ActorID {
		t.Fatalf("completion stamps are not native evidence: %+v vs %+v", m, confirmation)
	}
	nativeMergeReceipt(t, "fullpass-merge", m)
	nativeMergeReceipt(t, "fullpass-report", report)
	return true
}

// TestNativeMergeDependantRunnable proves confirmed completion releases
// an eligible dependant: B stays blocked on A's code until A's merge
// confirms, then B becomes runnable without any further input.
func TestNativeMergeDependantRunnable(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeDependantAttempt(t, c) {
			return
		}
		t.Logf("dependant reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("dependant never observed its scenario outcome")
}

// nativeMergeDependantAttempt runs one dependant scenario: false means
// host churn burned a proof submit, and only then may the caller
// reseed. The edge wires exact issues, so the publish seeding stays on
// the wired assignment and churn reseeds the whole attempt instead.
func nativeMergeDependantAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	ctx := context.Background()
	fx, nA := nativeMergeSetup(t, c)
	aA := nativeMergeAssignment(t, c, fx, nA)
	nB := nativeNewCandidate(t, c.nativeST09Config)
	aB := nativeMergeAssignment(t, c, fx, nB)
	var issueA, issueB struct {
		ID int64 `json:"id"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(aA.Issue, 10), nil, &issueA)
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(aB.Issue, 10), nil, &issueB)
	if issueA.ID <= 0 || issueB.ID <= 0 {
		t.Fatal("native issue identities unconfirmed")
	}
	nativeMergeInsertEdge(t, c, issueB.ID, issueA.ID)
	var edges []struct {
		ID int64 `json:"id"`
	}
	nativeAPI(t, c.nativeST09Config, c.CreatorTokenFile, http.MethodGet, "/api/v1/repos/"+c.Owner+"/"+c.Repo+"/issues/"+strconv.FormatInt(aB.Issue, 10)+"/dependencies", nil, &edges)
	if len(edges) != 1 {
		t.Fatalf("native edge not served: %+v", edges)
	}
	evidence, err := fx.coord.AcceptanceReads.ReadAcceptanceEvidence(ctx, strconv.FormatInt(c.Repository, 10), strconv.FormatInt(aB.Issue, 10), nil)
	nativeMust(t, err)
	if len(evidence.Dependencies) != 1 {
		t.Fatalf("snapshot does not carry the edge: %+v", evidence.Dependencies)
	}
	edge := evidence.Dependencies[0]
	second := factory.Acceptance{
		ID: "d" + factory.NewID()[:24], Repository: c.Repository, IssueIndex: strconv.FormatInt(aB.Issue, 10),
		Approver: c.CreatorID, NativeRev: evidence.Revision,
		TitleDigest: evidence.Issue.TitleDigest, ContentDigest: evidence.Issue.ContentDigest, ContentVersion: evidence.Issue.ContentVer,
		Predecessor: aB.Acceptance,
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: edge.Occurrence, DependsOn: edge.DependsOn,
			EndpointRepo: c.Repository, EndpointIssue: aA.Issue,
			Outcome: factory.PrereqCode, PrereqAcceptance: aA.Acceptance,
		}},
	}
	if _, err := fx.coord.AdmitAcceptance(ctx, factory.NewID(), "native:"+strconv.FormatInt(c.CreatorID, 10), second); err != nil {
		t.Fatalf("dependant acceptance refused: %v", err)
	}
	blocked, _, err := fx.coord.ObserveIssueEvent(ctx, control.IntakeHint{Delivery: "st12-dep-" + factory.NewID(), Repository: c.Repository, Issue: aB.Issue})
	nativeMust(t, err)
	if blocked.Readiness != factory.ReadinessBlocked {
		t.Fatalf("dependant runnable before completion: %+v", blocked)
	}
	pA := nativeMergePublishDrive(t, c, fx, aA)
	if pA.Stage == factory.PublicationFailed &&
		(pA.Publish.Reason == "stale_native_revision" || pA.PRCreate.Reason == "stale_native_revision") {
		return false
	}
	if pA.Stage != factory.PublicationPublished {
		t.Fatalf("seed publication unsettled: %+v", pA)
	}
	nativeMergeApprove(t, c, pA, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, pA.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx, pA); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	mA, _ := nativeMergeDrive(t, c, fx, pA.ID)
	if mA.Stage == factory.MergeFailed && mA.Reason == factory.MergeReasonRefused && mA.Operation.Reason == "stale_native_revision" {
		return false
	}
	if mA.Stage != factory.MergeMerged {
		t.Fatalf("endpoint merge unsettled: %+v", mA)
	}
	released, err := fx.db.IssueControl(ctx, c.Repository, aB.Issue)
	nativeMust(t, err)
	if released.Readiness != factory.ReadinessQueued || released.Reason != factory.ReasonEligible {
		t.Fatalf("confirmed completion did not release the dependant: %+v", released)
	}
	nativeMergeReceipt(t, "dependant-merge", mA)
	nativeMergeReceipt(t, "dependant-control", released)
	return true
}
