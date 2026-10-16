package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// FW38 proves T48/T51 admission without a database: cross-role,
// cross-run, expired, revoked and stopped calls refuse; terminal
// freeze holds; the handoff cap reserves terminal capacity.

type stubMCPJournal struct {
	terminal factory.MCPReceipt
	frozen   bool
	handoffs int
}

func (s stubMCPJournal) TerminalMCPProposal(context.Context, string) (factory.MCPReceipt, bool, error) {
	return s.terminal, s.frozen, nil
}

func (s stubMCPJournal) MCPHandoffCount(context.Context, string) (int, error) {
	return s.handoffs, nil
}

func testMCPCapability() factory.MCPCapability {
	digest := strings.Repeat("a", 64)
	return factory.MCPCapability{
		ID: factory.NewID(), AssignmentID: factory.NewID(),
		RunID: "run-1", Generation: 1,
		Profile: factory.MCPProfileBuilder, Subject: "implement:42", Repository: 7,
		Project:           "p" + strings.Repeat("d", 24),
		PreparationDigest: digest, SourceDigest: "abc", AcceptedRevision: "rev-1",
		Authority: digest, DeadlineUnix: time.Now().Add(time.Hour).Unix(),
		State: factory.MCPCapabilityActive,
	}
}

func testMCPObserved(capability factory.MCPCapability) MCPRunObservation {
	return MCPRunObservation{
		RunID: capability.RunID, Incarnation: 3,
		AssignmentID: capability.AssignmentID, Authority: capability.Authority,
	}
}

func TestAdmitMCPCapability(t *testing.T) {
	capability := testMCPCapability()
	now := time.Now()
	if err := AdmitMCPCapability(capability, factory.MCPToolFinishImplement, now, testMCPObserved(capability)); err != nil {
		t.Fatalf("admitted call refused: %v", err)
	}
	// Cross-role direct call refuses even though the tool exists.
	if err := AdmitMCPCapability(capability, factory.MCPToolFinishReview, now, testMCPObserved(capability)); err == nil {
		t.Fatal("cross-role direct call admitted")
	}
	// Undiscoverable tool refuses server-side.
	if err := AdmitMCPCapability(capability, "factory.merge", now, testMCPObserved(capability)); err == nil {
		t.Fatal("unknown tool admitted")
	}
	// Cross-run/incarnation binding refuses.
	observed := testMCPObserved(capability)
	observed.RunID = "run-2"
	if err := AdmitMCPCapability(capability, factory.MCPToolStatus, now, observed); err == nil {
		t.Fatal("cross-run call admitted")
	}
	observed = testMCPObserved(capability)
	observed.AssignmentID = factory.NewID()
	if err := AdmitMCPCapability(capability, factory.MCPToolStatus, now, observed); err == nil {
		t.Fatal("cross-assignment call admitted")
	}
	// Expired capability refuses; rate limits never extend it.
	expired := capability
	expired.DeadlineUnix = now.Add(-time.Minute).Unix()
	if err := AdmitMCPCapability(expired, factory.MCPToolStatus, now, testMCPObserved(expired)); err == nil {
		t.Fatal("expired capability admitted")
	}
	// Staged, revoked and retired capabilities admit nothing.
	for _, state := range []string{factory.MCPCapabilityStaged, factory.MCPCapabilityRevoked, factory.MCPCapabilityRetired} {
		revoked := capability
		revoked.State = state
		if err := AdmitMCPCapability(revoked, factory.MCPToolStatus, now, testMCPObserved(revoked)); err == nil {
			t.Fatalf("capability in state %s admitted", state)
		}
	}
	// Stop intent observed before admission permanently prevents it.
	stopped := testMCPObserved(capability)
	stopped.Stopped = true
	if err := AdmitMCPCapability(capability, factory.MCPToolStatus, now, stopped); err == nil {
		t.Fatal("stopped run admitted")
	}
}

func testMCPSubmission(capability factory.MCPCapability, tool factory.MCPTool, id string, payload []byte) factory.MCPSubmission {
	hash := factory.MCPInputDigest(payload)
	return factory.MCPSubmission{
		AssignmentID: capability.AssignmentID, SubmissionID: id, Tool: tool,
		ExpectedInputDigest: hash, PayloadHash: hash, Payload: payload,
	}
}

func TestAdmitMCPSubmissionFreeze(t *testing.T) {
	capability := testMCPCapability()
	payload := []byte(`{"outcome":"done"}`)
	submission := testMCPSubmission(capability, factory.MCPToolFinishImplement, factory.NewID(), payload)
	journal := stubMCPJournal{}
	if err := AdmitMCPSubmission(context.Background(), journal, capability, submission); err != nil {
		t.Fatalf("first terminal refused: %v", err)
	}
	// After the terminal records, exact replay stays allowed while
	// new terminal and nonterminal identities refuse.
	frozen := stubMCPJournal{frozen: true, terminal: factory.MCPReceipt{
		SubmissionID: submission.SubmissionID, Tool: submission.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
	}}
	if err := AdmitMCPSubmission(context.Background(), frozen, capability, submission); err != nil {
		t.Fatalf("terminal replay refused: %v", err)
	}
	second := testMCPSubmission(capability, factory.MCPToolFinishCorrection, factory.NewID(), payload)
	if err := AdmitMCPSubmission(context.Background(), frozen, capability, second); err == nil {
		t.Fatal("second terminal identity admitted")
	}
	late := testMCPSubmission(capability, factory.MCPToolSubmitTaskOutput, factory.NewID(), []byte(`{"task_id":"t1"}`))
	if err := AdmitMCPSubmission(context.Background(), frozen, capability, late); err == nil {
		t.Fatal("post-terminal output admitted")
	}
	// Cross-assignment submission refuses.
	foreign := testMCPSubmission(capability, factory.MCPToolStatus, factory.NewID(), payload)
	foreign.AssignmentID = factory.NewID()
	if err := AdmitMCPSubmission(context.Background(), journal, capability, foreign); err == nil {
		t.Fatal("cross-assignment submission admitted")
	}
}

func TestAdmitMCPHandoffCap(t *testing.T) {
	capability := testMCPCapability()
	payload := []byte(`{"predecessor_task":"a","successor_task":"b"}`)
	full := stubMCPJournal{handoffs: 256}
	capped := testMCPSubmission(capability, factory.MCPToolRequestHandoff, factory.NewID(), payload)
	if err := AdmitMCPSubmission(context.Background(), full, capability, capped); err == nil {
		t.Fatal("257th handoff admitted")
	}
	room := stubMCPJournal{handoffs: 255}
	if err := AdmitMCPSubmission(context.Background(), room, capability, capped); err != nil {
		t.Fatalf("256th handoff refused: %v", err)
	}
}
