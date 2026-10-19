package control

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// FW40 consumer tests: structured carriers settle typed through the
// existing phase-result joints, loose prose parks untouched, and
// structured-but-invalid carriers fence instead of fake-settling.

// fw40Stub builds one recorded-terminal stub store from a structured
// payload value.
func fw40Stub(t *testing.T, assigned factory.Assignment, tool factory.MCPTool, payload map[string]any) *stubSettleStore {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return &stubSettleStore{
		assignment: assigned, assignmentOK: true,
		submission: factory.MCPSubmission{
			AssignmentID: assigned.ID, SubmissionID: factory.NewID(), Tool: tool,
			ExpectedInputDigest: strings.Repeat("e", 64),
			PayloadHash:         factory.MCPInputDigest(raw), Payload: raw,
		},
		terminalOK: true, receiptState: factory.MCPReceiptRecorded,
	}
}

func fw40Settle(t *testing.T, stub *stubSettleStore, id string) bool {
	t.Helper()
	settled, err := SettleMCPTerminalProposal(context.Background(), stub, id, time.Unix(1700000200, 0))
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	return settled
}

func fw40Finished(t *testing.T, stub *stubSettleStore) factory.Assignment {
	t.Helper()
	if len(stub.finished) != 1 {
		t.Fatalf("finished %d assignments, want 1", len(stub.finished))
	}
	finished := stub.finished[0]
	if err := finished.Validate(); err != nil {
		t.Fatalf("finished assignment invalid: %v", err)
	}
	if stub.lastReceipt.State != factory.MCPReceiptAccepted {
		t.Fatalf("receipt state %q, want accepted", stub.lastReceipt.State)
	}
	return finished
}

