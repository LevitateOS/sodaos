package api

// FW38/T47/T48/T51 catalog proof.
//
// The pinned SDK server is driven over a real native child on stdio
// per admitted harness and over the fixed HTTP route: discovery of
// the fourteen tools, capability-bound reads, immutable mutations
// with exact-replay receipts, conflicting-byte refusal, terminal
// freezing, lost-response receipt recovery, cross-profile and
// unknown-tool refusal, unknown-field refusal, and the fixed-route
// shape (POST initializes as JSON, GET is 405 with no SSE, browser
// Origin refuses). The in-memory submission sink mirrors the store
// semantics the PG-gated store test proves against real PostgreSQL.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

// fakeCatalogCapability is the one admitted test capability.
func fakeCatalogCapability() factory.MCPCapability {
	digest := strings.Repeat("a", 64)
	return factory.MCPCapability{
		ID: strings.Repeat("a", 32), AssignmentID: strings.Repeat("b", 32),
		RunID: "run-test", Generation: 1,
		Profile: factory.MCPProfileBuilder, Subject: "implement:42", Repository: 7,
		Project:           "p" + strings.Repeat("d", 24),
		PreparationDigest: digest, SourceDigest: "abc", AcceptedRevision: "rev-1",
		Authority: digest, DeadlineUnix: time.Now().Add(time.Hour).Unix(),
		State: factory.MCPCapabilityActive,
	}
}

type fakeCatalogRecord struct {
	tool     factory.MCPTool
	input    string
	hash     string
	receipt  factory.MCPReceipt
	terminal bool
}

// fakeCatalogSink mirrors store.RecordMCPSubmission semantics in
// memory: exact replay returns the recorded receipt, changed bytes
// conflict, and the terminal freezes new identities except the same
// terminal content under another ID.
type fakeCatalogSink struct {
	mu          sync.Mutex
	capability  factory.MCPCapability
	submissions map[string]fakeCatalogRecord
	assignments map[string]factory.Assignment
}

func newFakeCatalogSink() *fakeCatalogSink {
	capability := fakeCatalogCapability()
	return &fakeCatalogSink{
		capability:  capability,
		submissions: map[string]fakeCatalogRecord{},
		assignments: map[string]factory.Assignment{
			capability.AssignmentID: {
				ID: capability.AssignmentID, Phase: string(factory.MCPPhaseOf(capability.Profile)),
				SourceCommit: strings.Repeat("c", 40), Preparation: "prep-1",
				PromptSHA: strings.Repeat("d", 64),
			},
		},
	}
}

func (f *fakeCatalogSink) Assignment(_ context.Context, id string) (factory.Assignment, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	assignment, ok := f.assignments[id]
	if !ok {
		return factory.Assignment{}, store.ErrNotFound
	}
	return assignment, nil
}

func (f *fakeCatalogSink) GetMCPCapability(_ context.Context, id string) (factory.MCPCapability, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id != f.capability.ID {
		return factory.MCPCapability{}, store.ErrNotFound
	}
	return f.capability, nil
}

func (f *fakeCatalogSink) record(submission factory.MCPSubmission, receipt factory.MCPReceipt) (factory.MCPReceipt, error) {
	key := submission.AssignmentID + "/" + submission.SubmissionID
	if current, ok := f.submissions[key]; ok {
		if current.tool != submission.Tool || current.input != submission.ExpectedInputDigest || current.hash != submission.PayloadHash {
			return factory.MCPReceipt{}, store.ErrMCPSubmissionConflict
		}
		return current.receipt, nil
	}
	for _, current := range f.submissions {
		if current.terminal {
			if current.tool == submission.Tool && current.receipt.InputDigest == submission.PayloadHash {
				return current.receipt, nil
			}
			return factory.MCPReceipt{}, store.ErrMCPSubmissionFrozen
		}
	}
	f.submissions[key] = fakeCatalogRecord{
		tool: submission.Tool, input: submission.ExpectedInputDigest,
		hash: submission.PayloadHash, receipt: receipt,
		terminal: factory.MCPIsTerminal(submission.Tool),
	}
	return receipt, nil
}

