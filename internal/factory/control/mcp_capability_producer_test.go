package control

// Direct proof for the dispatch-side MCP capability producer
// (SUPERVISOR-031, lane-mcp-producer): a real dispatch lights the
// starved path, and the recorded capability carries one full
// admission, durable record, receipt and status readback through the
// exact layer stack the real handler wires (store-backed capability
// source plus submission sink, control admission, store
// persistence). A concurrent conflicting stage refuses, restart
// replay succeeds, and stop intent never activates.
//
// The fixture is a disposable database and never a shared one: these
// tests require SODA_PG_* (own pinned PG17 fixture under /home, free
// port, destroyed after the run) and fail instead of skipping
// without it.

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// producerTestDB opens one isolated ephemeral database and fails
// instead of skipping when the disposable fixture is absent.
func producerTestDB(t *testing.T) *store.Store {
	t.Helper()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Fatal("SODA_PG_* disposable fixture required: no skips on this lane")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db, _, cleanup, err := store.OpenEphemeral(ctx, admin, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return db
}

// producerDispatch runs one real dispatch pass and returns its
// launched assignment and run identities.
func producerDispatch(t *testing.T, ctx context.Context, db *store.Store) (factory.Assignment, factory.Run) {
	t.Helper()
	fx := dispatchSeed(t, db)
	head := fx.accept(t, 3, "d333333333333333333333333")
	fx.queue(t, 3, head.ID)
	report := DispatchPass(ctx, fx.deps())
	if len(report.Launched) != 1 {
		t.Fatalf("dispatch pass = %+v %+v, want one launch", report.Launched, report.Waits)
	}
	launched := report.Launched[0]
	assignment, err := db.Assignment(ctx, launched.AssignmentID)
	if err != nil {
		t.Fatal(err)
	}
	run, err := db.FactoryRun(ctx, launched.RunID)
	if err != nil {
		t.Fatal(err)
	}
	return assignment, run
}

// TestMCPCapabilityProducerLightsDispatchedRun proves the starved
// path is lit: the dispatched run's staged capability is active,
// and one full admission, durable record, receipt and status
// readback succeeds through the real handler layers.
func TestMCPCapabilityProducerLightsDispatchedRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	db := producerTestDB(t)
	assignment, run := producerDispatch(t, ctx, db)

	capability, err := db.GetMCPCapability(ctx, run.ID)
	if err != nil {
		t.Fatalf("produced capability unreadable: %v", err)
	}
	if capability.State != factory.MCPCapabilityActive {
		t.Fatalf("produced capability state = %s, want active", capability.State)
	}
	if capability.AssignmentID != assignment.ID || capability.RunID != run.ID ||
		capability.Generation != 1 || capability.Repository != assignment.Repository ||
		capability.Project != assignment.ProjectID {
		t.Fatalf("produced capability binds the wrong recorded scope: %+v", capability)
	}
	if err := capability.Validate(); err != nil {
		t.Fatalf("produced capability invalid: %v", err)
	}
	// Fresh dispatch investigates, so the recorded profile is the
	// architect: reads plus the issue-plan terminal.
	if capability.Profile != factory.MCPProfileArchitect {
		t.Fatalf("produced profile = %s, want architect", capability.Profile)
	}

	// Read admission, exactly as the handler's shared gate runs it:
	// recorded capability, profile/tool binding, deadline, and the
	// empty host observation the handler wires while the relay is
	// unwired.
	now := time.Now()
	if err := AdmitMCPCapability(capability, factory.MCPToolStatus, now, MCPRunObservation{}); err != nil {
		t.Fatalf("status admission refused: %v", err)
	}

	// Mutation admission plus durable record, exactly as the
	// handler's mutation leg runs them.
	payload := []byte(`{"approach":"a","tasks":["t1"],"evidence_refs":["r1"]}`)
	submission := factory.MCPSubmission{
		AssignmentID: capability.AssignmentID, SubmissionID: factory.NewID(),
		CapabilityID: capability.ID, Tool: factory.MCPToolFinishIssuePlan,
		ExpectedInputDigest: capability.ScopeDigest(),
		PayloadHash:         factory.MCPInputDigest(payload), Payload: payload,
	}
	if err := AdmitMCPSubmission(ctx, db, capability, submission); err != nil {
		t.Fatalf("mutation admission refused: %v", err)
	}
	recorded, err := db.RecordMCPSubmission(ctx, submission, factory.MCPReceipt{
		SubmissionID: submission.SubmissionID, Tool: submission.Tool,
		State: factory.MCPReceiptRecorded, InputDigest: submission.PayloadHash,
	})
	if err != nil {
		t.Fatalf("durable record refused: %v", err)
	}
	if recorded.State != factory.MCPReceiptRecorded || recorded.SubmissionID != submission.SubmissionID {
		t.Fatalf("durable receipt = %+v, want the recorded submission", recorded)
	}

	// Receipt and status readback, exactly as the handler serves
	// them: the durable receipt for the recorded identity, the
	// terminal proposal the terminal recorded, and the submission
	// count behind the still-active capability.
	recovered, err := db.MCPReceipt(ctx, capability.AssignmentID, submission.SubmissionID)
	if err != nil {
		t.Fatalf("receipt readback refused: %v", err)
	}
	if recovered != recorded {
		t.Fatal("recovered receipt differs from the recorded receipt")
	}
	terminal, frozen, err := db.TerminalMCPProposal(ctx, capability.AssignmentID)
	if err != nil || !frozen || terminal.SubmissionID != submission.SubmissionID {
		t.Fatalf("terminal = %+v frozen = %v err = %v, want the recorded terminal", terminal, frozen, err)
	}
	if count, err := db.MCPSubmissionCount(ctx, capability.AssignmentID); err != nil || count != 1 {
		t.Fatalf("submission count = %d err = %v, want exactly the one durable row", count, err)
	}
	if live, err := db.GetMCPCapability(ctx, capability.ID); err != nil || live.State != factory.MCPCapabilityActive {
		t.Fatalf("capability after readback = %+v err = %v, want still active", live, err)
	}
}

