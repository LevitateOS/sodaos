package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// settleTestAssignment returns one valid assigned implement-phase
// issue assignment: the smallest row mapMCPTerminalFinish accepts.
func settleTestAssignment() factory.Assignment {
	id := factory.NewID()
	run := factory.NewID()
	prompt := []byte("build the checklist app")
	sum := sha256.Sum256(prompt)
	acceptance := "d" + strings.Repeat("1", 24)
	return factory.Assignment{
		ID: id, AttemptRoot: id, PublicationAssignment: factory.NewID(),
		ProjectID: "p" + strings.Repeat("2", 24), Role: project.RoleCoder,
		Phase:        project.ExecutionImplement,
		Acceptance:   acceptance,
		Preparation:  "f" + strings.Repeat("3", 24),
		Harness:      "codex",
		HarnessVers:  "1.2.3",
		Model:        "test-model",
		Connection:   "conn",
		SourceCommit: strings.Repeat("c", 40),
		Prompt:       prompt,
		PromptSHA:    hex.EncodeToString(sum[:]),
		Run:          run,
		RunHistory:   []string{run},
		Stage:        factory.AssignmentAssigned,
		Subject:      factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: 3, Acceptance: acceptance}},
		Repository:   7,
		Issue:        3,
		NativeRev:    1,
		Attempts:     1,
		ActorID:      7,
	}
}

// settleTestSubmission binds one terminal payload to an assignment.
func settleTestSubmission(t *testing.T, assignmentID string, tool factory.MCPTool, fields map[string]string) factory.MCPSubmission {
	t.Helper()
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return factory.MCPSubmission{
		AssignmentID: assignmentID, SubmissionID: factory.NewID(),
		Tool: tool, ExpectedInputDigest: strings.Repeat("e", 64),
		PayloadHash: factory.MCPInputDigest(raw), Payload: raw,
	}
}

