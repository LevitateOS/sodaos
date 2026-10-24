package control

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// The authenticated correction terminal must retain its whole typed report
// through the PostgreSQL finish/receipt transaction. Changed or omitted keys
// fence rather than silently dropping dispositions through the legacy mapper.
func TestSettleCorrectionTerminalPGDrive(t *testing.T) {
	db, dsn := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	for index, malformed := range []bool{false, true, false} {
		a := phaseTestAssignment(t, project.ExecutionCorrect, project.RoleCoder, false)
		a.Issue, a.Subject.Issue.Issue = int64(11+index), int64(11+index)
		finding := correctionTestFinding(1, true, factory.FindingFixCode)
		digest, err := factory.FindingSetDigest([]string{finding.Key})
		if err != nil {
			t.Fatal(err)
		}
		cycle := factory.ReviewCycle{PublicationID: factory.NewID(), AttemptRoot: a.AttemptRoot, Phase: project.ExecutionCorrect, Head: a.SourceCommit, BaseOID: strings.Repeat("b", 40), FindingDigest: digest, ReviewerAssignment: factory.NewID(), CorrectionAssignment: a.ID, Round: 1, TriggerDecision: "correction", TriggerReason: "answer the observed finding"}
		cycle.ID = factory.ReviewCycleID(cycle.PublicationID, cycle.AttemptRoot, cycle.Round, cycle.Phase)
		a.ReviewContext = &factory.CurrentReviewContext{Publication: factory.CurrentPublicationContext{ID: cycle.PublicationID, AssignmentID: a.PublicationAssignment}, AdmittedTuple: factory.CandidateTuple{Head: a.SourceCommit, Base: cycle.BaseOID, DiffBase: cycle.BaseOID, Revision: 1}, Cycle: cycle, CarriedFindings: []factory.Finding{finding}}
		if err := a.Validate(); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`, a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(raw)); err != nil {
			t.Fatal(err)
		}
		run := phaseTestRun(a, "")
		run.Image = "sha256:" + strings.Repeat("c", 64)
		run.Started, run.Deadline = time.Now().Add(-time.Second), time.Now().Add(time.Hour)
		if err := db.RecordFactoryRun(ctx, run); err != nil {
			t.Fatal(err)
		}
		staged := testMCPCapability()
		staged.AssignmentID, staged.Profile, staged.State = a.ID, factory.MCPProfileFixer, factory.MCPCapabilityStaged
		if err := db.StageMCPCapability(ctx, staged); err != nil {
			t.Fatal(err)
		}
		if err := db.ActivateMCPCapability(ctx, staged.ID, staged.Generation, staged.RunID, staged.AssignmentID, staged.Authority, time.Now().Unix(), false); err != nil {
			t.Fatal(err)
		}
		capability, err := db.GetMCPCapability(ctx, staged.ID)
		if err != nil {
			t.Fatal(err)
		}
		key, candidate := finding.Key, strings.Repeat("8", 40)
		if malformed {
			key = factory.NewID()
		}
		evidence := []factory.CorrectionFixEvidence{{Key: finding.Key, ChangedPaths: []string{"native-proof.txt"}, Commits: []string{candidate}}}
		payload, err := json.Marshal(map[string]any{"finding_digest": digest, "candidate": candidate, "dispositions": []factory.CorrectionDisposition{{Key: key, Disposition: factory.DispositionFixed}}, "fix_evidence": evidence, "evidence_refs": []string{"native checkpoint"}})
		if err != nil {
			t.Fatal(err)
		}
		if index == 2 {
			payload, _ = json.Marshal(map[string]any{"candidate": candidate})
		}
		submission := testMCPSubmission(capability, factory.MCPToolFinishCorrection, factory.NewID(), payload)
		if _, err := db.RecordMCPSubmission(ctx, submission, factory.MCPReceipt{SubmissionID: submission.SubmissionID, Tool: submission.Tool, State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash}); err != nil {
			t.Fatal(err)
		}
		// Native completion may arrive before the periodic MCP drive. Its
		// empty stdout must not consume an already-recorded typed terminal.
		retired := run
		retired.Outcome, retired.Reconciled = factory.Succeeded, true
		foreign := retired
		foreign.ID = factory.NewID()
		if handled, err := settleRecordedMCPBeforeRun(ctx, db, a, foreign, time.Now()); err != nil || handled {
			t.Fatalf("foreign retired run claimed terminal: %v/%v", handled, err)
		}
		if index != 1 {
			if err := finishFromRun(ctx, db, a, retired, "", time.Now()); err != nil {
				t.Fatal(err)
			}
		} else if _, accounted := AccountSettledRun(ctx, db, retired, "", time.Now()); !accounted {
			t.Fatal("retired MCP run was not accounted through its terminal")
		}
		if settled, err := SettleMCPTerminalProposal(ctx, db, a.ID, time.Now()); err != nil || settled {
			mapped, note, handled, mapErr := mapMCPTerminalFinish(a, submission, time.Now())
			receipt, found, receiptErr := db.TerminalMCPProposal(ctx, a.ID)
			t.Fatalf("correction case=%d settlement=%v err=%v map=%v/%v/%s outcome=%s result=%+v receipt=%+v/%v/%v", index, settled, err, handled, mapErr, note, mapped.Outcome, mapped.Result, receipt, found, receiptErr)
		}
		finished, err := db.Assignment(ctx, a.ID)
		if err != nil {
			t.Fatal(err)
		}
		if index == 2 {
			receipt, err := db.MCPReceipt(ctx, a.ID, submission.SubmissionID)
			if err != nil || finished.Stage != factory.AssignmentAssigned || receipt.State != factory.MCPReceiptRecorded {
				t.Fatal("parked strict carrier was replaced by native no-report finish")
			}
			continue
		}
		if malformed {
			if finished.Outcome != factory.NeedsHuman {
				t.Fatal("foreign finding key did not fence")
			}
		} else if finished.Outcome != factory.Succeeded || finished.Result == nil || finished.Result.PhaseResult == nil || finished.Result.PhaseResult.Output.Correction == nil || finished.Result.Candidate != candidate || len(finished.Result.PhaseResult.Output.Correction.Evidence) != 1 || finished.Result.PhaseResult.Output.Correction.Findings[0].Defect != finding.Defect {
			t.Fatalf("typed correction report lost: %+v", finished)
		}
		receipt, err := db.MCPReceipt(ctx, a.ID, submission.SubmissionID)
		if err != nil || receipt.State != factory.MCPReceiptAccepted {
			t.Fatalf("correction receipt: %+v err=%v", receipt, err)
		}
		if err := finishFromRun(ctx, db, a, retired, "", time.Now()); err != nil {
			t.Fatalf("retired finish replay: %v", err)
		}
		replayed, err := db.Assignment(ctx, a.ID)
		if err != nil || factory.ResultDigest(*replayed.Result) != factory.ResultDigest(*finished.Result) {
			t.Fatal("retired finish replay changed the original terminal")
		}
	}
}

// TestSettleMCPTerminalsPGDrive proves the coordinator drive against
// real PostgreSQL (SODA_PG_*; skips without it): a recorded FW40
// review terminal behind a persisted assigned assignment finishes
// the assignment typed and advances its receipt recorded→accepted,
// and a second drive settles nothing more.
func TestSettleMCPTerminalsPGDrive(t *testing.T) {
	db, dsn := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(raw)); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}

	staged := testMCPCapability()
	staged.AssignmentID = a.ID
	staged.Profile = factory.MCPProfileInspector
	staged.State = factory.MCPCapabilityStaged
	if err := db.StageMCPCapability(ctx, staged); err != nil {
		t.Fatalf("stage capability: %v", err)
	}
	if err := db.ActivateMCPCapability(ctx, staged.ID, staged.Generation, staged.RunID,
		staged.AssignmentID, staged.Authority, time.Now().Unix(), false); err != nil {
		t.Fatalf("activate capability: %v", err)
	}
	capability, err := db.GetMCPCapability(ctx, staged.ID)
	if err != nil {
		t.Fatalf("capability readback: %v", err)
	}
	payload, err := json.Marshal(map[string]any{
		"conclusion": "incomplete", "coverage": "partial", "evidence_refs": "r",
	})
	if err != nil {
		t.Fatal(err)
	}
	submission := testMCPSubmission(capability, factory.MCPToolFinishReview, factory.NewID(), payload)
	receipt := factory.MCPReceipt{
		SubmissionID: submission.SubmissionID, Tool: submission.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
	}
	if _, err := db.RecordMCPSubmission(ctx, submission, receipt); err != nil {
		t.Fatalf("record terminal: %v", err)
	}

	coordinator := &Coordinator{Store: db}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 1 {
		t.Fatalf("settled %d, want 1", settled)
	}
	finished, err := db.Assignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Stage != factory.AssignmentFinished || finished.Outcome != factory.NeedsHuman ||
		finished.Result == nil || finished.Result.Status != "blocked" ||
		finished.Result.PhaseResult == nil || finished.Result.PhaseResult.Status != "blocked" {
		t.Fatalf("driven finish not typed blocked: %+v", finished)
	}
	advanced, err := db.MCPReceipt(ctx, a.ID, submission.SubmissionID)
	if err != nil {
		t.Fatal(err)
	}
	if advanced.State != factory.MCPReceiptAccepted {
		t.Fatalf("receipt state %q, want accepted", advanced.State)
	}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 0 {
		t.Fatalf("second drive settled %d, want 0", settled)
	}
}

// TestSettleResearchTerminalPGDrive proves the coordinator drive settles
// a recorded finish_research terminal behind a persisted research root:
// the assignment finishes reported with the typed research report and
// no candidate, and the receipt advances recorded→accepted.
// Fenced terminals share one seeding drive: a recorded terminal on a
// code-phase assignment settles exactly once and fences blocked whether
// the tool is a verify finish with full carriers or a reassessment
// finish naming a repair report.
func TestSettleVerifyRepairWrongSubjectFencedPGDrive(t *testing.T) {
	db, dsn := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	seed := func(t *testing.T, sqlDB *sql.DB, issue int64, tool factory.MCPTool, profile factory.MCPProfile, payload map[string]any) (factory.Assignment, factory.MCPSubmission) {
		t.Helper()
		a := phaseTestAssignment(t, project.ExecutionInvestigate, project.RoleCoder, true)
		// One unfinished assignment per repository issue: spread the
		// fenced cases across issues, keeping scope and subject bound.
		a.Issue, a.Subject.Issue.Issue = issue, issue
		if err := a.Validate(); err != nil {
			t.Fatalf("seed assignment invalid: %v", err)
		}
		raw, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
			a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(raw)); err != nil {
			t.Fatalf("seed assignment: %v", err)
		}
		staged := testMCPCapability()
		staged.AssignmentID = a.ID
		staged.Profile = profile
		staged.State = factory.MCPCapabilityStaged
		if err := db.StageMCPCapability(ctx, staged); err != nil {
			t.Fatalf("stage capability: %v", err)
		}
		if err := db.ActivateMCPCapability(ctx, staged.ID, staged.Generation, staged.RunID,
			staged.AssignmentID, staged.Authority, time.Now().Unix(), false); err != nil {
			t.Fatalf("activate capability: %v", err)
		}
		capability, err := db.GetMCPCapability(ctx, staged.ID)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmission(capability, tool, factory.NewID(), encoded)
		receipt := factory.MCPReceipt{
			SubmissionID: submission.SubmissionID, Tool: submission.Tool,
			State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
		}
		if _, err := db.RecordMCPSubmission(ctx, submission, receipt); err != nil {
			t.Fatalf("record terminal: %v", err)
		}
		return a, submission
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()

	verifyPayload := map[string]any{
		"outcome": "verified", "delivery_id": factory.NewID(),
		"input_digest":   strings.Repeat("c", 64),
		"check_receipts": []any{map[string]any{}}, "coverage": []any{},
	}
	verifyAssignment, verifySubmission := seed(t, sqlDB, 11, factory.MCPToolFinishAppVerify, factory.MCPProfileTestPilot, verifyPayload)
	repairPayload := map[string]any{"subject": "stall:" + factory.NewID(), "report": map[string]any{}}
	repairAssignment, repairSubmission := seed(t, sqlDB, 12, factory.MCPToolFinishReassessment, factory.MCPProfileStrategist, repairPayload)

	coordinator := &Coordinator{Store: db}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 2 {
		t.Fatalf("settled %d, want 2", settled)
	}
	for _, tc := range []struct {
		name       string
		assignment factory.Assignment
		submission factory.MCPSubmission
	}{
		{"verify", verifyAssignment, verifySubmission},
		{"repair", repairAssignment, repairSubmission},
	} {
		finished, err := db.Assignment(ctx, tc.assignment.ID)
		if err != nil {
			t.Fatal(err)
		}
		if finished.Stage != factory.AssignmentFinished || finished.Outcome != factory.NeedsHuman ||
			finished.Result == nil || finished.Result.Status != "blocked" {
			t.Fatalf("%s wrong-subject finish not fenced blocked: %+v", tc.name, finished)
		}
		advanced, err := db.MCPReceipt(ctx, tc.assignment.ID, tc.submission.SubmissionID)
		if err != nil {
			t.Fatal(err)
		}
		if advanced.State != factory.MCPReceiptAccepted {
			t.Fatalf("%s receipt state %q, want accepted", tc.name, advanced.State)
		}
	}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 0 {
		t.Fatalf("second drive settled %d, want 0", settled)
	}
	// A real stall report arriving before native completion must survive the
	// generic empty-stdout finish. A foreign sealed inventory still fences.
	for _, foreign := range []bool{false, true} {
		a := phaseTestAssignment(t, project.ExecutionRepair, project.RoleCoder, false)
		a.Issue, a.Acceptance, a.PublicationAssignment = 0, "", ""
		a.Subject = factory.ExecutionSubject{Kind: factory.SubjectStall, Stall: &factory.StallSubject{StallID: factory.NewID(), CycleID: a.ID, InventoryDigest: strings.Repeat("9", 64)}}
		if err := a.Validate(); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(a)
		if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`, a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(raw)); err != nil {
			t.Fatal(err)
		}
		allowance := factory.AttemptAllowance{Limits: factory.AttemptLimits{ActiveMinutes: 30}, Stall: a.Subject.Stall, RootAssignment: a.ID, Repository: a.Repository, CheckpointUnix: time.Now().Unix(), Active: true}
		if err := allowance.Validate(); err != nil {
			t.Fatal(err)
		}
		raw, _ = json.Marshal(allowance)
		if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_attempt_allowances(repository,issue,root_assignment,revision,data) VALUES($1,$2,$3,$4,$5)`, a.Repository, a.Issue, a.ID, allowance.Revision, string(raw)); err != nil {
			t.Fatal(err)
		}
		staged := testMCPCapability()
		staged.AssignmentID, staged.RunID, staged.Profile, staged.State = a.ID, a.Run, factory.MCPProfileStrategist, factory.MCPCapabilityStaged
		if err := db.StageMCPCapability(ctx, staged); err != nil {
			t.Fatal(err)
		}
		if err := db.ActivateMCPCapability(ctx, staged.ID, staged.Generation, staged.RunID, staged.AssignmentID, staged.Authority, time.Now().Unix(), false); err != nil {
			t.Fatal(err)
		}
		capability, err := db.GetMCPCapability(ctx, staged.ID)
		if err != nil {
			t.Fatal(err)
		}
		report := factory.StallRepairReport{StallID: a.Subject.Stall.StallID, CycleID: a.ID, InventoryDigest: a.Subject.Stall.InventoryDigest, SourceCommit: a.SourceCommit, PreparationDigest: strings.Repeat("8", 64), Inspected: []factory.InspectedSource{{Path: "native-proof.txt", Commit: a.SourceCommit, Evidence: "Inspected retained requirement evidence"}}, Causes: []factory.CheckFailure{{CheckRef: "manual-result", Detail: "No authorized producer or accepted resolution"}}, RetainedIssueIndexes: []int64{1, 2}, Outcome: factory.RepairOutcomeIncomplete}
		if foreign {
			report.InventoryDigest = strings.Repeat("7", 64)
		}
		payload, _ := json.Marshal(map[string]any{"subject": "stall:" + report.StallID, "report": report})
		submission := testMCPSubmission(capability, factory.MCPToolFinishReassessment, factory.NewID(), payload)
		if _, err := db.RecordMCPSubmission(ctx, submission, factory.MCPReceipt{SubmissionID: submission.SubmissionID, Tool: submission.Tool, State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash}); err != nil {
			t.Fatal(err)
		}
		run := phaseTestRun(a, factory.Succeeded)
		run.Reconciled = true
		if err := finishFromRun(ctx, db, a, run, "", time.Now()); err != nil {
			t.Fatal(err)
		}
		finished, err := db.Assignment(ctx, a.ID)
		if err != nil {
			t.Fatal(err)
		}
		if foreign {
			if finished.Outcome != factory.NeedsHuman || finished.Result.Reported {
				t.Fatal("foreign stall inventory did not fence")
			}
		} else if finished.Outcome != factory.Succeeded || !finished.Result.Reported || finished.Result.PhaseResult.Output.Repair == nil || finished.Result.PhaseResult.Output.Repair.Outcome != factory.RepairOutcomeIncomplete {
			t.Fatalf("typed stall diagnosis lost behind native completion: %+v", finished.Result)
		}
		if settled, err := SettleMCPTerminalProposal(ctx, db, a.ID, time.Now()); err != nil || settled {
			t.Fatalf("stall terminal replay changed finish: %v/%v", settled, err)
		}
	}
}

func TestSettleResearchTerminalPGDrive(t *testing.T) {
	db, dsn := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	a := phaseTestAssignment(t, project.ExecutionResearch, project.RoleReviewer, true)
	// Native exit and a stdout research block cannot replace the durable
	// Researcher finish, even though the execution principal is reviewer.
	stdout := fencedResult(t, factory.PhaseResult{Version: factory.PhaseResultVersion, Phase: project.ExecutionResearch, Status: "completed", Summary: "Research completed", Output: &factory.PhaseOutput{Research: &factory.ResearchReport{Questions: []string{"What restarts the worker?"}, Findings: []string{"The supervisor."}, Evidence: []string{"ops.runbooks#3"}, Conclusion: "Use the supervisor."}}})
	unreported, err := finishAssignmentFromRun(a, phaseTestRun(a, factory.Succeeded), stdout, time.Now())
	if err != nil || unreported.Outcome != factory.NeedsHuman || unreported.Result.Reported || unreported.Result.Candidate != "" {
		t.Fatalf("research accepted stdout without its MCP terminal: %+v %v", unreported.Result, err)
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(raw)); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	// A reported research finish closes its root admission, which the
	// store requires as a recorded attempt allowance like production
	// dispatch records. Seed the root allowance the packet would own.
	allowance := factory.AttemptAllowance{
		Limits:         factory.AttemptLimits{ActiveMinutes: 10, ReviewRounds: 1, MaxChildren: 1},
		RootAssignment: a.ID, Repository: a.Repository, Issue: a.Issue,
		CheckpointUnix: time.Now().Unix(), Active: true,
	}
	if err := allowance.Validate(); err != nil {
		t.Fatalf("seed allowance invalid: %v", err)
	}
	allowanceRaw, err := json.Marshal(allowance)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_attempt_allowances(repository,issue,root_assignment,revision,data) VALUES($1,$2,$3,$4,$5)`,
		a.Repository, a.Issue, a.ID, allowance.Revision, string(allowanceRaw)); err != nil {
		t.Fatalf("seed allowance: %v", err)
	}

	staged := testMCPCapability()
	staged.AssignmentID = a.ID
	staged.Profile = factory.MCPProfileResearcher
	staged.State = factory.MCPCapabilityStaged
	if err := db.StageMCPCapability(ctx, staged); err != nil {
		t.Fatalf("stage capability: %v", err)
	}
	if err := db.ActivateMCPCapability(ctx, staged.ID, staged.Generation, staged.RunID,
		staged.AssignmentID, staged.Authority, time.Now().Unix(), false); err != nil {
		t.Fatalf("activate capability: %v", err)
	}
	capability, err := db.GetMCPCapability(ctx, staged.ID)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"questions":   []string{"What restarts the worker?"},
		"findings":    []string{"The supervisor restarts it."},
		"evidence":    []string{"ops.runbooks#3"},
		"conclusion":  "Restart through the supervisor.",
		"uncertainty": "None.",
	})
	if err != nil {
		t.Fatal(err)
	}
	submission := testMCPSubmission(capability, factory.MCPToolFinishResearch, factory.NewID(), payload)
	receipt := factory.MCPReceipt{
		SubmissionID: submission.SubmissionID, Tool: submission.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
	}
	if _, err := db.RecordMCPSubmission(ctx, submission, receipt); err != nil {
		t.Fatalf("record terminal: %v", err)
	}

	coordinator := &Coordinator{Store: db}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 1 {
		t.Fatalf("settled %d, want 1", settled)
	}
	finished, err := db.Assignment(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Stage != factory.AssignmentFinished || finished.Outcome != factory.Succeeded ||
		finished.Result == nil || finished.Result.Status != "completed" || finished.Result.Candidate != "" ||
		finished.Result.PhaseResult == nil || finished.Result.PhaseResult.Phase != project.ExecutionResearch ||
		finished.Result.PhaseResult.Output == nil || finished.Result.PhaseResult.Output.Research == nil ||
		len(finished.Result.PhaseResult.Output.Research.Questions) != 1 {
		t.Fatalf("driven research finish not typed completed: %+v", finished)
	}
	// Wrong-subject research terminals never settle as research: a
	// finish_research receipt against a code-phase assignment fences
	// the assignment blocked for human attention while the receipt
	// still advances recorded→accepted.
	wrong := phaseTestAssignment(t, project.ExecutionInvestigate, project.RoleCoder, true)
	wrongRaw, err := json.Marshal(wrong)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
		wrong.ID, wrong.Repository, wrong.Issue, wrong.Run, wrong.Stage, wrong.Revision, string(wrongRaw)); err != nil {
		t.Fatalf("seed wrong-subject assignment: %v", err)
	}
	wrongStaged := testMCPCapability()
	wrongStaged.AssignmentID = wrong.ID
	wrongStaged.Profile = factory.MCPProfileResearcher
	wrongStaged.State = factory.MCPCapabilityStaged
	if err := db.StageMCPCapability(ctx, wrongStaged); err != nil {
		t.Fatalf("stage wrong-subject capability: %v", err)
	}
	if err := db.ActivateMCPCapability(ctx, wrongStaged.ID, wrongStaged.Generation, wrongStaged.RunID,
		wrongStaged.AssignmentID, wrongStaged.Authority, time.Now().Unix(), false); err != nil {
		t.Fatalf("activate wrong-subject capability: %v", err)
	}
	wrongCapability, err := db.GetMCPCapability(ctx, wrongStaged.ID)
	if err != nil {
		t.Fatal(err)
	}
	wrongSubmission := testMCPSubmission(wrongCapability, factory.MCPToolFinishResearch, factory.NewID(), payload)
	wrongReceipt := factory.MCPReceipt{
		SubmissionID: wrongSubmission.SubmissionID, Tool: wrongSubmission.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: wrongSubmission.PayloadHash,
	}
	if _, err := db.RecordMCPSubmission(ctx, wrongSubmission, wrongReceipt); err != nil {
		t.Fatalf("record wrong-subject terminal: %v", err)
	}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 1 {
		t.Fatalf("wrong-subject settled %d, want 1", settled)
	}
	wrongFinished, err := db.Assignment(ctx, wrong.ID)
	if err != nil {
		t.Fatal(err)
	}
	if wrongFinished.Stage != factory.AssignmentFinished || wrongFinished.Outcome != factory.NeedsHuman ||
		wrongFinished.Result == nil || wrongFinished.Result.Status != "blocked" {
		t.Fatalf("wrong-subject research finish not fenced blocked: %+v", wrongFinished)
	}
	advanced, err := db.MCPReceipt(ctx, a.ID, submission.SubmissionID)
	if err != nil {
		t.Fatal(err)
	}
	if advanced.State != factory.MCPReceiptAccepted {
		t.Fatalf("receipt state %q, want accepted", advanced.State)
	}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 0 {
		t.Fatalf("second drive settled %d, want 0", settled)
	}
}

