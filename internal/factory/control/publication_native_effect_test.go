package control_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/store"
)

func nativeUnexpectedRefs(t *testing.T, extra bool) {
	t.Helper()
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	w := fx.observe(t, fx.work(t, a, n, factory.NewID()))
	label := "wrong-ref"
	reason := "stale_base_or_result"
	if extra {
		label = "extra-ref"
		reason = "unexpected_ref_effects"
	}
	nativeReceipt(t, label+"-intent", w.Intent())
	registered, err := fx.pub.SubmitPublish(context.Background(), w)
	nativeMust(t, err)
	if registered.Effect != factory.OpEffectPending {
		t.Fatalf("operation not registered: %+v", registered)
	}
	env, err := fx.background.PublishPushEnv(context.Background(), w.OperationID, false)
	nativeMust(t, err)
	target := factory.PublishBranchName(a.ID)
	other := target + "-unexpected"
	args := []string{"push", "--no-follow-tags", nativeRepoURL(c), n.head + ":" + other}
	if extra {
		args = append(args, n.head+":"+target)
	}
	if _, err := nativeGit(t, c, n.dir, env, args...); err == nil {
		t.Fatal("unexpected ref command set succeeded")
	}
	outcome := fx.terminal(t, w.OperationID)
	nativeReceipt(t, label, struct {
		Intent  factory.PublicationIntent
		Outcome factory.OperationOutcome
	}{w.Intent(), outcome})
	if outcome.Effect != factory.OpEffectNotCommitted || outcome.Reason != reason || outcome.OperationID != w.OperationID || outcome.Kind != factory.OpRefPublish || outcome.ActorID != w.ActorID || outcome.RepositoryID != w.Repository || outcome.InstallationID != registered.InstallationID {
		t.Fatalf("unexpected refs lack native refusal: %+v", outcome)
	}
	if nativeTip(t, c, target) != "" || nativeTip(t, c, other) != "" {
		t.Fatal("refused operation changed a ref")
	}
}

type nativeLostReplies struct {
	*forgejo.Publisher
	submits, pushes, creates int
	submitOnly               bool
}

func (p *nativeLostReplies) SubmitPublish(ctx context.Context, w factory.PublicationWork) (factory.OperationOutcome, error) {
	outcome, err := p.Publisher.SubmitPublish(ctx, w)
	if err != nil {
		return outcome, err
	}
	p.submits++
	return factory.OperationOutcome{}, errors.New("injected lost branch-registration reply")
}
func (p *nativeLostReplies) PushBranch(ctx context.Context, w factory.PublicationWork) (factory.OperationOutcome, error) {
	outcome, err := p.Publisher.PushBranch(ctx, w)
	if err != nil {
		return outcome, err
	}
	p.pushes++
	if p.submitOnly {
		return outcome, nil
	}
	return factory.OperationOutcome{}, errors.New("injected lost committed-branch reply")
}
func (p *nativeLostReplies) SubmitPRCreate(ctx context.Context, w factory.PublicationWork) (factory.OperationOutcome, error) {
	outcome, err := p.Publisher.SubmitPRCreate(ctx, w)
	if err != nil {
		return outcome, err
	}
	p.creates++
	if p.submitOnly {
		return outcome, nil
	}
	return factory.OperationOutcome{}, errors.New("injected lost PR-creation reply")
}

// The transport wrapper discards actual native answers after each operation
// returns. Every subsequent pass uses a reopened Soda database and must recover
// the original native identity without replaying a primary mutation.
func TestNativePublishLostReplies(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	drops := &nativeLostReplies{Publisher: fx.pub}
	fx.coord.Publication = drops
	var p factory.Publication
	var branchIntent, createIntent *factory.PublicationIntent
	for i := 0; i < 20; i++ {
		_ = fx.coord.PublishPass(context.Background())
		var err error
		p, err = fx.db.PublicationByAssignment(context.Background(), a.ID)
		nativeMust(t, err)
		if branchIntent == nil && p.Publish.Work != nil {
			copy := *p.Publish.Work
			branchIntent = &copy
		}
		if createIntent == nil && p.PRCreate.Work != nil {
			copy := *p.PRCreate.Work
			createIntent = &copy
		}
		if branchIntent != nil && (p.Publish.Work == nil || *branchIntent != *p.Publish.Work) {
			t.Fatal("branch replay changed its persisted intent")
		}
		if createIntent != nil && (p.PRCreate.Work == nil || *createIntent != *p.PRCreate.Work) {
			t.Fatal("creation replay changed its persisted intent")
		}
		if p.Stage == factory.PublicationPublished {
			break
		}
		if p.Stage != factory.PublicationOpen {
			t.Fatalf("lost reply advanced incorrectly: %+v", p)
		}
		nativeMust(t, fx.db.Close())
		fx.db, err = store.Open(fx.path)
		nativeMust(t, err)
		fx.wire()
		fx.coord.Publication = drops
		time.Sleep(100 * time.Millisecond)
	}
	nativeReceipt(t, "lost-replies", struct {
		Publication              factory.Publication
		Submits, Pushes, Creates int
	}{p, drops.submits, drops.pushes, drops.creates})
	if p.Stage != factory.PublicationPublished || drops.submits != 1 || drops.pushes != 1 || drops.creates != 1 {
		t.Fatalf("lost-reply recovery did not preserve one operation per effect: stage=%s calls=%d/%d/%d", p.Stage, drops.submits, drops.pushes, drops.creates)
	}
	if p.Publish.Receipt == "" || p.PRCreate.Receipt == "" {
		t.Fatal("lost reply recovery lacks exact receipts")
	}
}