func TestMapMCPTerminalFinish(t *testing.T) {
	now := time.Now()
	commit := strings.Repeat("a", 40)
	for _, tc := range []struct {
		name    string
		tool    factory.MCPTool
		fields  map[string]string
		raw     []byte
		park    bool
		outcome factory.Outcome
		reason  string
		status  string
	}{
		{name: "incomplete blocked", tool: factory.MCPToolFinishIncomplete, fields: map[string]string{"outcome": "blocked", "cause": "needs input", "evidence_refs": "r", "affected_task_ids": "t", "requested_resolution": "retry"}, outcome: factory.NeedsHuman, reason: factory.AssignReasonNoReport, status: "blocked"},
		{name: "incomplete failed", tool: factory.MCPToolFinishIncomplete, fields: map[string]string{"outcome": "failed", "cause": "boom", "evidence_refs": "r", "affected_task_ids": "t", "requested_resolution": "retry"}, outcome: factory.Failed, reason: factory.AssignReasonRunFailed, status: "failed"},
		{name: "incomplete cancelled", tool: factory.MCPToolFinishIncomplete, fields: map[string]string{"outcome": "cancelled", "cause": "stop", "evidence_refs": "r", "affected_task_ids": "t", "requested_resolution": "none"}, outcome: factory.Cancelled, reason: factory.AssignReasonCancelled, status: "cancelled"},
		{name: "incomplete unknown outcome fences blocked", tool: factory.MCPToolFinishIncomplete, fields: map[string]string{"outcome": "whatever", "cause": "x", "evidence_refs": "r", "affected_task_ids": "t", "requested_resolution": "retry"}, outcome: factory.NeedsHuman, reason: factory.AssignReasonNoReport, status: "blocked"},
		{name: "implementation nominates candidate", tool: factory.MCPToolFinishImplement, fields: map[string]string{"task_id": "t", "manifest_digest": strings.Repeat("b", 64), "output_digest": strings.Repeat("c", 64), "evidence_refs": "r", "candidate": commit}, outcome: factory.Succeeded, reason: factory.AssignReasonReported, status: "completed"},
		{name: "implementation unchanged fences", tool: factory.MCPToolFinishImplement, fields: map[string]string{"task_id": "t", "manifest_digest": strings.Repeat("b", 64), "output_digest": strings.Repeat("c", 64), "evidence_refs": "r", "unchanged": "true"}, outcome: factory.NeedsHuman, reason: factory.AssignReasonNoReport, status: "blocked"},
		{name: "correction nominates candidate", tool: factory.MCPToolFinishCorrection, fields: map[string]string{"finding_digest": strings.Repeat("d", 64), "dispositions": "fix", "evidence_refs": "r", "candidate": commit}, outcome: factory.Succeeded, reason: factory.AssignReasonReported, status: "completed"},
		{name: "correction without candidate fences", tool: factory.MCPToolFinishCorrection, fields: map[string]string{"finding_digest": strings.Repeat("d", 64), "dispositions": "fix", "evidence_refs": "r"}, outcome: factory.NeedsHuman, reason: factory.AssignReasonNoReport, status: "blocked"},
		{name: "review parks", tool: factory.MCPToolFinishReview, fields: map[string]string{"conclusion": "clear", "coverage": "all", "evidence_refs": "r"}, park: true},
		{name: "product plan parks", tool: factory.MCPToolFinishProductPlan, fields: map[string]string{"candidates": "g", "evidence_refs": "r", "coverage": "c"}, park: true},
		{name: "issue plan parks", tool: factory.MCPToolFinishIssuePlan, fields: map[string]string{"evidence_refs": "r", "approach": "a", "tasks": "t"}, park: true},
		{name: "reassessment parks", tool: factory.MCPToolFinishReassessment, fields: map[string]string{"subject": "s", "report": "r"}, park: true},
		{name: "app verification parks", tool: factory.MCPToolFinishAppVerify, fields: map[string]string{"delivery_id": "d", "outcome": "verified", "coverage": "c"}, park: true},
		{name: "garbage payload fences", tool: factory.MCPToolFinishImplement, raw: []byte("{nope"), outcome: factory.NeedsHuman, reason: factory.AssignReasonNoReport, status: "blocked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assigned := settleTestAssignment()
			submission := settleTestSubmission(t, assigned.ID, tc.tool, tc.fields)
			if tc.raw != nil {
				submission.Payload = tc.raw
				submission.PayloadHash = factory.MCPInputDigest(tc.raw)
			}
			finished, _, ok, err := mapMCPTerminalFinish(assigned, submission, now)
			if err != nil {
				t.Fatalf("map: %v", err)
			}
			if tc.park {
				if ok {
					t.Fatal("parked tool settled")
				}
				return
			}
			if !ok {
				t.Fatal("handled tool parked")
			}
			if finished.Outcome != tc.outcome || finished.Reason != tc.reason {
				t.Fatalf("outcome %v/%v, want %v/%v", finished.Outcome, finished.Reason, tc.outcome, tc.reason)
			}
			if finished.Result == nil || finished.Result.Status != tc.status {
				t.Fatalf("result status %+v, want %v", finished.Result, tc.status)
			}
			if finished.Stage != factory.AssignmentFinished || finished.FinishedUnix <= 0 {
				t.Fatal("finish lacks its terminal stage stamp")
			}
			if err := finished.Validate(); err != nil {
				t.Fatalf("settled finish invalid: %v", err)
			}
			if tc.status == "completed" && (!finished.Result.Reported || finished.Result.Candidate != commit) {
				t.Fatalf("nomination lost: %+v", finished.Result)
			}
		})
	}
}

// stubSettleStore is the in-memory MCPTerminalSettleStore: assignment
// and terminal rows plus the receipt-state guard.
type stubSettleStore struct {
	assignment   factory.Assignment
	assignmentOK bool
	assignmentEr error
	submission   factory.MCPSubmission
	terminalOK   bool
	terminalErr  error
	receiptState string
	advanceErr   error
	advances     int
	lastReceipt  factory.MCPReceipt
	finishErr    error
	finished     []factory.Assignment
	byRun        map[string]factory.Assignment
}

