package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	sdk "forgejo.org/extension-sdk"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// TestServeStallRoundAdmitsBoundedEpisode proves the R36 production
// joint on the real store path: a no-launch dispatch pass runs
// detection at the standing generation, persists the bound episode
// durably, and admits exactly one cycle-1 charge under the explicit
// grant. A redetected round replays without spending; a pass that
// launched work skips detection entirely.
func TestServeStallRoundAdmitsBoundedEpisode(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	coord := &Coordinator{Store: db}
	const repo = 44
	grantID := "d1000000000000000000000000000044"
	grant := factory.RepairGrant{
		ID: grantID, Repository: repo, Sponsor: 7, Revision: 0,
		InvestigatorRuns: 4, ProposalJudgments: 4, ActiveMinutes: 120,
		Active: true, CreatedUnix: 1000,
	}
	raw, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	cmd := factory.Command{
		ID: grantID, Type: factory.CommandRepairGrant, Target: factory.RepairTarget(repo),
		Principal: "soda-tester", Payload: string(raw),
	}
	cmd.Digest = factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload)
	if _, created, err := db.ApplyRepairGrantCommand(ctx, cmd, grant, time.Now().UTC()); err != nil || !created {
		t.Fatalf("apply grant: created=%v err=%v", created, err)
	}
	stallSeedControl(t, db, repo, 1, factory.ReadinessBlocked,
		factory.BlockerCycle, stallBlocker(factory.BlockerCycle, 0, 0))

	round, err := coord.ServeStallRound(ctx, DispatchReport{})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	if len(round.Repositories) != 1 || round.Repositories[0].Kind != StallRepositoryStall {
		t.Fatalf("round=%+v, want one bound stall repository", round)
	}
	standing, err := db.StandingStallEvents(ctx)
	if err != nil {
		t.Fatalf("standing: %v", err)
	}
	entry, ok := standing[repo]
	if !ok || entry.Generation != 0 || entry.EventID == "" || entry.Digest == "" {
		t.Fatalf("standing=%+v, want the durable generation-0 observation", standing)
	}
	if charges, err := db.RepairCharges(ctx, repo); err != nil || len(charges) != 1 {
		t.Fatalf("charges=%d err=%v, want exactly one admitted cycle", len(charges), err)
	}

	again, err := coord.ServeStallRound(ctx, DispatchReport{})
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if len(again.Repositories) != 1 {
		t.Fatalf("round=%+v, want the replayed episode", again)
	}
	if charges, err := db.RepairCharges(ctx, repo); err != nil || len(charges) != 1 {
		t.Fatalf("charges=%d err=%v, want redetection to replay without spending", len(charges), err)
	}

	skipped, err := coord.ServeStallRound(ctx, DispatchReport{
		Launched: []DispatchLaunch{{Repository: repo, Issue: 9}},
	})
	if err != nil {
		t.Fatalf("skip: %v", err)
	}
	if len(skipped.Repositories) != 0 {
		t.Fatalf("round=%+v, want no detection when the pass launched work", skipped)
	}
	t.Run("retained Strategist launch and required context", func(t *testing.T) {
		serving, dsn := dispatchTestDB(t)
		fx := dispatchSeed(t, serving)
		fx.accept(t, 3, "d333333333333333333333333")
		stallSeedControl(t, serving, fx.repo, 3, factory.ReadinessBlocked, factory.BlockerCycle, stallBlocker(factory.BlockerCycle, 0, 0))
		grant := factory.DefaultRepairGrant()
		grant.ID, grant.Repository, grant.Sponsor, grant.Active, grant.CreatedUnix = factory.NewID(), fx.repo, 7, true, time.Now().Unix()
		encoded, _ := json.Marshal(grant)
		command := factory.Command{ID: grant.ID, Type: factory.CommandRepairGrant, Target: factory.RepairTarget(fx.repo), Principal: "soda-tester", Payload: string(encoded)}
		command.Digest = factory.SettingsDigest(command.Type, command.Target, command.Payload)
		if _, created, err := serving.ApplyRepairGrantCommand(ctx, command, grant, time.Now()); err != nil || !created {
			t.Fatal("repair grant", err)
		}
		enumerator := &fixtureEnumerator{revision: 41, idle: true, pages: map[string]NativeInventoryPage{"": {Issues: []NativeInventoryIssue{inventoryIssue(t, 3)}, Complete: true}}}
		worker := &Coordinator{Store: serving, Host: fx.host, Broker: fx.broker, DispatchReads: fx.reads, CompletionEnumerator: enumerator, CompletionLocalCapture: serving.CaptureCompletionLocal, MCPKey: []byte(strings.Repeat("k", 32)), Priority: &PriorityJudgments{Journal: serving, Chooser: &bundleGapChooser{pick: factory.ContextBundleCurrent}}}
		first, err := worker.ServeStallRound(ctx, DispatchReport{})
		if err != nil || len(first.Repositories) != 1 || first.Repositories[0].Event == nil {
			t.Fatal("bound repair episode", err)
		}
		event := *first.Repositories[0].Event
		record, err := serving.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
		if err != nil || len(record.Repair.Cycles) != 1 || record.Repair.Cycles[0].State != factory.RepairReady {
			t.Fatalf("repair input not sealed: %+v %v", record.Repair, err)
		}
		if _, err = worker.ServeStallRound(ctx, DispatchReport{}); err != nil {
			t.Fatal(err)
		}
		record, err = serving.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
		if err != nil || record.Repair.Cycles[0].State != factory.RepairInvestigating {
			t.Fatalf("repair did not retain sole investigator: state=%s reason=%s %v", record.Repair.Cycles[0].State, record.Repair.Cycles[0].Reason, err)
		}
		cycle := record.Repair.Cycles[0]
		if len(fx.host.launches) != 1 || fx.host.launches[0].Run.ID != cycle.RunID || fx.host.launches[0].Run.ExecutionPhase != project.ExecutionRepair || fx.host.launches[0].MCP == nil {
			t.Fatalf("repair native launch missing or unbound: launches=%d cycle=%+v", len(fx.host.launches), cycle)
		}
		assignment, err := serving.Assignment(ctx, cycle.AssignmentID)
		if err != nil || assignment.Subject.Stall == nil || assignment.Subject.Stall.CycleID != cycle.ID || assignment.Run != cycle.RunID {
			t.Fatal("repair assignment lost cycle", err)
		}
		capability, err := serving.GetMCPCapability(ctx, cycle.RunID)
		if err != nil || capability.Profile != factory.MCPProfileStrategist {
			t.Fatal("repair MCP profile", err)
		}
		current, err := serving.FactoryCurrentContext(ctx, capability, time.Now())
		if err != nil || current.Evidence.Availability != "complete" || current.Role.Repair == nil || !current.Role.Repair.Complete || current.Role.Repair.InventoryDigest != cycle.InventoryDigest || current.Allowance == nil || current.Allowance.MaxChildren != 0 {
			t.Fatalf("repair current input incomplete: %+v %v", current.Evidence, err)
		}
		if current.Role.Repair.Budget == nil || current.Role.Repair.Budget.RunsRemaining != 3 || current.Role.Repair.Budget.ActiveMinutesRemaining != 90 || current.Role.Repair.Budget.CyclesRemaining != 1 || current.Role.Repair.State != factory.RepairInvestigating {
			t.Fatal("repair current context lost canonical consumed grant budget")
		}
		for i := 0; i < 2; i++ {
			if _, err = worker.ServeStallRound(ctx, DispatchReport{}); err != nil {
				t.Fatal(err)
			}
		}
		charges, err := serving.RepairCharges(ctx, fx.repo)
		if err != nil || len(charges) != 1 || len(fx.host.launches) != 1 {
			t.Fatalf("redetection refilled repair: charges=%d launches=%d %v", len(charges), len(fx.host.launches), err)
		}
		run, err := serving.FactoryRun(ctx, cycle.RunID)
		if err != nil {
			t.Fatal(err)
		}
		run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "repair no-change result", true
		if err = serving.SaveFactoryRun(ctx, run); err != nil {
			t.Fatal(err)
		}
		proposal := factory.StallRepairReport{StallID: record.ID, CycleID: cycle.ID, InventoryDigest: cycle.InventoryDigest, SourceCommit: cycle.Local.SourceCommit, PreparationDigest: cycle.Local.PreparationDigest, Causes: []factory.CheckFailure{{CheckRef: "dependency_cycle", Detail: "The recorded dependency cycle still has no viable producer."}}, RetainedIssueIndexes: []int64{3}, Outcome: factory.RepairOutcomeNoChange}
		output := fencedResult(t, factory.PhaseResult{Version: factory.PhaseResultVersion, Phase: project.ExecutionRepair, Status: "completed", Summary: "No graph change proposed; the real stall remains.", Output: &factory.PhaseOutput{Repair: &proposal}})
		if a, ok := AccountSettledRun(ctx, serving, run, output, time.Now()); !ok || a.Outcome != factory.Succeeded {
			t.Fatalf("repair typed finish refused: %s %s %v", a.Outcome, a.Reason, ok)
		}
		for i := 0; i < 3; i++ {
			_, _ = worker.ServeStallRound(ctx, DispatchReport{})
		}
		record, err = serving.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
		if err != nil || len(record.Repair.Cycles) != 1 || record.Repair.Cycles[0].State != factory.RepairAttention {
			t.Fatalf("no-change cleared or refilled real stall: cycles=%d state=%s %v", len(record.Repair.Cycles), record.Repair.Cycles[0].State, err)
		}
		// A native revision or replaced issue identity alone does not supply facts.
		enumerator.revision++
		_, _ = worker.ServeStallRound(ctx, DispatchReport{})
		if charges, _ = serving.RepairCharges(ctx, fx.repo); len(charges) != 1 {
			t.Fatal("native revision refilled repair")
		}
		changed := factory.CompletionIssueEvidence{Index: 3, Title: "New manual factual evidence", Body: "The missing producer can now be traced to a distinct external contract.", TitleDigest: evidenceDigest("New manual factual evidence"), ContentDigest: evidenceDigest("The missing producer can now be traced to a distinct external contract.")}
		body, _ := changed.CanonicalData()
		enumerator.revision++
		enumerator.pages[""] = NativeInventoryPage{Issues: []NativeInventoryIssue{{Index: 3, Data: body}}, Complete: true}
		_, _ = worker.ServeStallRound(ctx, DispatchReport{})
		record, err = serving.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
		if err != nil || len(record.Repair.Cycles) != 2 || record.Repair.Cycles[1].State != factory.RepairReady {
			t.Fatalf("material new evidence did not admit bounded next cycle: cycles=%d %v", len(record.Repair.Cycles), err)
		}
		if charges, _ = serving.RepairCharges(ctx, fx.repo); len(charges) != 2 {
			t.Fatal("new evidence did not retain cumulative charges")
		}
		_, _ = worker.ServeStallRound(ctx, DispatchReport{})
		record, err = serving.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
		if err != nil {
			t.Fatal(err)
		}
		second := record.Repair.Cycles[1]
		if second.State != factory.RepairInvestigating || len(fx.host.launches) != 2 {
			t.Fatalf("second bounded cycle did not launch: %s %s", second.State, second.Reason)
		}
		run, err = serving.FactoryRun(ctx, second.RunID)
		if err != nil {
			t.Fatal(err)
		}
		// A structured terminal may be fenced before native retirement.
		// Its exact typed root closes admission but keeps its active clock
		// until SaveFactoryRun records the actual stopped worker.
		secondCapability, err := serving.GetMCPCapability(ctx, second.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if err = serving.ActivateObservedMCPCapability(ctx, secondCapability); err != nil {
			t.Fatal(err)
		}
		secondCapability, err = serving.GetMCPCapability(ctx, second.RunID)
		if err != nil {
			t.Fatal(err)
		}
		invalidReport, _ := json.Marshal(map[string]any{"subject": secondCapability.Subject, "report": map[string]any{}})
		submission := testMCPSubmission(secondCapability, factory.MCPToolFinishReassessment, factory.NewID(), invalidReport)
		receipt := factory.MCPReceipt{SubmissionID: submission.SubmissionID, Tool: submission.Tool, State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash}
		if _, err = serving.RecordMCPSubmission(ctx, submission, receipt); err != nil {
			t.Fatal(err)
		}
		if finished, err := SettleMCPTerminalProposal(ctx, serving, second.AssignmentID, time.Now()); err != nil || !finished {
			t.Fatal("early fenced Strategist terminal", finished, err)
		}
		sqlDB, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Fatal(err)
		}
		defer sqlDB.Close()
		var beforeAllowance, afterAllowance factory.AttemptAllowance
		var allowanceRaw []byte
		if err = sqlDB.QueryRowContext(ctx, `SELECT data FROM factory_attempt_allowances WHERE root_assignment=$1`, second.AssignmentID).Scan(&allowanceRaw); err != nil || json.Unmarshal(allowanceRaw, &beforeAllowance) != nil || !beforeAllowance.Active || !beforeAllowance.Closed {
			t.Fatal("fenced root must retain active clock before native retirement", err)
		}
		run.Outcome, run.Reconciled = factory.Succeeded, true
		if err = serving.SaveFactoryRun(ctx, run); err != nil {
			t.Fatal(err)
		}
		if err = sqlDB.QueryRowContext(ctx, `SELECT data FROM factory_attempt_allowances WHERE root_assignment=$1`, second.AssignmentID).Scan(&allowanceRaw); err != nil || json.Unmarshal(allowanceRaw, &afterAllowance) != nil || afterAllowance.Active || !afterAllowance.Closed || afterAllowance.Limits != beforeAllowance.Limits || afterAllowance.ActiveSeconds < beforeAllowance.ActiveSeconds {
			t.Fatal("retired typed root did not freeze the same cumulative allowance", err)
		}
		if _, created := AccountSettledRun(ctx, serving, run, "", time.Now()); created {
			t.Fatal("retirement replaced an already fenced terminal")
		}
		finishedSecond, err := serving.Assignment(ctx, second.AssignmentID)
		if err != nil || finishedSecond.Stage != factory.AssignmentFinished || finishedSecond.Outcome != factory.NeedsHuman {
			t.Fatal("original fenced terminal changed after native settlement", err)
		}
		for i := 0; i < 3; i++ {
			_, _ = worker.ServeStallRound(ctx, DispatchReport{})
		}
		changed.Body += " A further external fact cannot buy a third automatic run."
		changed.ContentDigest = evidenceDigest(changed.Body)
		body, _ = changed.CanonicalData()
		enumerator.revision++
		enumerator.pages[""] = NativeInventoryPage{Issues: []NativeInventoryIssue{{Index: 3, Data: body}}, Complete: true}
		_, _ = worker.ServeStallRound(ctx, DispatchReport{})
		record, err = serving.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
		if err != nil || len(record.Repair.Cycles) != 2 || record.Repair.Cycles[1].State != factory.RepairAttention {
			t.Fatal("two unsuccessful cycles did not retain owner attention", err)
		}
		if charges, _ = serving.RepairCharges(ctx, fx.repo); len(charges) != 2 || len(fx.host.launches) != 2 {
			t.Fatal("new facts refilled exhausted automatic cycles")
		}
		ready, err := serving.IssueControl(ctx, fx.repo, 3)
		if err != nil {
			t.Fatal(err)
		}
		ready.Readiness, ready.Reason, ready.Blockers, ready.Fingerprint = factory.ReadinessQueued, factory.ReasonEligible, nil, strings.Repeat("9", 64)
		if _, _, err = serving.RecordIssueAssessment(ctx, ready, time.Now()); err != nil {
			t.Fatal(err)
		}
		_, _ = worker.ServeStallRound(ctx, DispatchReport{})
		record, err = serving.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
		if err != nil || record.Repair.Cycles[1].State != factory.RepairStale || len(fx.host.launches) != 2 {
			t.Fatal("ready work did not retire obsolete diagnosis", err)
		}
	})
	t.Run("protected repair adoption recovers native lost reply", stallRepairGraphIntegration)
}