func (f *fakeCatalogSink) RecordMCPSubmission(_ context.Context, submission factory.MCPSubmission, receipt factory.MCPReceipt) (factory.MCPReceipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.record(submission, receipt)
}

func (f *fakeCatalogSink) MCPReceipt(_ context.Context, assignmentID, submissionID string) (factory.MCPReceipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	current, ok := f.submissions[assignmentID+"/"+submissionID]
	if !ok {
		return factory.MCPReceipt{}, store.ErrNotFound
	}
	return current.receipt, nil
}

func (f *fakeCatalogSink) TerminalMCPProposal(_ context.Context, assignmentID string) (factory.MCPReceipt, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key, current := range f.submissions {
		if strings.HasPrefix(key, assignmentID+"/") && current.terminal {
			return current.receipt, true, nil
		}
	}
	return factory.MCPReceipt{}, false, nil
}

func (f *fakeCatalogSink) MCPHandoffCount(_ context.Context, assignmentID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for key, current := range f.submissions {
		if strings.HasPrefix(key, assignmentID+"/") && current.tool == factory.MCPToolRequestHandoff {
			count++
		}
	}
	return count, nil
}

func (f *fakeCatalogSink) MCPSubmissionCount(_ context.Context, assignmentID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for key := range f.submissions {
		if strings.HasPrefix(key, assignmentID+"/") {
			count++
		}
	}
	return count, nil
}

// TestFactoryCatalogServerProcess is the stdio helper: catalog tests
// re-execute the test binary with SODA_FACTORY_MCP_FW38 set and
// speak MCP to this server over the child's stdin/stdout.
func TestFactoryCatalogServerProcess(t *testing.T) {
	if os.Getenv("SODA_FACTORY_MCP_FW38") == "" {
		return
	}
	sink := newFakeCatalogSink()
	server := newFactoryCatalogServer(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink})
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		t.Logf("catalog helper server exited: %v", err)
	}
}

func catalogDial(t *testing.T, ctx context.Context) *mcp.ClientSession {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestFactoryCatalogServerProcess")
	cmd.Env = append(os.Environ(), "SODA_FACTORY_MCP_FW38=1")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "fw38-proof"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("initialize over stdio: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func catalogCall(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()
	return session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
}

func catalogRefused(res *mcp.CallToolResult, err error) bool {
	return err != nil || (res != nil && res.IsError)
}

func catalogText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("expected one content frame, got %d", len(res.Content))
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", res.Content[0])
	}
	return text.Text
}

// catalogPayload returns a caller payload bound to the actually
// admitted fixture capability: the caller passes the scope digest
// the server served for that capability, never a recomputed one.
// Recomputing from a fresh fakeCatalogCapability re-reads the wall
// clock, and ScopeDigest includes DeadlineUnix, so crossing a second
// between the fixture admission and the request produces a false
// stale-input refusal.
func catalogPayload(args map[string]any, scopeDigest string) (map[string]any, string) {
	return args, scopeDigest
}

// catalogScopeDigest reads the scope digest the server actually
// admitted: factory.context serves the recorded capability's current
// scope digest in its body.
func catalogScopeDigest(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(catalogText(t, res)), &body); err != nil {
		t.Fatalf("context body is not JSON: %v", err)
	}
	digest, _ := body["scope_digest"].(string)
	if len(digest) != 64 {
		t.Fatalf("context body carries no admitted scope digest: %q", body)
	}
	return digest
}