func TestNativePublishWithdrawal(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	fx.coord.Publication = &nativeLostReplies{Publisher: fx.pub, submitOnly: true}
	_ = fx.coord.PublishPass(context.Background())
	p, err := fx.db.PublicationByAssignment(context.Background(), a.ID)
	nativeMust(t, err)
	if p.Publish.Work == nil || p.Publish.OperationID == "" {
		t.Fatal("withdrawal fixture did not persist native registration")
	}
	nativeReceipt(t, "withdrawal-registered", p)
	registered, err := fx.background.GetOperation(context.Background(), p.Publish.OperationID)
	nativeMust(t, err)
	nativeReceipt(t, "withdrawal-registered-native", registered)
	_, err = fx.db.WithdrawDispatch(context.Background(), c.Repository, "ST09 withdrawal fixture", "native-maintainer")
	nativeMust(t, err)
	fx.coord.Publication = fx.pub
	report := fx.coord.PublishPass(context.Background())
	p, err = fx.db.PublicationByAssignment(context.Background(), a.ID)
	nativeMust(t, err)
	nativeReceipt(t, "withdrawal-cancel-pass", struct {
		Publication factory.Publication
		Report      control.PublishReport
	}{p, report})
	cancelled, err := fx.background.GetOperation(context.Background(), p.Publish.OperationID)
	nativeMust(t, err)
	nativeReceipt(t, "withdrawal-cancel-native", cancelled)
	if p.Publish.Cancellation == factory.OpCancelPending && p.Stage != factory.PublicationOpen {
		t.Fatal("pending native cancellation was presented as a finished withdrawal")
	}
	w := p.Publish.Work.Apply(fx.work(t, a, n, p.ID))
	outcome, err := fx.pub.PushBranch(context.Background(), w)
	nativeMust(t, err)
	nativeReceipt(t, "withdrawal-delayed-push", outcome)
	if outcome.Effect != factory.OpEffectNotCommitted || outcome.Cancellation != factory.OpCancelCancelled || outcome.Reason != "cancelled_before_admission" || nativeTip(t, c, factory.PublishBranchName(a.ID)) != "" {
		t.Fatal("delayed push escaped confirmed cancellation")
	}
	p = fx.drive(t, a)
	nativeReceipt(t, "withdrawal", p)
	if p.Stage != factory.PublicationWithdrawn || p.Publish.Effect != factory.OpEffectNotCommitted || p.Publish.Cancellation != factory.OpCancelCancelled {
		t.Fatalf("withdrawal not confirmed: %+v", p)
	}
}

func TestNativePublishBranchCommittedPRFailed(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	rival := &nativeLookalike{Publisher: fx.pub, t: t, cfg: c}
	fx.coord.Publication = rival
	p := fx.drive(t, a)
	nativeReceipt(t, "branch-committed-pr-failed", struct {
		Publication factory.Publication
		LookalikePR int64
	}{p, rival.number})
	if p.Stage != factory.PublicationFailed || p.Publish.Effect != factory.OpEffectCommitted || p.PRCreate.Effect != factory.OpEffectNotCommitted || p.PRCreate.Reason != "duplicate_pull_request" || p.PRNumber != 0 {
		t.Fatalf("partial outcome adopted lookalike or lost branch effect: %+v", p)
	}
	if p.Publish.Receipt == "" || p.PRCreate.OperationID == "" || rival.number <= 0 || nativeTip(t, c, factory.PublishBranchName(a.ID)) != n.head {
		t.Fatal("partial outcome lacks linked native evidence")
	}
}
