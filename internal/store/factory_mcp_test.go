package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	if !strings.Contains(FactoryMCPAssignmentGuardDDL, "factory_mcp_submission_guards") ||
		!strings.Contains(FactoryMCPAssignmentGuardDDL, "PRIMARY KEY(assignment)") {
		t.Fatal("per-assignment serialization guard missing")
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

// testMCPSubmissionRow binds a submission to its server-admitted
// capability: the capability ID plus the capability's current scope
// digest, which persistence rechecks against the authentic row.
func testMCPSubmissionRow(capability factory.MCPCapability, tool factory.MCPTool, payload []byte) factory.MCPSubmission {
	return factory.MCPSubmission{
		AssignmentID: capability.AssignmentID, SubmissionID: factory.NewID(),
		CapabilityID: capability.ID, Tool: tool,
		ExpectedInputDigest: capability.ScopeDigest(),
		PayloadHash:         factory.MCPInputDigest(payload), Payload: payload,
	}
}

// activateMCPCapability stages and activates a capability for
// submission tests: persistence only records under an active row.
func activateMCPCapability(t *testing.T, ctx context.Context, db *Store, capability factory.MCPCapability) factory.MCPCapability {
	t.Helper()
	staged := capability
	staged.State = factory.MCPCapabilityStaged
	if err := db.StageMCPCapability(ctx, staged); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := db.ActivateMCPCapability(ctx, staged.ID, staged.Generation, staged.RunID,
		staged.AssignmentID, staged.Authority, time.Now().Unix(), false); err != nil {
		t.Fatalf("activate: %v", err)
	}
	active, err := db.GetMCPCapability(ctx, staged.ID)
	if err != nil {
		t.Fatalf("readback: %v", err)
	}
	return active
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
		FactoryMCPAssignmentGuardDDL,
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
	// Submissions bind the activated capability above; the scope is
	// unchanged since activation.
	payload := []byte(`{"outcome":"done"}`)
	submission := testMCPSubmissionRow(got, factory.MCPToolFinishImplement, payload)
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
	if _, err := db.RecordMCPSubmission(ctx, changed, testMCPRecordedReceipt(changed)); !errors.Is(err, ErrMCPSubmissionConflict) {
		t.Fatalf("changed bytes under recorded ID = %v, want conflict", err)
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
	second := testMCPSubmissionRow(got, factory.MCPToolFinishCorrection, payload)
	if _, err := db.RecordMCPSubmission(ctx, second, testMCPRecordedReceipt(second)); err == nil {
		t.Fatal("second terminal identity recorded")
	}
	// Post-terminal nonterminal identities refuse atomically: the
	// assignment is settled.
	late := testMCPSubmissionRow(got, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t1"}`))
	if _, err := db.RecordMCPSubmission(ctx, late, testMCPRecordedReceipt(late)); err == nil {
		t.Fatal("post-terminal output recorded")
	}
	// Handoff counting for the admission cap, on a live assignment
	// under its own activated capability.
	liveCap := testMCPCapabilityRow()
	liveCap.AssignmentID = factory.NewID()
	live := activateMCPCapability(t, ctx, db, liveCap)
	edge := testMCPSubmissionRow(live, factory.MCPToolRequestHandoff, []byte(`{"predecessor_task":"a","successor_task":"b"}`))
	if _, err := db.RecordMCPSubmission(ctx, edge, testMCPRecordedReceipt(edge)); err != nil {
		t.Fatalf("live handoff refused: %v", err)
	}
	if count, err := db.MCPHandoffCount(ctx, live.AssignmentID); err != nil || count != 1 {
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

// TestFactoryMCPSubmissionConcurrentFreeze proves the F2 guard on
// real PostgreSQL (SODA_PG_*; skips without it). Barrier-released
// goroutines force the race; every assertion is a schedule-free
// final-state invariant, so no timing assumption can pass it.
func TestFactoryMCPSubmissionConcurrentFreeze(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	for _, stmt := range []string{
		FactoryMCPCapabilitiesDDL, FactoryMCPCapabilityDeleteTrigger,
		FactoryMCPSubmissionsDDL, FactoryMCPSubmissionTriggers,
		FactoryMCPAssignmentGuardDDL,
	} {
		if _, err := db.db.ExecContext(context.Background(), stmt); err != nil && !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("MCP DDL: %v", err)
		}
	}

	// runRace releases n goroutines at once and collects one result
	// per goroutine after all finish.
	runRace := func(n int, fn func(i int) (factory.MCPReceipt, error)) ([]factory.MCPReceipt, []error) {
		start := make(chan struct{})
		var wg sync.WaitGroup
		receipts := make([]factory.MCPReceipt, n)
		errs := make([]error, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				receipts[i], errs[i] = fn(i)
			}(i)
		}
		close(start)
		wg.Wait()
		return receipts, errs
	}
	// liveAssignment activates a capability and returns it with a
	// recorder bound to it.
	liveAssignment := func(t *testing.T, ctx context.Context) (factory.MCPCapability, func(factory.MCPTool, []byte) (factory.MCPReceipt, error)) {
		t.Helper()
		active := activateMCPCapability(t, ctx, db, testMCPCapabilityRow())
		return active, func(tool factory.MCPTool, payload []byte) (factory.MCPReceipt, error) {
			submission := testMCPSubmissionRow(active, tool, payload)
			return db.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		}
	}

	// Terminal-vs-terminal: concurrent distinct terminal IDs with
	// conflicting content record exactly one row; losers refuse.
	t.Run("distinct terminals", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		active, record := liveAssignment(t, ctx)
		const racers = 8
		receipts, errs := runRace(racers, func(i int) (factory.MCPReceipt, error) {
			return record(factory.MCPToolFinishImplement,
				[]byte(fmt.Sprintf(`{"candidate":"cand-%d"}`, i)))
		})
		wins := 0
		var winner factory.MCPReceipt
		for i := range errs {
			if errs[i] == nil {
				wins++
				winner = receipts[i]
			} else if !errors.Is(errs[i], ErrMCPSubmissionFrozen) {
				t.Fatalf("racer %d: err = %v, want frozen or success", i, errs[i])
			}
		}
		if wins != 1 {
			t.Fatalf("wins = %d, want exactly one recorded terminal", wins)
		}
		if count, err := db.MCPSubmissionCount(ctx, active.AssignmentID); err != nil || count != 1 {
			t.Fatalf("submission count = %d, err = %v, want exactly one row", count, err)
		}
		terminal, frozen, err := db.TerminalMCPProposal(ctx, active.AssignmentID)
		if err != nil || !frozen || terminal != winner {
			t.Fatalf("terminal = %+v, frozen = %v, err = %v, want the single winner", terminal, frozen, err)
		}
	})

	// Same terminal content under distinct IDs: every caller gets
	// the identical receipt and exactly one row exists.
	t.Run("same terminal content", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		active, record := liveAssignment(t, ctx)
		payload := []byte(`{"candidate":"cand-1"}`)
		const racers = 8
		receipts, errs := runRace(racers, func(i int) (factory.MCPReceipt, error) {
			return record(factory.MCPToolFinishImplement, payload)
		})
		for i := range errs {
			if errs[i] != nil {
				t.Fatalf("racer %d refused same-content replay: %v", i, errs[i])
			}
			if receipts[i] != receipts[0] {
				t.Fatalf("racer %d receipt %+v differs from %+v", i, receipts[i], receipts[0])
			}
		}
		if count, err := db.MCPSubmissionCount(ctx, active.AssignmentID); err != nil || count != 1 {
			t.Fatalf("submission count = %d, err = %v, want exactly one row", count, err)
		}
	})

	// Handoff cap at insert: 250 recorded, 16 racing for the last 6
	// slots. Exactly 6 record; the rest hit the cap; the total is
	// exactly 256 under any schedule.
	t.Run("handoff cap", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		active, record := liveAssignment(t, ctx)
		edge := []byte(`{"predecessor_task":"a","successor_task":"b"}`)
		for i := 0; i < 250; i++ {
			if _, err := record(factory.MCPToolRequestHandoff, edge); err != nil {
				t.Fatalf("prefill %d: %v", i, err)
			}
		}
		const racers = 16
		_, errs := runRace(racers, func(i int) (factory.MCPReceipt, error) {
			return record(factory.MCPToolRequestHandoff, edge)
		})
		recorded, capped := 0, 0
		for i := range errs {
			switch {
			case errs[i] == nil:
				recorded++
			case errors.Is(errs[i], ErrMCPSubmissionHandoffCap):
				capped++
			default:
				t.Fatalf("racer %d: err = %v, want cap or success", i, errs[i])
			}
		}
		if recorded != 6 || capped != racers-6 {
			t.Fatalf("recorded = %d, capped = %d, want 6 and %d", recorded, capped, racers-6)
		}
		if count, err := db.MCPHandoffCount(ctx, active.AssignmentID); err != nil || count != factory.MCPMaxHandoffs {
			t.Fatalf("handoff count = %d, err = %v, want %d", count, err, factory.MCPMaxHandoffs)
		}
	})

	// Controlled revoke/insert orderings: a revocation committing
	// ahead of an in-flight insert refuses that insert at
	// persistence, and an insert committing ahead of revocation
	// records before the capability retires.
	t.Run("revoke then insert refuses", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		active, record := liveAssignment(t, ctx)
		if err := db.RevokeMCPCapability(ctx, active.ID); err != nil {
			t.Fatalf("revoke: %v", err)
		}
		if _, err := record(factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-1"}`)); !errors.Is(err, ErrMCPSubmissionCapabilityInactive) {
			t.Fatalf("post-revoke insert = %v, want capability-inactive refusal", err)
		}
		if count, err := db.MCPSubmissionCount(ctx, active.AssignmentID); err != nil || count != 0 {
			t.Fatalf("submission count = %d, err = %v, want no rows", count, err)
		}
	})
	t.Run("insert then revoke records", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		active, record := liveAssignment(t, ctx)
		if _, err := record(factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-1"}`)); err != nil {
			t.Fatalf("pre-revoke insert: %v", err)
		}
		if err := db.RevokeMCPCapability(ctx, active.ID); err != nil {
			t.Fatalf("revoke: %v", err)
		}
		if count, err := db.MCPSubmissionCount(ctx, active.AssignmentID); err != nil || count != 1 {
			t.Fatalf("submission count = %d, err = %v, want one row", count, err)
		}
		got, err := db.GetMCPCapability(ctx, active.ID)
		if err != nil || got.State != factory.MCPCapabilityRevoked {
			t.Fatalf("capability state = %v, err = %v, want revoked", got.State, err)
		}
	})
	t.Run("revoke races inserts", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		active, _ := liveAssignment(t, ctx)
		const writers = 8
		start := make(chan struct{})
		var wg sync.WaitGroup
		errs := make([]error, writers)
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				submission := testMCPSubmissionRow(active, factory.MCPToolSubmitTaskOutput,
					[]byte(fmt.Sprintf(`{"task_id":"t-%d"}`, i)))
				_, errs[i] = db.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
			}(i)
		}
		var revokeErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			revokeErr = db.RevokeMCPCapability(ctx, active.ID)
		}()
		close(start)
		wg.Wait()
		if revokeErr != nil {
			t.Fatalf("revoke: %v", revokeErr)
		}
		// Every writer either recorded ahead of the revocation or
		// refused behind it; nothing else is possible.
		recorded := 0
		for i := range errs {
			switch {
			case errs[i] == nil:
				recorded++
			case errors.Is(errs[i], ErrMCPSubmissionCapabilityInactive):
			default:
				t.Fatalf("writer %d: %v, want success or capability-inactive refusal", i, errs[i])
			}
		}
		if count, err := db.MCPSubmissionCount(ctx, active.AssignmentID); err != nil || count != recorded {
			t.Fatalf("submission count = %d, err = %v, want %d recorded rows", count, err, recorded)
		}
		got, err := db.GetMCPCapability(ctx, active.ID)
		if err != nil || got.State != factory.MCPCapabilityRevoked {
			t.Fatalf("capability state = %v, err = %v, want revoked", got.State, err)
		}
	})
}
