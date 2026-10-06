package control_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// TestNativeMergeStaleHead proves a moved candidate never merges: even
// with a valid approval on the new head, the merge refuses its stale
// binding without submitting anything.
func TestNativeMergeStaleHead(t *testing.T) {
	c := loadNativeST12(t)
	fx, n := nativeMergeSetup(t, c)
	p, a := nativeMergeSeedPublication(t, c, fx, n)
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	nativeMergeOpenRow(t, fx, p)
	head2 := nativeMergeAdvanceBranch(t, c, factory.PublicationBranch(a.ID))
	if head2 == p.PRCreate.HeadOID {
		t.Fatal("advance did not move the candidate")
	}
	// A fresh approval on the new head does not rescue the stale binding.
	bg := forgejo.NewServiceBackground(c.Socket, uint32(os.Getuid()), "")
	r := forgejo.NewReviewer(bg, forgejo.New(c.FountainURL), c.ReviewerTokenFile)
	nativeMergeSubmitReviewCommitted(t, r, bg, factory.ReviewWork{
		AuthRevision: "st12-native-proof",
		Repository:   c.Repository, ActorID: c.ReviewerID, PRNumber: p.PRNumber, PRID: p.PRID,
		IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef,
		HeadOID: head2, BaseOID: p.PRCreate.BaseOID,
		Event: "APPROVED", Body: "ST12 stale-head approval on the new head.",
	})
	m, _ := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage != factory.MergeFailed || m.Reason != factory.MergeReasonInvalid || m.Operation.Attempts != 0 {
		t.Fatalf("stale head misadvanced: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("stale merge moved the base")
	}
	nativeMergeReceipt(t, "stalehead-merge", m)
}

// TestNativeMergeStaleBase proves a moved base never merges: the merge
// refuses its unverified base without submitting anything.
func TestNativeMergeStaleBase(t *testing.T) {
	c := loadNativeST12(t)
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	nativeMergeOpenRow(t, fx, p)
	base2 := nativeMergeAdvanceBranch(t, c, strings.TrimPrefix(c.BaseBranch, "refs/heads/"))
	if base2 == p.PRCreate.BaseOID {
		t.Fatal("advance did not move the base")
	}
	m, _ := nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage != factory.MergeFailed || m.Reason != factory.MergeReasonInvalid || m.Operation.Attempts != 0 {
		t.Fatalf("stale base misadvanced: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != base2 {
		t.Fatal("base tip is not the advanced base")
	}
	nativeMergeReceipt(t, "stalebase-merge", m)
}

// TestNativeMergeWithdrawBeforeSubmit proves withdrawal before any
// submit leaves no native effect: the merge withdraws, nothing commits.
func TestNativeMergeWithdrawBeforeSubmit(t *testing.T) {
	c := loadNativeST12(t)
	ctx := context.Background()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	nativeMergeOpenRow(t, fx, p)
	m, err := fx.db.MergeByPublication(ctx, p.ID)
	nativeMust(t, err)
	if m.Stage != factory.MergeOpen || m.Operation.Work != nil {
		t.Fatalf("seed merge misadvanced: %+v", m)
	}
	if _, err := fx.db.WithdrawDispatch(ctx, c.Repository, "test withdrawal", "soda-tester"); err != nil {
		t.Fatal(err)
	}
	m, _ = nativeMergeDrive(t, c, fx, p.ID)
	if m.Stage != factory.MergeWithdrawn || m.Operation.Attempts != 0 {
		t.Fatalf("withdrawal unconfirmed: %+v", m)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("withdrawn merge moved the base")
	}
	nativeMergeReceipt(t, "withdraw-merge", m)
}

// TestNativeMergeAuthorityChanged proves the merge stays under current
// authority: a rotated merge binding waits without submitting. The
// CAS policy revision pins recorded authority, so the rotated
// bracket stays orphaned; restoring the binding lets a fresh cycle
// proceed.
func TestNativeMergeAuthorityChanged(t *testing.T) {
	c := loadNativeST12(t)
	for attempt := 0; attempt < 3; attempt++ {
		if nativeMergeAuthorityAttempt(t, c) {
			return
		}
		t.Logf("authority reseeded after host churn (attempt %d)", attempt+1)
	}
	t.Fatal("authority never observed its scenario outcome")
}

func nativeMergeAuthorityAttempt(t *testing.T, c nativeST12Config) bool {
	t.Helper()
	ctx := context.Background()
	fx, n := nativeMergeSetup(t, c)
	p, _ := nativeMergeSeedPublication(t, c, fx, n)
	before := nativeTip(t, c.nativeST09Config, c.BaseBranch)
	nativeMergeOpenRow(t, fx, p)
	m, err := fx.db.MergeByPublication(ctx, p.ID)
	nativeMust(t, err)
	if m.Stage != factory.MergeOpen || m.Operation.Work != nil {
		t.Fatalf("seed merge misadvanced: %+v", m)
	}
	policy, err := fx.db.RepositoryPolicy(ctx, c.Repository)
	nativeMust(t, err)
	original := policy.Merge.ActorID
	policy.Merge.ActorID = original + 1000
	nativeMust(t, fx.db.SaveRepositoryPolicy(ctx, policy))
	sawWait := false
	for i := 0; i < 5; i++ {
		report := fx.coord.MergePass(ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				t.Fatalf("merge errors: %+v", report)
			}
		}
		for _, wait := range report.Waits {
			if wait.Reason == "authority_changed" {
				sawWait = true
			}
		}
	}
	m, err = fx.db.MergeByPublication(ctx, p.ID)
	nativeMust(t, err)
	if m.Stage != factory.MergeOpen || m.Operation.Attempts != 0 || m.Operation.Work != nil || !sawWait {
		t.Fatalf("rotated authority misadvanced: %+v sawWait=%v", m, sawWait)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != before {
		t.Fatal("unauthorized merge moved the base")
	}
	nativeMergeReceipt(t, "authority-wait", m)
	policy, err = fx.db.RepositoryPolicy(ctx, c.Repository)
	nativeMust(t, err)
	policy.Merge.ActorID = original
	nativeMust(t, fx.db.SaveRepositoryPolicy(ctx, policy))
	// The fresh cycle needs its own fixture: the fixture host serves
	// the seed candidate's bundle, so a second candidate on the same
	// host cannot validate.
	fx2, n2 := nativeMergeSetup(t, c)
	p2, _ := nativeMergeSeedPublication(t, c, fx2, n2)
	nativeMergeApprove(t, c, p2, "APPROVED", "ST12 proof approval.")
	nativePostStatus(t, c.nativeST09Config, p2.PRCreate.HeadOID, "verify", "success")
	if assessment := nativeMergeAssess(t, c, fx2, p2); assessment.Verdict != factory.CheckPass {
		t.Fatalf("seed assessment did not pass: %+v", assessment)
	}
	m2, _ := nativeMergeDrive(t, c, fx2, p2.ID)
	if m2.Stage == factory.MergeFailed && m2.Reason == factory.MergeReasonRefused && m2.Operation.Reason == "stale_native_revision" {
		return false
	}
	if m2.Stage != factory.MergeMerged || m2.Operation.Attempts != 1 {
		t.Fatalf("restored authority unsettled: %+v", m2)
	}
	if nativeTip(t, c.nativeST09Config, c.BaseBranch) != p2.PRCreate.HeadOID {
		t.Fatal("restored merge did not move the base")
	}
	nativeMergeReceipt(t, "authority-merge", m2)
	return true
}