// TestFactoryCatalogTransportPerHarness proves T47/T48/T51 through
// the catalog: fourteen-tool discovery, capability-bound reads,
// immutable mutation with replay/conflict semantics, terminal
// freezing, receipt recovery and server-side refusals.
// The transport admits by recorded capability, not by harness
// string: P6 proved the exact harness pins, and the FW39 relay owns
// parent/harness authentication. Capability binding, profile
// admission and submission semantics are harness-independent here.
func TestFactoryCatalogTransportPerHarness(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	session := catalogDial(t, ctx)
	capability := strings.Repeat("a", 32)

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("discovery: %v", err)
	}
	seen := map[string]bool{}
	for _, tool := range tools.Tools {
		seen[tool.Name] = true
	}
	if len(seen) != 14 {
		t.Fatalf("discovery advertises %d tools, want 14", len(seen))
	}
	for _, tool := range factory.MCPTools {
		if !seen[string(tool)] {
			t.Fatalf("catalog tool %s missing from discovery", tool)
		}
	}

	reads := map[string]any{"capability_id": capability}
	res, err := catalogCall(t, ctx, session, "factory.context", reads)
	if err != nil {
		t.Fatalf("allowed read: %v", err)
	}
	if body := catalogText(t, res); !strings.Contains(body, strings.Repeat("b", 32)) || !strings.Contains(body, "builder") {
		t.Fatalf("context body %q lacks assignment/profile", body)
	}
	// Bind every later mutation to the actually admitted fixture
	// capability's scope digest. Recomputing a fresh fixture digest
	// here would re-read the wall clock across a possible second
	// boundary and refuse as stale.
	admittedDigest := catalogScopeDigest(t, res)
	res, err = catalogCall(t, ctx, session, "factory.status", reads)
	if err != nil {
		t.Fatalf("status read: %v", err)
	}
	if body := catalogText(t, res); !strings.Contains(body, `"terminal":false`) {
		t.Fatalf("status body %q lacks live terminal state", body)
	}
	res, err = catalogCall(t, ctx, session, "factory.evidence", reads)
	if err != nil {
		t.Fatalf("evidence read: %v", err)
	}
	if body := catalogText(t, res); !strings.Contains(body, `"complete":false`) {
		t.Fatalf("evidence body %q is not an honest incomplete", body)
	}

	// Unknown capability refuses without disclosure.
	if res, err := catalogCall(t, ctx, session, "factory.context", map[string]any{"capability_id": strings.Repeat("c", 32)}); !catalogRefused(res, err) {
		t.Fatal("unknown capability admitted")
	}
	// Unknown tool refuses server-side.
	if res, err := catalogCall(t, ctx, session, "factory.merge", reads); !catalogRefused(res, err) {
		t.Fatal("unknown tool admitted")
	}
	// Unknown top-level field refuses server-side.
	extra := map[string]any{"capability_id": capability, "role": "admin"}
	if res, err := catalogCall(t, ctx, session, "factory.context", extra); !catalogRefused(res, err) {
		t.Fatal("unknown envelope field admitted")
	}
	// Cross-profile direct call refuses: builder cannot finish reviews.
	reviewPayload, reviewDigest := catalogPayload(map[string]any{
		"conclusion": "clear", "coverage": "full", "evidence_refs": []any{"r1"},
	}, admittedDigest)
	if res, err := catalogCall(t, ctx, session, "factory.finish_review", map[string]any{
		"capability_id": capability, "submission_id": strings.Repeat("1", 32),
		"expected_input_digest": reviewDigest, "payload": reviewPayload,
	}); !catalogRefused(res, err) {
		t.Fatal("cross-profile direct call admitted")
	}

	// Immutable mutation: record, replay, conflict, recovery.
	implPayload, implDigest := catalogPayload(map[string]any{"candidate": "cand-1"}, admittedDigest)
	finishID := strings.Repeat("2", 32)
	finish := map[string]any{
		"capability_id": capability, "submission_id": finishID,
		"expected_input_digest": implDigest, "payload": implPayload,
	}
	first, err := catalogCall(t, ctx, session, "factory.finish_implementation", finish)
	if err != nil {
		t.Fatalf("typed finish: %v", err)
	}
	if body := catalogText(t, first); !strings.Contains(body, `"state":"recorded"`) {
		t.Fatalf("finish receipt %q not recorded", body)
	}
	replay, err := catalogCall(t, ctx, session, "factory.finish_implementation", finish)
	if err != nil {
		t.Fatalf("exact replay: %v", err)
	}
	if catalogText(t, replay) != catalogText(t, first) {
		t.Fatal("replay receipt differs from recorded receipt")
	}
	clashPayload, clashDigest := catalogPayload(map[string]any{"unchanged": true}, admittedDigest)
	clash := map[string]any{
		"capability_id": capability, "submission_id": finishID,
		"expected_input_digest": clashDigest, "payload": clashPayload,
	}
	if res, err := catalogCall(t, ctx, session, "factory.finish_implementation", clash); !catalogRefused(res, err) {
		t.Fatal("conflicting bytes under recorded ID admitted")
	}
	lost, err := catalogCall(t, ctx, session, "factory.receipt", map[string]any{
		"capability_id": capability, "submission_id": finishID,
	})
	if err != nil {
		t.Fatalf("receipt recovery: %v", err)
	}
	if catalogText(t, lost) != catalogText(t, first) {
		t.Fatal("recovered receipt differs from recorded receipt")
	}
	if res, err := catalogCall(t, ctx, session, "factory.receipt", map[string]any{
		"capability_id": capability, "submission_id": strings.Repeat("3", 32),
	}); !catalogRefused(res, err) {
		t.Fatal("unknown receipt disclosed")
	}

	// Terminal freeze: a second terminal identity and post-terminal
	// output both refuse.
	otherPayload, otherDigest := catalogPayload(map[string]any{"candidate": "cand-2"}, admittedDigest)
	if res, err := catalogCall(t, ctx, session, "factory.finish_implementation", map[string]any{
		"capability_id": capability, "submission_id": strings.Repeat("4", 32),
		"expected_input_digest": otherDigest, "payload": otherPayload,
	}); !catalogRefused(res, err) {
		t.Fatal("second terminal identity admitted")
	}
	outputPayload, outputDigest := catalogPayload(map[string]any{
		"task_id": "t1", "manifest_digest": strings.Repeat("d", 64),
		"output_digest": strings.Repeat("e", 64), "evidence_refs": []any{"r1"},
	}, admittedDigest)
	if res, err := catalogCall(t, ctx, session, "factory.submit_task_output", map[string]any{
		"capability_id": capability, "submission_id": strings.Repeat("5", 32),
		"expected_input_digest": outputDigest, "payload": outputPayload,
	}); !catalogRefused(res, err) {
		t.Fatal("post-terminal output admitted")
	}

	// Manifest proposal without an approved catalog names the gate.
	proposalPayload, proposalDigest := catalogPayload(map[string]any{
		"candidates":    []any{"g1"},
		"evidence_refs": []any{"r1"},
		"coverage":      "both outcomes",
		"verification_manifest_proposal": map[string]any{
			"outcomes": []any{"checkout"},
			"checks": []any{map[string]any{
				"outcome": "checkout", "catalog_id": "web.journey.smoke.v3",
				"params":    map[string]any{"journey": "guest"},
				"artifacts": []any{"log"},
			}},
		},
	}, admittedDigest)
	product := map[string]any{
		"capability_id": capability, "submission_id": strings.Repeat("6", 32),
		"expected_input_digest": proposalDigest, "payload": proposalPayload,
	}
	// The test capability is a builder: product plans refuse on
	// profile first. The catalog-gate leg runs in-process below.
	if res, err := catalogCall(t, ctx, session, "factory.finish_product_plan", product); !catalogRefused(res, err) {
		t.Fatal("cross-profile product plan admitted")
	}
}

