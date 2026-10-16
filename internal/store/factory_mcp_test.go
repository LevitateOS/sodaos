package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// FW38 freezes the D14 capability/submission storage contract: the
// table owners, the idempotency keys and the immutability guards.
// Pure checks run everywhere; the transaction proof needs the
// disposable SODA_PG_* fixture and skips without it.
func TestFactoryMCPSchemaContract(t *testing.T) {
	if TableFactoryMCPCapabilities != "factory_mcp_capabilities" ||
		TableFactoryMCPSubmissions != "factory_mcp_submissions" {
		t.Fatal("MCP table owners renamed")
	}
	for _, stmt := range []string{FactoryMCPCapabilitiesDDL, FactoryMCPSubmissionsDDL} {
		if !strings.Contains(stmt, "JSONB NOT NULL") {
			t.Fatalf("MCP table outside its storage contract: %.60s", stmt)
		}
	}
	if !strings.Contains(FactoryMCPSubmissionsDDL, "PRIMARY KEY(assignment,submission)") {
		t.Fatal("submission idempotency key missing")
	}
	if !strings.Contains(FactoryMCPCapabilitiesDDL, "PRIMARY KEY") || !strings.Contains(FactoryMCPCapabilitiesDDL, "CHECK(id ~") {
		t.Fatal("capability identity key missing")
	}
	for _, trigger := range []string{FactoryMCPCapabilityDeleteTrigger, FactoryMCPSubmissionTriggers} {
		if !strings.Contains(trigger, "soda_reject_immutable()") {
			t.Fatalf("MCP history guard missing: %.60s", trigger)
		}
	}
}

func testMCPCapabilityRow() factory.MCPCapability {
	digest := strings.Repeat("a", 64)
	return factory.MCPCapability{
		ID: factory.NewID(), AssignmentID: factory.NewID(), RunID: "run-1", Generation: 1,
		Profile: factory.MCPProfileBuilder, Subject: "implement:42", Repository: 7,
		Project:           "p" + strings.Repeat("d", 24),
		PreparationDigest: digest, SourceDigest: "abc", AcceptedRevision: "rev-1",
		Authority: digest, DeadlineUnix: time.Now().Add(time.Hour).Unix(),
		State: factory.MCPCapabilityStaged,
	}
}

func testMCPSubmissionRow(assignmentID string, tool factory.MCPTool, payload []byte) factory.MCPSubmission {
	hash := factory.MCPInputDigest(payload)
	return factory.MCPSubmission{
		AssignmentID: assignmentID, SubmissionID: factory.NewID(), Tool: tool,
		ExpectedInputDigest: hash, PayloadHash: hash, Payload: payload,
	}
}

func testMCPRecordedReceipt(submission factory.MCPSubmission) factory.MCPReceipt {
	return factory.MCPReceipt{
		SubmissionID: submission.SubmissionID, Tool: submission.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
	}
}