// AssignmentByRun serves the repair-chain joint: no recorded failed
// verification exists behind a stub run unless a test records one.
func (s *stubSettleStore) AssignmentByRun(_ context.Context, runID string) (factory.Assignment, error) {
	if s.byRun != nil {
		if a, ok := s.byRun[runID]; ok {
			return a, nil
		}
	}
	return factory.Assignment{}, store.ErrNotFound
}

func (s *stubSettleStore) Assignment(context.Context, string) (factory.Assignment, error) {
	if s.assignmentEr != nil {
		return factory.Assignment{}, s.assignmentEr
	}
	if !s.assignmentOK {
		return factory.Assignment{}, store.ErrNotFound
	}
	return s.assignment, nil
}

func (s *stubSettleStore) TerminalMCPSubmission(context.Context, string) (factory.MCPSubmission, bool, error) {
	if s.terminalErr != nil {
		return factory.MCPSubmission{}, false, s.terminalErr
	}
	return s.submission, s.terminalOK, nil
}

func (s *stubSettleStore) TerminalMCPProposal(context.Context, string) (factory.MCPReceipt, bool, error) {
	if !s.terminalOK {
		return factory.MCPReceipt{}, false, nil
	}
	return factory.MCPReceipt{
		SubmissionID: s.submission.SubmissionID, Tool: s.submission.Tool,
		State: s.receiptState, InputDigest: s.submission.PayloadHash,
	}, true, nil
}

func (s *stubSettleStore) AdvanceMCPReceipt(_ context.Context, _, _ string, expectedState string, receipt factory.MCPReceipt) error {
	if s.advanceErr != nil {
		return s.advanceErr
	}
	if s.receiptState != expectedState {
		return store.ErrStaleRevision
	}
	s.receiptState = receipt.State
	s.lastReceipt = receipt
	s.advances++
	return nil
}

func (s *stubSettleStore) FinishAssignment(_ context.Context, a factory.Assignment) error {
	if s.finishErr != nil {
		return s.finishErr
	}
	s.finished = append(s.finished, a)
	return nil
}

func settleStub(assigned factory.Assignment, tool factory.MCPTool, fields map[string]string) *stubSettleStore {
	raw, _ := json.Marshal(fields)
	submission := factory.MCPSubmission{
		AssignmentID: assigned.ID, SubmissionID: factory.NewID(), Tool: tool,
		ExpectedInputDigest: strings.Repeat("e", 64),
		PayloadHash:         factory.MCPInputDigest(raw), Payload: raw,
	}
	return &stubSettleStore{
		assignment: assigned, assignmentOK: true,
		submission: submission, terminalOK: true,
		receiptState: factory.MCPReceiptRecorded,
	}
}