// TestFactoryCatalogClockBoundaryDigest pins the R47 fixture repair
// without a flaky sleep: a one-second deadline tick — the recompute
// a clock-boundary crossing would produce — changes ScopeDigest, so
// a request built from the recomputed digest refuses as stale, while
// the actually admitted digest records. Intentional stale-input
// refusal behavior is unchanged.
func TestFactoryCatalogClockBoundaryDigest(t *testing.T) {
	sink := newFakeCatalogSink()
	server := &factoryCatalogServer{
		deps:    FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink},
		limiter: newMCPRateLimiter(),
	}
	ctx := context.Background()
	admitted := sink.capability.ScopeDigest()
	ticked := sink.capability
	ticked.DeadlineUnix++
	if recomputed := ticked.ScopeDigest(); recomputed == admitted {
		t.Fatal("deadline tick keeps scope digest: clock boundary untested")
	} else {
		payload := map[string]any{
			"task_id": "t1", "manifest_digest": strings.Repeat("d", 64),
			"output_digest": strings.Repeat("e", 64), "evidence_refs": []any{"r1"},
		}
		stale := mcpCallArgs{
			CapabilityID: sink.capability.ID, SubmissionID: factory.NewID(),
			ExpectedInputDigest: recomputed, Payload: payload,
		}
		if _, _, err := server.serve(ctx, factory.MCPToolSubmitTaskOutput, stale); err == nil {
			t.Fatal("recomputed clock-boundary digest admitted")
		} else if !strings.Contains(err.Error(), "stale") {
			t.Fatalf("clock-boundary refusal %q does not name stale inputs", err)
		}
	}
	fresh := mcpCallArgs{
		CapabilityID: sink.capability.ID, SubmissionID: factory.NewID(),
		ExpectedInputDigest: admitted, Payload: map[string]any{
			"task_id": "t1", "manifest_digest": strings.Repeat("d", 64),
			"output_digest": strings.Repeat("e", 64), "evidence_refs": []any{"r1"},
		},
	}
	if _, _, err := server.serve(ctx, factory.MCPToolSubmitTaskOutput, fresh); err != nil {
		t.Fatalf("admitted scope digest refused: %v", err)
	}
}

