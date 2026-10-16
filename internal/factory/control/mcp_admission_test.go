package control

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// errJournalNotFound is the stub's unrecorded-identity signal.
// AdmitMCPSubmission treats any lookup error as "not a replay" and
// still runs the freeze and freshness gates below.
var errJournalNotFound = errors.New("stub journal: submission not recorded")

// FW38 proves T48/T51 admission without a database: cross-role,
// cross-run, expired, revoked and stopped calls refuse; terminal
// freeze holds; the handoff cap reserves terminal capacity.

type stubMCPJournal struct {
	terminal factory.MCPReceipt
	frozen   bool
	handoffs int
	recorded map[string]factory.MCPReceipt
}

func (s stubMCPJournal) TerminalMCPProposal(context.Context, string) (factory.MCPReceipt, bool, error) {
	return s.terminal, s.frozen, nil
}

func (s stubMCPJournal) MCPReceipt(_ context.Context, assignmentID, submissionID string) (factory.MCPReceipt, error) {
	if receipt, ok := s.recorded[assignmentID+"/"+submissionID]; ok {
		return receipt, nil
	}
	return factory.MCPReceipt{}, errJournalNotFound
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

// testMCPSubmission builds a submission against the capability's
// current admitted scope: CapabilityID binds the server-admitted
// capability, ExpectedInputDigest binds its scope digest,
// PayloadHash binds the payload bytes.
func testMCPSubmission(capability factory.MCPCapability, tool factory.MCPTool, id string, payload []byte) factory.MCPSubmission {
	return factory.MCPSubmission{
		AssignmentID: capability.AssignmentID, SubmissionID: id,
		CapabilityID: capability.ID, Tool: tool,
		ExpectedInputDigest: capability.ScopeDigest(),
		PayloadHash:         factory.MCPInputDigest(payload), Payload: payload,
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
	terminal := factory.MCPReceipt{
		SubmissionID: submission.SubmissionID, Tool: submission.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
	}
	frozen := stubMCPJournal{frozen: true, terminal: terminal}
	// After the terminal records, exact replay of the terminal ID
	// stays allowed while conflicting terminal and nonterminal
	// identities refuse.
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
	// Cross-capability submission refuses: the bound capability ID
	// must be the admitted one.
	borrowed := testMCPSubmission(capability, factory.MCPToolStatus, factory.NewID(), payload)
	borrowed.CapabilityID = factory.NewID()
	if err := AdmitMCPSubmission(context.Background(), journal, capability, borrowed); err == nil {
		t.Fatal("cross-capability submission admitted")
	}
}

// TestAdmitMCPSubmissionReplayRoutes proves the F3 replay half:
// exact replay of a recorded nonterminal ID and the same terminal
// content under another ID route to store replay instead of
// refusing after the freeze.
func TestAdmitMCPSubmissionReplayRoutes(t *testing.T) {
	capability := testMCPCapability()
	ctx := context.Background()
	outputPayload := []byte(`{"task_id":"t1"}`)
	output := testMCPSubmission(capability, factory.MCPToolSubmitTaskOutput, factory.NewID(), outputPayload)
	terminalPayload := []byte(`{"outcome":"done"}`)
	terminalSubmission := testMCPSubmission(capability, factory.MCPToolFinishImplement, factory.NewID(), terminalPayload)
	terminal := factory.MCPReceipt{
		SubmissionID: terminalSubmission.SubmissionID, Tool: terminalSubmission.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: terminalSubmission.PayloadHash,
	}
	frozen := stubMCPJournal{
		frozen:   true,
		terminal: terminal,
		recorded: map[string]factory.MCPReceipt{
			capability.AssignmentID + "/" + output.SubmissionID: {
				SubmissionID: output.SubmissionID, Tool: output.Tool,
				State: factory.MCPReceiptRecorded, InputDigest: output.PayloadHash,
			},
		},
	}
	// Previously recorded nonterminal ID replays after the freeze.
	if err := AdmitMCPSubmission(ctx, frozen, capability, output); err != nil {
		t.Fatalf("recorded nonterminal replay refused: %v", err)
	}
	// Same terminal content under another ID routes to replay.
	clone := testMCPSubmission(capability, terminalSubmission.Tool, factory.NewID(), terminalPayload)
	if err := AdmitMCPSubmission(ctx, frozen, capability, clone); err != nil {
		t.Fatalf("same-terminal-content replay refused: %v", err)
	}
	// Same tool but different terminal content still refuses.
	rival := testMCPSubmission(capability, terminalSubmission.Tool, factory.NewID(), []byte(`{"outcome":"blocked"}`))
	if err := AdmitMCPSubmission(ctx, frozen, capability, rival); err == nil {
		t.Fatal("conflicting terminal content admitted")
	}
}

// TestAdmitMCPSubmissionFreshness proves the F3 freshness half:
// new submissions must be built against the capability's current
// admitted inputs; stale snapshots refuse while replay of a
// recorded ID never faces the freshness gate.
func TestAdmitMCPSubmissionFreshness(t *testing.T) {
	capability := testMCPCapability()
	ctx := context.Background()
	payload := []byte(`{"task_id":"t1"}`)
	stale := testMCPSubmission(capability, factory.MCPToolSubmitTaskOutput, factory.NewID(), payload)
	stale.ExpectedInputDigest = strings.Repeat("b", 64)
	if err := AdmitMCPSubmission(ctx, stubMCPJournal{}, capability, stale); err == nil {
		t.Fatal("stale-input submission admitted")
	} else if !errors.Is(err, ErrMCPSubmissionStale) {
		t.Fatalf("stale refusal = %v, want stale-input refusal", err)
	}
	fresh := testMCPSubmission(capability, factory.MCPToolSubmitTaskOutput, factory.NewID(), payload)
	if err := AdmitMCPSubmission(ctx, stubMCPJournal{}, capability, fresh); err != nil {
		t.Fatalf("current-input submission refused: %v", err)
	}
	// A recorded ID replays even when its inputs have rotated since:
	// history is immutable, not fresh.
	rotated := capability
	rotated.PreparationDigest = strings.Repeat("c", 64)
	replay := testMCPSubmission(capability, factory.MCPToolSubmitTaskOutput, fresh.SubmissionID, payload)
	journal := stubMCPJournal{recorded: map[string]factory.MCPReceipt{
		capability.AssignmentID + "/" + fresh.SubmissionID: {
			SubmissionID: fresh.SubmissionID, Tool: fresh.Tool,
			State: factory.MCPReceiptRecorded, InputDigest: fresh.PayloadHash,
		},
	}}
	if err := AdmitMCPSubmission(ctx, journal, rotated, replay); err != nil {
		t.Fatalf("post-rotation replay refused: %v", err)
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
