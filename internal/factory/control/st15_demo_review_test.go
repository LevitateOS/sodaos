package control_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/project"
)

// reviewLeg consumes the reviewer child produced by PublishPass and submits
// its recorded report through the separate reviewer actor.
func (fx *st15Fixture) reviewLeg(head string) (factory.ReviewOutcome, string) {
	fx.t.Helper()
	_, run, err := fx.publishChildRun(project.RoleReviewer, head)
	if err != nil {
		fx.t.Fatal(err)
	}
	assignment, output := fx.settleDispatched(run.ID)
	if assignment.Result == nil || assignment.Result.Review == nil {
		st15Receipt(fx.t, "review-output-"+head[:12], map[string]any{"output_tail": tailLines(output, 40)})
		fx.t.Fatal("ST15 recorded reviewer child lacks its parsed review report")
	}
	report := *assignment.Result.Review
	adopted, event := fx.submitReview(run.ID)
	st15Receipt(fx.t, "review-"+head[:12], map[string]any{
		"run": run.ID, "assignment": assignment.ID, "verdict": report.Verdict, "event": event,
		"review": adopted.ReviewID, "summary": report.Summary, "findings": report.Findings,
	})
	return adopted, event
}

func (fx *st15Fixture) reviewLeg1() error {
	adopted, event := fx.reviewLeg(fx.head1)
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

// correctA observes the correction child admitted from recorded review and
// CI evidence, then verifies its result advances the existing PR head.
func (fx *st15Fixture) correctA() error {
	assignment, run, err := fx.publishChildRun(project.RoleCoder, fx.head1)
	if err != nil {
		return err
	}
	if assignment.AttemptRoot != fx.assignA || assignment.PublicationAssignment != fx.assignA || assignment.ID == fx.assignA {
		return fmt.Errorf("ST15 correction did not retain the original publication attempt: %+v", assignment)
	}
	fx.runC = run.ID
	settled, output := fx.settleDispatched(run.ID)
	if settled.ID != assignment.ID || settled.AttemptRoot != assignment.AttemptRoot || settled.Result == nil {
		return fmt.Errorf("ST15 correction result lost its recorded child binding: %+v", settled)
	}
	allowance, err := fx.db.AttemptAllowance(fx.ctx, fx.cfg.Repository, fx.issueAIndex)
	if err != nil || allowance.RootAssignment != fx.assignA || allowance.Closed || !allowance.Active {
		return fmt.Errorf("ST15 correction publication attempt unavailable: %+v %v", allowance, err)
	}
	remaining := allowance.RemainingSeconds(time.Now())
	if remaining <= 0 {
		return errors.New("ST15 correction publication exceeded the existing attempt budget")
	}
	deadline := time.Now().Add(2 * time.Minute)
	attemptDeadline := time.Now().Add(time.Duration(remaining) * time.Second)
	if attemptDeadline.Before(deadline) {
		deadline = attemptDeadline
	}
	ctx, cancel := context.WithDeadline(fx.ctx, deadline)
	defer cancel()
	var publicationReport control.PublishReport
	for time.Now().Before(deadline) {
		publicationReport = fx.coord.PublishPass(ctx)
		for _, entry := range publicationReport.Errors {
			if entry.Reason != "native_unavailable" {
				st15Receipt(fx.t, "correction-output", map[string]any{"output_tail": tailLines(output, 40), "publish": publicationReport})
				return fmt.Errorf("correction publication failed: %+v", publicationReport)
			}
		}
		p, err := fx.db.PublicationByAssignment(ctx, fx.assignA)
		if err != nil {
			return err
		}
		if p.Candidate != fx.head1 && factory.ValidCommit(p.Candidate) {
			fx.head2 = p.Candidate
			fx.requireContentStatus(fx.head2, "tests/test_total_regression.py", http.StatusOK)
			st15Receipt(fx.t, "correct-A", map[string]any{"head": fx.head2, "run": run.ID, "assignment": assignment.ID, "root": assignment.AttemptRoot})
			return nil
		}
		wait := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			wait.Stop()
			st15Receipt(fx.t, "correction-output", map[string]any{"output_tail": tailLines(output, 40), "publish": publicationReport})
			return fmt.Errorf("correction publication deadline: %w; final report: %+v", ctx.Err(), publicationReport)
		case <-wait.C:
		}
	}
	st15Receipt(fx.t, "correction-output", map[string]any{"output_tail": tailLines(output, 40), "report": publicationReport})
	return fmt.Errorf("correction unpublished: %+v", publicationReport)
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
	adopted, event := fx.reviewLeg(fx.head2)
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
