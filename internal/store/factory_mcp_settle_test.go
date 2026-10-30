package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

// TestRecordedTerminalMCPAssignments proves the settlement work list
// against real PostgreSQL (SODA_PG_*; skips without it): only
// assignments with a recorded terminal proposal list, non-terminal
// traffic never lists, and an advanced receipt drops off the list.
func TestRecordedTerminalMCPAssignments(t *testing.T) {
	db, _ := postgresFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for _, stmt := range []string{
		FactoryMCPCapabilitiesDDL, FactoryMCPCapabilityDeleteTrigger,
		FactoryMCPSubmissionsDDL, FactoryMCPSubmissionTriggers,
		FactoryMCPAssignmentGuardDDL,
	} {
		if _, err := db.db.ExecContext(ctx, stmt); err != nil && !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("MCP DDL: %v", err)
		}
	}
	active := activateMCPCapability(t, ctx, db, testMCPCapabilityRow())
	traffic := testMCPSubmissionRow(active, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-1"}`))
	if _, err := db.RecordMCPSubmission(ctx, traffic); err != nil {
		t.Fatalf("record task output: %v", err)
	}
	ids, err := db.RecordedTerminalMCPAssignments(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("non-terminal traffic listed: %v", ids)
	}
	payload := []byte(`{"candidate":"` + strings.Repeat("a", 40) + `"}`)
	terminal := testMCPSubmissionRow(active, factory.MCPToolFinishImplement, payload)
	if _, err := db.RecordMCPSubmission(ctx, terminal); err != nil {
		t.Fatalf("record terminal: %v", err)
	}
	ids, err = db.RecordedTerminalMCPAssignments(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != active.AssignmentID {
		t.Fatalf("recorded terminal list = %v, want [%s]", ids, active.AssignmentID)
	}
	if _, err := db.RecordedTerminalMCPAssignments(ctx, 0); err == nil {
		t.Fatal("unbounded listing admitted")
	}
	accepted := factory.MCPReceipt{
		SubmissionID: terminal.SubmissionID, Tool: terminal.Tool,
		State: factory.MCPReceiptAccepted, InputDigest: terminal.PayloadHash,
		Reason: "settled in test",
	}
	if err := db.AdvanceMCPReceipt(ctx, active.AssignmentID, terminal.SubmissionID, factory.MCPReceiptRecorded, accepted); err != nil {
		t.Fatalf("advance receipt: %v", err)
	}
	ids, err = db.RecordedTerminalMCPAssignments(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("consumed terminal still listed: %v", ids)
	}
}
