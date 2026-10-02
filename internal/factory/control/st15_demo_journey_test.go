package control_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	hostexec "github.com/levitateos/sodaos/internal/host"
	"github.com/levitateos/sodaos/internal/project"
)

// providerGate fails closed without a real provider credential: every
// input leg above runs credential-free, and no synthetic run ever
// substitutes for the composed execution below.
func (fx *st15Fixture) providerGate() error {
	if fx.synthetic {
		st15Receipt(fx.t, "provider-gate", map[string]any{"credential": "absent", "legs": fx.checks})
		return errors.New("ST15 requires SODA_ST15_PROVIDER_CREDENTIAL: a 0600 file of codex auth.json bytes; refusing synthetic execution")
	}
	return nil
}

// runForIssue returns one recorded run ID for an issue index, or "".
func (fx *st15Fixture) runForIssue(index int64) string {
	fx.t.Helper()
	runs, err := fx.db.FactoryRuns(fx.ctx, 50)
	if err != nil {
		fx.t.Fatal(err)
	}
	for _, run := range runs {
		view, err := fx.db.FactoryRunView(fx.ctx, run.ID)
		if err != nil {
			continue
		}
		if view.Issue == index {
			return run.ID
		}
	}
	return ""
}

// pollRunForIssue waits for a recorded run on one issue and returns it.
func (fx *st15Fixture) pollRunForIssue(index int64, timeout time.Duration) string {
	fx.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if id := fx.runForIssue(index); id != "" {
			return id
		}
		time.Sleep(2 * time.Second)
	}
	fx.t.Fatalf("ST15 no run recorded for issue #%d", index)
	return ""
}

// runBrowser executes the Playwright check in one mode. Secrets cross by
// file path only; the receipt carries IDs and counts.
func (fx *st15Fixture) runBrowser(mode, runID, role string, issue int64) {
	fx.t.Helper()
	out := filepath.Join(st15ReceiptDir(fx.t), "browser-"+mode+".json")
	cmd := exec.CommandContext(fx.ctx, "/home/vince/.bun/bin/bun", "/home/vince/Projects/sodaos/.artifacts/st15-demo/spaces-check.ts")
	cmd.Env = append(os.Environ(),
		"ST15_FOUNTAIN="+fx.cfg.BrowserURL,
		"ST15_USER=soda-maintainer",
		"ST15_PASS_FILE="+fx.cfg.MaintainerPass,
		"ST15_MODE="+mode,
		"ST15_OUT="+out,
		"ST15_RUN_ID="+runID,
		"ST15_ROLE="+role,
		"ST15_ISSUE="+strconv.FormatInt(issue, 10),
		"ST15_REPO_ID="+strconv.FormatInt(fx.cfg.Repository, 10),
	)
	cmd.Dir = "/home/vince/Projects/sodaos"
	combined, err := cmd.CombinedOutput()
	fx.t.Logf("ST15 browser %s: %s", mode, strings.TrimSpace(string(combined)))
	if err != nil {
		fx.t.Fatalf("ST15 browser %s: %v", mode, err)
	}
}

// dispatchA launches the coding run through production intake
// auto-dispatch while the live browser watches: the re-observation records
// A queued now that authority is effective, the intake pass launches
// exactly one run, the watch attaches, observes output bytes, then
// detaches, and the run continues to completion without it.
func (fx *st15Fixture) dispatchA() error {
	// Dispatch scans recorded queued controls: re-observe through the
	// real intake handler so A flips from not_authorized to queued and
	// the intake pass auto-launches. Closed P stays skipped by plan
	// (WaitIssueClosed); B/C stay unqueued/withdrawn on record.
	fx.commentHint(fx.issueAIndex)
	ctrl := fx.observeControl(fx.issueAIndex)
	if ctrl.Readiness != factory.ReadinessQueued && ctrl.Readiness != factory.ReadinessActive {
		return fmt.Errorf("A not queued after authority activation: %+v", ctrl)
	}
	runID := fx.pollRunForIssue(fx.issueAIndex, 10*time.Minute)
	fx.t.Logf("ST15 auto-dispatch launched run %s for A", runID)
	fx.runBrowser("watch", runID, project.RoleCoder, fx.issueAIndex)
	fx.t.Log("ST15 browser detached; the coding run continues without it")
	// An explicit pass must not duplicate the launch.
	report := fx.coord.Dispatch(fx.ctx)
	if len(report.Launched) != 0 {
		return fmt.Errorf("explicit dispatch duplicated the launch: %+v", report)
	}
	if len(report.Errors) != 0 {
		return fmt.Errorf("explicit dispatch errors: %+v", report)
	}
	assignment, output := fx.settleDispatched(runID)
	if assignment.Stage != factory.AssignmentFinished || assignment.Outcome != factory.Succeeded ||
		assignment.Result == nil || !assignment.Result.Reported || assignment.Result.Status != "completed" ||
		!factory.ValidCommit(assignment.Result.Candidate) {
		st15Receipt(fx.t, "coder-output", map[string]any{"output_tail": tailLines(output, 40)})
		return fmt.Errorf("coding run reported no completed candidate: %+v", assignment.Result)
	}
	fx.assignA, fx.runA = assignment.ID, runID
	st15Receipt(fx.t, "dispatch-A", map[string]any{
		"assignment": assignment.ID, "run": runID, "candidate": assignment.Result.Candidate,
	})
	st15Receipt(fx.t, "coder-output", map[string]any{"output_tail": tailLines(output, 40)})
	return nil
}

