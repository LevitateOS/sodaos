package control_test

// These tests require an explicitly supplied, task-owned native fixture.
// They exercise the real Soda controller, native observation adapter, Git
// publisher, and Fountain operations. Stopped-run/preparation records and the
// host bundle response are seeded fixtures; they do not prove CLI execution.
// Absence of SODA_ST09_NATIVE skips native work and is never passing evidence.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/forgejo"
)

func TestNativePublishCandidate(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	p := fx.drive(t, a)
	nativeReceipt(t, "candidate", p)
	if p.Stage != factory.PublicationPublished || p.Publish.Receipt == "" || p.PRCreate.Receipt == "" || p.PRNumber <= 0 {
		t.Fatalf("candidate not linked: %+v", p)
	}
	if nativeTip(t, c, factory.PublishBranchName(a.ID)) != n.head {
		t.Fatal("published branch is not exact candidate")
	}
	var pr struct {
		ID     int64  `json:"id"`
		Number int64  `json:"number"`
		State  string `json:"state"`
		Head   struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	}
	nativeAPI(t, c, c.TokenFile, http.MethodGet, fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", c.Owner, c.Repo, p.PRNumber), nil, &pr)
	if pr.ID != p.PRID || pr.Number != p.PRNumber || pr.Head.Ref != factory.PublicationBranch(a.ID) || pr.Head.SHA != n.head || pr.Base.Ref != strings.TrimPrefix(c.BaseBranch, "refs/heads/") || pr.State != "open" {
		t.Fatalf("native PR differs from receipts: %+v", pr)
	}
	again := fx.coord.PublishPass(context.Background())
	if len(again.Published) != 0 {
		t.Fatal("finished publication advanced twice")
	}
}

// Correction exercises the retained production publisher primitive. Its exact
// intent is retained before dispatch and its native receipt afterward. Automatic
// reviewer/coder correction selection belongs to ST10 and is not simulated here.
func TestNativePublishCorrection(t *testing.T) {
	c := loadNativeST09(t)
	n := nativeNewCandidate(t, c)
	fx := nativeSeed(t, c, n)
	a := fx.assignment(t, n)
	first := fx.drive(t, a)
	if first.Stage != factory.PublicationPublished {
		t.Fatalf("initial publication: %+v", first)
	}
	corrected := nativeAppendCandidate(t, c, n)
	w := fx.work(t, a, corrected, factory.NewID())
	w.ExpectedOld = n.head
	w.CorrectionNumber, w.CorrectionAuthor = first.PRNumber, c.ActorID
	w = fx.observe(t, w)
	nativeReceipt(t, "correction-intent", w.Intent())
	registered, err := fx.pub.SubmitPublish(context.Background(), w)
	nativeMust(t, err)
	if registered.Effect != factory.OpEffectPending {
		t.Fatalf("correction not registered: %+v", registered)
	}
	pushed, err := fx.pub.PushBranch(context.Background(), w)
	nativeMust(t, err)
	if pushed.Effect == factory.OpEffectPending {
		pushed = fx.terminal(t, w.OperationID)
	}
	nativeReceipt(t, "correction", struct {
		Initial factory.Publication
		Intent  factory.PublicationIntent
		Outcome factory.OperationOutcome
	}{first, w.Intent(), pushed})
	_, err = fx.pub.AdoptBranch(w, pushed)
	nativeMust(t, err)
	if nativeTip(t, c, factory.PublishBranchName(a.ID)) != corrected.head {
		t.Fatal("correction did not update exact recorded branch")
	}
	var pr struct {
		ID   int64 `json:"id"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	nativeAPI(t, c, c.TokenFile, http.MethodGet, fmt.Sprintf("/api/v1/repos/%s/%s/pulls/%d", c.Owner, c.Repo, first.PRNumber), nil, &pr)
	if pr.ID != first.PRID || pr.Head.SHA != corrected.head {
		t.Fatalf("correction did not retain exact PR: %+v", pr)
	}
}

func TestNativePublishWrongRefs(t *testing.T) { nativeUnexpectedRefs(t, false) }
func TestNativePublishExtraRefs(t *testing.T) { nativeUnexpectedRefs(t, true) }

type nativeLookalike struct {
	*forgejo.Publisher
	t      *testing.T
	cfg    nativeST09Config
	number int64
}

func (p *nativeLookalike) ObservePublication(ctx context.Context, w factory.PublicationWork) (factory.PublicationObservation, error) {
	if w.ExpectedOld != "absent" && p.number == 0 {
		var pr struct {
			Number int64 `json:"number"`
		}
		// This is an independent ordinary native writer in the test fixture,
		// not a factory fallback. It races before the fresh creation bracket.
		nativeAPI(p.t, p.cfg, p.cfg.TokenFile, http.MethodPost, "/api/v1/repos/"+p.cfg.Owner+"/"+p.cfg.Repo+"/pulls", map[string]any{"head": factory.PublicationBranch(w.AssignmentID), "base": strings.TrimPrefix(w.TargetBranch, "refs/heads/"), "title": w.PRTitle, "body": w.PRBody, "allow_maintainer_edit": false}, &pr)
		if pr.Number <= 0 {
			p.t.Fatal("lookalike fixture has no native PR")
		}
		p.number = pr.Number
	}
	return p.Publisher.ObservePublication(ctx, w)
}
