package control_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	hostexec "github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
)

// reviewLeg runs one genuine reviewer agent and submits its verdict
// through the separate reviewer actor.
// prepareReviewer stages a fresh reviewer checkout of the exact
// published candidate: each review reads the candidate in its own
// preparation, never the coder checkout or a stale shared one. It
// exports the candidate bundle from the coder run that produced it
// and prepares a fresh reviewer identity from the ready reviewer
// source preparation. It returns the fresh preparation ID.
func (fx *st15Fixture) prepareReviewer(head, coderRunID string) string {
	fx.t.Helper()
	ctx := fx.ctx
	exported, err := fx.coord.Host.FactoryExport(ctx, project.FactoryExport{
		Project: fx.projectID, ID: coderRunID, Role: project.RoleCoder,
		Preparation: fx.coderPrep, Candidate: head,
	})
	if err != nil {
		fx.t.Fatalf("ST15 reviewer export %s: %v", head[:12], err)
	}
	if exported.ID != coderRunID || exported.Project != fx.projectID || exported.Candidate != head {
		fx.t.Fatalf("ST15 reviewer export identity differs: %+v", exported)
	}
	bundle, err := base64.StdEncoding.DecodeString(exported.Bundle)
	if err != nil || len(bundle) == 0 {
		fx.t.Fatalf("ST15 reviewer export bundle invalid: %v", err)
	}
	source, err := fx.db.Preparation(ctx, fx.reviewPrep)
	if err != nil {
		fx.t.Fatal(err)
	}
	prep := source.Preparation
	prep.ID = "f" + st15RandHex(12)
	prep.SourceCommit = head
	if _, admitted, err := fx.db.AdmitPreparation(ctx, project.StoredPreparation{Preparation: prep}); err != nil || !admitted {
		fx.t.Fatalf("ST15 admit reviewer %s: admitted=%v err=%v", prep.ID, admitted, err)
	}
	client := hostexec.NewClient(fx.cfg.HostSocket)
	state, err := client.PrepareCandidate(ctx, project.FactoryCandidate{
		Preparation: prep, SourcePreparation: fx.reviewPrep, Bundle: bundle,
	})
	if err != nil {
		fx.t.Fatalf("ST15 prepare reviewer %s: %v", prep.ID, err)
	}
	if state.ID != prep.ID {
		fx.t.Fatalf("ST15 reviewer preparation identity differs: %+v", state)
	}
	deadline := time.Now().Add(15 * time.Minute)
	for {
		state, err = client.InspectPreparation(ctx, project.PrepareInspect{Project: fx.projectID, ID: prep.ID})
		if err != nil {
			fx.t.Fatal(err)
		}
		if state.Phase == project.PrepareReady {
			stored, err := fx.db.Preparation(ctx, prep.ID)
			if err != nil {
				fx.t.Fatal(err)
			}
			state.ID, state.Project, state.Role = prep.ID, fx.projectID, project.RoleReviewer
			stored.State = state
			if err := fx.db.ObservePreparation(ctx, stored); err != nil {
				fx.t.Fatal(err)
			}
			fx.t.Logf("ST15 reviewer preparation %s ready at %s", prep.ID, head[:12])
			return prep.ID
		}
		if state.Phase == project.PrepareFailed || state.Phase == project.PrepareStopped {
			fx.t.Fatalf("ST15 reviewer preparation %s ended %s", prep.ID, state.Phase)
		}
		if time.Now().After(deadline) {
			fx.t.Fatalf("ST15 reviewer preparation %s not ready: %s", prep.ID, state.Phase)
		}
		time.Sleep(2 * time.Second)
	}
}

func (fx *st15Fixture) reviewLeg(head, prep string) (factory.ReviewOutcome, string) {
	fx.t.Helper()
	title, body, _ := fx.readIssue(fx.issueAIndex)
	answer := fx.readComment(fx.answerA)
	prompt := fx.reviewPrompt(title, body, answer, head, fx.sourceHead)
	// The review run itself is still driver-sequenced: reviewer dispatch
	// is the remaining production gap. Its verdict submits through the
	// production review path, never a fixture-native trio.
	run, _ := fx.launchDirect(project.RoleReviewer, prep, fx.assignA, prompt, head)
	output := fx.settleDirect(run, fx.assignA)
	report, ok := factory.ParseReviewReport(output)
	if !ok {
		fx.t.Fatal("ST15 review lacks its fenced report")
	}
	adopted, event := fx.submitReview(run.ID, output)
	st15Receipt(fx.t, "review-"+head[:12], map[string]any{
		"run": run.ID, "verdict": report.Verdict, "event": event,
		"review": adopted.ReviewID, "summary": report.Summary, "findings": report.Findings,
	})
	return adopted, event
}

func (fx *st15Fixture) reviewLeg1() error {
	adopted, event := fx.reviewLeg(fx.head1, fx.prepareReviewer(fx.head1, fx.runA))
	fx.review1, fx.verdict1 = adopted, event
	return nil
}

// ciFail seeds the true fixture verdict on the first head (the required
// regression path is absent) and records the failed ST11 assessment.
func (fx *st15Fixture) ciFail() error {
	fx.requireContentStatus(fx.head1, "tests/test_total_regression.py", http.StatusNotFound)
	fx.seedCIStatus(fx.head1, "st15-build", "success", "widget compiles")
	fx.seedCIStatus(fx.head1, "st15-test", "failure", "required regression test tests/test_total_regression.py missing")
	report := fx.coord.CheckPass(fx.ctx)
	link, ok := st15CheckLink(report, fx.assignA)
	if !ok || len(report.Errors) != 0 || link.Verdict != factory.CheckFailed {
		return fmt.Errorf("CI failure unrecorded: %+v", report)
	}
	fx.ciFailRev = link.Revision
	st15Receipt(fx.t, "ci-fail", link)
	return nil
}

