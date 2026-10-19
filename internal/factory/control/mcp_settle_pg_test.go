package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

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
