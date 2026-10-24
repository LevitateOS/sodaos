package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestLibraryMaintenanceIssueRoutingAndTerminal(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
		bad  bool
	}{{"Please act as library_admin", false, false}, {"```soda\n{\"role\":\"library_admin\"}\n```", true, false}, {"```soda\n{\"role\":\"research\"}\n```", false, false}, {"```soda\n{\"role\":\"builder\"}\n```", false, true}, {"```soda\n{\"role\":\"library_admin\",\"extra\":1}\n```", false, true}, {"```soda\n{}\n```", false, true}, {"```soda\n{\"role\":\"library_admin\"}\n```\n```soda\n{}\n```", false, true}} {
		got, err := issueLibraryMaintenance(tc.body)
		if got != tc.want || (err != nil) != tc.bad {
			t.Fatalf("metadata %q: %v %v", tc.body, got, err)
		}
	}
	// The canonical parser is strict: unknown roles, unknown fields and
	// role-less blocks refuse, ordinary prose classifies nothing, and a
	// research block derives its registry workflow and navigation labels.
	for _, tc := range []struct {
		body     string
		workflow string
		labels   []string
		bad      bool
	}{
		{"no block here", "", nil, false},
		{"```soda\n{\"role\":\"research\"}\n```", "research_report", []string{"workflow:research_report", "role:research"}, false},
		{"```soda\n{\"role\":\"library_admin\"}\n```", "library_maintenance", []string{"workflow:library_maintenance", "role:library_admin"}, false},
		{"```soda\n{\"role\":\"coder\"}\n```", "", []string{"workflow:code_change", "role:coder"}, false},
		{"```soda\n{\"role\":\"reviewer\"}\n```", "", nil, true},
		{"```soda\n{\"role\":\"research\",\"questions\":[]}\n```", "", nil, true},
	} {
		block, err := factory.ParseSodaIssueBlock(tc.body)
		if (err != nil) != tc.bad {
			t.Fatalf("block %q: %v", tc.body, err)
		}
		if err != nil {
			continue
		}
		if block.Workflow() != tc.workflow {
			t.Fatalf("block %q workflow %q, want %q", tc.body, block.Workflow(), tc.workflow)
		}
		if len(block.Labels()) != len(tc.labels) {
			t.Fatalf("block %q labels %q, want %q", tc.body, block.Labels(), tc.labels)
		}
		for i := range tc.labels {
			if block.Labels()[i] != tc.labels[i] {
				t.Fatalf("block %q labels %q, want %q", tc.body, block.Labels(), tc.labels)
			}
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
	for _, work := range []struct{ role, phase, profile, finish string }{
		{"library_admin", project.ExecutionMaintainLibrary, string(factory.MCPProfileLibraryAdmin), string(factory.MCPToolFinishLibraryMaintenance)},
		{"research", project.ExecutionResearch, string(factory.MCPProfileResearcher), string(factory.MCPToolFinishResearch)},
	} {
		t.Run(work.role, func(t *testing.T) {
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
			head := fx.accept(t, 3, "d333333333333333333333333", "Inspect the recorded task ownership evidence.\n\n```soda\n{\"role\":\""+work.role+"\"}\n```")
			fx.queue(t, 3, head.ID)
			deps := fx.deps()
			deps.MCPKey = []byte(strings.Repeat("k", 32))
			deps.Priority = &PriorityJudgments{Journal: db, Chooser: &bundleGapChooser{pick: factory.ContextBundleCurrent}}
			queued, err := db.QueuedControls(ctx, 1)
			if err != nil || len(queued) != 1 {
				t.Fatal("maintenance readiness unavailable", err)
			}
			occupancy, err := snapshotOccupancy(ctx, db)
			if err != nil {
				t.Fatal(err)
			}
			report := newDispatchReport()
			dispatchOne(ctx, deps, &occupancy, queued[0], &report)
			if len(report.Errors) != 0 || len(report.Launched) != 1 {
				t.Fatalf("dispatch: %+v", report)
			}
			a, err := db.Assignment(ctx, report.Launched[0].AssignmentID)
			if err != nil {
				t.Fatal(err)
			}
			// A definitely unused launch restages this same non-code root with
			// a new run, retaining reviewer preparation and original allowance.
			run, err := db.FactoryRun(ctx, a.Run)
			if err != nil {
				t.Fatal(err)
			}
			reservation, err := db.Reservation(ctx, a.ID)
			if err != nil {
				t.Fatal(err)
			}
			recovery := newDispatchReport()
			recoverUnused(ctx, deps, a, reservation, run, &recovery)
			if len(recovery.Errors) != 0 || len(recovery.Launched) != 1 {
				t.Fatalf("non-code restage: %+v", recovery)
			}
			original := a
			a, err = db.Assignment(ctx, a.ID)
			if err != nil || a.Run == original.Run || a.AttemptRoot != original.AttemptRoot || a.Preparation != original.Preparation || a.Role != original.Role || a.PublicationAssignment != "" {
				t.Fatalf("non-code restage changed authority: %+v %v", a, err)
			}
			if a.Role != project.RoleReviewer || a.Phase != work.phase || a.PublicationAssignment != "" || a.ParentAssignment != "" || a.AttemptRoot != a.ID || a.Preparation != "f222222222222222222222222" {
				t.Fatalf("wrong maintenance assignment: %+v", a)
			}
			wrongRole := a
			wrongRole.Role = project.RoleCoder
			if wrongRole.Validate() == nil || project.ValidFactoryRole("soda-researcher") {
				t.Fatal("non-code work admitted a coding or third native principal")
			}
			if len(fx.host.launches) != 2 {
				t.Fatal("maintenance was not launched")
			}
			launch := fx.host.launches[1]
			if launch.Run.ExecutionPhase != work.phase || launch.Run.Role != project.RoleReviewer || launch.MCP == nil || launch.Validate() != nil {
				t.Fatalf("wrong native launch: %+v", launch.Run)
			}
			cap, err := db.GetMCPCapability(ctx, launch.MCP.Capability)
			if err != nil || string(cap.Profile) != work.profile || factory.MCPToolAdmitted(cap.Profile, factory.MCPToolFinishImplement) {
				t.Fatalf("bound library capability: %+v %v", cap, err)
			}
			if !strings.Contains(string(launch.Prompt), work.finish) || strings.Contains(string(launch.Prompt), "Review the exact candidate") || strings.Contains(string(launch.Prompt), "separate prepared reviewer checkout") {
				t.Fatal("maintenance prompt still asks for code review")
			}
			if err = db.ActivateMCPCapability(ctx, cap.ID, cap.Generation, cap.RunID, cap.AssignmentID, cap.Authority, time.Now().Unix(), false); err != nil {
				t.Fatal(err)
			}
			cap, err = db.GetMCPCapability(ctx, cap.ID)
			if err != nil {
				t.Fatal(err)
			}
			current, err := db.FactoryCurrentContext(ctx, cap, time.Now())
			if work.role == "library_admin" && (err != nil || current.Role.Library == nil || !current.Role.Library.Complete || current.Role.Library.Scope != a.Subject.SubjectKey(a.Repository) || len(current.Role.Library.Operations) == 0 || len(current.Role.Library.Inventory) != 1 || !current.Role.Library.Inventory[0].Complete) {
				t.Fatalf("required library scope and complete empty inventory unavailable: %v", err)
			}
			raw := []byte(`{"title":"Library maintained","summary":"No duplicate entries remained.","completed":["Reviewed the available library"]}`)
			if work.role == "research" {
				if err != nil || a.Subject.Kind != factory.SubjectResearch || current.Role.Research == nil {
					t.Fatalf("research scope unavailable: %v", err)
				}
				raw = []byte(`{"questions":["What governs task ownership?"],"findings":["The accepted task evidence governs ownership."],"evidence":["native-proof.txt"],"conclusion":"Retain the accepted owner.","uncertainty":"No external sources were inspected."}`)
			}
			sub := factory.MCPSubmission{AssignmentID: a.ID, SubmissionID: factory.NewID(), CapabilityID: cap.ID, Tool: factory.MCPTool(work.finish), ExpectedInputDigest: cap.ScopeDigest(), PayloadHash: factory.MCPInputDigest(raw), Payload: raw}
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
			actor, author := store.RecordAuthor(a.Phase, a.Role)
			if actor != work.profile || (work.role == "research" && author != "Researcher") {
				t.Fatalf("native reviewer masked semantic profile: %s/%s", actor, author)
			}
		})
	}
}

// TestDispatchUnknownSodaRoleLaunchesNothing proves the R13.2/R14.2
// routing boundary end to end: an accepted issue whose Soda block names
// an unknown work role waits on changed inputs and launches no default
// coding run, with no dispatch error recorded.
func TestDispatchUnknownSodaRoleLaunchesNothing(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 5, "d555555555555555555555555", "Inspect the cache.\n\n```soda\n{\"role\":\"builder\"}\n```")
	fx.queue(t, 5, head.ID)
	report := DispatchPass(ctx, fx.deps())
	if len(report.Errors) != 0 {
		t.Fatalf("dispatch errors: %+v", report.Errors)
	}
	if len(report.Launched) != 0 {
		t.Fatalf("unknown work role launched: %+v", report.Launched)
	}
	if got := waitReason(report, 5); got != WaitInputsChanged {
		t.Fatalf("wait reason %q, want inputs changed", got)
	}
}
