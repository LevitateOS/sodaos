package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestDeliveredIntentAdmitsScopedTestPilotOnce(t *testing.T) {
	for _, early := range []bool{false, true} {
		t.Run(fmt.Sprintf("early-terminal-%t", early), func(t *testing.T) {
			ctx := context.Background()
			db, dsn := dispatchTestDB(t)
			fx := dispatchSeed(t, db)
			sponsor, err := db.Sponsorship(ctx, fx.repo, "conn")
			if err != nil {
				t.Fatal(err)
			}
			sponsor.Roles = []string{project.RoleCoder, project.RoleReviewer}
			if err = db.SaveSponsorship(ctx, sponsor); err != nil {
				t.Fatal(err)
			}
			idea := factory.ProductIdea{ID: phasedDecisionID(51), Description: "A starter saves a task", Consequence: "Tasks persist", Decision: "accept", Source: "requested", Provenance: "human", Actor: 7}
			task := factory.IntentTask{Key: phasedDecisionID(52), Objective: "Implement task storage", Outcomes: []string{"Tasks persist"}, IdeaIDs: []string{idea.ID}}
			revision := factory.IntentRevision{ID: phasedDecisionID(41), Intent: phasedDecisionID(40), Owner: 7, Repository: 7, Objective: "Task app", State: "accepted", MaxIssues: 10, CreatedUnix: 1, Ideas: []factory.ProductIdea{idea}, Tasks: []factory.IntentTask{task}, EffectiveOutcomes: []string{idea.ID}}
			if err = db.SaveIntentRevision(ctx, revision); err != nil {
				t.Fatal(err)
			}
			binding := factory.IntentTaskBinding{RevisionID: revision.ID, TaskKey: task.Key, IssueID: 3, IssueIndex: 3, AcceptanceID: phasedDecisionID(53), CreatedUnix: 1}
			if err = db.BindIntentTaskAcceptance(ctx, binding); err != nil {
				t.Fatal(err)
			}
			candidates, err := db.DeliveredIntentCandidates(ctx, 16)
			if err != nil || len(candidates) != 0 {
				t.Fatalf("undelivered work admitted: %+v %v", candidates, err)
			}
			sqlDB, err := sql.Open("pgx", dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()
			event := factory.CompletionEvent{ID: factory.NewID(), Repository: 7, Issue: 3, Generation: 1, PublicationID: factory.NewID(), AcceptanceID: binding.AcceptanceID, MergedCommit: fx.tip, ClosingOccurrence: "1"}
			raw, _ := json.Marshal(event)
			if _, err = sqlDB.Exec(`INSERT INTO factory_completions(repository,generation,event,publication,occurrence,data) VALUES(7,1,$1,$2,'1',$3)`, event.ID, event.PublicationID, string(raw)); err != nil {
				t.Fatal(err)
			}
			candidates, err = db.DeliveredIntentCandidates(ctx, 16)
			if err != nil || len(candidates) != 1 {
				t.Fatalf("delivered work missing: %+v %v", candidates, err)
			}
			prep, err := db.Preparation(ctx, "f222222222222222222222222")
			if err != nil {
				t.Fatal(err)
			}
			manifest := factory.ApprovedAppCheck{IntentID: revision.Intent, AcceptedRevision: revision.ID, Outcomes: []string{idea.ID}}
			encoded, _ := json.Marshal(manifest)
			script := []byte("#!/bin/sh\n" + factory.ApprovedAppCheckPrefix + string(encoded) + "\nexit 0\n")
			zero := 0
			fx.host.inspectPreparation = func(project.PrepareInspect) (project.PrepareState, error) {
				return project.PrepareState{ID: prep.Preparation.ID, Project: fx.proj, Role: project.RoleReviewer, SourceCommit: fx.tip, SetupDigest: prep.Preparation.SetupDigest, Phase: project.PrepareReady, Ready: true, CheckExit: &zero, Output: "task storage passed"}, nil
			}
			fx.host.contextRead = func(request project.FactoryPreparationContextRequest, role string) (project.FactoryPreparationContext, error) {
				material := fixturePreparationContext(request, role)
				material.Check = script
				return material, nil
			}
			c := &Coordinator{Store: db, Host: fx.host, DispatchReads: &appVerificationReads{tip: fx.tip}}
			subject, preparation, prompt, err := c.deliveryAdmission(ctx, candidates[0])
			if err != nil {
				t.Fatal(err)
			}
			if preparation != prep.Preparation.ID || subject.Delivery == nil || len(prompt) == 0 {
				t.Fatal("missing delivery scope")
			}
			deps := fx.deps()
			deps.MCPKey = []byte("test-delivery-key-32-bytes-long!!")
			deps.ContextVerification = c.ApplicationVerificationEvidence
			deps.Priority = &PriorityJudgments{Journal: db, Chooser: &bundleGapChooser{pick: factory.ContextBundleCurrent}}
			occupancy, err := snapshotOccupancy(ctx, db)
			if err != nil {
				t.Fatal(err)
			}
			report := newDispatchReport()
			fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
				a, err := db.AssignmentByRun(ctx, in.Run.ID)
				if err != nil || a.Role != project.RoleReviewer || a.Phase != project.ExecutionVerify || in.MCP == nil || in.Run.ExecutionPhase != project.ExecutionVerify {
					t.Fatalf("unbound Test Pilot: role=%s phase=%s mcp=%t error=%v", a.Role, a.Phase, in.MCP != nil, err)
				}
				var data []byte
				if err = sqlDB.QueryRow(`SELECT data FROM factory_attempt_allowances WHERE root_assignment=$1`, a.ID).Scan(&data); err != nil {
					t.Fatal(err)
				}
				var allowance factory.AttemptAllowance
				if err = json.Unmarshal(data, &allowance); err != nil || allowance.Delivery == nil || allowance.Delivery.DeliveryID != subject.Delivery.DeliveryID {
					t.Fatalf("delivery unfunded: %+v %v", allowance, err)
				}
				return project.FactoryState{ID: in.Run.ID, Project: in.Run.Project, Role: in.Run.Role, Phase: project.FactoryCompleted}, nil
			}
			DispatchTimeOnlyRoot(ctx, deps, &occupancy, 7, subject, fx.tip, prompt, preparation, &report)
			if len(report.Launched) != 1 || len(report.Errors) != 0 || len(report.Waits) != 0 {
				t.Fatalf("verification dispatch: %+v", report)
			}
			seen, err := db.DeliveryAlreadyAssigned(ctx, 7, subject.Delivery.DeliveryID)
			if err != nil || !seen {
				t.Fatal("delivery replay is unbound", err)
			}
			// The normal current-format reopen must preserve the new packet and its allowance.
			if deadline, err := db.CapAttemptDeadline(ctx, report.Launched[0].AssignmentID, time.Now().Add(time.Minute).Unix()); err != nil || deadline <= time.Now().Unix() {
				t.Fatal("delivery lifetime is unavailable", err)
			}
			a, err := db.Assignment(ctx, report.Launched[0].AssignmentID)
			if err != nil {
				t.Fatal(err)
			}
			run, err := db.FactoryRun(ctx, a.Run)
			if err != nil {
				t.Fatal(err)
			}
			if early {
				capability, err := db.GetMCPCapability(ctx, run.ID)
				if err != nil {
					t.Fatal(err)
				}
				if err := db.ActivateObservedMCPCapability(ctx, capability); err != nil {
					t.Fatal(err)
				}
				capability, err = db.GetMCPCapability(ctx, run.ID)
				if err != nil {
					t.Fatal(err)
				}
				submission := testMCPSubmission(capability, factory.MCPToolFinishIncomplete, factory.NewID(), []byte(`{"outcome":"blocked","cause":"Application evidence is incomplete."}`))
				if _, err := db.RecordMCPSubmission(ctx, submission, factory.MCPReceipt{SubmissionID: submission.SubmissionID, Tool: submission.Tool, State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash}); err != nil {
					t.Fatal(err)
				}
				if settled, err := SettleMCPTerminalProposal(ctx, db, a.ID, time.Now()); err != nil || !settled {
					t.Fatal("early Test Pilot terminal did not settle", settled, err)
				}
				var before factory.AttemptAllowance
				var raw []byte
				if err := sqlDB.QueryRow(`SELECT data FROM factory_attempt_allowances WHERE root_assignment=$1`, a.ID).Scan(&raw); err != nil || json.Unmarshal(raw, &before) != nil || !before.Active || !before.Closed {
					t.Fatal("early Test Pilot terminal must keep its actual native clock", err)
				}
				run.Outcome, run.Reconciled = factory.Succeeded, true
				if err := db.SaveFactoryRun(ctx, run); err != nil {
					t.Fatal("retired Test Pilot native outcome failed to save", err)
				}
				var after factory.AttemptAllowance
				if err := sqlDB.QueryRow(`SELECT data FROM factory_attempt_allowances WHERE root_assignment=$1`, a.ID).Scan(&raw); err != nil || json.Unmarshal(raw, &after) != nil || after.Active || !after.Closed || after.Limits != before.Limits || after.ActiveSeconds < before.ActiveSeconds {
					t.Fatal("retired Test Pilot did not checkpoint the same closed allowance", err)
				}
				finished, err := db.Assignment(ctx, a.ID)
				if err != nil || finished.Outcome != factory.NeedsHuman || finished.Result == nil || finished.Result.Status != "blocked" {
					t.Fatal("native retirement changed the recorded verification result", err)
				}
				if err := db.SaveFactoryRun(ctx, run); err != nil {
					t.Fatal("native Test Pilot save replay failed", err)
				}
				var replayed factory.AttemptAllowance
				if err := sqlDB.QueryRow(`SELECT data FROM factory_attempt_allowances WHERE root_assignment=$1`, a.ID).Scan(&raw); err != nil || json.Unmarshal(raw, &replayed) != nil || replayed.Revision != after.Revision || replayed.ActiveSeconds != after.ActiveSeconds {
					t.Fatal("native Test Pilot replay charged or reopened its root", err)
				}
				return
			}
			run.Outcome, run.Reconciled = factory.Succeeded, true
			if err = db.SaveFactoryRun(ctx, run); err != nil {
				t.Fatal(err)
			}
			verification := factory.AppVerificationReport{DeliveryID: subject.Delivery.DeliveryID, InputDigest: strings.Repeat("9", 64), Outcome: factory.AppVerificationVerified, CheckReceipts: []factory.VerificationCheckReceipt{{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: factory.MCPInputDigest([]byte("task storage passed")), LogExcerpt: "Native assertions passed"}}, Coverage: []factory.ReassessmentCoverage{{IntentID: revision.Intent, OutcomeID: idea.ID, TaskKeys: []string{task.Key}, DeliveredAcceptanceIDs: []string{binding.AcceptanceID}}}}
			a.Stage, a.Outcome, a.Reason, a.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, time.Now().Unix()
			a.Result = &factory.AssignmentResult{AssignmentID: a.ID, RunID: a.Run, Phase: project.ExecutionVerify, Status: "completed", Summary: "Application assertions passed", Reported: true, RecordedUnix: time.Now().Unix(), PhaseResult: &factory.PhaseResult{Version: 2, Phase: project.ExecutionVerify, Status: "completed", Summary: "Application assertions passed", Output: &factory.PhaseOutput{Verification: &verification}}}
			if err = db.FinishAssignment(ctx, a); err != nil {
				t.Fatal("delivery settlement", err)
			}
			var closed bool
			if err = sqlDB.QueryRow(`SELECT (data->>'closed')::boolean FROM factory_attempt_allowances WHERE root_assignment=$1`, a.ID).Scan(&closed); err != nil || !closed {
				t.Fatal("delivery allowance remained open", err)
			}
			reopened, err := store.Open(dsn)
			if err != nil {
				t.Fatal("current-format reopen", err)
			}
			defer reopened.Close()
			persisted, err := reopened.Assignment(ctx, a.ID)
			if err != nil || persisted.Stage != factory.AssignmentFinished {
				t.Fatal("startup extension lost retained delivery", err)
			}
		})
	}
}
