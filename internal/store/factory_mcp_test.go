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
	if !strings.Contains(FactoryMCPSubmissionsDDL, "capability TEXT NOT NULL") {
		t.Fatal("submission author binding missing")
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

// TestFactoryMCPSubmissionRecordsAuthorizingCapability proves the
// R32.1 author binding against real PostgreSQL: the recorded row
// durably names the authorizing capability, and the terminal readback
// returns it, so author/role/subject attribution never assumes one
// capability per assignment.
func TestFactoryMCPSubmissionRecordsAuthorizingCapability(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	capability := activateMCPCapability(t, ctx, db, testMCPCapabilityRow())
	terminal := testMCPSubmissionRow(capability, factory.MCPToolFinishResearch, []byte(`{"phase":"research","status":"complete","summary":"Bound author."}`))
	if _, err := db.RecordMCPSubmission(ctx, terminal, testMCPRecordedReceipt(terminal)); err != nil {
		t.Fatalf("record terminal: %v", err)
	}
	readback, found, err := db.TerminalMCPSubmission(ctx, capability.AssignmentID)
	if err != nil || !found {
		t.Fatalf("terminal readback: %v %v", found, err)
	}
	if readback.CapabilityID != capability.ID {
		t.Fatalf("terminal author=%q, want %q", readback.CapabilityID, capability.ID)
	}
	var stored string
	if err := db.db.QueryRowContext(ctx, `SELECT capability FROM factory_mcp_submissions WHERE assignment=$1 AND submission=$2`, terminal.AssignmentID, terminal.SubmissionID).Scan(&stored); err != nil {
		t.Fatalf("author column: %v", err)
	}
	if stored != capability.ID {
		t.Fatalf("stored author=%q, want %q", stored, capability.ID)
	}
}

// TestFactoryMCPTransactions proves the T48/T51 store joints against
// real PostgreSQL (SODA_PG_*; skips without it): staged replay
// identity, scope-conflict refusal, CAS activation binding, revoke
// and retire transitions, exact-replay receipts, changed-bytes
// conflicts, terminal freezing and handoff counting.
// TestFactoryMCPSubmissionRefusesFinishedAssignment proves the R35
// late-answer rule against real PostgreSQL: new submission identities
// against a finished assignment refuse, while exact replay of a
// submission recorded before the finish still returns its receipt.
func TestFactoryMCPSubmissionRefusesFinishedAssignment(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	seedAssignment := func(id, stage string) {
		t.Helper()
		if _, err := db.db.ExecContext(ctx, `INSERT INTO factory_assignments(id,repository,issue,run,stage,revision,data) VALUES($1,7,7,'run-9',$2,0,'{}')`, id, stage); err != nil {
			t.Fatalf("seed assignment: %v", err)
		}
	}

	capability := activateMCPCapability(t, ctx, db, testMCPCapabilityRow())
	seedAssignment(capability.AssignmentID, string(factory.AssignmentFinished))
	late := testMCPSubmissionRow(capability, factory.MCPToolFlag, []byte(`{"title":"Late flag","summary":"Arrived after the finish.","severity":"critical"}`))
	if _, err := db.RecordMCPSubmission(ctx, late, testMCPRecordedReceipt(late)); !errors.Is(err, ErrMCPSubmissionAssignmentFinished) {
		t.Fatalf("late submission recorded: %v", err)
	}

	live := activateMCPCapability(t, ctx, db, testMCPCapabilityRow())
	seedAssignment(live.AssignmentID, string(factory.AssignmentAssigned))
	early := testMCPSubmissionRow(live, factory.MCPToolFlag, []byte(`{"title":"Early flag","summary":"Recorded while assigned.","severity":"info"}`))
	recorded, err := db.RecordMCPSubmission(ctx, early, testMCPRecordedReceipt(early))
	if err != nil {
		t.Fatalf("record before finish: %v", err)
	}
	if _, err := db.db.ExecContext(ctx, `UPDATE factory_assignments SET stage=$1 WHERE id=$2`, string(factory.AssignmentFinished), live.AssignmentID); err != nil {
		t.Fatalf("finish assignment: %v", err)
	}
	again, err := db.RecordMCPSubmission(ctx, early, testMCPRecordedReceipt(early))
	if err != nil {
		t.Fatalf("replay refused after finish: %v", err)
	}
	if again.SubmissionID != recorded.SubmissionID {
		t.Fatal("replay returned a different receipt")
	}
}

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

// TestFactoryMCPSubmissionExpiryAtInsert proves the F3-expiry half
// against real PostgreSQL (SODA_PG_*; skips without it): the insert
// transaction rechecks the capability deadline against insertion
// time, so a submission queued past expiry records nothing. Ordered
// pause->authority-change/expiry/stop->insert cases refuse with no
// new rows, while already-recorded receipts stay readable through
// the separate MCPReceipt route without granting new authority.
func TestFactoryMCPSubmissionExpiryAtInsert(t *testing.T) {
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

	// shortLived stages and activates a capability expiring soon, so
	// the test can queue an insert across the deadline without
	// touching the immutable scope.
	shortLived := func(t *testing.T, ctx context.Context, lifetime time.Duration) factory.MCPCapability {
		t.Helper()
		capability := testMCPCapabilityRow()
		capability.DeadlineUnix = time.Now().Add(lifetime).Unix()
		return activateMCPCapability(t, ctx, db, capability)
	}
	// sleepPastExpiry waits until the named deadline is a full
	// second past, so the expiry assertion never depends on
	// scheduling tightness: lateness only strengthens it.
	sleepPastExpiry := func(deadline int64) {
		if wait := time.Until(time.Unix(deadline, 0).Add(time.Second)); wait > 0 {
			time.Sleep(wait)
		}
	}

	// A submission queued past its deadline records nothing.
	t.Run("queued past expiry records nothing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		active := shortLived(t, ctx, 2*time.Second)
		submission := testMCPSubmissionRow(active, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-1"}`))
		sleepPastExpiry(active.DeadlineUnix)
		if _, err := db.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission)); !errors.Is(err, ErrMCPSubmissionCapabilityExpired) {
			t.Fatalf("post-expiry insert = %v, want capability-expired refusal", err)
		}
		if count, err := db.MCPSubmissionCount(ctx, active.AssignmentID); err != nil || count != 0 {
			t.Fatalf("submission count = %d, err = %v, want no rows", count, err)
		}
	})

	// Held-guard determinism: the insert passes the capability
	// lock while live, blocks on a fixture-held assignment guard,
	// crosses the deadline there, and records nothing once the
	// guard releases. The blocked-backend observation is the
	// barrier, not a timing assumption: the test cannot proceed
	// past it without the insert waiting on the guard lock.
	t.Run("held guard past expiry records nothing", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		active := shortLived(t, ctx, 5*time.Second)
		if _, err := db.db.ExecContext(ctx, `INSERT INTO factory_mcp_submission_guards(assignment) VALUES($1) ON CONFLICT DO NOTHING`, active.AssignmentID); err != nil {
			t.Fatalf("guard setup: %v", err)
		}
		holder, err := db.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("guard holder begin: %v", err)
		}
		defer func() { _ = holder.Rollback() }()
		var guard string
		if err := holder.QueryRowContext(ctx, `SELECT assignment FROM factory_mcp_submission_guards WHERE assignment=$1 FOR UPDATE`, active.AssignmentID).Scan(&guard); err != nil {
			t.Fatalf("guard holder lock: %v", err)
		}
		submission := testMCPSubmissionRow(active, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-held"}`))
		type outcome struct {
			receipt factory.MCPReceipt
			err     error
		}
		done := make(chan outcome, 1)
		go func() {
			receipt, err := db.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
			done <- outcome{receipt, err}
		}()
		blocked := false
		for limit := time.Now().Add(10 * time.Second); time.Now().Before(limit); {
			var n int
			if err := db.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND wait_event_type='Lock' AND query ILIKE '%factory_mcp_submission_guards%'`).Scan(&n); err != nil {
				t.Fatalf("lock observation: %v", err)
			}
			if n > 0 {
				blocked = true
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if !blocked {
			t.Fatal("insert never blocked on the assignment guard")
		}
		// Cross the deadline while blocked, then release the
		// guard: the insert must refuse instead of recording.
		sleepPastExpiry(active.DeadlineUnix)
		if err := holder.Commit(); err != nil {
			t.Fatalf("guard release: %v", err)
		}
		select {
		case out := <-done:
			if !errors.Is(out.err, ErrMCPSubmissionCapabilityExpired) {
				t.Fatalf("held-guard insert = %v, want capability-expired refusal", out.err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("held-guard insert never returned after guard release")
		}
		if count, err := db.MCPSubmissionCount(ctx, active.AssignmentID); err != nil || count != 0 {
			t.Fatalf("submission count = %d, err = %v, want no rows", count, err)
		}
	})

	// History recorded before expiry stays readable through the
	// receipt route after expiry, while new inserts still refuse.
	t.Run("recorded receipt survives expiry", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		active := shortLived(t, ctx, 2*time.Second)
		early := testMCPSubmissionRow(active, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-early"}`))
		first, err := db.RecordMCPSubmission(ctx, early, testMCPRecordedReceipt(early))
		if err != nil {
			t.Fatalf("pre-expiry record: %v", err)
		}
		sleepPastExpiry(active.DeadlineUnix)
		late := testMCPSubmissionRow(active, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-late"}`))
		if _, err := db.RecordMCPSubmission(ctx, late, testMCPRecordedReceipt(late)); !errors.Is(err, ErrMCPSubmissionCapabilityExpired) {
			t.Fatalf("post-expiry insert = %v, want capability-expired refusal", err)
		}
		if count, err := db.MCPSubmissionCount(ctx, active.AssignmentID); err != nil || count != 1 {
			t.Fatalf("submission count = %d, err = %v, want the one pre-expiry row", count, err)
		}
		recovered, err := db.MCPReceipt(ctx, active.AssignmentID, early.SubmissionID)
		if err != nil {
			t.Fatalf("post-expiry receipt read: %v", err)
		}
		if recovered != first {
			t.Fatal("post-expiry receipt differs from recorded receipt")
		}
	})

	// Pause, then authority rotation: the old authority records
	// nothing further, the assignment stays live under the new
	// authority, and history stays readable.
	t.Run("authority rotation retires the old capability", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		old := activateMCPCapability(t, ctx, db, testMCPCapabilityRow())
		recorded := testMCPSubmissionRow(old, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-1"}`))
		first, err := db.RecordMCPSubmission(ctx, recorded, testMCPRecordedReceipt(recorded))
		if err != nil {
			t.Fatalf("pre-rotation record: %v", err)
		}
		if err := db.RevokeMCPCapability(ctx, old.ID); err != nil {
			t.Fatalf("revoke old authority: %v", err)
		}
		rotated := testMCPCapabilityRow()
		rotated.AssignmentID = old.AssignmentID
		rotated.Authority = strings.Repeat("b", 64)
		live := activateMCPCapability(t, ctx, db, rotated)
		stale := testMCPSubmissionRow(old, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-2"}`))
		if _, err := db.RecordMCPSubmission(ctx, stale, testMCPRecordedReceipt(stale)); !errors.Is(err, ErrMCPSubmissionCapabilityInactive) {
			t.Fatalf("rotated-away insert = %v, want capability-inactive refusal", err)
		}
		if count, err := db.MCPSubmissionCount(ctx, old.AssignmentID); err != nil || count != 1 {
			t.Fatalf("submission count = %d, err = %v, want the one pre-rotation row", count, err)
		}
		fresh := testMCPSubmissionRow(live, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-3"}`))
		if _, err := db.RecordMCPSubmission(ctx, fresh, testMCPRecordedReceipt(fresh)); err != nil {
			t.Fatalf("new-authority record refused: %v", err)
		}
		recovered, err := db.MCPReceipt(ctx, old.AssignmentID, recorded.SubmissionID)
		if err != nil {
			t.Fatalf("post-rotation receipt read: %v", err)
		}
		if recovered != first {
			t.Fatal("post-rotation receipt differs from recorded receipt")
		}
	})

	// Pause, then stop-ordered revocation: nothing records, and the
	// capability stays revoked.
	t.Run("stop revocation freezes inserts", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		active := activateMCPCapability(t, ctx, db, testMCPCapabilityRow())
		submission := testMCPSubmissionRow(active, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-1"}`))
		if err := db.RevokeMCPCapability(ctx, active.ID); err != nil {
			t.Fatalf("stop revocation: %v", err)
		}
		if _, err := db.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission)); !errors.Is(err, ErrMCPSubmissionCapabilityInactive) {
			t.Fatalf("post-stop insert = %v, want capability-inactive refusal", err)
		}
		if count, err := db.MCPSubmissionCount(ctx, active.AssignmentID); err != nil || count != 0 {
			t.Fatalf("submission count = %d, err = %v, want no rows", count, err)
		}
		got, err := db.GetMCPCapability(ctx, active.ID)
		if err != nil || got.State != factory.MCPCapabilityRevoked {
			t.Fatalf("capability state = %v, err = %v, want revoked", got.State, err)
		}
	})
}