// TestFactoryCatalogReadsSpendNothingAndDiscloseNothing proves the R31
// read contract through the actual handlers: context, status, evidence
// and receipt record no submission rows, unknown capabilities refuse on
// every read, receipts never cross assignments, and evidence stays
// honestly incomplete instead of an empty success.
func TestFactoryCatalogReadsSpendNothingAndDiscloseNothing(t *testing.T) {
	sink := newFakeCatalogSink()
	server := &factoryCatalogServer{
		deps:    FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink},
		limiter: newMCPRateLimiter(),
	}
	ctx := context.Background()
	admitted := sink.capability.ScopeDigest()
	reads := func(tool factory.MCPTool, capabilityID, submissionID string) mcpCallArgs {
		return mcpCallArgs{CapabilityID: capabilityID, SubmissionID: submissionID, ExpectedInputDigest: admitted}
	}
	count := func() int {
		n, err := sink.MCPSubmissionCount(ctx, sink.capability.AssignmentID)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	for _, tool := range []factory.MCPTool{factory.MCPToolContext, factory.MCPToolStatus, factory.MCPToolEvidence} {
		res, _, err := server.serve(ctx, tool, reads(tool, sink.capability.ID, ""))
		if err != nil {
			t.Fatalf("%s read refused: %v", tool, err)
		}
		if tool == factory.MCPToolEvidence {
			var body map[string]any
			text, ok := res.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("evidence content is %T, not text", res.Content[0])
			}
			if err := json.Unmarshal([]byte(text.Text), &body); err != nil {
				t.Fatalf("evidence body is not JSON: %v", err)
			}
			if complete, _ := body["complete"].(bool); complete {
				t.Fatalf("evidence claims completeness: %q", text.Text)
			}
			if reason, _ := body["reason"].(string); reason == "" {
				t.Fatalf("incomplete evidence names no reason: %q", text.Text)
			}
		}
	}
	if _, _, err := server.serve(ctx, factory.MCPToolReceipt, reads(factory.MCPToolReceipt, sink.capability.ID, factory.NewID())); err == nil {
		t.Fatal("unknown receipt admitted")
	} else if !strings.Contains(err.Error(), "receipt not found") {
		t.Fatalf("unknown receipt refusal %q discloses", err)
	}
	if count() != 0 {
		t.Fatalf("reads recorded %d submissions", count())
	}
	unknown := strings.Repeat("f", 32)
	for _, tool := range []factory.MCPTool{factory.MCPToolContext, factory.MCPToolStatus, factory.MCPToolEvidence, factory.MCPToolReceipt} {
		if _, _, err := server.serve(ctx, tool, reads(tool, unknown, factory.NewID())); err == nil {
			t.Fatalf("%s read admitted an unknown capability", tool)
		}
	}
	if count() != 0 {
		t.Fatalf("refused reads recorded %d submissions", count())
	}
	submissionID := factory.NewID()
	mutPayload, mutDigest := catalogPayload(map[string]any{
		"task_id": "t1", "manifest_digest": strings.Repeat("d", 64),
		"output_digest": strings.Repeat("e", 64), "evidence_refs": []any{"r1"},
	}, admitted)
	if _, _, err := server.serve(ctx, factory.MCPToolSubmitTaskOutput, mcpCallArgs{
		CapabilityID: sink.capability.ID, SubmissionID: submissionID,
		ExpectedInputDigest: mutDigest, Payload: mutPayload,
	}); err != nil {
		t.Fatalf("recording mutation refused: %v", err)
	}
	if _, _, err := server.serve(ctx, factory.MCPToolReceipt, reads(factory.MCPToolReceipt, sink.capability.ID, submissionID)); err != nil {
		t.Fatalf("own receipt refused: %v", err)
	}
	other := newFakeCatalogSink()
	other.capability.ID = strings.Repeat("c", 32)
	other.capability.AssignmentID = strings.Repeat("e", 32)
	otherAssignment := other.assignments[sink.capability.AssignmentID]
	otherAssignment.ID = other.capability.AssignmentID
	delete(other.assignments, sink.capability.AssignmentID)
	other.assignments[other.capability.AssignmentID] = otherAssignment
	otherServer := &factoryCatalogServer{
		deps:    FactoryMCPDeps{Capabilities: other, Submissions: other, Assignments: other},
		limiter: newMCPRateLimiter(),
	}
	otherDigest := other.capability.ScopeDigest()
	if _, _, err := otherServer.serve(ctx, factory.MCPToolReceipt, mcpCallArgs{
		CapabilityID: other.capability.ID, SubmissionID: submissionID, ExpectedInputDigest: otherDigest,
	}); err == nil {
		t.Fatal("receipt crossed assignments")
	} else if !strings.Contains(err.Error(), "receipt not found") {
		t.Fatalf("cross-assignment refusal %q discloses", err)
	}
	if count() != 1 {
		t.Fatalf("recorded %d submissions, want exactly the mutation", count())
	}
}