// TestFactoryMCPTransactions proves the T48/T51 store joints against
// real PostgreSQL (SODA_PG_*; skips without it): staged replay
// identity, scope-conflict refusal, CAS activation binding, revoke
// and retire transitions, exact-replay receipts, changed-bytes
// conflicts, terminal freezing and handoff counting.
func TestFactoryMCPTransactions(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, stmt := range []string{
		FactoryMCPCapabilitiesDDL, FactoryMCPCapabilityDeleteTrigger,
		FactoryMCPSubmissionsDDL, FactoryMCPSubmissionTriggers,
	} {
		if _, err := db.db.ExecContext(ctx, stmt); err != nil && !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("MCP DDL: %v", err)
		}
	}

	capability := testMCPCapabilityRow()
	if err := db.StageMCPCapability(ctx, capability); err != nil {
		t.Fatalf("stage: %v", err)
	}
	// Exact restart delivery replays identically.
	if err := db.StageMCPCapability(ctx, capability); err != nil {
		t.Fatalf("staged replay refused: %v", err)
	}
	// Changed scope under the same ID conflicts.
	clash := capability
	clash.Generation = 2
	if err := db.StageMCPCapability(ctx, clash); err == nil {
		t.Fatal("conflicting capability scope staged")
	}
	// Activation binds the observed run state.
	now := time.Now().Unix()
	if err := db.ActivateMCPCapability(ctx, capability.ID, 99, capability.RunID, capability.AssignmentID, capability.Authority, now, false); err == nil {
		t.Fatal("wrong-generation activation admitted")
	}
	if err := db.ActivateMCPCapability(ctx, capability.ID, capability.Generation, "run-2", capability.AssignmentID, capability.Authority, now, false); err == nil {
		t.Fatal("cross-run activation admitted")
	}
	if err := db.ActivateMCPCapability(ctx, capability.ID, capability.Generation, capability.RunID, capability.AssignmentID, capability.Authority, now, true); err == nil {
		t.Fatal("stopped activation admitted")
	}
	if err := db.ActivateMCPCapability(ctx, capability.ID, capability.Generation, capability.RunID, capability.AssignmentID, capability.Authority, now, false); err != nil {
		t.Fatalf("activation refused: %v", err)
	}
	got, err := db.GetMCPCapability(ctx, capability.ID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	if got.State != factory.MCPCapabilityActive || got.ScopeDigest() != capability.ScopeDigest() {
		t.Fatal("activated capability lost state or scope")
	}
	// Second activation refuses: only staged activates.
	if err := db.ActivateMCPCapability(ctx, capability.ID, capability.Generation, capability.RunID, capability.AssignmentID, capability.Authority, now, false); err == nil {
		t.Fatal("double activation admitted")
	}

	// Immutable submissions: exact replay, conflict, recovery.
	payload := []byte(`{"outcome":"done"}`)
	submission := testMCPSubmissionRow(capability.AssignmentID, factory.MCPToolFinishImplement, payload)
	first, err := db.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	replay, err := db.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
	if err != nil {
		t.Fatalf("exact replay refused: %v", err)
	}
	if replay != first {
		t.Fatal("replay receipt differs from recorded receipt")
	}
	changed := submission
	changed.Payload = []byte(`{"outcome":"blocked"}`)
	changed.PayloadHash = factory.MCPInputDigest(changed.Payload)
	changed.ExpectedInputDigest = changed.PayloadHash
	if _, err := db.RecordMCPSubmission(ctx, changed, testMCPRecordedReceipt(changed)); err == nil {
		t.Fatal("changed bytes under recorded ID admitted")
	}
	recovered, err := db.MCPReceipt(ctx, capability.AssignmentID, submission.SubmissionID)
	if err != nil {
		t.Fatalf("receipt recovery: %v", err)
	}
	if recovered != first {
		t.Fatal("recovered receipt differs from recorded receipt")
	}
	if _, err := db.MCPReceipt(ctx, capability.AssignmentID, factory.NewID()); err == nil {
		t.Fatal("unknown receipt disclosed")
	}
	// Terminal freezes new identities; same terminal content under
	// another ID returns the existing receipt without another record.
	terminal, frozen, err := db.TerminalMCPProposal(ctx, capability.AssignmentID)
	if err != nil || !frozen || terminal.SubmissionID != submission.SubmissionID {
		t.Fatalf("terminal proposal = %v, frozen = %v, err = %v", terminal, frozen, err)
	}
	second := testMCPSubmissionRow(capability.AssignmentID, factory.MCPToolFinishCorrection, payload)
	if _, err := db.RecordMCPSubmission(ctx, second, testMCPRecordedReceipt(second)); err == nil {
		t.Fatal("second terminal identity recorded")
	}
	// Post-terminal nonterminal identities refuse atomically: the
	// assignment is settled.
	late := testMCPSubmissionRow(capability.AssignmentID, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t1"}`))
	if _, err := db.RecordMCPSubmission(ctx, late, testMCPRecordedReceipt(late)); err == nil {
		t.Fatal("post-terminal output recorded")
	}
	// Handoff counting for the admission cap, on a live assignment.
	live := factory.NewID()
	edge := testMCPSubmissionRow(live, factory.MCPToolRequestHandoff, []byte(`{"predecessor_task":"a","successor_task":"b"}`))
	if _, err := db.RecordMCPSubmission(ctx, edge, testMCPRecordedReceipt(edge)); err != nil {
		t.Fatalf("live handoff refused: %v", err)
	}
	if count, err := db.MCPHandoffCount(ctx, live); err != nil || count != 1 {
		t.Fatalf("handoff count = %d, err = %v", count, err)
	}
	// Revoke then retire transitions.
	if err := db.RetireMCPCapability(ctx, capability.ID); err != nil {
		t.Fatalf("retire: %v", err)
	}
	retired, err := db.GetMCPCapability(ctx, capability.ID)
	if err != nil || retired.State != factory.MCPCapabilityRetired {
		t.Fatalf("retired state = %v, err = %v", retired.State, err)
	}
	if err := db.RevokeMCPCapability(ctx, capability.ID); err == nil {
		t.Fatal("revoke after retire admitted")
	}
}