// TestTerminalMCPSubmissionRead proves the FW40 settlement read
// against real PostgreSQL (SODA_PG_*; skips without it): the
// recorded terminal submission round-trips its immutable payload
// bytes beside a non-terminal row, and an assignment with no
// terminal reports false.
func TestTerminalMCPSubmissionRead(t *testing.T) {
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
	active := activateMCPCapability(t, ctx, db, testMCPCapabilityRow())
	traffic := testMCPSubmissionRow(active, factory.MCPToolSubmitTaskOutput, []byte(`{"task_id":"t-1"}`))
	if _, err := db.RecordMCPSubmission(ctx, traffic, testMCPRecordedReceipt(traffic)); err != nil {
		t.Fatalf("record task output: %v", err)
	}
	if _, ok, err := db.TerminalMCPSubmission(ctx, active.AssignmentID); err != nil || ok {
		t.Fatalf("terminal = %v, err = %v, want no terminal yet", ok, err)
	}
	payload := []byte(`{"task_id":"t-1","manifest_digest":"` + strings.Repeat("b", 64) + `","output_digest":"` + strings.Repeat("c", 64) + `","evidence_refs":"r"}`)
	terminal := testMCPSubmissionRow(active, factory.MCPToolFinishImplement, payload)
	if _, err := db.RecordMCPSubmission(ctx, terminal, testMCPRecordedReceipt(terminal)); err != nil {
		t.Fatalf("record terminal: %v", err)
	}
	got, ok, err := db.TerminalMCPSubmission(ctx, active.AssignmentID)
	if err != nil || !ok {
		t.Fatalf("terminal = %v, err = %v, want the recorded proposal", ok, err)
	}
	if got.SubmissionID != terminal.SubmissionID || got.Tool != factory.MCPToolFinishImplement ||
		string(got.Payload) != string(payload) || got.PayloadHash != factory.MCPInputDigest(payload) {
		t.Fatalf("terminal mismatch: %+v", got)
	}
	if _, ok, err := db.TerminalMCPSubmission(ctx, factory.NewID()); err != nil || ok {
		t.Fatalf("unknown assignment terminal = %v, err = %v, want false", ok, err)
	}
}

