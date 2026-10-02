package control_test

import (
	"context"
	"encoding/json"
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
		Image: fx.image, Harness: project.FactoryHarnessCodex, Model: "gpt-6-luna",
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
			Harness: project.FactoryHarnessCodex, HarnessVers: fx.versions,
			Model:      "gpt-6-luna",
			Assignment: project.FactoryPromptDigest([]byte(prompt)), SourceCommit: source,
			Connection: "st15-codex",
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

// settleDirect settles one driver-sequenced run exactly like the
// coordinator settle path: host stop, broker close, recorded outcome,
// usage row. It returns the run output.
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
		RunID: run.ID, Repository: fx.cfg.Repository, Connection: "st15-codex",
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

// settleDispatched settles one production-dispatched run through the
// production accounting path and returns its assignment and output.
func (fx *st15Fixture) settleDispatched(runID string) (factory.Assignment, string) {
	fx.t.Helper()
	run, err := fx.db.FactoryRun(fx.ctx, runID)
	if err != nil {
		fx.t.Fatal(err)
	}
	state := fx.stopRunSettled(runID)
	var outcome factory.Outcome
	switch {
	case state.Retirement == "uncertain":
		fx.t.Fatalf("ST15 run %s retired uncertain", runID)
	case state.Phase == project.FactoryCompleted:
		outcome = factory.Succeeded
	case state.Phase == project.FactoryFailed:
		outcome = factory.Failed
	case state.Phase == project.FactoryStopped:
		outcome = factory.Cancelled
	default:
		fx.t.Fatalf("ST15 run %s unsettled: %+v", runID, state)
	}
	if err := fx.coord.Broker.CloseExecution(fx.ctx, identity.Factory, runID); err != nil {
		fx.t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = outcome, state.Output, true
	if len(run.Summary) > 16*1024 {
		run.Summary = run.Summary[:16*1024]
	}
	if err := fx.db.SaveFactoryRun(fx.ctx, run); err != nil {
		fx.t.Fatal(err)
	}
	// Production accounting: usage, reservation consume, attempt finish.
	// Reconcile itself stays out of the journey: it would record merges
	// eagerly, before review and checks evidence exists.
	probe, perr := fx.db.AssignmentByRun(fx.ctx, runID)
	if perr != nil {
		fx.t.Logf("ST15 settle probe run=%s AssignmentByRun err: %v", runID, perr)
	} else {
		fx.t.Logf("ST15 settle probe run=%s assignment=%s stage=%s head=%s history=%v attempts=%d",
			runID, probe.ID, probe.Stage, probe.Run, probe.RunHistory, probe.Attempts)
		if res, rerr := fx.db.Reservation(fx.ctx, probe.ID); rerr != nil {
			fx.t.Logf("ST15 settle probe reservation err: %v", rerr)
		} else {
			fx.t.Logf("ST15 settle probe reservation state=%s revision=%d", res.State, res.Revision)
		}
	}
	if _, uerr := fx.db.RunUsage(fx.ctx, runID); uerr != nil {
		fx.t.Logf("ST15 settle probe no prior usage for run=%s: %v", runID, uerr)
	} else {
		fx.t.Logf("ST15 settle probe prior usage already recorded for run=%s", runID)
	}
	assignment, ok := control.AccountSettledRun(fx.ctx, fx.db, run, state.Output, time.Now())
	if !ok {
		// The shared control plane may have settled first (dashboard
		// settle/reconcile racing the journey's explicit sequencing):
		// adopt the finished assignment instead of failing. The
		// journey's outcome checks below still judge the result.
		adopted, aerr := fx.db.AssignmentByRun(fx.ctx, runID)
		if aerr != nil || adopted.Stage != factory.AssignmentFinished {
			fx.t.Fatalf("ST15 run %s accounting refused", runID)
		}
		fx.t.Logf("ST15 run %s already settled externally; adopting assignment %s outcome=%s",
			runID, adopted.ID, adopted.Outcome)
		assignment = adopted
	}
	st15Receipt(fx.t, "run-"+runID, map[string]any{
		"role": run.Role, "outcome": outcome, "container": state.Container,
		"assignment": assignment.ID, "attempts": assignment.Attempts,
	})
	return assignment, state.Output
}

type st15Review struct {
	Verdict  string   `json:"verdict"`
	Summary  string   `json:"summary"`
	Body     string   `json:"body"`
	Findings []string `json:"findings"`
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

func parseReview(t interface {
	Helper()
	Fatalf(string, ...any)
}, output string,
) st15Review {
	t.Helper()
	start := strings.LastIndex(output, "```review-json")
	if start < 0 {
		t.Fatalf("ST15 review lacks its fenced report")
	}
	rest := output[start+len("```review-json"):]
	end := strings.Index(rest, "```")
	if end < 0 {
		t.Fatalf("ST15 review report unterminated")
	}
	var review st15Review
	if err := json.Unmarshal([]byte(rest[:end]), &review); err != nil {
		t.Fatalf("ST15 review report invalid: %v", err)
	}
	if review.Verdict != "approve" && review.Verdict != "request-changes" {
		t.Fatalf("ST15 review verdict invalid: %q", review.Verdict)
	}
	if review.Verdict == "request-changes" && strings.TrimSpace(review.Body) == "" {
		t.Fatalf("ST15 requested changes lack findings")
	}
	return review
}

// submitReview observes the exact head and submits the agent's genuine
// verdict through the separate reviewer actor. It returns the adopted
// outcome and the event submitted.
func (fx *st15Fixture) submitReview(head, base string, review st15Review) (factory.ReviewOutcome, string) {
	fx.t.Helper()
	ctx := fx.ctx
	p, err := fx.db.PublicationByAssignment(ctx, fx.assignA)
	if err != nil {
		fx.t.Fatal(err)
	}
	event := "APPROVED"
	if review.Verdict == "request-changes" {
		event = "REQUEST_CHANGES"
	}
	body := review.Body
	if strings.TrimSpace(body) == "" {
		body = review.Summary
	}
	w := factory.ReviewWork{
		OperationID: "st15-review-" + factory.NewID(), AuthRevision: "st15-composed-demo",
		Repository: fx.cfg.Repository, ActorID: fx.cfg.ReviewerID,
		PRNumber: p.PRNumber, PRID: p.PRID, IssueID: p.PRCreate.IssueID, PRAuthorID: p.PRCreate.Work.ActorID,
		HeadRef: p.PRCreate.HeadRef, BaseRef: p.PRCreate.BaseRef, HeadOID: head, BaseOID: base,
		Event: event, Body: body,
	}
	observed, err := fx.coord.Reviews.ObserveReview(ctx, w)
	if err != nil {
		fx.t.Fatalf("ST15 review observe %s: %v", head[:12], err)
	}
	w.NativeRev, w.NotAfter = observed.NativeRev, time.Now().Unix()+600
	outcome, err := fx.coord.Reviews.SubmitReview(ctx, w)
	if err != nil {
		fx.t.Fatalf("ST15 review submit %s: %v", head[:12], err)
	}
	adopted, err := fx.coord.Reviews.AdoptReview(w, outcome)
	if err != nil {
		fx.t.Fatal(err)
	}
	if outcome.Completion != factory.OpCompletionComplete || adopted.ReviewerID != fx.cfg.ReviewerID {
		fx.t.Fatalf("ST15 review incomplete: %+v %+v", outcome, adopted)
	}
	return adopted, event
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