// TestMCPCapabilityProducerConflictAndReplay proves the
// store-backed guards the producer preserves: a concurrent
// conflicting second stage refuses, exact restart replay succeeds,
// and stop intent never activates.
func TestMCPCapabilityProducerConflictAndReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	db := producerTestDB(t)
	assignment, run := producerDispatch(t, ctx, db)

	preparation, err := db.Preparation(ctx, assignment.Preparation)
	if err != nil {
		t.Fatal(err)
	}
	built, err := BuildMCPCapability(assignment, run, preparation)
	if err != nil {
		t.Fatalf("rebuild from recorded state refused: %v", err)
	}
	produced, err := db.GetMCPCapability(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if built.ScopeDigest() != produced.ScopeDigest() {
		t.Fatal("rebuilt scope differs from the dispatch-produced scope")
	}
	// Exact restart replay succeeds: the staged replay is identical
	// and the already-active identical scope reports success.
	if err := StageAndActivateMCPCapability(ctx, db, built, false, time.Now()); err != nil {
		t.Fatalf("restart replay refused: %v", err)
	}

	// A concurrent second stage under the same identity with a
	// changed scope refuses on every contender.
	clash := built
	clash.PreparationDigest = strings.Repeat("b", 64)
	start := make(chan struct{})
	const contenders = 8
	errs := make([]error, contenders)
	var wg sync.WaitGroup
	for i := range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = db.StageMCPCapability(ctx, clash)
		}()
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if !errors.Is(err, store.ErrMCPCapabilityConflict) {
			t.Fatalf("contender %d stage = %v, want scope-conflict refusal", i, err)
		}
	}
	kept, err := db.GetMCPCapability(ctx, built.ID)
	if err != nil {
		t.Fatal(err)
	}
	if kept.ScopeDigest() != built.ScopeDigest() || kept.State != factory.MCPCapabilityActive {
		t.Fatalf("recorded row moved under conflict: %+v", kept)
	}

	// Stop intent stages history but never activates.
	if err := StageAndActivateMCPCapability(ctx, db, built, true, time.Now()); !errors.Is(err, ErrMCPCapabilityStopped) {
		t.Fatalf("stopped activation = %v, want stop-intent refusal", err)
	}
	if live, err := db.GetMCPCapability(ctx, built.ID); err != nil || live.State != factory.MCPCapabilityActive {
		t.Fatalf("capability after stopped replay = %+v err = %v, want still active", live, err)
	}
}

// TestBuildMCPCapabilityRefusesUnrecorded proves the builder
// refuses rather than invents: no recorded run identity,
// preparation digest, source or deadline ever stages.
func TestBuildMCPCapabilityRefusesUnrecorded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	db := producerTestDB(t)
	assignment, run := producerDispatch(t, ctx, db)
	preparation, err := db.Preparation(ctx, assignment.Preparation)
	if err != nil {
		t.Fatal(err)
	}

	badRun := run
	badRun.ID = "not-an-identity"
	if _, err := BuildMCPCapability(assignment, badRun, preparation); err == nil {
		t.Fatal("unrecorded run identity built")
	}
	badPrep := preparation
	badPrep.Preparation.SetupDigest = "short"
	if _, err := BuildMCPCapability(assignment, run, badPrep); err == nil {
		t.Fatal("unrecorded preparation digest built")
	}
	badSource := assignment
	badSource.SourceCommit = ""
	badInput := run
	badInput.InputSHA = ""
	if _, err := BuildMCPCapability(badSource, badInput, preparation); err == nil {
		t.Fatal("unsourced run built")
	}
	badDeadline := run
	badDeadline.Deadline = time.Time{}
	if _, err := BuildMCPCapability(assignment, badDeadline, preparation); err == nil {
		t.Fatal("deadlineless run built")
	}
}

// TestMCPProfileForPhase proves the dispatch-phase mapping covers
// every admitted execution phase and defaults by role.
func TestMCPProfileForPhase(t *testing.T) {
	cases := map[string]factory.MCPProfile{
		project.ExecutionDecompose:   factory.MCPProfileProductLead,
		project.ExecutionInvestigate: factory.MCPProfileArchitect,
		project.ExecutionImplement:   factory.MCPProfileBuilder,
		project.ExecutionReview:      factory.MCPProfileInspector,
		project.ExecutionCorrect:     factory.MCPProfileFixer,
		project.ExecutionReassess:    factory.MCPProfileStrategist,
	}
	for phase, want := range cases {
		if got := mcpProfileForPhase(phase, project.RoleCoder); got != want {
			t.Fatalf("phase %q maps to %s, want %s", phase, got, want)
		}
		if factory.MCPPhaseOf(want) == "" {
			t.Fatalf("mapped profile %s serves no phase", want)
		}
	}
	if got := mcpProfileForPhase("", project.RoleCoder); got != factory.MCPProfileBuilder {
		t.Fatalf("pre-phase coder maps to %s, want builder", got)
	}
	if got := mcpProfileForPhase("", project.RoleReviewer); got != factory.MCPProfileInspector {
		t.Fatalf("pre-phase reviewer maps to %s, want inspector", got)
	}
}