// TestSettleVerifyArtifactRefsResolvedPGDrive proves the R32.6 finish
// revalidation against real PostgreSQL: an app verification finish
// citing this assignment's complete artifact digest settles with the
// report recorded intact, while a finish citing an unknown digest
// fences blocked with its proposal preserved for inspection.
func TestSettleVerifyArtifactRefsResolvedPGDrive(t *testing.T) {
	db, dsn := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	seed := func(t *testing.T, deliveryID string) (factory.Assignment, factory.MCPCapability) {
		t.Helper()
		a := phaseTestAssignment(t, project.ExecutionVerify, project.RoleReviewer, false)
		a.Issue = 0
		// App verification runs read-only with no publication slot
		// and no issue authority, mirroring dispatch launch.
		a.PublicationAssignment = ""
		a.Acceptance = ""
		a.Subject = factory.ExecutionSubject{Kind: factory.SubjectDelivery, Delivery: &factory.DeliverySubject{
			DeliveryID: deliveryID, IntentID: "d" + strings.Repeat("1", 24), AcceptedRevision: "d" + strings.Repeat("2", 24),
			ScopeDigest: strings.Repeat("a", 64), PreparationDigest: strings.Repeat("b", 64),
			VerificationManifestDigest: strings.Repeat("c", 64), SourceCommit: a.SourceCommit,
		}}
		if err := a.Validate(); err != nil {
			t.Fatalf("verify assignment invalid: %v", err)
		}
		raw, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
			a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(raw)); err != nil {
			t.Fatalf("seed assignment: %v", err)
		}
		// A delivery finish closes its root admission, which the store
		// requires as a recorded attempt allowance like production
		// dispatch records. Seed the root allowance carrying this
		// assignment's exact delivery subject.
		allowance := factory.AttemptAllowance{
			Limits:         factory.AttemptLimits{ActiveMinutes: 10},
			RootAssignment: a.AttemptRoot, Repository: a.Repository, Issue: 0,
			Delivery:       a.Subject.Delivery,
			CheckpointUnix: time.Now().Unix(), Active: true,
		}
		if err := allowance.Validate(); err != nil {
			t.Fatalf("seed allowance invalid: %v", err)
		}
		allowanceRaw, err := json.Marshal(allowance)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_attempt_allowances(repository,issue,root_assignment,revision,data) VALUES($1,$2,$3,$4,$5)`,
			a.Repository, 0, a.AttemptRoot, allowance.Revision, string(allowanceRaw)); err != nil {
			t.Fatalf("seed allowance: %v", err)
		}
		staged := testMCPCapability()
		staged.AssignmentID = a.ID
		staged.Profile = factory.MCPProfileTestPilot
		staged.State = factory.MCPCapabilityStaged
		if err := db.StageMCPCapability(ctx, staged); err != nil {
			t.Fatalf("stage capability: %v", err)
		}
		if err := db.ActivateMCPCapability(ctx, staged.ID, staged.Generation, staged.RunID,
			staged.AssignmentID, staged.Authority, time.Now().Unix(), false); err != nil {
			t.Fatalf("activate capability: %v", err)
		}
		capability, err := db.GetMCPCapability(ctx, staged.ID)
		if err != nil {
			t.Fatal(err)
		}
		return a, capability
	}
	upload := func(t *testing.T, capability factory.MCPCapability, uploadID, body string) string {
		t.Helper()
		raw, err := json.Marshal(map[string]any{
			"upload_id": uploadID, "name": "evidence.txt", "media_type": "text/plain",
			"offset": 0, "content_base64": base64.StdEncoding.EncodeToString([]byte(body)), "final": true,
		})
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmission(capability, factory.MCPToolPublishArtifact, factory.NewID(), raw)
		receipt, err := db.RecordMCPSubmission(ctx, submission, factory.MCPReceipt{
			SubmissionID: submission.SubmissionID, Tool: submission.Tool,
			State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
		})
		if err != nil {
			t.Fatalf("upload: %v", err)
		}
		if receipt.ArtifactDigest == "" {
			t.Fatal("final upload returned no attested digest")
		}
		if _, err := db.ResolveArtifactDigest(ctx, capability.Repository, capability.AssignmentID, receipt.ArtifactDigest); err != nil {
			t.Fatalf("resolve attested digest: %v", err)
		}
		return receipt.ArtifactDigest
	}
	finish := func(t *testing.T, capability factory.MCPCapability, deliveryID string, refs []string) factory.MCPSubmission {
		t.Helper()
		raw, err := json.Marshal(map[string]any{
			"outcome": "verified", "delivery_id": deliveryID, "input_digest": strings.Repeat("c", 64),
			"check_receipts": []any{map[string]any{"check_id": "app-check", "outcome": "passed", "log_excerpt": "checks green"}},
			"coverage":       []any{},
			"artifact_refs":  refs,
		})
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmission(capability, factory.MCPToolFinishAppVerify, factory.NewID(), raw)
		if _, err := db.RecordMCPSubmission(ctx, submission, factory.MCPReceipt{
			SubmissionID: submission.SubmissionID, Tool: submission.Tool,
			State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
		}); err != nil {
			t.Fatalf("record verify finish: %v", err)
		}
		return submission
	}
	goodDelivery := factory.NewID()
	goodAssignment, goodCap := seed(t, goodDelivery)
	digest := upload(t, goodCap, factory.NewID(), "hello verify")
	goodSubmission := finish(t, goodCap, goodDelivery, []string{digest})
	badDelivery := factory.NewID()
	badAssignment, badCap := seed(t, badDelivery)
	_ = upload(t, badCap, factory.NewID(), "hello verify")
	badSubmission := finish(t, badCap, badDelivery, []string{strings.Repeat("f", 64)})
	coordinator := &Coordinator{Store: db}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 2 {
		t.Fatalf("settled %d, want 2", settled)
	}
	goodFinished, err := db.Assignment(ctx, goodAssignment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if goodFinished.Stage != factory.AssignmentFinished || goodFinished.Outcome == factory.NeedsHuman ||
		goodFinished.Result == nil || goodFinished.Result.PhaseResult == nil ||
		goodFinished.Result.PhaseResult.Output == nil || goodFinished.Result.PhaseResult.Output.Verification == nil ||
		len(goodFinished.Result.PhaseResult.Output.Verification.ArtifactRefs) != 1 ||
		goodFinished.Result.PhaseResult.Output.Verification.ArtifactRefs[0] != digest {
		t.Fatalf("verified finish did not record its artifact ref: %+v", goodFinished)
	}
	goodReceipt, err := db.MCPReceipt(ctx, goodAssignment.ID, goodSubmission.SubmissionID)
	if err != nil {
		t.Fatal(err)
	}
	if goodReceipt.State != factory.MCPReceiptAccepted {
		t.Fatalf("verified receipt state %q, want accepted", goodReceipt.State)
	}
	badFinished, err := db.Assignment(ctx, badAssignment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if badFinished.Stage != factory.AssignmentFinished || badFinished.Outcome != factory.NeedsHuman ||
		badFinished.Result == nil || badFinished.Result.Status != "blocked" {
		t.Fatalf("unknown digest finish not fenced blocked: %+v", badFinished)
	}
	badReceipt, err := db.MCPReceipt(ctx, badAssignment.ID, badSubmission.SubmissionID)
	if err != nil {
		t.Fatal(err)
	}
	if badReceipt.State != factory.MCPReceiptAccepted {
		t.Fatalf("fenced receipt state %q, want accepted", badReceipt.State)
	}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 0 {
		t.Fatalf("second drive settled %d, want 0", settled)
	}
}

// TestSettleNonCodeAssessmentPGDrive proves the R34.3 assessment
// route against real PostgreSQL: an Inspector finish over a research
// subject settles with the deliverable-bound verdict intact and no
// code carrier; a clear verdict settles empty; a deliverable bound to
// the wrong subject fences blocked. The seed assignment validates,
// proving the review-phase matrix admits non-code subjects.
func TestSettleNonCodeAssessmentPGDrive(t *testing.T) {
	db, dsn := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()
	seedIssue := 11
	seed := func(t *testing.T, subject factory.ExecutionSubject) (factory.Assignment, factory.MCPCapability) {
		t.Helper()
		a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, false)
		a.Issue = int64(seedIssue)
		seedIssue++
		a.PublicationAssignment = ""
		a.Acceptance = ""
		a.Subject = subject
		if err := a.Validate(); err != nil {
			t.Fatalf("assessment assignment invalid: %v", err)
		}
		raw, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,$2,$3,$4,$5,$6,$7)`,
			a.ID, a.Repository, a.Issue, a.Run, a.Stage, a.Revision, string(raw)); err != nil {
			t.Fatalf("seed assignment: %v", err)
		}
		allowance := factory.AttemptAllowance{
			Limits:         factory.AttemptLimits{ActiveMinutes: 10, ReviewRounds: 1, MaxChildren: 1},
			RootAssignment: a.AttemptRoot, Repository: a.Repository, Issue: a.Issue,
			CheckpointUnix: time.Now().Unix(), Active: true,
		}
		if err := allowance.Validate(); err != nil {
			t.Fatalf("seed allowance invalid: %v", err)
		}
		allowanceRaw, err := json.Marshal(allowance)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := sqlDB.ExecContext(ctx, `INSERT INTO factory_attempt_allowances(repository,issue,root_assignment,revision,data) VALUES($1,$2,$3,$4,$5)`,
			a.Repository, a.Issue, a.AttemptRoot, allowance.Revision, string(allowanceRaw)); err != nil {
			t.Fatalf("seed allowance: %v", err)
		}
		staged := testMCPCapability()
		staged.AssignmentID = a.ID
		staged.Profile = factory.MCPProfileInspector
		staged.State = factory.MCPCapabilityStaged
		if err := db.StageMCPCapability(ctx, staged); err != nil {
			t.Fatalf("stage capability: %v", err)
		}
		if err := db.ActivateMCPCapability(ctx, staged.ID, staged.Generation, staged.RunID,
			staged.AssignmentID, staged.Authority, time.Now().Unix(), false); err != nil {
			t.Fatalf("activate capability: %v", err)
		}
		capability, err := db.GetMCPCapability(ctx, staged.ID)
		if err != nil {
			t.Fatal(err)
		}
		return a, capability
	}
	finish := func(t *testing.T, capability factory.MCPCapability, payload map[string]any) {
		t.Helper()
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		submission := testMCPSubmission(capability, factory.MCPToolFinishReview, factory.NewID(), raw)
		if _, err := db.RecordMCPSubmission(ctx, submission, factory.MCPReceipt{
			SubmissionID: submission.SubmissionID, Tool: submission.Tool,
			State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
		}); err != nil {
			t.Fatalf("record assessment finish: %v", err)
		}
	}
	reportID := factory.NewID()
	researchSubject := factory.ExecutionSubject{Kind: factory.SubjectResearch, Research: &factory.ResearchSubject{Issue: 7, Acceptance: "d" + strings.Repeat("1", 24)}}
	findAssignment, findCap := seed(t, researchSubject)
	finish(t, findCap, map[string]any{
		"conclusion": "findings", "evidence_refs": []string{"report:" + reportID},
		"deliverable": map[string]any{"kind": "research", "id": reportID},
		"findings": []any{map[string]any{
			"key": factory.NewID(), "origin_run": factory.NewID(), "severity": "major",
			"defect":           "The report answers one of two declared questions.",
			"evidence":         "recorded questions show two, findings cover one.",
			"required_outcome": "Answer the second question or mark it unanswered with evidence.",
			"fix_kind":         "code", "actionable": true,
		}},
	})
	clearAssignment, clearCap := seed(t, researchSubject)
	finish(t, clearCap, map[string]any{
		"conclusion": "clear", "evidence_refs": []string{"report:" + reportID},
		"deliverable": map[string]any{"kind": "research", "id": reportID},
	})
	collectionSubject := factory.ExecutionSubject{Kind: factory.SubjectCollection, Collection: &factory.CollectionSubject{CollectionID: "knowledge"}}
	badAssignment, badCap := seed(t, collectionSubject)
	finish(t, badCap, map[string]any{
		"conclusion": "findings", "evidence_refs": []string{"report:" + reportID},
		"deliverable": map[string]any{"kind": "research", "id": reportID},
		"findings": []any{map[string]any{
			"key": factory.NewID(), "origin_run": factory.NewID(), "severity": "minor",
			"defect": "Wrong subject binding.", "evidence": "Deliverable kind mismatches.",
			"required_outcome": "Bind the collection item instead.",
			"fix_kind":         "verification", "actionable": true,
		}},
	})
	coordinator := &Coordinator{Store: db}
	if settled := coordinator.settleMCPTerminals(ctx); settled != 3 {
		t.Fatalf("settled %d, want 3", settled)
	}
	assessed, err := db.Assignment(ctx, findAssignment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if assessed.Stage != factory.AssignmentFinished || assessed.Result == nil || assessed.Result.PhaseResult == nil ||
		assessed.Result.PhaseResult.Output == nil || assessed.Result.PhaseResult.Output.Assessment == nil {
		t.Fatalf("findings assessment did not settle typed: %+v", assessed)
	}
	verdict := assessed.Result.PhaseResult.Output.Assessment
	if verdict.Clear || verdict.Deliverable.Kind != "research" || verdict.Deliverable.ID != reportID || len(verdict.Findings) != 1 {
		t.Fatalf("findings assessment lost its verdict: %+v", verdict)
	}
	cleared, err := db.Assignment(ctx, clearAssignment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Stage != factory.AssignmentFinished || cleared.Result == nil || cleared.Result.PhaseResult == nil ||
		cleared.Result.PhaseResult.Output == nil || cleared.Result.PhaseResult.Output.Assessment == nil ||
		!cleared.Result.PhaseResult.Output.Assessment.Clear {
		t.Fatalf("clear assessment did not settle empty: %+v", cleared)
	}
	fenced, err := db.Assignment(ctx, badAssignment.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fenced.Stage != factory.AssignmentFinished || fenced.Outcome != factory.NeedsHuman {
		t.Fatalf("mismatched deliverable did not fence: %+v", fenced)
	}
}
