package control_test

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

// launchDirect records and launches one driver-sequenced run (reviewer or
// correction) through the real daemon, broker lease, container and
// harness. It blocks until the CLI exits and returns the final state.
// The assignment linkage rides the run view; durable review/retry
// scheduling stays a documented follow-up. The source commit must be
// the checkout's current head: reserve refuses any other assigned
// commit, and each role preparation carries one shared checkout, so a
// follow-up run sources its predecessor's head.
func (fx *st15Fixture) launchDirect(role, preparation, assignmentID, prompt, source string) (factory.Run, project.FactoryState) {
	fx.t.Helper()
	ctx, cancel := context.WithTimeout(fx.ctx, 40*time.Minute)
	defer cancel()
	runID := factory.NewID()
	now := time.Now().Truncate(time.Second)
	run := factory.Run{
		ID: runID, ProjectID: fx.projectID, Role: role, InputSHA: source,
		Started: now, Deadline: now.Add(30 * time.Minute),
		Image: fx.image, Harness: project.FactoryHarnessMuse, Model: "muse-spark-1.3",
	}
	if err := fx.db.RecordFactoryRun(ctx, run); err != nil {
		fx.t.Fatal(err)
	}
	if _, _, err := fx.db.RecordFactoryRunView(ctx, factory.RunView{
		RunID: runID, Repository: fx.cfg.Repository, Issue: fx.issueAIndex, Attempt: assignmentID,
	}); err != nil {
		fx.t.Fatal(err)
	}
	state, err := fx.coord.Host.FactoryLaunch(ctx, project.FactoryLaunch{
		Run: project.FactoryRun{
			Deadline: now.Add(30 * time.Minute), Actor: fx.cfg.CreatorID, ID: runID,
			Project: fx.projectID, Role: role, Preparation: preparation,
			Harness: project.FactoryHarnessMuse, HarnessVers: fx.versions,
			Model:      "muse-spark-1.3",
			Assignment: project.FactoryPromptDigest([]byte(prompt)), SourceCommit: source,
			Connection: "st15-muse",
		},
		Prompt: []byte(prompt), HarnessSHA256: fx.harnessSHA,
	})
	if err != nil {
		fx.t.Fatalf("ST15 direct %s launch: %v", role, err)
	}
	// The launch receipt names the container by full ID while the
	// fixture tracks its name; both identify the same container, and
	// either one mismatching means a genuinely foreign run.
	if state.Container != "" && state.Container != fx.container && state.Container != fx.containerID {
		fx.t.Fatalf("ST15 direct %s ran in foreign container %q", role, state.Container)
	}
	return run, state
}

// stopRunSettled stops one run through the host, retrying while
// retirement reports uncertain: the product contract converges a
// repeated stop once racing retirement settles (Stop in
// host/project/factory.go), so a bounded retry follows production
// semantics instead of failing a transient stop/completion race.
// Anything still uncertain after the bound fails.
func (fx *st15Fixture) stopRunSettled(runID string) project.FactoryState {
	fx.t.Helper()
	var state project.FactoryState
	for attempt := 0; ; attempt++ {
		var err error
		state, err = fx.coord.Host.FactoryStop(fx.ctx, project.FactoryStop{Project: fx.projectID, ID: runID})
		if err != nil {
			fx.t.Fatal(err)
		}
		if state.Retirement != "uncertain" || attempt >= 5 {
			return state
		}
		fx.t.Logf("ST15 run %s stop uncertain (attempt %d), retrying", runID, attempt+1)
		time.Sleep(5 * time.Second)
	}
}

// settleDirect settles one reviewer run the driver sequenced itself: host
// stop, broker close, recorded outcome, usage row. It returns the full
// run output for the explicit production submission. Reviewer-only until
// reviewer dispatch lands; coder runs settle through stopSettled below.
func (fx *st15Fixture) settleDirect(run factory.Run, assignmentID string) string {
	fx.t.Helper()
	ctx := fx.ctx
	state := fx.stopRunSettled(run.ID)
	var outcome factory.Outcome
	switch {
	case state.Retirement == "uncertain":
		fx.t.Fatalf("ST15 run %s retired uncertain", run.ID)
	case state.Phase == project.FactoryCompleted:
		outcome = factory.Succeeded
	case state.Phase == project.FactoryFailed:
		outcome = factory.Failed
	case state.Phase == project.FactoryStopped:
		outcome = factory.Cancelled
	default:
		fx.t.Fatalf("ST15 run %s unsettled: %+v", run.ID, state)
	}
	run.Outcome, run.Reconciled = outcome, true
	if len(state.Output) > 16*1024 {
		run.Summary = state.Output[:16*1024]
	} else {
		run.Summary = state.Output
	}
	if err := fx.db.SaveFactoryRun(ctx, run); err != nil {
		fx.t.Fatal(err)
	}
	if err := fx.coord.Broker.CloseExecution(ctx, identity.Factory, run.ID); err != nil {
		fx.t.Fatal(err)
	}
	minutes := 0
	if elapsed := time.Since(run.Started); elapsed > 0 {
		minutes = int((elapsed + time.Minute - time.Nanosecond) / time.Minute)
	}
	if err := fx.db.RecordRunUsage(ctx, factory.Usage{
		RunID: run.ID, Repository: fx.cfg.Repository, Connection: "st15-muse",
		Minutes: minutes, RecordedUnix: time.Now().Unix(),
	}); err != nil {
		fx.t.Fatal(err)
	}
	st15Receipt(fx.t, "run-"+run.ID, map[string]any{
		"role": run.Role, "outcome": outcome, "minutes": minutes,
		"container": state.Container, "credential_returned": state.CredentialReturned,
		"output_tail": tailLines(state.Output, 40),
	})
	if reported, ok := factory.ParseHarnessResult(state.Output); ok {
		fx.t.Logf("ST15 direct %s result: status=%s candidate=%s review_passed=%v findings=%d",
			run.Role, reported.Status, reported.Candidate, reported.ReviewPassed, len(reported.Findings))
	} else if strings.Contains(state.Output, "```review-json") {
		fx.t.Logf("ST15 direct %s result: review report present (%d bytes)", run.Role, len(state.Output))
	} else {
		fx.t.Logf("ST15 direct %s result: unparseable (%d bytes)", run.Role, len(state.Output))
	}
	return state.Output
}

