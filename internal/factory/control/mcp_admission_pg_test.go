package control

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// TestAdmitMCPSubmissionReplayEndToEnd proves F3 against real
// PostgreSQL (SODA_PG_*; skips without it): admission routes exact
// replay and same-terminal-content to store replay instead of
// refusing, refuses conflicting identities at both layers, binds
// fresh submissions to the capability scope digest, and refuses a
// persistence that a revocation overtook after admission. Store
// persistence is the backstop throughout: every admitted replay is
// also persisted and compared byte-for-byte.
func TestAdmitMCPSubmissionReplayEndToEnd(t *testing.T) {
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG_* fixture unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	s, _, cleanup, err := store.OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	// The guard table arrives through schema startup (v53):
	// OpenEphemeral already installs it. No fixture DDL here,
	// never request-time DDL in production.

	// The assignment's capability starts staged, activates, and is
	// read back active: persistence only records under an active
	// row.
	staged := testMCPCapability()
	staged.State = factory.MCPCapabilityStaged
	if err := s.StageMCPCapability(ctx, staged); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := s.ActivateMCPCapability(ctx, staged.ID, staged.Generation, staged.RunID,
		staged.AssignmentID, staged.Authority, time.Now().Unix(), false); err != nil {
		t.Fatalf("activate: %v", err)
	}
	capability, err := s.GetMCPCapability(ctx, staged.ID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	admit := func(submission factory.MCPSubmission) error {
		return AdmitMCPSubmission(ctx, s, capability, submission)
	}
	persist := func(submission factory.MCPSubmission) factory.MCPReceipt {
		t.Helper()
		receipt := factory.MCPReceipt{
			SubmissionID: submission.SubmissionID, Tool: submission.Tool,
			State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
		}
		recorded, err := s.RecordMCPSubmission(ctx, submission, receipt)
		if err != nil {
			t.Fatalf("record %s: %v", submission.SubmissionID, err)
		}
		return recorded
	}
	fresh := func(tool factory.MCPTool, payload []byte) factory.MCPSubmission {
		return testMCPSubmission(capability, tool, factory.NewID(), payload)
	}

	// Fresh nonterminal output admits and records.
	output := fresh(factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t1"}`))
	if err := admit(output); err != nil {
		t.Fatalf("fresh output refused: %v", err)
	}
	first := persist(output)
	// Exact replay admits and returns the identical receipt.
	if err := admit(output); err != nil {
		t.Fatalf("exact replay refused: %v", err)
	}
	if replay := persist(output); replay != first {
		t.Fatal("replay receipt differs from recorded receipt")
	}
	// A stale scope digest refuses on a new identity.
	stale := fresh(factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t2"}`))
	stale.ExpectedInputDigest = strings.Repeat("b", 64)
	if err := admit(stale); !errors.Is(err, ErrMCPSubmissionStale) {
		t.Fatalf("stale admission = %v, want stale-input refusal", err)
	}
	// Terminal records and freezes the assignment.
	terminal := fresh(factory.MCPToolFinishImplement, []byte(`{"candidate":"cand-1"}`))
	if err := admit(terminal); err != nil {
		t.Fatalf("terminal refused: %v", err)
	}
	terminalReceipt := persist(terminal)
	// Post-freeze, the previously recorded nonterminal ID replays
	// end-to-end with its identical receipt.
	if err := admit(output); err != nil {
		t.Fatalf("post-freeze recorded replay refused: %v", err)
	}
	if replay := persist(output); replay != first {
		t.Fatal("post-freeze replay receipt differs from recorded receipt")
	}
	// Post-freeze, the same terminal content under another ID
	// returns the terminal receipt without another record.
	clone := fresh(factory.MCPToolFinishImplement, []byte(`{"candidate":"cand-1"}`))
	if err := admit(clone); err != nil {
		t.Fatalf("same-content replay refused: %v", err)
	}
	if got := persist(clone); got != terminalReceipt {
		t.Fatalf("same-content receipt %+v differs from terminal %+v", got, terminalReceipt)
	}
	// Conflicting terminal content refuses at admission, and a call
	// forged past admission still refuses at persistence.
	rival := fresh(factory.MCPToolFinishImplement, []byte(`{"candidate":"cand-2"}`))
	if err := admit(rival); !errors.Is(err, ErrMCPSubmissionFrozen) {
		t.Fatalf("rival admission = %v, want frozen refusal", err)
	}
	rivalReceipt := factory.MCPReceipt{
		SubmissionID: rival.SubmissionID, Tool: rival.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: rival.PayloadHash,
	}
	if _, err := s.RecordMCPSubmission(ctx, rival, rivalReceipt); !errors.Is(err, store.ErrMCPSubmissionFrozen) {
		t.Fatalf("rival persistence = %v, want frozen refusal", err)
	}
	// Post-freeze nonterminal new identities refuse at admission.
	late := fresh(factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t9"}`))
	if err := admit(late); !errors.Is(err, ErrMCPSubmissionFrozen) {
		t.Fatalf("late admission = %v, want frozen refusal", err)
	}
	// Exactly two rows exist: no replay or refusal recorded
	// anything further.
	if count, err := s.MCPSubmissionCount(ctx, capability.AssignmentID); err != nil || count != 2 {
		t.Fatalf("submission count = %d, err = %v, want 2", count, err)
	}
	// Paused admission on a live assignment: a revocation landing
	// between admission and persistence refuses the persistence at
	// the capability recheck, recording nothing.
	live := testMCPCapability()
	live.State = factory.MCPCapabilityStaged
	if err := s.StageMCPCapability(ctx, live); err != nil {
		t.Fatalf("live stage: %v", err)
	}
	if err := s.ActivateMCPCapability(ctx, live.ID, live.Generation, live.RunID,
		live.AssignmentID, live.Authority, time.Now().Unix(), false); err != nil {
		t.Fatalf("live activate: %v", err)
	}
	liveActive, err := s.GetMCPCapability(ctx, live.ID)
	if err != nil {
		t.Fatalf("live readback: %v", err)
	}
	paused := testMCPSubmission(liveActive, factory.MCPToolSubmitTaskOutput, factory.NewID(), []byte(`{"task_id":"t10"}`))
	if err := AdmitMCPSubmission(ctx, s, liveActive, paused); err != nil {
		t.Fatalf("pre-revoke admission refused: %v", err)
	}
	if err := s.RevokeMCPCapability(ctx, live.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	pausedReceipt := factory.MCPReceipt{
		SubmissionID: paused.SubmissionID, Tool: paused.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: paused.PayloadHash,
	}
	if _, err := s.RecordMCPSubmission(ctx, paused, pausedReceipt); !errors.Is(err, store.ErrMCPSubmissionCapabilityInactive) {
		t.Fatalf("post-revoke persistence = %v, want capability-inactive refusal", err)
	}
	if count, err := s.MCPSubmissionCount(ctx, live.AssignmentID); err != nil || count != 0 {
		t.Fatalf("live submission count = %d, err = %v, want no rows", count, err)
	}
}