// TestEvidenceStatusReceiptReadsExplicitWithoutSideEffects proves the
// R31.4 read contract against real PostgreSQL: inbox, discussion,
// comment and receipt reads return current observed state and
// original operation outcomes without recording anything — paging
// answers never marks them incorporated. Missing receipts and posts
// refuse explicitly instead of returning empty success.
func TestEvidenceStatusReceiptReadsExplicitWithoutSideEffects(t *testing.T) {
	db, _ := postgresFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	capability := activateMCPCapability(t, ctx, db, testMCPCapabilityRow())
	post := func(tool factory.MCPTool, payload string) factory.MCPReceipt {
		t.Helper()
		submission := testMCPSubmissionRow(capability, tool, []byte(payload))
		receipt, err := db.RecordMCPSubmission(ctx, submission, testMCPRecordedReceipt(submission))
		if err != nil {
			t.Fatalf("%s: %v", tool, err)
		}
		if !factory.ValidID(receipt.FeedPostID) {
			t.Fatalf("%s recorded no post: %+v", tool, receipt)
		}
		return receipt
	}
	ask := post(factory.MCPToolAsk, `{"title":"Who can view a task?","summary":"The owner policy needs clarification.","question":"May team members view?"}`)
	answer := post(factory.MCPToolAnswer, `{"title":"Policy clarified","summary":"Only owners may view.","reply_to":"`+ask.FeedPostID+`"}`)
	suggest := post(factory.MCPToolSuggest, `{"title":"Cache the policy","summary":"Reading the file is slow.","rationale":"Cache at dispatch."}`)
	read := func() {
		t.Helper()
		if _, err := db.MCPInbox(ctx, capability, ""); err != nil {
			t.Fatalf("inbox: %v", err)
		}
		if _, err := db.MCPDiscussion(ctx, capability, ask.FeedPostID, ""); err != nil {
			t.Fatalf("discussion: %v", err)
		}
		if _, err := db.FeedComment(ctx, ask.FeedPostID); err != nil {
			t.Fatalf("comment: %v", err)
		}
		if _, err := db.FeedPostDetails(ctx, []string{answer.FeedPostID, suggest.FeedPostID}); err != nil {
			t.Fatalf("details: %v", err)
		}
	}
	read()
	read()
	var resolutions int
	if err := db.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM factory_post_resolutions`).Scan(&resolutions); err != nil {
		t.Fatal(err)
	}
	if resolutions != 0 {
		t.Fatalf("reads recorded dispositions: %d", resolutions)
	}
	question, err := db.FeedComment(ctx, ask.FeedPostID)
	if err != nil || question.Payload.Communication.QuestionState != "answered" {
		t.Fatalf("question state misreported: %+v %v", question, err)
	}
	if question.Payload.Communication.Disposition != "" {
		t.Fatalf("unresolved question carries disposition: %+v", question)
	}
	for _, tc := range []struct {
		name string
		post string
	}{
		{"answer", answer.FeedPostID},
		{"suggestion", suggest.FeedPostID},
	} {
		detail, err := db.FeedComment(ctx, tc.post)
		if err != nil || detail.Payload.Communication.Disposition != "" {
			t.Fatalf("%s misreported: %+v %v", tc.name, detail, err)
		}
	}
	answerReceipt, err := db.MCPReceipt(ctx, capability.AssignmentID, answer.SubmissionID)
	if err != nil || answerReceipt.State != factory.MCPReceiptRecorded || answerReceipt.FeedPostID != answer.FeedPostID {
		t.Fatalf("receipt lost its original outcome: %+v %v", answerReceipt, err)
	}
	if _, err := db.MCPReceipt(ctx, capability.AssignmentID, factory.NewID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing receipt not explicit: %v", err)
	}
	if _, err := db.FeedComment(ctx, factory.NewID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing post not explicit: %v", err)
	}
	if _, err := db.MCPReceipt(ctx, factory.NewID(), answer.SubmissionID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign receipt disclosed: %v", err)
	}
}