func stallRepairGraphIntegration(t *testing.T) {
	for _, mode := range []string{"lost_reply", "grant_hold_new_effect", "grant_withdrawn_after_admission", "append_producer"} {
		t.Run(mode, func(t *testing.T) { stallRepairGraphMode(t, mode) })
	}
}

func stallRepairGraphMode(t *testing.T, mode string) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	graph := &generatedGraphFixture{revision: 41, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
	coord := NewCoordinator(db, fx.host, fx.broker)
	coord.DispatchReads = fx.reads
	coord.Graph, coord.GraphActorID, coord.AcceptanceReads = graph, graph.actor, graph
	revision := broadJoinAcceptedRevision()
	revision.Repository, revision.Tasks, revision.Ideas = fx.repo, revision.Tasks[:2], revision.Ideas[:1]
	if err := db.SaveIntentRevision(ctx, revision); err != nil {
		t.Fatal(err)
	}
	if generation, err := generateChoiceMechanics(t, coord, ctx, revision.Intent, revision.Owner, chosenFor(revision)); err != nil || generation.State != "completed" {
		t.Fatalf("initial accepted graph: state=%s %v", generation.State, err)
	}
	initial, err := db.IntentTaskBindings(ctx, revision.ID)
	if err != nil || len(initial) != 2 {
		t.Fatal("initial bindings", err)
	}
	graph.issues[2].Issue.Content += "\nRetained human note."
	graph.issues[2].Issue.ContentDigest = evidenceDigest(graph.issues[2].Issue.Content)
	graph.issues[2].Issue.ContentVersion++
	graph.revision++
	protectedBody := graph.issues[2].Issue.Content
	protectedHeads := map[int64]string{}
	var protectedDependencies []byte
	if mode == "append_producer" {
		manual := &sdk.SnapshotIssue{ID: "103", Index: "3", Title: "Supply the requested fixture", Content: "A maintainer must accept the exact fixture result.", Visible: true, Complete: true, Provenance: sdk.SnapshotCreationProvenance{PosterID: "8"}}
		manual.TitleDigest, manual.ContentDigest = evidenceDigest(manual.Title), evidenceDigest(manual.Content)
		graph.issues[3] = sdk.NativeSnapshot{Issue: manual, Dependencies: &sdk.SnapshotDependencyPage{IssueID: "103", Complete: true, Items: []sdk.SnapshotDependency{}}}
		// These ordinary human edges and their readopted result requirements
		// protect both existing consumers. The missing manual producer has no
		// factory acceptance; adoption may add work but cannot claim its result.
		for _, binding := range initial {
			index := binding.IssueIndex
			endpoint, native := int64(2), "102"
			if index == 2 {
				endpoint, native = 3, "103"
			}
			snapshot := graph.issues[index]
			snapshot.Dependencies.Items = []sdk.SnapshotDependency{{OccurrenceID: "801", IssueID: snapshot.Issue.ID, DependencyID: native, Visible: true, Complete: true}}
			snapshot.Dependencies.Total = 1
			graph.issues[index] = snapshot
			decision, err := db.AcceptanceDecision(ctx, binding.AcceptanceID)
			if err != nil {
				t.Fatal(err)
			}
			decision.ID, decision.Predecessor = generatedDecisionID("human-result", revision.ID, binding.TaskKey), decision.ID
			decision.NativeRev, decision.ContentDigest, decision.ContentVersion = graph.revision, snapshot.Issue.ContentDigest, snapshot.Issue.ContentVersion
			decision.Prerequisites = []factory.AcceptedPrerequisite{{Occurrence: "801", DependsOn: native, EndpointRepo: fx.repo, EndpointIssue: endpoint, Outcome: factory.PrereqResult}}
			if _, err = coord.AdmitAcceptance(ctx, factory.NewID(), "soda-tester", decision); err != nil {
				t.Fatal("human result requirement", err)
			}
			protectedHeads[index] = decision.ID
		}
		protectedDependencies, _ = json.Marshal([]*sdk.SnapshotDependencyPage{graph.issues[1].Dependencies, graph.issues[2].Dependencies, graph.issues[3].Dependencies})
	}
	baseline := graph.submissions
	coord.Graph = &reassessmentLostReplyGraph{generatedGraphFixture: graph, loseReply: true}
	enumerator := reassessmentGraphInventory{graph: graph}
	coord.CompletionEnumerator, coord.CompletionLocalCapture = enumerator, db.CaptureCompletionLocal
	chooser := &reassessmentLifecycleChooser{}
	coord.Planning = &PlanningJudgments{Journal: db, Funding: db, Chooser: chooser}
	coord.MCPKey = []byte(strings.Repeat("k", 32))
	coord.Priority = &PriorityJudgments{Journal: db, Chooser: &bundleGapChooser{pick: factory.ContextBundleCurrent}}
	grant := factory.DefaultRepairGrant()
	grant.ID, grant.Repository, grant.Sponsor, grant.Active, grant.CreatedUnix = factory.NewID(), fx.repo, 7, true, time.Now().Unix()
	raw, _ := json.Marshal(grant)
	command := factory.Command{ID: grant.ID, Type: factory.CommandRepairGrant, Target: factory.RepairTarget(fx.repo), Principal: "soda-tester", Payload: string(raw)}
	command.Digest = factory.SettingsDigest(command.Type, command.Target, command.Payload)
	if _, _, err = db.ApplyRepairGrantCommand(ctx, command, grant, time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, binding := range initial {
		if mode == "append_producer" {
			stallSeedControl(t, db, fx.repo, binding.IssueIndex, factory.ReadinessBlocked, factory.BlockerResultPending, stallBlocker(factory.BlockerResultPending, fx.repo, binding.IssueIndex+1))
		} else {
			stallSeedControl(t, db, fx.repo, binding.IssueIndex, factory.ReadinessBlocked, factory.BlockerCycle, stallBlocker(factory.BlockerCycle, 0, 0))
		}
	}
	first, err := coord.ServeStallRound(ctx, DispatchReport{})
	if err != nil || len(first.Repositories) != 1 || first.Repositories[0].Event == nil {
		t.Fatal("stall observation", err)
	}
	event := *first.Repositories[0].Event
	_, _ = coord.ServeStallRound(ctx, DispatchReport{})
	record, err := db.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
	if err != nil || len(record.Repair.Cycles) != 1 {
		t.Fatal("repair stage", err)
	}
	cycle := record.Repair.Cycles[0]
	if cycle.State != factory.RepairInvestigating {
		t.Fatalf("repair launch state=%s reason=%s", cycle.State, cycle.Reason)
	}
	tasks := append([]factory.IntentTask(nil), revision.Tasks...)
	retained := []int64{2}
	if mode == "append_producer" {
		tasks = append(tasks, factory.IntentTask{Key: choiceTaskKey(3), Objective: "Supply the missing in-scope validation fixture", Outcomes: []string{"The requested fixture is available for independent maintainer acceptance"}, IdeaIDs: []string{revision.Ideas[0].ID}})
		retained = []int64{1, 2, 3}
	} else {
		tasks[0].Objective += " with the clarified validation boundary"
	}
	keys := submittedKeys(tasks)
	successor := generatedDecisionID("revision", cycle.ID, revision.ID)
	proposal := factory.StallRepairReport{StallID: record.ID, CycleID: cycle.ID, InventoryDigest: cycle.InventoryDigest, SourceCommit: cycle.Local.SourceCommit, PreparationDigest: cycle.Local.PreparationDigest, Causes: []factory.CheckFailure{{CheckRef: "dependency_cycle", Detail: "The idle task lacks its precise validation boundary."}}, RetainedIssueIndexes: retained, IntentChanges: []factory.ReassessmentIntentChange{{IntentID: revision.Intent, ExpectedRevision: successor, ScopeDigest: revision.Digest(), Reason: "Clarify the idle validation task while retaining protected human work.", Tasks: tasks}}, Coverage: []factory.ReassessmentCoverage{{IntentID: revision.Intent, OutcomeID: revision.Ideas[0].ID, TaskKeys: keys}}, Outcome: factory.RepairOutcomeRepair}
	if mode == "append_producer" {
		proposal.Causes = []factory.CheckFailure{{CheckRef: "missing_producer", Detail: "The manual fixture requirement has no accepted producer."}}
		proposal.IntentChanges[0].Reason = "Add the missing fixture producer within the accepted scope; preserve both human prerequisite routes."
	}
	if err = proposal.Validate(); err != nil {
		t.Fatal(err)
	}
	run, err := db.FactoryRun(ctx, cycle.RunID)
	if err != nil {
		t.Fatal(err)
	}
	run.Outcome, run.Summary, run.Reconciled = factory.Succeeded, "repair proposal ready", true
	if err = db.SaveFactoryRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	output := fencedResult(t, factory.PhaseResult{Version: factory.PhaseResultVersion, Phase: project.ExecutionRepair, Status: "completed", Summary: "Repair the idle accepted graph and preserve protected work.", Output: &factory.PhaseOutput{Repair: &proposal}})
	if a, ok := AccountSettledRun(ctx, db, run, output, time.Now()); !ok || a.Outcome != factory.Succeeded {
		t.Fatalf("repair finish outcome=%s reason=%s recorded=%v", a.Outcome, a.Reason, ok)
	}
	grantChanged := false
	changeGrant := func(active bool) {
		t.Helper()
		prior := grant
		grant.ID, grant.Predecessor, grant.Revision, grant.Active = factory.NewID(), prior.ID, prior.Revision+1, active
		raw, _ := json.Marshal(grant)
		command := factory.Command{ID: grant.ID, Type: factory.CommandRepairGrant, Target: factory.RepairTarget(fx.repo), Principal: "soda-tester", Payload: string(raw)}
		command.Digest = factory.SettingsDigest(command.Type, command.Target, command.Payload)
		if _, _, err := db.ApplyRepairGrantCommand(ctx, command, grant, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	for pass := 0; pass < 14; pass++ {
		coord.driveStallRepairs(ctx, map[int64]bool{fx.repo: true})
		record, err = db.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
		if err != nil {
			t.Fatal(err)
		}
		cycle = record.Repair.Cycles[0]
		if (mode == "grant_hold_new_effect" || mode == "grant_withdrawn_after_admission") && !grantChanged && cycle.State == factory.RepairApplying {
			progress, err := loadGraphRepairAdoption(cycle.AdoptionProgress, record.Repository, "", record.ID, cycle.ID, cycle.InventoryDigest)
			if err != nil {
				t.Fatal(err)
			}
			step, _ := progress.Journal.CurrentStep()
			if mode == "grant_hold_new_effect" && step.State == reassessmentStepStaged {
				changeGrant(false)
				coord.driveStallRepairs(ctx, map[int64]bool{fx.repo: true})
				held, err := db.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
				if err != nil || held.Repair.Cycles[0].Reason != "repair_sponsor_grant_unavailable" || graph.submissions != baseline {
					t.Fatal("withdrawn grant admitted new effect", err)
				}
				if _, err = db.IntentGeneration(ctx, successor); !errors.Is(err, store.ErrNotFound) {
					t.Fatal("held successor activated")
				}
				changeGrant(true)
				grantChanged = true
			} else if mode == "grant_withdrawn_after_admission" && step.State == reassessmentStepActive {
				changeGrant(false)
				grantChanged = true
			}
		}
		if cycle.State == factory.RepairAdopted {
			break
		}
		// Restart from retained rows after the immutable current operation exists.
		if pass == 5 {
			fresh := NewCoordinator(db, coord.Host, coord.Broker)
			fresh.Graph, fresh.GraphActorID, fresh.AcceptanceReads = coord.Graph, coord.GraphActorID, coord.AcceptanceReads
			fresh.CompletionEnumerator, fresh.CompletionLocalCapture, fresh.Planning = enumerator, db.CaptureCompletionLocal, coord.Planning
			coord = fresh
		}
	}
	if cycle.State != factory.RepairAdopted {
		charges, _ := db.RepairCharges(ctx, fx.repo)
		t.Fatalf("repair adoption state=%s reason=%s charges=%d calls=%d reserved=%v", cycle.State, cycle.Reason, len(charges), chooser.calls, cycle.JudgmentFingerprint != "")
	}
	head, err := db.IntentRevision(ctx, revision.Intent, revision.Owner)
	if err != nil || head.ID != successor {
		t.Fatal("repair successor missing", err)
	}
	bindings, err := db.IntentTaskBindings(ctx, successor)
	if err != nil || len(bindings) != len(tasks) {
		t.Fatal("repair readback missing", err)
	}
	for _, binding := range bindings {
		if binding.TaskKey == tasks[1].Key && binding.AcceptanceID != initial[1].AcceptanceID {
			t.Fatal("protected acceptance changed")
		}
	}
	if mode == "append_producer" {
		for _, binding := range initial {
			head, err := db.AcceptanceHead(ctx, fx.repo, binding.IssueIndex)
			if err != nil || head != protectedHeads[binding.IssueIndex] {
				t.Fatal("repair replaced human result acceptance", err)
			}
		}
		if _, err := db.AcceptanceHead(ctx, fx.repo, 3); !errors.Is(err, store.ErrNotFound) {
			t.Fatal("repair invented an acceptance for the manual result", err)
		}
		after, _ := json.Marshal([]*sdk.SnapshotDependencyPage{graph.issues[1].Dependencies, graph.issues[2].Dependencies, graph.issues[3].Dependencies})
		if string(after) != string(protectedDependencies) || len(graph.issues) != 4 {
			t.Fatal("repair changed protected prerequisite routes or omitted the new producer")
		}
		generation, err := db.IntentGeneration(ctx, successor)
		var payload sdk.IssueGraphPayload
		if err != nil || json.Unmarshal(generation.Payload, &payload) != nil || len(payload.Creates) != 1 || len(payload.Edits) != 0 || len(payload.Dependencies) != 0 {
			t.Fatal("append producer did not remain one independent create", err)
		}
	}
	charges, err := db.RepairCharges(ctx, fx.repo)
	if err != nil || len(charges) != 2 || chooser.calls != 1 || graph.submissions != baseline+1 || graph.issues[2].Issue.Content != protectedBody {
		t.Fatalf("repair duplicated spend/effect or changed protected work: charges=%d calls=%d submissions=%d %v", len(charges), chooser.calls, graph.submissions, err)
	}
	if generation, e := db.IntentGeneration(ctx, successor); e != nil || generation.RepairCycle != cycle.ID || generation.ReassessmentCycle != "" || generation.State != "completed" {
		t.Fatal("repair borrowed completion identity", e)
	}
	coord.driveStallRepairs(ctx, map[int64]bool{fx.repo: true})
	record, err = db.RepairEpisodeRecord(ctx, fx.repo, event.EpisodeID)
	charges, _ = db.RepairCharges(ctx, fx.repo)
	if err != nil || len(record.Repair.Cycles) != 1 || len(charges) != 2 {
		t.Fatal("graph reshuffle alone refilled repair", err)
	}
}

// TestServeStallRoundGenerationReuseAndIsolation proves the R36 serving
// joints across rounds and repositories on the real store path: the
// detection generation reuses until verified delivery advances it, and a
// stalled repository's bound episode and single charge never leak into a
// sibling repository that still holds ready work.
func TestServeStallRoundGenerationReuseAndIsolation(t *testing.T) {
	ctx := context.Background()
	db, _ := postgresFixture(t, nil)
	coord := &Coordinator{Store: db}
	const stalled, ready = 61, 62
	grantID := "d1000000000000000000000000000061"
	grant := factory.RepairGrant{
		ID: grantID, Repository: stalled, Sponsor: 7, Revision: 0,
		InvestigatorRuns: 4, ProposalJudgments: 4, ActiveMinutes: 120,
		Active: true, CreatedUnix: 1000,
	}
	raw, err := json.Marshal(grant)
	if err != nil {
		t.Fatal(err)
	}
	cmd := factory.Command{
		ID: grantID, Type: factory.CommandRepairGrant, Target: factory.RepairTarget(stalled),
		Principal: "soda-tester", Payload: string(raw),
	}
	cmd.Digest = factory.SettingsDigest(cmd.Type, cmd.Target, cmd.Payload)
	if _, created, err := db.ApplyRepairGrantCommand(ctx, cmd, grant, time.Now().UTC()); err != nil || !created {
		t.Fatalf("apply grant: created=%v err=%v", created, err)
	}
	stallSeedControl(t, db, stalled, 1, factory.ReadinessBlocked,
		factory.BlockerCycle, stallBlocker(factory.BlockerCycle, 0, 0))
	stallSeedControl(t, db, ready, 1, factory.ReadinessQueued, factory.ReasonEligible)

	byRepo := func(round StallClassification) map[int64]StallRepositoryResult {
		out := map[int64]StallRepositoryResult{}
		for _, result := range round.Repositories {
			out[result.Repository] = result
		}
		return out
	}
	for round := 1; round <= 2; round++ {
		served, err := coord.ServeStallRound(ctx, DispatchReport{})
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		if served.Generation != 0 {
			t.Fatalf("round %d generation = %d, want reused 0", round, served.Generation)
		}
		repos := byRepo(served)
		stalledResult, ok := repos[stalled]
		if !ok || stalledResult.Kind != StallRepositoryStall || !stalledResult.Investigate || stalledResult.Event == nil {
			t.Fatalf("round %d stalled=%+v, want one bound episode", round, stalledResult)
		}
		readyResult, ok := repos[ready]
		if !ok || readyResult.Kind != StallRepositoryReady || readyResult.Investigate || readyResult.Event != nil {
			t.Fatalf("round %d ready=%+v, want resumed selection without an episode", round, readyResult)
		}
		if charges, err := db.RepairCharges(ctx, stalled); err != nil || len(charges) != 1 {
			t.Fatalf("round %d stall charges=%d err=%v, want exactly one", round, len(charges), err)
		}
		if charges, err := db.RepairCharges(ctx, ready); err != nil || len(charges) != 0 {
			t.Fatalf("round %d ready charges=%d err=%v, want none", round, len(charges), err)
		}
	}
	standing, err := db.StandingStallEvents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if entry, ok := standing[stalled]; !ok || entry.Generation != 0 {
		t.Fatalf("standing=%+v, want the stalled generation-0 observation", standing)
	}
	if _, ok := standing[ready]; ok {
		t.Fatalf("standing=%+v, want no observation for the ready repository", standing)
	}
}