// TestFactoryCatalogContextProjectsAssignmentCurrent proves the R31
// current-context projection through the actual handler: context serves
// the recorded assignment's prompt digest, exact source/preparation
// and, once set, outcome/reason — never prompt bytes, run history or
// actor identity. A capability without its assignment refuses instead
// of serving context without it.
func TestFactoryCatalogContextProjectsAssignmentCurrent(t *testing.T) {
	sink := newFakeCatalogSink()
	server := &factoryCatalogServer{
		deps:    FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink},
		limiter: newMCPRateLimiter(),
	}
	ctx := context.Background()
	admitted := sink.capability.ScopeDigest()
	contextBody := func() map[string]any {
		t.Helper()
		res, _, err := server.serve(ctx, factory.MCPToolContext, mcpCallArgs{
			CapabilityID: sink.capability.ID, ExpectedInputDigest: admitted,
		})
		if err != nil {
			t.Fatalf("context refused: %v", err)
		}
		text, ok := res.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("context content is %T, not text", res.Content[0])
		}
		var body map[string]any
		if err := json.Unmarshal([]byte(text.Text), &body); err != nil {
			t.Fatalf("context body is not JSON: %v", err)
		}
		return body
	}
	body := contextBody()
	for key, want := range map[string]string{
		"prompt_sha":    strings.Repeat("d", 64),
		"source_commit": strings.Repeat("c", 40),
		"preparation":   "prep-1",
	} {
		if body[key] != want {
			t.Fatalf("context %s = %v, want %s", key, body[key], want)
		}
	}
	for _, absent := range []string{"outcome", "reason", "prompt", "run_history", "actor_id"} {
		if _, ok := body[absent]; ok {
			t.Fatalf("unset context carries %q: %v", absent, body[absent])
		}
	}
	stored := sink.assignments[sink.capability.AssignmentID]
	stored.Outcome, stored.Reason = factory.Succeeded, "settled"
	sink.assignments[sink.capability.AssignmentID] = stored
	body = contextBody()
	if body["outcome"] != string(factory.Succeeded) || body["reason"] != "settled" {
		t.Fatalf("settled context lacks outcome/reason: %v", body)
	}
	delete(sink.assignments, sink.capability.AssignmentID)
	if _, _, err := server.serve(ctx, factory.MCPToolContext, mcpCallArgs{
		CapabilityID: sink.capability.ID, ExpectedInputDigest: admitted,
	}); err == nil {
		t.Fatal("context without its assignment admitted")
	}
}

