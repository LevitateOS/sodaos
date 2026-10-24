package control

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
			// A real merged completion and paid, low-confidence decision
			// retain Attention. Non-code inquiry must neither buy a code
			// plan nor release that old decision to enter this repository.
			completionChooser := &native92LowConfidenceChooser{}
			merged, completionDeps := native92DrivePGFixture(t, completionChooser)
			coord, fx := merged.publish.coord, &merged.publish.seed
			db := merged.publish.db
			cycleID := pendingCycleID(t, merged)
			if report := coord.DriveCompletions(ctx, completionDeps); len(report.Sealed) != 1 {
				t.Fatalf("completion seal: %+v", report)
			}
			if report := coord.DriveCompletions(ctx, completionDeps); !native92ReportHas(report.Held, cycleID+":decision_not_actionable") {
				t.Fatalf("completion Attention: %+v", report)
			}
			attention, err := db.ReassessmentRecord(ctx, cycleID)
			if err != nil || attention.Cycle.State != factory.ReassessmentAttention || completionChooser.calls != 1 {
				t.Fatalf("retained Attention: %+v %v", attention, err)
			}
			attentionBefore, err := json.Marshal(attention)
			if err != nil {
				t.Fatal(err)
			}
			// Observe the old control at the same current native revision;
			// its completed assignment/closed allowance still excludes it.
			fx.queue(t, 3, fx.decision[3].ID)
			decision := fx.accept(t, 9, "d999999999999999999999999", "Inspect the recorded task ownership evidence.\n\n```soda\n{\"role\":\""+work.role+"\"}\n```")
			fx.queue(t, 9, decision.ID)
			input := fx.reads.inputs[fmt.Sprintf("%d/9", fx.repo)]
			coord.AcceptanceReads = &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{
				fmt.Sprintf("%d/9", fx.repo): {Revision: input.Revision, Issue: AcceptanceIssueView{Index: "9", TitleOccurrence: input.Issue.TitleOccurrence, TitleDigest: input.Issue.TitleDigest, ContentDigest: input.Issue.ContentDigest, ContentVer: input.Issue.ContentVer, Visible: true}, Comments: []AcceptanceComment{{ID: "11", Digest: input.Comments[0].Digest, Visible: true}}},
			}}
			coord.DispatchReads, coord.Host, coord.Broker = fx.reads, fx.host, fx.broker
			coord.Publication, coord.Checks, coord.Merges = nil, nil, nil
			planning := &planningChooser{}
			coord.Planning = &PlanningJudgments{Journal: db, Funding: db, Chooser: planning}
			sponsorship, err := db.Sponsorship(ctx, fx.repo, "conn")
			if err != nil {
				t.Fatal(err)
			}
			sponsorship.Roles = []string{project.RoleReviewer}
			if err = db.SaveSponsorship(ctx, sponsorship); err != nil {
				t.Fatal(err)
			}
			coord.MCPKey = []byte(strings.Repeat("k", 32))
			coord.Priority = &PriorityJudgments{Journal: db, Chooser: &bundleGapChooser{pick: factory.ContextBundleCurrent}}
			coord.Readiness = &fakeObserver{revision: 41, idle: true}
			deps := coord.dispatchDeps()
			control, err := db.IssueControl(ctx, fx.repo, 9)
			if err != nil {
				t.Fatal("maintenance readiness unavailable", err)
			}
			report := coord.Dispatch(ctx)
			if len(report.Errors) != 0 || len(report.Launched) != 1 {
				t.Fatalf("dispatch: %+v", report)
			}
			if planning.calls != 0 {
				t.Fatal("non-code inquiry bought an implementation-planning judgment")
			}
			a, err := db.Assignment(ctx, report.Launched[0].AssignmentID)
			if err != nil {
				t.Fatal(err)
			}
			candidate := factory.PriorityCandidate{Control: control, Acceptance: decision, Phase: a.Phase, Role: a.Role, Subject: &a.Subject}
			wrongSubject := factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: 8, Acceptance: decision.ID}}
			if work.role == "research" {
				wrongSubject = factory.ExecutionSubject{Kind: factory.SubjectResearch, Research: &factory.ResearchSubject{Issue: 8, Acceptance: decision.ID}}
			}
			if open, wait := splitFencedCandidates([]factory.PriorityCandidate{candidate}, map[int64]bool{fx.repo: true}); len(open) != 1 || len(wait) != 0 {
				t.Fatal("accepted reviewer inquiry remained fenced at priority")
			}
			for _, invalid := range []factory.PriorityCandidate{
				{Control: candidate.Control, Acceptance: decision},
				{Control: candidate.Control, Acceptance: decision, Phase: a.Phase, Role: project.RoleCoder, Subject: &a.Subject},
				{Control: candidate.Control, Acceptance: decision, Phase: project.ExecutionReview, Role: a.Role, Subject: &a.Subject},
				{Control: candidate.Control, Acceptance: decision, Phase: a.Phase, Role: a.Role, Subject: &wrongSubject},
				{Control: candidate.Control, Acceptance: factory.Acceptance{ID: decision.ID, Repository: fx.repo, IssueIndex: "8"}, Phase: a.Phase, Role: a.Role, Subject: &a.Subject},
			} {
				if open, wait := splitFencedCandidates([]factory.PriorityCandidate{invalid}, map[int64]bool{fx.repo: true}); len(open) != 0 || len(wait) != 1 || wait[0].Reason != WaitReassessmentPending {
					t.Fatal("unknown or mismatched inquiry escaped the priority fence")
				}
			}
			run, err := db.FactoryRun(ctx, a.Run)
			if err != nil {
				t.Fatal(err)
			}
			// The atomic gate still refuses code work even when it carries
			// the exact same accepted issue; no new packet rows survive.
			code := a
			code.ID, code.Run = factory.NewID(), factory.NewID()
			code.AttemptRoot, code.PublicationAssignment = code.ID, code.ID
			code.AcceptedContext, code.RunHistory = nil, []string{code.Run}
			code.Stage, code.Outcome, code.Reason, code.Result, code.FinishedUnix = factory.AssignmentAssigned, "", "", nil, 0
			code.Role, code.Phase, code.Preparation = project.RoleCoder, project.ExecutionInvestigate, "f111111111111111111111111"
			code.Subject = factory.ExecutionSubject{Kind: factory.SubjectIssue, Issue: &factory.IssueSubject{Issue: code.Issue, Acceptance: code.Acceptance}}
			codeRun := run
			codeRun.ID, codeRun.Role = code.Run, code.Role
			codeRun.Outcome, codeRun.Reconciled = "", false
			codeReservation := factory.Reservation{AssignmentID: code.ID, Repository: code.Repository, Connection: code.Connection, State: factory.ReservationHeld, PlannedMinutes: 30}
			codeView := factory.RunView{RunID: code.Run, Repository: code.Repository, Issue: code.Issue, Attempt: code.ID}
			if err := db.RecordDispatchPacket(ctx, factory.DispatchRegistration{ID: code.ID, Repository: code.Repository, Authority: code.Authority}, control, code, codeReservation, codeRun, codeView); !errors.Is(err, store.ErrCompletionFenced) {
				t.Fatalf("code packet escaped retained Attention: %v", err)
			}
			if _, err := db.Assignment(ctx, code.ID); !errors.Is(err, store.ErrNotFound) {
				t.Fatal("fenced code packet left an assignment")
			}
			// A definitely unused launch restages this same non-code root with
			// a new run, retaining reviewer preparation and original allowance.
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
			retained, err := db.ReassessmentRecord(ctx, cycleID)
			attentionAfter, marshalErr := json.Marshal(retained)
			if err != nil || marshalErr != nil || !bytes.Equal(attentionBefore, attentionAfter) || completionChooser.calls != 1 {
				t.Fatal("non-code dispatch/restage/finish altered the held completion or paid judgment")
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