func TestFW40ProductPlanSettlesTyped(t *testing.T) {
	a := phaseTestIntentAssignment(t)
	candidates := phasedDecomposition().Decomposition.Candidates
	stub := fw40Stub(t, a, factory.MCPToolFinishProductPlan, map[string]any{
		"candidates": candidates, "evidence_refs": "r", "coverage": "all",
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("structured product plan parked")
	}
	finished := fw40Finished(t, stub)
	if finished.Outcome != factory.Succeeded || finished.Reason != factory.AssignReasonReported {
		t.Fatalf("outcome %v/%v", finished.Outcome, finished.Reason)
	}
	out := finished.Result.PhaseResult.Output.Decomposition
	if out == nil || len(out.Candidates) != 1 || out.Candidates[0].Key != "keep" {
		t.Fatalf("decomposition lost: %+v", finished.Result.PhaseResult)
	}
	if finished.Result.Candidate != "" {
		t.Fatal("product plan named a publishable candidate")
	}
}

func TestFW40ProductPlanLooseCandidatesPark(t *testing.T) {
	a := phaseTestIntentAssignment(t)
	stub := fw40Stub(t, a, factory.MCPToolFinishProductPlan, map[string]any{
		"candidates": []any{"g1"}, "evidence_refs": "r", "coverage": "all",
	})
	if fw40Settle(t, stub, a.ID) {
		t.Fatal("loose product plan settled")
	}
	if len(stub.finished) != 0 || stub.advances != 0 {
		t.Fatal("parked product plan wrote")
	}
}

func TestFW40IssuePlanSettlesTyped(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionInvestigate, project.RoleCoder, true)
	plan := phaseTestPlan(t, a.AttemptRoot, a.SourceCommit, a.Preparation)
	stub := fw40Stub(t, a, factory.MCPToolFinishIssuePlan, map[string]any{
		"tasks": plan, "approach": plan.Approach, "evidence_refs": "r",
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("structured issue plan parked")
	}
	finished := fw40Finished(t, stub)
	got := finished.Result.PhaseResult.Output.Plan
	if got == nil || got.ID != plan.ID || got.Digest != plan.Digest {
		t.Fatalf("plan lost: %+v", finished.Result.PhaseResult)
	}
}

func TestFW40IssuePlanForeignRootFences(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionInvestigate, project.RoleCoder, true)
	plan := phaseTestPlan(t, factory.NewID(), a.SourceCommit, a.Preparation)
	stub := fw40Stub(t, a, factory.MCPToolFinishIssuePlan, map[string]any{
		"tasks": plan, "approach": plan.Approach, "evidence_refs": "r",
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("foreign issue plan parked instead of fencing")
	}
	finished := fw40Finished(t, stub)
	if finished.Outcome != factory.NeedsHuman || finished.Result.Status != "blocked" {
		t.Fatalf("foreign plan outcome %v/%v", finished.Outcome, finished.Result.Status)
	}
}

func TestFW40ReviewClearSettlesTyped(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	clear := phasedReviewClear(a.SourceCommit, strings.Repeat("b", 40)).Clear
	stub := fw40Stub(t, a, factory.MCPToolFinishReview, map[string]any{
		"conclusion": "clear", "coverage": clear.CoverageDigest, "evidence_refs": "r",
		"tuple":               map[string]any{"cycle_id": clear.CycleID, "head": clear.Head, "base_oid": clear.BaseOID},
		"carried_assessments": clear.Assessments,
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("structured clear review parked")
	}
	finished := fw40Finished(t, stub)
	got := finished.Result.PhaseResult.Output.Clear
	if got == nil || got.Head != a.SourceCommit || finished.Result.Candidate != "" {
		t.Fatalf("clear verdict lost: %+v", finished.Result)
	}
}

func TestFW40ReviewFindingsSettlesTyped(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	verdict := phasedReviewVerdict().Review
	stub := fw40Stub(t, a, factory.MCPToolFinishReview, map[string]any{
		"conclusion": "findings", "coverage": "full", "evidence_refs": "r",
		"tuple":               map[string]any{"cycle_id": verdict.CycleID, "head": a.SourceCommit, "base_oid": strings.Repeat("b", 40)},
		"findings":            verdict.NewFindings,
		"carried_assessments": verdict.Assessments,
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("structured findings review parked")
	}
	finished := fw40Finished(t, stub)
	got := finished.Result.PhaseResult.Output.Review
	if got == nil || len(got.NewFindings) != 1 {
		t.Fatalf("findings verdict lost: %+v", finished.Result)
	}
}

func TestFW40ReviewIncompleteSettlesBlocked(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	stub := fw40Stub(t, a, factory.MCPToolFinishReview, map[string]any{
		"conclusion": "incomplete", "coverage": "partial", "evidence_refs": "r",
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("incomplete review parked")
	}
	finished := fw40Finished(t, stub)
	if finished.Outcome != factory.NeedsHuman || finished.Result.Status != "blocked" || finished.Result.Reported {
		t.Fatalf("incomplete outcome %v/%+v", finished.Outcome, finished.Result)
	}
	if finished.Result.PhaseResult == nil || finished.Result.PhaseResult.Status != "blocked" {
		t.Fatalf("incomplete phased result lost: %+v", finished.Result.PhaseResult)
	}
}

func TestFW40ReviewClearWithoutTupleParks(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	stub := fw40Stub(t, a, factory.MCPToolFinishReview, map[string]any{
		"conclusion": "clear", "coverage": strings.Repeat("b", 64), "evidence_refs": "r",
	})
	if fw40Settle(t, stub, a.ID) {
		t.Fatal("tuple-less clear settled")
	}
	if len(stub.finished) != 0 || stub.advances != 0 {
		t.Fatal("parked clear wrote")
	}
}

func TestFW40ReviewClearWrongHeadFences(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	clear := phasedReviewClear(strings.Repeat("d", 40), strings.Repeat("b", 40)).Clear
	stub := fw40Stub(t, a, factory.MCPToolFinishReview, map[string]any{
		"conclusion": "clear", "coverage": clear.CoverageDigest, "evidence_refs": "r",
		"tuple":               map[string]any{"cycle_id": clear.CycleID, "head": clear.Head, "base_oid": clear.BaseOID},
		"carried_assessments": clear.Assessments,
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("wrong-head clear parked instead of fencing")
	}
	finished := fw40Finished(t, stub)
	if finished.Outcome != factory.NeedsHuman {
		t.Fatalf("wrong-head outcome %v", finished.Outcome)
	}
}

// fw40ReassessAssignment builds one valid completion-subject reassess
// assignment from the shared phase fixture.
func fw40ReassessAssignment(t *testing.T) factory.Assignment {
	t.Helper()
	a := phaseTestAssignment(t, project.ExecutionReassess, project.RoleCoder, false)
	a.Subject = factory.ExecutionSubject{
		Kind: factory.SubjectCompletion,
		Completion: &factory.CompletionSubject{
			CompletionID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
			InventoryDigest: strings.Repeat("f", 64),
		},
	}
	a.Issue, a.Acceptance, a.PublicationAssignment = 0, "", ""
	if err := a.Validate(); err != nil {
		t.Fatalf("reassess fixture invalid: %v", err)
	}
	return a
}

func TestFW40ReassessmentSettlesTyped(t *testing.T) {
	a := fw40ReassessAssignment(t)
	report := factory.ReassessmentReport{
		CompletionID: strings.Repeat("1", 32), CycleID: strings.Repeat("2", 32),
		InventoryDigest: strings.Repeat("f", 64), SourceCommit: a.SourceCommit,
		PreparationDigest: strings.Repeat("0", 64), LocalInputDigest: strings.Repeat("9", 64),
		Summary:              "No graph change needed",
		RetainedIssueIndexes: []int64{1},
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("report fixture invalid: %v", err)
	}
	stub := fw40Stub(t, a, factory.MCPToolFinishReassessment, map[string]any{
		"subject": "completion", "report": report,
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("structured reassessment parked")
	}
	finished := fw40Finished(t, stub)
	if finished.Result.PhaseResult.Output.Reassessment == nil || finished.Result.Candidate != "" {
		t.Fatalf("reassessment lost: %+v", finished.Result)
	}
}

func TestFW40ReassessmentProseReportParks(t *testing.T) {
	a := fw40ReassessAssignment(t)
	stub := fw40Stub(t, a, factory.MCPToolFinishReassessment, map[string]any{
		"subject": "completion", "report": "looks done",
	})
	if fw40Settle(t, stub, a.ID) {
		t.Fatal("prose reassessment settled")
	}
	if len(stub.finished) != 0 || stub.advances != 0 {
		t.Fatal("parked reassessment wrote")
	}
}

func TestFW40StallRepairSettlesTyped(t *testing.T) {
	a := phaseTestRepairAssignment(t, project.ExecutionRepair, project.RoleCoder, phasedStallSubject())
	stub := fw40Stub(t, a, factory.MCPToolFinishReassessment, map[string]any{
		"subject": "stall", "report": phasedStallRepair().Repair,
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("structured stall repair parked")
	}
	finished := fw40Finished(t, stub)
	if finished.Result.PhaseResult.Output.Repair == nil {
		t.Fatalf("stall repair lost: %+v", finished.Result)
	}
}

func TestFW40AppVerificationSettlesTyped(t *testing.T) {
	a := phaseTestRepairAssignment(t, project.ExecutionVerify, project.RoleReviewer, phasedDeliverySubject())
	report := phasedAppVerification().Verification
	stub := fw40Stub(t, a, factory.MCPToolFinishAppVerify, map[string]any{
		"delivery_id": report.DeliveryID, "outcome": report.Outcome,
		"input_digest": report.InputDigest, "coverage": []any{},
		"check_receipts": report.CheckReceipts,
	})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("structured app verification parked")
	}
	finished := fw40Finished(t, stub)
	got := finished.Result.PhaseResult.Output.Verification
	if got == nil || got.Outcome != factory.AppVerificationVerified || finished.Result.Candidate != "" {
		t.Fatalf("verification lost: %+v", finished.Result)
	}
}

func TestFW40AppVerificationWithoutReceiptsParks(t *testing.T) {
	a := phaseTestRepairAssignment(t, project.ExecutionVerify, project.RoleReviewer, phasedDeliverySubject())
	stub := fw40Stub(t, a, factory.MCPToolFinishAppVerify, map[string]any{
		"delivery_id": strings.Repeat("7", 32), "outcome": "verified", "coverage": "all journeys",
	})
	if fw40Settle(t, stub, a.ID) {
		t.Fatal("receipt-less verification settled")
	}
	if len(stub.finished) != 0 || stub.advances != 0 {
		t.Fatal("parked verification wrote")
	}
}

// stubDriveStore extends the settle stub with the recorded-terminal
// work list for the coordinator drive.
type stubDriveStore struct {
	*stubSettleStore
	ids []string
}

func (s *stubDriveStore) RecordedTerminalMCPAssignments(_ context.Context, limit int) ([]string, error) {
	if limit < len(s.ids) {
		return s.ids[:limit], nil
	}
	return s.ids, nil
}

func TestDriveMCPTerminalsSettlesRecorded(t *testing.T) {
	a := phaseTestAssignment(t, project.ExecutionReview, project.RoleReviewer, true)
	stub := fw40Stub(t, a, factory.MCPToolFinishReview, map[string]any{
		"conclusion": "incomplete", "coverage": "partial", "evidence_refs": "r",
	})
	drive := &stubDriveStore{stubSettleStore: stub, ids: []string{a.ID}}
	if settled := driveMCPTerminals(context.Background(), drive, time.Unix(1700000200, 0)); settled != 1 {
		t.Fatalf("settled %d, want 1", settled)
	}
	_ = fw40Finished(t, stub)
}