// stallForBrowserDiag is a diagnostic-only hook (default off): when
// ST15_DIAG_STALL_DIR names a directory, it records one synthetic run
// row plus view (never a provider run, never proof), writes the fixture
// connection info for a manual spaces-check.ts run, and sleeps so the
// fixture stays alive for browser/API debugging.
func (fx *st15Fixture) stallForBrowserDiag() {
	fx.t.Helper()
	dir := os.Getenv("ST15_DIAG_STALL_DIR")
	if dir == "" {
		return
	}
	now := time.Now().Truncate(time.Second)
	runID := factory.NewID()
	run := factory.Run{
		ID: runID, ProjectID: fx.projectID, Role: project.RoleCoder, InputSHA: strings.Repeat("d", 40),
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: fx.image, Harness: project.FactoryHarnessCodex, Model: "gpt-6-luna",
	}
	nativeMust(fx.t, fx.db.RecordFactoryRun(fx.ctx, run))
	_, _, err := fx.db.RecordFactoryRunView(fx.ctx, factory.RunView{
		RunID: runID, Repository: fx.cfg.Repository, Issue: 2, Attempt: factory.NewID(),
	})
	nativeMust(fx.t, err)
	info := map[string]string{
		"fountain_url": fx.cfg.BrowserURL, "pass_file": fx.cfg.MaintainerPass,
		"repository": strconv.FormatInt(fx.cfg.Repository, 10), "run_id": runID,
		"role": project.RoleCoder, "issue": "2",
	}
	raw, err := json.MarshalIndent(info, "", "  ")
	nativeMust(fx.t, err)
	nativeMust(fx.t, os.MkdirAll(dir, 0o700))
	nativeMust(fx.t, os.WriteFile(filepath.Join(dir, "stall.json"), raw, 0o600))
	minutes := 15
	if raw := os.Getenv("ST15_DIAG_STALL_MINUTES"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 30 {
			minutes = parsed
		}
	}
	fx.t.Logf("ST15 DIAG stall: %s for %d minutes (synthetic run %s, never proof)", dir, minutes, runID)
	deadline := time.Now().Add(time.Duration(minutes) * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(dir, "release")); err == nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func tailLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// publishA drives the production publication pass to the linked PR.
func (fx *st15Fixture) publishA() error {
	nativeMergeIdle(fx.t, forgejo.NewServiceBackground(fx.cfg.Socket, uint32(os.Getuid()), ""))
	last := ""
	for i := 0; i < 120; i++ {
		report := fx.coord.PublishPass(fx.ctx)
		for _, entry := range report.Errors {
			if entry.Reason != "native_unavailable" {
				return fmt.Errorf("publication errors: %+v", report)
			}
		}
		p, err := fx.db.PublicationByAssignment(fx.ctx, fx.assignA)
		if err != nil {
			return err
		}
		state := fmt.Sprintf("stage=%s publish=%d/%s/%s prcreate=%d/%s/%s waits=%v errors=%v",
			p.Stage, p.Publish.Attempts, p.Publish.Effect, p.Publish.Completion,
			p.PRCreate.Attempts, p.PRCreate.Effect, p.PRCreate.Completion,
			report.Waits, report.Errors)
		if i%20 == 0 || state != last {
			fx.t.Logf("ST15 publish-A poll %d: %s", i, state)
			last = state
		}
		if p.Stage == factory.PublicationPublished {
			fx.pubA, fx.head1, fx.prNumber = p.ID, p.Candidate, p.PRNumber
			st15Receipt(fx.t, "publish-A", map[string]any{
				"publication": p.ID, "head": p.Candidate, "pr": p.PRNumber, "revision": p.Revision,
			})
			return nil
		}
		if p.Stage != factory.PublicationOpen && p.Stage != factory.PublicationFenced {
			return fmt.Errorf("publication terminal: %+v", p)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errors.New("publication did not settle")
}

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
		state, err = fx.rt.InspectPreparation(ctx, project.PrepareInspect{Project: fx.projectID, ID: prep.ID})
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
	run, _ := fx.launchDirect(project.RoleReviewer, prep, fx.assignA, prompt, head)
	output := fx.settleDirect(run, fx.assignA)
	review := parseReview(fx.t, output)
	adopted, event := fx.submitReview(head, fx.sourceHead, review)
	st15Receipt(fx.t, "review-"+head[:12], map[string]any{
		"run": run.ID, "verdict": review.Verdict, "event": event,
		"review": adopted.ReviewID, "summary": review.Summary, "findings": review.Findings,
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
	assessor := forgejo.NewCheckAssessor(fx.bg, fx.rest, fx.cfg.TokenFile)
	report := fx.coord.AssessPublicationChecks(fx.ctx, assessor, fx.cfg.ActorID, fx.assignA)
	if len(report.Errors) != 0 || len(report.Assessed) != 1 || report.Assessed[0].Verdict != factory.CheckFailed {
		return fmt.Errorf("CI failure unrecorded: %+v", report)
	}
	fx.ciFailRev = report.Assessed[0].Revision
	st15Receipt(fx.t, "ci-fail", report.Assessed[0])
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
// recorded CI and review appendix, and publishes it to the same PR.
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
	output := fx.settleDirect(run, fx.assignA)
	// A failed run must fail here with its own cause, never as a
	// candidate wait inside the correction pass.
	if reported, ok := factory.ParseHarnessResult(output); !ok || reported.Status != "completed" {
		st15Receipt(fx.t, "correction-output", map[string]any{"output_tail": tailLines(output, 40)})
		return fmt.Errorf("correction run produced no completed result (parse=%v)", ok)
	}
	report := fx.coord.PublishCorrection(fx.ctx, fx.assignA, run.ID, output)
	if len(report.Errors) != 0 || len(report.Corrected) != 1 {
		st15Receipt(fx.t, "correction-output", map[string]any{"output_tail": tailLines(output, 40), "report": report})
		return fmt.Errorf("correction unpublished: %+v", report)
	}
	fx.head2 = report.Corrected[0].HeadOID
	fx.requireContentStatus(fx.head2, "tests/test_total_regression.py", http.StatusOK)
	st15Receipt(fx.t, "correct-A", report.Corrected[0])
	return nil
}

func (fx *st15Fixture) reviewLeg2() error {
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

// ciPass seeds the passing verdict on the corrected head and records it.
func (fx *st15Fixture) ciPass() error {
	fx.seedCIStatus(fx.head2, "st15-build", "success", "widget compiles")
	fx.seedCIStatus(fx.head2, "st15-test", "success", "required regression test present")
	assessor := forgejo.NewCheckAssessor(fx.bg, fx.rest, fx.cfg.TokenFile)
	report := fx.coord.AssessPublicationChecks(fx.ctx, assessor, fx.cfg.ActorID, fx.assignA)
	if len(report.Errors) != 0 || len(report.Assessed) != 1 || report.Assessed[0].Verdict != factory.CheckPass {
		return fmt.Errorf("CI pass unrecorded: %+v", report)
	}
	fx.ciPassRev = report.Assessed[0].Revision
	st15Receipt(fx.t, "ci-pass", report.Assessed[0])
	return nil
}

// mergeAndDependants drives the conditional merge while the dependant's
// automatic pickup runs under live browser control: the browser stops
// the dependant run, then latches and reopens the dispatch gate.
func (fx *st15Fixture) mergeAndDependants() error {
	nativeMergeIdle(fx.t, forgejo.NewServiceBackground(fx.cfg.Socket, uint32(os.Getuid()), ""))
	type mergeOutcome struct {
		merge  factory.Merge
		report control.MergeReport
	}
	done := make(chan mergeOutcome, 1)
	driven := make(chan error, 1)
	go func() {
		ctx := context.Background()
		var last control.MergeReport
		for i := 0; i < 240; i++ {
			last = fx.coord.MergePass(ctx)
			for _, entry := range last.Errors {
				if entry.Reason != "native_unavailable" {
					driven <- fmt.Errorf("merge errors: %+v", last)
					return
				}
			}
			m, err := fx.db.MergeByPublication(ctx, fx.pubA)
			if err == nil && m.Stage != factory.MergeOpen && m.Stage != factory.MergeFenced {
				done <- mergeOutcome{m, last}
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		driven <- errors.New("merge did not settle")
	}()
	// The dependant starts automatically inside the completing pass.
	// Poll for its run, but fail fast when the merge drive already
	// reported terminally: a failed merge never dispatches B, so
	// waiting out the run timeout would mask the real failure.
	var runB string
	var pending *mergeOutcome
	trigger := "completion-cascade"
	var mergedAt time.Time
	fallbackDone := false
	deadlineB := time.Now().Add(45 * time.Minute)
pollB:
	for time.Now().Before(deadlineB) {
		select {
		case outcome := <-done:
			if outcome.merge.Stage != factory.MergeMerged {
				return fmt.Errorf("merge terminal without merging: %+v", outcome.merge)
			}
			pending = &outcome
			if mergedAt.IsZero() {
				mergedAt = time.Now()
			}
		case err := <-driven:
			return err
		default:
		}
		if id := fx.runForIssue(fx.issueBIndex); id != "" {
			runB = id
			break pollB
		}
		if pending != nil && !fallbackDone && !mergedAt.IsZero() && time.Since(mergedAt) > 90*time.Second {
			// The completion cascade normally releases B within
			// seconds. If a transient cascade error was swallowed,
			// the fixture has no ambient intake traffic to supply
			// production's normal next trigger — so synthesize one
			// real intake event and re-observe exactly like
			// dispatch-A. The release itself stays fully automatic.
			fallbackDone = true
			trigger = "intake-fallback"
			fx.t.Log("ST15 B not released by completion cascade; firing intake fallback")
			fx.commentHint(fx.issueBIndex)
			fx.observeControl(fx.issueBIndex)
		}
		time.Sleep(2 * time.Second)
	}
	if runB == "" {
		return errors.New("ST15 no run recorded for dependant issue")
	}
	fx.assignB = mustViewAttempt(fx, runB)
	fx.runBrowser("control", runB, project.RoleCoder, fx.issueBIndex)
	if pending == nil {
		select {
		case o := <-done:
			pending = &o
		case err := <-driven:
			return err
		case <-time.After(50 * time.Minute):
			return errors.New("merge drive timed out")
		}
	}
	if pending.merge.Stage != factory.MergeMerged {
		return fmt.Errorf("merge terminal without merging: %+v", pending.merge)
	}
	st15Receipt(fx.t, "merge-A", map[string]any{
		"merge": pending.merge.ID, "commit": pending.merge.MergedCommit,
		"pr": pending.merge.PRNumber, "report": pending.report,
		"dependant_trigger": trigger,
	})
	// The browser stop settled B through the dashboard coordinator: the
	// run is reconciled-cancelled and its assignment finished without a
	// result, so it never publishes.
	deadline := time.Now().Add(5 * time.Minute)
	for {
		assignment, err := fx.db.Assignment(fx.ctx, fx.assignB)
		if err != nil {
			return err
		}
		if assignment.Stage == factory.AssignmentFinished {
			if assignment.Outcome != factory.Cancelled {
				return fmt.Errorf("stopped dependant finished %s, want cancelled", assignment.Outcome)
			}
			st15Receipt(fx.t, "dependant-B", map[string]any{
				"assignment": assignment.ID, "run": runB, "outcome": assignment.Outcome,
			})
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("dependant assignment unsettled: %+v", assignment)
		}
		time.Sleep(2 * time.Second)
	}
}

func mustViewAttempt(fx *st15Fixture, runID string) string {
	fx.t.Helper()
	view, err := fx.db.FactoryRunView(fx.ctx, runID)
	if err != nil {
		fx.t.Fatal(err)
	}
	return view.Attempt
}

// proveRetention verifies the human side survived: dirty files, service
// data and process, member execution, the same container, bounded usage,
// and the decoy's permanent silence.
func (fx *st15Fixture) proveRetention() error {
	ctx := fx.ctx
	var snapshot map[string]string
	raw, err := os.ReadFile(filepath.Join(st15ReceiptDir(fx.t), "human-snapshot.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return err
	}
	for _, path := range []string{"/home/soda-tester/work/dirty.txt", "/home/soda-tester/service-data/rows.txt", "/home/soda-tester/.ssh/authorized_keys"} {
		out, err := st15Pexec(ctx, fx.container, "", nil, "/usr/bin/sha256sum", path)
		if err != nil {
			return err
		}
		if got := strings.Fields(strings.TrimSpace(string(out)))[0]; got != snapshot[path] {
			return fmt.Errorf("retained file %s changed", path)
		}
	}
	if _, err := st15Pexec(ctx, fx.container, "soda-tester", nil, "/usr/bin/true"); err != nil {
		return fmt.Errorf("member execution lost: %v", err)
	}
	if _, err := st15Pexec(ctx, fx.container, "", nil, "/usr/bin/sh", "-c", "kill -0 "+snapshot["service_pid"]); err != nil {
		return fmt.Errorf("retained service process lost: %v", err)
	}
	id, err := st15Podman(ctx, nil, "inspect", "--format", "{{.ID}}", fx.container)
	if err != nil {
		return err
	}
	fx.t.Logf("ST15 container identity stable: %.12s", strings.TrimSpace(string(id)))
	runs, err := fx.db.FactoryRuns(ctx, 50)
	if err != nil {
		return err
	}
	total := 0
	rows := 0
	for _, run := range runs {
		view, err := fx.db.FactoryRunView(ctx, run.ID)
		if err != nil {
			continue
		}
		if view.Issue == fx.issueCIndex {
			return fmt.Errorf("withdrawn decoy ran: %s", run.ID)
		}
		usage, err := fx.db.RunUsage(ctx, run.ID)
		if err != nil {
			return fmt.Errorf("run %s lacks its usage row: %w", run.ID, err)
		}
		total += usage.Minutes
		rows++
	}
	if rows == 0 || total > 60 {
		return fmt.Errorf("usage out of bounds: %d minutes over %d runs", total, rows)
	}
	st15Receipt(fx.t, "retention", map[string]any{
		"usage_minutes": total, "usage_rows": rows, "runs": len(runs),
		"container": strings.TrimSpace(string(id)),
	})
	return nil
}

func (fx *st15Fixture) writeFinalReceipt() {
	st15Receipt(fx.t, "journey", map[string]any{
		"status": "passed", "qualification": "development-only",
		"checks":      fx.checks,
		"issues":      map[string]any{"P": fx.issuePIndex, "A": fx.issueAIndex, "B": fx.issueBIndex, "C": fx.issueCIndex},
		"acceptances": map[string]any{"P": fx.acceptP, "A": fx.acceptA, "B": fx.acceptB},
		"assignments": map[string]any{"A": fx.assignA, "B": fx.assignB},
		"publication": fx.pubA, "pr": fx.prNumber, "head1": fx.head1, "head2": fx.head2,
		"reviews": map[string]any{"r1": fx.review1.ReviewID, "v1": fx.verdict1, "r2": fx.review2.ReviewID, "v2": fx.verdict2},
		"ci":      map[string]any{"fail_rev": fx.ciFailRev, "pass_rev": fx.ciPassRev},
	})
}

func TestST15ComposedDemo(t *testing.T) {
	cfg := loadST15(t)
	t.Setenv("ST09_RECEIPT_DIR", st15ReceiptDir(t))
	fx := &st15Fixture{t: t, ctx: context.Background(), cfg: cfg}
	fx.check("host-stack", fx.setupHostStack())
	fx.check("factory-store", fx.openFactoryStore())
	fx.check("project", fx.setupProject())
	fx.check("seed-repository", fx.seedRepository())
	fx.check("prepare-roles", fx.prepareRoles())
	fx.check("human-work", fx.seedHumanWork())
	fx.check("wire-coordinator", fx.wireCoordinator())
	fx.stallForBrowserDiag()
	fx.check("grant-withdrawal", fx.proveGrantWithdrawal())
	fx.check("seed-issues", fx.seedIssues())
	fx.check("accept-P", fx.admitPLeg())
	fx.check("accept-A", fx.admitALeg())
	fx.check("accept-BC", fx.admitBCLegs())
	fx.check("closure-not-outcome", fx.proveClosureNotOutcome())
	fx.check("provider-gate", fx.providerGate())
	fx.check("activate-authority", fx.activateAuthority())
	fx.check("dispatch-A", fx.dispatchA())
	fx.check("publish-A", fx.publishA())
	fx.check("review-1", fx.reviewLeg1())
	fx.check("ci-fail", fx.ciFail())
	fx.check("correct-A", fx.correctA())
	fx.check("review-2", fx.reviewLeg2())
	fx.check("ci-pass", fx.ciPass())
	fx.check("merge-and-dependants", fx.mergeAndDependants())
	fx.check("retention", fx.proveRetention())
	fx.writeFinalReceipt()
}