func TestSettleMCPTerminalProposal(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	commit := strings.Repeat("a", 40)

	t.Run("implementation success finishes and accepts", func(t *testing.T) {
		assigned := settleTestAssignment()
		stub := settleStub(assigned, factory.MCPToolFinishImplement,
			map[string]string{"task_id": "t", "manifest_digest": strings.Repeat("b", 64), "output_digest": strings.Repeat("c", 64), "evidence_refs": "r", "candidate": commit})
		settled, err := SettleMCPTerminalProposal(ctx, stub, assigned.ID, now)
		if err != nil || !settled {
			t.Fatalf("settled=%v err=%v", settled, err)
		}
		if len(stub.finished) != 1 {
			t.Fatalf("finishes=%d", len(stub.finished))
		}
		got := stub.finished[0]
		if got.Outcome != factory.Succeeded || got.Reason != factory.AssignReasonReported || got.Result.Candidate != commit {
			t.Fatalf("finish %+v", got)
		}
		if stub.lastReceipt.State != factory.MCPReceiptAccepted || stub.lastReceipt.SubmissionID != stub.submission.SubmissionID {
			t.Fatalf("receipt %+v", stub.lastReceipt)
		}
	})

	t.Run("no terminal parks without writes", func(t *testing.T) {
		assigned := settleTestAssignment()
		stub := settleStub(assigned, factory.MCPToolFinishImplement, map[string]string{})
		stub.terminalOK = false
		settled, err := SettleMCPTerminalProposal(ctx, stub, assigned.ID, now)
		if err != nil || settled {
			t.Fatalf("settled=%v err=%v", settled, err)
		}
		if len(stub.finished) != 0 || stub.advances != 0 {
			t.Fatal("parked proposal wrote")
		}
	})

	t.Run("parked tool touches nothing", func(t *testing.T) {
		assigned := settleTestAssignment()
		stub := settleStub(assigned, factory.MCPToolFinishReview,
			map[string]string{"conclusion": "clear", "coverage": "all", "evidence_refs": "r"})
		settled, err := SettleMCPTerminalProposal(ctx, stub, assigned.ID, now)
		if err != nil || settled {
			t.Fatalf("settled=%v err=%v", settled, err)
		}
		if len(stub.finished) != 0 || stub.advances != 0 {
			t.Fatal("parked review wrote")
		}
	})

	t.Run("finished assignment consumes its recorded receipt", func(t *testing.T) {
		assigned := settleTestAssignment()
		assigned.Stage = factory.AssignmentFinished
		stub := settleStub(assigned, factory.MCPToolFinishImplement, map[string]string{})
		settled, err := SettleMCPTerminalProposal(ctx, stub, assigned.ID, now)
		if err != nil || !settled {
			t.Fatalf("settled=%v err=%v", settled, err)
		}
		if len(stub.finished) != 0 {
			t.Fatal("already-finished assignment finished again")
		}
		if stub.lastReceipt.State != factory.MCPReceiptAccepted {
			t.Fatalf("receipt %+v", stub.lastReceipt)
		}
	})

	t.Run("concurrent foreign finish consumes the proposal", func(t *testing.T) {
		assigned := settleTestAssignment()
		stub := settleStub(assigned, factory.MCPToolFinishIncomplete,
			map[string]string{"outcome": "blocked", "cause": "x", "evidence_refs": "r", "affected_task_ids": "t", "requested_resolution": "retry"})
		stub.finishErr = store.ErrNotFound
		settled, err := SettleMCPTerminalProposal(ctx, stub, assigned.ID, now)
		if err != nil || !settled {
			t.Fatalf("settled=%v err=%v", settled, err)
		}
		if !strings.Contains(stub.lastReceipt.Reason, "during settlement") {
			t.Fatalf("receipt %+v", stub.lastReceipt)
		}
	})

	t.Run("transient finish error retries without touching the receipt", func(t *testing.T) {
		assigned := settleTestAssignment()
		stub := settleStub(assigned, factory.MCPToolFinishIncomplete,
			map[string]string{"outcome": "failed", "cause": "x", "evidence_refs": "r", "affected_task_ids": "t", "requested_resolution": "retry"})
		stub.finishErr = errors.New("connection reset")
		settled, err := SettleMCPTerminalProposal(ctx, stub, assigned.ID, now)
		if err == nil || settled {
			t.Fatalf("settled=%v err=%v", settled, err)
		}
		if stub.advances != 0 {
			t.Fatal("transient error moved the receipt")
		}
	})

	t.Run("consumed receipt stays settled", func(t *testing.T) {
		assigned := settleTestAssignment()
		stub := settleStub(assigned, factory.MCPToolFinishIncomplete,
			map[string]string{"outcome": "blocked", "cause": "x", "evidence_refs": "r", "affected_task_ids": "t", "requested_resolution": "retry"})
		stub.receiptState = factory.MCPReceiptAccepted
		settled, err := SettleMCPTerminalProposal(ctx, stub, assigned.ID, now)
		if err != nil || settled {
			t.Fatalf("settled=%v err=%v", settled, err)
		}
		if len(stub.finished) != 0 {
			t.Fatal("consumed proposal finished again")
		}
	})

	t.Run("bad assignment refuses", func(t *testing.T) {
		if _, err := SettleMCPTerminalProposal(ctx, settleStub(settleTestAssignment(), factory.MCPToolFinishIncomplete, map[string]string{}), "nope", now); err == nil {
			t.Fatal("bad assignment admitted")
		}
	})
}