// TestFactoryCatalogManifestGate proves the D16 release leg: a plan
// carrying a manifest proposal validates against the catalog, and a
// missing catalog names the unwired gate instead of inventing
// approval.
func TestFactoryCatalogManifestGate(t *testing.T) {
	sink := newFakeCatalogSink()
	sink.capability.Profile = factory.MCPProfileProductLead
	server := &factoryCatalogServer{
		deps:    FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink},
		limiter: newMCPRateLimiter(),
	}
	ctx := context.Background()
	argsFor := func(payload map[string]any) mcpCallArgs {
		return mcpCallArgs{
			CapabilityID: sink.capability.ID, SubmissionID: factory.NewID(),
			ExpectedInputDigest: sink.capability.ScopeDigest(), Payload: payload,
		}
	}
	plan := map[string]any{
		"candidates": []any{"g1"}, "evidence_refs": []any{"r1"}, "coverage": "all",
		"verification_manifest_proposal": map[string]any{
			"outcomes": []any{"checkout"},
			"checks": []any{map[string]any{
				"outcome": "checkout", "catalog_id": "web.journey.smoke.v3",
				"params": map[string]any{"journey": "guest"}, "artifacts": []any{"log"},
			}},
		},
	}
	if _, _, err := server.serve(ctx, factory.MCPToolFinishProductPlan, argsFor(plan)); err == nil {
		t.Fatal("manifest proposal approved without a catalog")
	} else if !strings.Contains(err.Error(), "catalog") {
		t.Fatalf("refusal %q does not name the catalog gate", err)
	}
	plain := map[string]any{
		"candidates": []any{"g1"}, "evidence_refs": []any{"r1"}, "coverage": "all",
	}
	if _, _, err := server.serve(ctx, factory.MCPToolFinishProductPlan, argsFor(plain)); err != nil {
		t.Fatalf("plan without proposal refused: %v", err)
	}
}

// TestFactoryCatalogRateLimit bounds traffic per run without
// recording receipts.
func TestFactoryCatalogRateLimit(t *testing.T) {
	limiter := newMCPRateLimiter()
	now := time.Now()
	limiter.now = func() time.Time { return now }
	var releases []func()
	for range 60 {
		release, err := limiter.admit("run-1")
		if err != nil {
			t.Fatalf("admitted call refused: %v", err)
		}
		releases = append(releases, release)
		for _, release := range releases {
			release()
		}
		releases = releases[:0]
	}
	if _, err := limiter.admit("run-1"); err == nil {
		t.Fatal("61st call in the minute admitted")
	}
	limiter.now = func() time.Time { return now.Add(61 * time.Second) }
	release, err := limiter.admit("run-1")
	if err != nil {
		t.Fatalf("fresh window refused: %v", err)
	}
	release()
	if _, err := limiter.admit("unknown"); err != nil {
		t.Fatalf("other run refused: %v", err)
	}
}

// TestFactoryCatalogHTTPRoute proves the fixed serving route: POST
// /mcp initializes through the SDK handler with a JSON response,
// GET returns 405 with no SSE stream, and browser Origin refuses.
func TestFactoryCatalogHTTPRoute(t *testing.T) {
	sink := newFakeCatalogSink()
	srv := httptest.NewServer(factoryCatalogMux(newFactoryCatalogServer(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink})))
	defer srv.Close()

	if res, err := http.Get(srv.URL + FactoryMCPRoute); err != nil {
		t.Fatal(err)
	} else {
		defer res.Body.Close()
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("GET /mcp = %d, want 405", res.StatusCode)
		}
		if ct := res.Header.Get("Content-Type"); strings.Contains(ct, "text/event-stream") {
			t.Fatalf("GET /mcp serves SSE content %q", ct)
		}
	}

	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"fw38-proof","version":"1.0"}}}`
	post := func(origin string) *http.Response {
		req, err := http.NewRequest(http.MethodPost, srv.URL+FactoryMCPRoute, strings.NewReader(initBody))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	res := post("")
	raw, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /mcp initialize = %d (%s), want 200", res.StatusCode, raw)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("POST /mcp content-type = %q, want JSON", ct)
	}
	res = post("https://example.test")
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("POST /mcp with Origin = %d, want 403", res.StatusCode)
	}
}