func (fx *st15Fixture) requireContentStatus(head, path string, status int) {
	fx.t.Helper()
	token, err := os.ReadFile(fx.cfg.CreatorTokenFile)
	if err != nil {
		fx.t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(fx.ctx, http.MethodGet,
		fx.cfg.FountainURL+"/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/contents/"+path+"?ref="+head, nil)
	if err != nil {
		fx.t.Fatal(err)
	}
	request.Header.Set("Authorization", "token "+strings.TrimSpace(string(token)))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		fx.t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != status {
		fx.t.Fatalf("ST15 %s@%.12s status %d, want %d", path, head, response.StatusCode, status)
	}
}

// correctA authorizes the explicit retry, runs the correction with the
// recorded CI and review appendix, and observes its publication to the
// same PR. The correction run itself is still driver-sequenced with its
// findings appendix: findings-aware correction dispatch is the remaining
// production gap. Settling runs through the production stop, which
// publishes the correction automatically.
func (fx *st15Fixture) correctA() error {
	if _, err := fx.coord.RetryRun(fx.ctx, factory.NewID(), "soda-maintainer", fx.runA); err != nil {
		return fmt.Errorf("explicit retry: %w", err)
	}
	a, err := fx.db.Assignment(fx.ctx, fx.assignA)
	if err != nil {
		return err
	}
	var appendix strings.Builder
	appendix.WriteString("\n\n## Correction\n\nThe candidate drew recorded findings; address every one and keep the fix:\n")
	appendix.WriteString("- CI assessment revision " + strconv.FormatInt(fx.ciFailRev, 10) + ": required check st15-test failed on this head: required regression test tests/test_total_regression.py missing. Add that exact regression test.\n")
	if fx.verdict1 == "REQUEST_CHANGES" {
		appendix.WriteString("- Review " + strconv.FormatInt(fx.review1.ReviewID, 10) + " requested changes: " + fx.review1.Event + ". Address its findings.\n")
	} else {
		appendix.WriteString("- Review " + strconv.FormatInt(fx.review1.ReviewID, 10) + " approved the fix; the CI requirement above is the remaining work.\n")
	}
	appendix.WriteString("- After committing, run `git rev-parse HEAD`: the printed candidate must be the new head, never the head this run started from. If they match, the fix is uncommitted; commit it and re-read the head before printing.\n")
	prompt := append(a.Prompt, appendix.String()...)
	run, _ := fx.launchDirect(project.RoleCoder, fx.coderPrep, fx.assignA, string(prompt), fx.head1)
	fx.runC = run.ID
	receipt := fx.stopSettled(run.ID)
	if receipt.Outcome != string(factory.Succeeded) {
		st15Receipt(fx.t, "correction-output", map[string]any{"output_tail": tailLines(receipt.Reason, 40)})
		return fmt.Errorf("correction run ended %s: %s", receipt.Outcome, receipt.Reason)
	}
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		p, err := fx.db.PublicationByAssignment(fx.ctx, fx.assignA)
		if err != nil {
			return err
		}
		if p.Candidate != fx.head1 && factory.ValidCommit(p.Candidate) {
			fx.head2 = p.Candidate
			fx.requireContentStatus(fx.head2, "tests/test_total_regression.py", http.StatusOK)
			st15Receipt(fx.t, "correct-A", map[string]any{"head": fx.head2, "run": run.ID})
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	// Surface the production correction report once for the failure; when
	// the stop already linked the head this replays as recorded.
	report := fx.coord.PublishCorrection(fx.ctx, fx.assignA, run.ID, receipt.Reason)
	st15Receipt(fx.t, "correction-output", map[string]any{"output_tail": tailLines(receipt.Reason, 40), "report": report})
	return fmt.Errorf("correction unpublished: %+v", report)
}

func (fx *st15Fixture) reviewLeg2() error {
	// Boot the dependant-stop browser now, in the background: it polls
	// for B's run while review-2, ci-pass, and the merge drive run, so
	// the stop lands within seconds of record. Starting it after B
	// records would race a fast CLI past completion.
	fx.controlBrowser = make(chan error, 1)
	go func() {
		fx.controlBrowser <- fx.runBrowser("control", "", project.RoleCoder, fx.issueBIndex)
	}()
	// The head-1 approval cannot observe the new tip: staleness refuses.
	p, err := fx.db.PublicationByAssignment(fx.ctx, fx.assignA)
	if err != nil {
		return err
	}
	stale := factory.ReviewWork{
		OperationID: "st15-review-stale-" + factory.NewID(), AuthRevision: "st15-composed-demo",
		Repository: fx.cfg.Repository, ActorID: fx.cfg.ReviewerID,
		PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef, HeadOID: fx.head1, BaseOID: p.PRCreate.BaseOID,
		Event: "APPROVED", Body: "stale approval replay",
	}
	if _, err := fx.coord.Reviews.ObserveReview(fx.ctx, stale); err == nil {
		return errors.New("stale head-1 review observed the corrected tip")
	}
	adopted, event := fx.reviewLeg(fx.head2, fx.prepareReviewer(fx.head2, fx.runC))
	if event != "APPROVED" {
		return fmt.Errorf("fresh review did not approve the corrected head: %s", event)
	}
	fx.review2, fx.verdict2 = adopted, event
	return nil
}

// st15CheckLink finds one assignment's assessment inside a production
// check sweep.
func st15CheckLink(report control.CheckReport, assignmentID string) (control.CheckLink, bool) {
	for _, link := range report.Assessed {
		if link.AssignmentID == assignmentID {
			return link, true
		}
	}
	return control.CheckLink{}, false
}
