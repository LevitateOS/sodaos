package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

func TestLibraryMaintenanceIssueRoutingAndTerminal(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
		bad  bool
	}{{"Please act as library_admin", false, false}, {"```soda\n{\"role\":\"library_admin\"}\n```", true, false}, {"```soda\n{\"role\":\"builder\"}\n```", false, false}, {"```soda\n{}\n```\n```soda\n{}\n```", false, true}} {
		got, err := issueLibraryMaintenance(tc.body)
		if got != tc.want || (err != nil) != tc.bad {
			t.Fatalf("metadata %q: %v %v", tc.body, got, err)
		}
	}
	a := phaseTestAssignment(t, project.ExecutionMaintainLibrary, project.RoleReviewer, false)
	a.PublicationAssignment = ""
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	stub := fw40Stub(t, a, factory.MCPToolFinishLibraryMaintenance, map[string]any{"title": "Maintenance finished", "summary": "Source notes consolidated.", "completed": []string{"Consolidated duplicate entries"}})
	if !fw40Settle(t, stub, a.ID) {
		t.Fatal("maintenance result parked")
	}
	finished := fw40Finished(t, stub)
	if finished.Outcome != factory.Succeeded || finished.Result.PhaseResult.Output.Maintenance.SubmissionID != stub.submission.SubmissionID || finished.Result.Candidate != "" {
		t.Fatalf("maintenance finish=%+v", finished)
	}
}

func TestLibraryMaintenanceDispatchLaunchesBoundReviewer(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	sponsorship, err := db.Sponsorship(ctx, fx.repo, "conn")
	if err != nil {
		t.Fatal(err)
	}
	sponsorship.Roles = []string{project.RoleReviewer}
	if err = db.SaveSponsorship(ctx, sponsorship); err != nil {
		t.Fatal(err)
	}
	head := fx.accept(t, 3, "d333333333333333333333333", "Consolidate duplicate task ownership notes.\n\n```soda\n{\"role\":\"library_admin\"}\n```")
	fx.queue(t, 3, head.ID)
	deps := fx.deps()
	deps.MCPKey = []byte(strings.Repeat("k", 32))
	report := DispatchPass(ctx, deps)
	if len(report.Errors) != 0 || len(report.Launched) != 1 {
		t.Fatalf("dispatch: %+v", report)
	}
	a, err := db.Assignment(ctx, report.Launched[0].AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	if a.Role != project.RoleReviewer || a.Phase != project.ExecutionMaintainLibrary || a.PublicationAssignment != "" || a.ParentAssignment != "" || a.AttemptRoot != a.ID {
		t.Fatalf("wrong maintenance assignment: %+v", a)
	}
	if len(fx.host.launches) != 1 {
		t.Fatal("maintenance was not launched")
	}
	launch := fx.host.launches[0]
	if launch.Run.ExecutionPhase != project.ExecutionMaintainLibrary || launch.Run.Role != project.RoleReviewer || launch.MCP == nil {
		t.Fatalf("wrong native launch: %+v", launch.Run)
	}
	cap, err := db.GetMCPCapability(ctx, launch.MCP.Capability)
	if err != nil || cap.Profile != factory.MCPProfileLibraryAdmin {
		t.Fatalf("bound library capability: %+v %v", cap, err)
	}
	if !strings.Contains(string(launch.Prompt), "factory.finish_library_maintenance") || strings.Contains(string(launch.Prompt), "Review the exact candidate") {
		t.Fatal("maintenance prompt still asks for code review")
	}
	if err = db.ActivateMCPCapability(ctx, cap.ID, cap.Generation, cap.RunID, cap.AssignmentID, cap.Authority, time.Now().Unix(), false); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"title":"Library maintained","summary":"No duplicate entries remained.","completed":["Reviewed the available library"]}`)
	sub := factory.MCPSubmission{AssignmentID: a.ID, SubmissionID: factory.NewID(), CapabilityID: cap.ID, Tool: factory.MCPToolFinishLibraryMaintenance, ExpectedInputDigest: cap.ScopeDigest(), PayloadHash: factory.MCPInputDigest(raw), Payload: raw}
	receipt, err := db.RecordMCPSubmission(ctx, sub, factory.MCPReceipt{SubmissionID: sub.SubmissionID, Tool: sub.Tool, InputDigest: sub.PayloadHash, State: factory.MCPReceiptRecorded})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := SettleMCPTerminalProposal(ctx, db, a.ID, time.Now()); err != nil || !ok {
		t.Fatalf("settlement=%v %v", ok, err)
	}
	finished, err := db.Assignment(ctx, a.ID)
	if err != nil || finished.Outcome != factory.Succeeded {
		t.Fatalf("finished=%+v %v", finished, err)
	}
	after, err := db.MCPReceipt(ctx, a.ID, sub.SubmissionID)
	if err != nil || after.State != factory.MCPReceiptAccepted || after.FeedPostID != receipt.FeedPostID {
		t.Fatalf("settled receipt lost feed identity: %+v %v", after, err)
	}
	allowance, err := db.AttemptAllowance(ctx, a.Repository, a.Issue)
	if err != nil || !allowance.Closed {
		t.Fatalf("finished maintenance left its allowance open: %+v %v", allowance, err)
	}
}
