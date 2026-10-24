package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestDeliveredIntentAdmitsScopedTestPilotOnce(t *testing.T) {
	for _, fixture := range []struct {
		early  bool
		state  string
		failed bool
	}{{false, factory.ReassessmentCollecting, false}, {true, factory.ReassessmentCollecting, false}, {false, factory.ReassessmentAttention, false}, {true, factory.ReassessmentAttention, false}, {true, factory.ReassessmentCollecting, true}, {true, factory.ReassessmentAttention, true}} {
		early := fixture.early
		t.Run(fmt.Sprintf("%s/early-terminal-%t/reported-failed-%t", fixture.state, early, fixture.failed), func(t *testing.T) {
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
			// A real merge also seeds its reassessment cycle. Verification of the
			// delivered application must not require changing that independent
			// cycle or repeating its already-recorded judgment to open the fence.
			cycle := factory.ReassessmentCycle{CycleID: factory.NewID(), EventID: event.ID, Repository: 7, Generation: 1, State: fixture.state}
			if fixture.state == factory.ReassessmentAttention {
				cycle.Decision = factory.ReassessmentOptionContinue
				cycle.Ledger.DecisionCalls = []string{factory.NewID()}
			}
			reassessment := store.ReassessmentRecord{Cycle: cycle, Completion: event, Issue: event.Issue}
			if fixture.state == factory.ReassessmentAttention {
				reassessment.WaitReason = "decision_not_actionable"
			}
			if err = reassessment.Validate(); err != nil {
				t.Fatal(err)
			}
			raw, _ = json.Marshal(reassessment)
			if _, err = sqlDB.Exec(`INSERT INTO factory_reassessments(cycle,repository,generation,event,state,data) VALUES($1,7,1,$2,$3,$4)`, cycle.CycleID, event.ID, cycle.State, string(raw)); err != nil {
				t.Fatal(err)
			}
			var originalCycle string
			if err = sqlDB.QueryRow(`SELECT data::text FROM factory_reassessments WHERE cycle=$1`, cycle.CycleID).Scan(&originalCycle); err != nil {
				t.Fatal(err)
			}
			if pending, err := db.CompletionPending(ctx, 7); err != nil || len(pending) != 1 || pending[0] != 1 {
				t.Fatal("real completion must keep its fence closed", pending, err)
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
			for _, invalid := range []struct {
				name string
				edit func(*factory.DeliverySubject)
			}{
				{"foreign-intent", func(s *factory.DeliverySubject) { s.IntentID = phasedDecisionID(61) }},
				{"stale-accepted-head", func(s *factory.DeliverySubject) { s.AcceptedRevision = phasedDecisionID(62) }},
				{"changed-accepted-scope", func(s *factory.DeliverySubject) { s.ScopeDigest = strings.Repeat("e", 64) }},
				{"changed-ready-preparation", func(s *factory.DeliverySubject) { s.PreparationDigest = strings.Repeat("d", 64) }},
			} {
				wrong := subject
				payload := *subject.Delivery
				wrong.Delivery = &payload
				invalid.edit(wrong.Delivery)
				refused := newDispatchReport()
				DispatchTimeOnlyRoot(ctx, deps, &occupancy, 7, wrong, fx.tip, prompt, preparation, &refused)
				if len(refused.Launched) != 0 || len(refused.Errors)+len(refused.Waits) == 0 {
					t.Fatalf("%s admitted an unbound verifier: %+v", invalid.name, refused)
				}
				var recorded int
				if err = sqlDB.QueryRow(`SELECT count(*) FROM factory_assignments WHERE data->>'phase'='verify'`).Scan(&recorded); err != nil || recorded != 0 {
					t.Fatalf("%s wrote a verification packet: %d %v", invalid.name, recorded, err)
				}
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
			var retainedCycle string
			if err = sqlDB.QueryRow(`SELECT data::text FROM factory_reassessments WHERE cycle=$1`, cycle.CycleID).Scan(&retainedCycle); err != nil || retainedCycle != originalCycle {
				t.Fatal("verification changed the completion cycle or its judgment", err)
			}
			if pending, err := db.CompletionPending(ctx, 7); err != nil || len(pending) != 1 || pending[0] != 1 {
				t.Fatal("verification released the independent completion fence", pending, err)
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
				if fixture.failed {
					verification := factory.AppVerificationReport{DeliveryID: subject.Delivery.DeliveryID, InputDigest: strings.Repeat("9", 64), Outcome: factory.AppVerificationFailed, CheckReceipts: []factory.VerificationCheckReceipt{{CheckID: factory.ApprovedAppCheckID, Outcome: factory.CheckReceiptPassed, ArtifactDigest: factory.MCPInputDigest([]byte("task storage passed")), LogExcerpt: "Native assertions passed"}}, Failures: []factory.CheckFailure{{CheckRef: idea.ID, Detail: "The approved assertions omit required task behavior."}}}
					a.Stage, a.Outcome, a.Reason, a.FinishedUnix = factory.AssignmentFinished, factory.Succeeded, factory.AssignReasonReported, time.Now().Unix()
					a.Result = &factory.AssignmentResult{AssignmentID: a.ID, RunID: a.Run, Phase: project.ExecutionVerify, Status: "completed", Summary: "Application coverage failed", Reported: true, RecordedUnix: time.Now().Unix(), PhaseResult: &factory.PhaseResult{Version: 2, Phase: project.ExecutionVerify, Status: "completed", Summary: "Application coverage failed", Output: &factory.PhaseOutput{Verification: &verification}}}
					if err := db.FinishAssignment(ctx, a); err != nil {
						t.Fatal("reported application failure", err)
					}
				} else {
					submission := testMCPSubmission(capability, factory.MCPToolFinishIncomplete, factory.NewID(), []byte(`{"outcome":"blocked","cause":"Application evidence is incomplete."}`))
					if _, err := db.RecordMCPSubmission(ctx, submission, factory.MCPReceipt{SubmissionID: submission.SubmissionID, Tool: submission.Tool, State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash}); err != nil {
						t.Fatal(err)
					}
					if settled, err := SettleMCPTerminalProposal(ctx, db, a.ID, time.Now()); err != nil || !settled {
						t.Fatal("early Test Pilot terminal did not settle", settled, err)
					}
				}
				if _, err := c.RetryRun(ctx, factory.NewID(), "native:7", run.ID); err == nil {
					t.Fatal("verification Retry admitted before native retirement")
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
				wantOutcome, wantStatus := factory.NeedsHuman, "blocked"
				if fixture.failed {
					wantOutcome, wantStatus = factory.Succeeded, "completed"
				}
				if err != nil || finished.Outcome != wantOutcome || finished.Result == nil || finished.Result.Status != wantStatus {
					t.Fatal("native retirement changed the recorded verification result", err)
				}
				if err := db.SaveFactoryRun(ctx, run); err != nil {
					t.Fatal("native Test Pilot save replay failed", err)
				}
				var replayed factory.AttemptAllowance
				if err := sqlDB.QueryRow(`SELECT data FROM factory_attempt_allowances WHERE root_assignment=$1`, a.ID).Scan(&raw); err != nil || json.Unmarshal(raw, &replayed) != nil || replayed.Revision != after.Revision || replayed.ActiveSeconds != after.ActiveSeconds {
					t.Fatal("native Test Pilot replay charged or reopened its root", err)
				}
				reservation, err := db.Reservation(ctx, a.ID)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = db.RunUsage(ctx, run.ID); errors.Is(err, store.ErrNotFound) && reservation.State != factory.ReservationReleased {
					if err = db.RecordRunUsage(ctx, factory.Usage{RunID: run.ID, Repository: 7, Connection: a.Connection, Minutes: 1, StartedAt: run.Started, EndedAt: run.Started.Add(time.Millisecond)}); err != nil {
						t.Fatal(err)
					}
				} else if err != nil && !errors.Is(err, store.ErrNotFound) {
					t.Fatal(err)
				}
				if reservation.State == factory.ReservationHeld {
					if err = db.ConsumeReservation(ctx, a.ID); err != nil {
						t.Fatal(err)
					}
				}
				retained := func() string {
					var rows string
					if err := sqlDB.QueryRow(`SELECT jsonb_build_object(
						'assignment',(SELECT to_jsonb(x) FROM factory_assignments x WHERE id=$1),
						'run',(SELECT to_jsonb(x) FROM factory_runs x WHERE id=$2),
						'allowance',(SELECT to_jsonb(x) FROM factory_attempt_allowances x WHERE root_assignment=$1),
						'reservation',(SELECT to_jsonb(x) FROM factory_reservations x WHERE assignment=$1),
						'usage',(SELECT to_jsonb(x) FROM factory_usage x WHERE run=$2),
						'cycle',(SELECT to_jsonb(x) FROM factory_reassessments x WHERE cycle=$3))::text`, a.ID, run.ID, cycle.CycleID).Scan(&rows); err != nil {
						t.Fatal(err)
					}
					return rows
				}
				oldRows := retained()
				if available, err := db.DeliveryRetryAvailable(ctx, run.ID); err != nil || !available {
					t.Fatal("retired verifier Retry unavailable", available, err)
				}
				c.DispatchReads = &appVerificationReads{tip: strings.Repeat("e", 40)}
				if available, err := c.DeliveryRetryAvailable(ctx, run.ID); err != nil || available {
					t.Fatal("old-source verifier advertised Retry", available, err)
				}
				if _, err := c.RetryRun(ctx, factory.NewID(), "native:7", run.ID); !errors.Is(err, store.ErrExplicitRetryIneligible) {
					t.Fatal("old-source verifier Retry queued", err)
				}
				c.DispatchReads = &appVerificationReads{tip: fx.tip}
				if available, err := c.DeliveryRetryAvailable(ctx, run.ID); err != nil || !available {
					t.Fatal("current-source verifier Retry unavailable", available, err)
				}
				unowned := subject
				unownedPayload := *subject.Delivery
				unownedPayload.DeliveryID = factory.NewID()
				unowned.Delivery = &unownedPayload
				unownedOccupancy, err := snapshotOccupancy(ctx, db)
				if err != nil {
					t.Fatal(err)
				}
				unownedReport := newDispatchReport()
				DispatchTimeOnlyRoot(ctx, deps, &unownedOccupancy, 7, unowned, fx.tip, prompt, preparation, &unownedReport)
				if len(unownedReport.Launched) != 0 || len(unownedReport.Waits)+len(unownedReport.Errors) == 0 {
					t.Fatal("verification polled into an unrequested retry")
				}
				command := factory.NewID()
				decision, err := c.RetryRun(ctx, command, "native:7", run.ID)
				if err != nil || !decision.Queued {
					t.Fatal("explicit verifier Retry refused", decision, err)
				}
				if replay, err := c.RetryRun(ctx, command, "native:7", run.ID); err != nil || replay != decision {
					t.Fatal("verifier Retry lost-reply recovery", replay, err)
				}
				if _, err := c.RetryRun(ctx, factory.NewID(), "native:7", run.ID); !errors.Is(err, store.ErrExplicitRetryIneligible) {
					t.Fatal("duplicate queued verifier Retry", err)
				}
				if retained() != oldRows {
					t.Fatal("queuing verification Retry changed old result, spending or completion decision")
				}
				if available, err := db.DeliveryRetryAvailable(ctx, run.ID); err != nil || available {
					t.Fatal("queued verifier Retry still advertised", available, err)
				}
				pending, err := db.PendingDeliveryRetry(ctx, 7, *subject.Delivery)
				if err != nil || pending != decision {
					t.Fatal("verification Retry was not selected", pending, err)
				}
				foreign := *subject.Delivery
				foreign.ScopeDigest = strings.Repeat("f", 64)
				if _, err := db.PendingDeliveryRetry(ctx, 7, foreign); !errors.Is(err, store.ErrNotFound) {
					t.Fatal("foreign version selected verification Retry", err)
				}
				payload := *subject.Delivery
				payload.DeliveryID = factory.DeliveryRetryID(payload, command)
				subject = factory.ExecutionSubject{Kind: factory.SubjectDelivery, Delivery: &payload}
				retryPrompt, err := c.deliveryRetryPrompt(ctx, candidates[0], subject, preparation)
				if err != nil || !strings.Contains(string(retryPrompt), payload.DeliveryID) {
					t.Fatal("verification Retry prompt is unbound", err)
				}
				occupancy, err = snapshotOccupancy(ctx, db)
				if err != nil {
					t.Fatal(err)
				}
				retryReport := newDispatchReport()
				dispatchTimeOnlyRetry(ctx, deps, &occupancy, 7, subject, fx.tip, retryPrompt, preparation, &decision, &retryReport)
				if len(retryReport.Launched) != 1 || len(retryReport.Errors) != 0 || len(retryReport.Waits) != 0 || retryReport.Launched[0].AssignmentID != command {
					t.Fatalf("explicit verifier successor: %+v", retryReport)
				}
				if retained() != oldRows {
					t.Fatal("new verifier packet rewrote the old incomplete result or budget")
				}
				if replay, err := c.RetryRun(ctx, command, "native:7", run.ID); err != nil || replay != decision {
					t.Fatal("consumed verifier Retry lost its immutable receipt", replay, err)
				}
				if _, err := db.PendingDeliveryRetry(ctx, 7, payload); !errors.Is(err, store.ErrNotFound) {
					t.Fatal("consumed verifier Retry stayed pending", err)
				}
				repeated := newDispatchReport()
				dispatchTimeOnlyRetry(ctx, deps, &occupancy, 7, subject, fx.tip, retryPrompt, preparation, &decision, &repeated)
				if len(repeated.Launched) != 0 {
					t.Fatal("verifier Retry launched twice")
				}
				if _, err := c.RetryRun(ctx, factory.NewID(), "native:7", run.ID); !errors.Is(err, store.ErrExplicitRetryIneligible) {
					t.Fatal("old verifier remained retryable after successor", err)
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
			if available, err := db.DeliveryRetryAvailable(ctx, run.ID); err != nil || available {
				t.Fatal("verified delivery advertised Retry", available, err)
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
