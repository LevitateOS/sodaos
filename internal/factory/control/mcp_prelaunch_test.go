package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func TestMCPPreparedBeforeHostLaunchAndStopFencesCalls(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := producerTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	deps := fx.deps()
	deps.MCPKey = []byte(strings.Repeat("k", 32))
	oldLaunch := fx.host.launch
	var bound factory.MCPCapability
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		if in.MCP == nil {
			t.Fatal("host started without prepared MCP")
		}
		cap, err := db.GetMCPCapability(ctx, in.MCP.Capability)
		if err != nil {
			t.Fatal(err)
		}
		if cap.State != factory.MCPCapabilityStaged || cap.RunID != in.Run.ID {
			t.Fatal("MCP was not staged for this run before host launch")
		}
		expected, err := factory.DeriveMCPCapabilitySecret(deps.MCPKey, cap)
		if err != nil || in.MCP.Secret != expected {
			t.Fatal("host received another capability binding")
		}
		if strings.Contains(string(in.Prompt), in.MCP.Secret) {
			t.Fatal("MCP secret leaked into model prompt")
		}
		bound = cap
		return oldLaunch(in)
	}
	report := DispatchPass(ctx, deps)
	if len(report.Launched) != 1 {
		t.Fatalf("launch=%+v errors=%+v waits=%+v", report.Launched, report.Errors, report.Waits)
	}
	// The API has independently observed the same live host run before calling
	// this production store joint. It still checks the current stored stop state.
	if err := db.ActivateObservedMCPCapability(ctx, bound); err != nil {
		t.Fatal(err)
	}
	active, err := db.GetMCPCapability(ctx, bound.ID)
	if err != nil || active.State != factory.MCPCapabilityActive {
		t.Fatal("live-bound capability did not activate")
	}
	if err := db.CheckMCPWorkActive(ctx, active); err != nil {
		t.Fatal(err)
	}
	if _, err := db.WithdrawDispatch(ctx, active.Repository, "project_stop", "7"); err != nil {
		t.Fatal(err)
	}
	if err := db.CheckMCPWorkActive(ctx, active); err == nil {
		t.Fatal("stored stop admitted read")
	}
	raw := []byte(`{"title":"Cannot proceed","summary":"A dependency is unavailable.","resolution":"Restore the dependency"}`)
	submission := factory.MCPSubmission{AssignmentID: active.AssignmentID, SubmissionID: factory.NewID(), CapabilityID: active.ID, Tool: factory.MCPToolBlocked, ExpectedInputDigest: active.ScopeDigest(), PayloadHash: factory.MCPInputDigest(raw), Payload: raw}
	receipt := factory.MCPReceipt{SubmissionID: submission.SubmissionID, Tool: submission.Tool, State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash}
	if _, err := db.RecordMCPSubmission(ctx, submission, receipt); err == nil {
		t.Fatal("cached live capability wrote after recorded stop")
	}
}

func TestMCPObservedActivationRefusesAlreadyStoppedWork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := producerTestDB(t)
	_, run := producerDispatch(t, ctx, db)
	staged, err := db.GetMCPCapability(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.WithdrawDispatch(ctx, staged.Repository, "project_stop", "7"); err != nil {
		t.Fatal(err)
	}
	if err := db.ActivateObservedMCPCapability(ctx, staged); err == nil {
		t.Fatal("cached live observation activated across committed stop")
	}
	retained, err := db.GetMCPCapability(ctx, staged.ID)
	if err != nil || retained.State != factory.MCPCapabilityStaged {
		t.Fatal("refused activation changed capability")
	}
}