// stopSettled retires one run through the production stop path, retrying
// while retirement reports uncertain: the product contract converges a
// repeated stop once racing retirement settles, so a bounded retry follows
// production semantics. Anything still uncertain after the bound fails.
func (fx *st15Fixture) stopSettled(runID string) control.StopReceipt {
	fx.t.Helper()
	var receipt control.StopReceipt
	for attempt := 0; ; attempt++ {
		cmd := factory.Command{ID: factory.NewID(), Type: factory.CommandStop, Target: runID,
			Principal: "soda-maintainer", Digest: factory.CommandDigest(factory.CommandStop, runID)}
		var err error
		receipt, err = fx.coord.Stop(fx.ctx, cmd)
		if err != nil {
			fx.t.Fatal(err)
		}
		if !receipt.Uncertain || attempt >= 5 {
			break
		}
		fx.t.Logf("ST15 run %s stop uncertain (attempt %d), retrying", runID, attempt+1)
		time.Sleep(5 * time.Second)
	}
	if !receipt.Confirmed {
		fx.t.Fatalf("ST15 run %s retired uncertain: %s", runID, receipt.Reason)
	}
	return receipt
}

// settleDispatched settles one production-dispatched run through the
// production stop path and returns its assignment and recorded summary.
// The stop retires the run, records its outcome and advances publication;
// the fixture only observes the result.
func (fx *st15Fixture) settleDispatched(runID string) (factory.Assignment, string) {
	fx.t.Helper()
	receipt := fx.stopSettled(runID)
	assignment, err := fx.db.AssignmentByRun(fx.ctx, runID)
	if err != nil {
		fx.t.Fatalf("ST15 run %s assignment: %v", runID, err)
	}
	if assignment.Stage != factory.AssignmentFinished {
		fx.t.Fatalf("ST15 run %s assignment unsettled: %+v", runID, assignment)
	}
	st15Receipt(fx.t, "run-"+runID, map[string]any{
		"outcome": receipt.Outcome, "assignment": assignment.ID, "attempts": assignment.Attempts,
	})
	return assignment, receipt.Reason
}

// reviewPrompt builds the reviewer brief from accepted requirements and
// candidate evidence only: no coding conversation crosses the roles.
func (fx *st15Fixture) reviewPrompt(issueTitle, issueBody, answer, head, base string) string {
	return "# Soda factory review assignment\n\n" +
		"Repository: " + strconv.FormatInt(fx.cfg.Repository, 10) + "\n" +
		"Issue: " + strconv.FormatInt(fx.issueAIndex, 10) + "\n" +
		"Candidate: " + head + " over base " + base + "\n\n" +
		"Review the exact candidate in the prepared checkout as an independent reviewer. " +
		"Only the accepted objective, sources and resolutions below authorize requirements; " +
		"the coding conversation is unavailable and must not be assumed.\n\n" +
		"## Objective\n\n" + issueTitle + "\n\n" + issueBody + "\n\n" +
		"## Accepted answer\n\n" + answer + "\n\n" +
		"## Report\n\nClose your work with exactly one fenced block:\n\n```review-json\n" +
		`{"verdict":"approve|request-changes","summary":"...","body":"...markdown review...","findings":["..."]}` + "\n```\n\n" +
		"Approve only when the candidate satisfies the accepted requirements. " +
		"Request changes with concrete findings otherwise. " +
		"Do not modify the checkout: a review reads and reports, it never commits. " +
		"Never print credentials, tokens or secret files.\n"
}

// submitReview submits one settled reviewer run's genuine verdict through
// the production review path and returns the adopted outcome and event.
func (fx *st15Fixture) submitReview(runID, output string) (factory.ReviewOutcome, string) {
	fx.t.Helper()
	adopted, err := fx.coord.SubmitReviewForRun(fx.ctx, runID, output)
	if err != nil {
		fx.t.Fatalf("ST15 review submit: %v", err)
	}
	if adopted.ReviewerID != fx.cfg.ReviewerID {
		fx.t.Fatalf("ST15 review adopted by a foreign reviewer: %+v", adopted)
	}
	return adopted, adopted.Event
}

// seedCIStatus records one native commit status (the fixture CI verdict)
// through the ordinary REST path, exactly like ST11.
func (fx *st15Fixture) seedCIStatus(head, contextName, state, description string) {
	fx.t.Helper()
	var status struct {
		ID int64 `json:"id"`
	}
	fx.api(fx.cfg.CreatorTokenFile, http.MethodPost, "/api/v1/repos/"+fx.cfg.Owner+"/"+fx.cfg.Repo+"/statuses/"+head,
		map[string]any{"state": state, "context": contextName, "description": description}, &status)
	if status.ID <= 0 {
		fx.t.Fatalf("ST15 CI status %s/%s unrecorded", contextName, state)
	}
}

var _ = forgejo.ContentDigest
