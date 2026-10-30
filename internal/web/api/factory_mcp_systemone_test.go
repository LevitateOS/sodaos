package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeSystemOne struct {
	mu    sync.Mutex
	calls int
	kinds []string
	fail  bool
}

func (f *fakeSystemOne) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeSystemOne) EvaluateQuestions(_ context.Context, k string, _ any, questions map[string]factory.SystemOneWorkerQuestion) (map[string]any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	answers := make(map[string]any, len(questions))
	for id := range questions {
		var answer map[string]any
		switch k {
		case "noul":
			answer = map[string]any{"noul": 0.7}
		case "choice":
			answer = map[string]any{"choice": "a"}
		default:
			answer = map[string]any{"score": 1.0}
		}
		answers[id] = answer
	}
	return map[string]any{"answers": answers, "model": "fixture"}, nil
}

func (f *fakeSystemOne) Evaluate(ctx context.Context, k string, state, instructions, criteria any) (map[string]any, error) {
	f.mu.Lock()
	f.calls++
	f.kinds = append(f.kinds, k)
	fail := f.fail
	f.mu.Unlock()
	if fail {
		return nil, errors.New("provider exploded")
	}
	switch k {
	case "noul":
		return map[string]any{"answer": map[string]any{"noul": 0.7}, "model": "fixture"}, nil
	case "choice":
		return map[string]any{"answer": map[string]any{"choice": "a"}, "model": "fixture"}, nil
	default:
		return map[string]any{"answer": map[string]any{"score": 1.0}, "model": "fixture"}, nil
	}
}

func TestBoundGenericSystemOneTools(t *testing.T) {
	ctx := context.Background()
	sink := newFakeCatalogSink()
	provider := &fakeSystemOne{}
	key := []byte(strings.Repeat("k", 32))
	secret, _ := factory.DeriveMCPCapabilitySecret(key, sink.capability)
	srv := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink, PriorityJudgments: sink, SystemOne: provider}, key))
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "generic-test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{sink.capability.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	for _, kind := range []string{"noul", "choice", "score"} {
		args := map[string]any{"state": map[string]any{"task": "demo"}, "instructions": "Judge the supplied task"}
		if kind == "choice" {
			args["criteria"] = map[string]any{"a": "A", "b": "B"}
		}
		if kind == "score" {
			args["criteria"] = []string{"low", "high"}
		}
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone." + kind, Arguments: args})
		if catalogRefused(result, err) {
			t.Fatalf("%s: %v %v", kind, result, err)
		}
	}
	if provider.count() != 3 {
		t.Fatalf("calls=%d", provider.count())
	}
	bad, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.score", Arguments: map[string]any{"state": "task", "instructions": "Score", "criteria": []string{"only"}}})
	if !catalogRefused(bad, err) || provider.count() != 3 {
		t.Fatal("invalid score called provider")
	}
}

// Consultations call the provider directly, including repeated questions and
// a new request after a provider failure. No reservation can strand the worker.
func TestBoundSystemOneConsultationWithoutReplayReceipts(t *testing.T) {
	ctx := context.Background()
	sink := newFakeCatalogSink()
	provider := &fakeSystemOne{}
	key := []byte(strings.Repeat("k", 32))
	secret, _ := factory.DeriveMCPCapabilitySecret(key, sink.capability)
	srv := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink, PriorityJudgments: sink, SystemOne: provider}, key))
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "consultation-test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{sink.capability.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	args := map[string]any{"state": map[string]any{"task": "demo"}, "instructions": "Judge the supplied task"}
	first, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.noul", Arguments: args})
	if catalogRefused(first, err) {
		t.Fatalf("first call: %v %v", first, err)
	}
	repeat, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.noul", Arguments: args})
	if catalogRefused(repeat, err) {
		t.Fatalf("replay: %v %v", repeat, err)
	}
	if provider.count() != 2 {
		t.Fatalf("repeated question did not call the provider: calls=%d", provider.count())
	}
	if catalogText(t, first) != catalogText(t, repeat) {
		t.Fatal("replay returned a different answer")
	}

	provider.fail = true
	failing := map[string]any{"state": map[string]any{"task": "broken"}, "instructions": "Judge the broken task"}
	failed, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.noul", Arguments: failing})
	if !catalogRefused(failed, err) {
		t.Fatal("failed consultation reported success")
	}
	held, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.noul", Arguments: failing})
	if !catalogRefused(held, err) {
		t.Fatal("failed consultation reported success on a second request")
	}
	if provider.count() != 4 {
		t.Fatalf("calls=%d, want two successes and two failures", provider.count())
	}
	provider.fail = false
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.noul", Arguments: failing})
	if catalogRefused(result, err) || provider.count() != 5 {
		t.Fatal("a failed consultation left a reservation blocking the next request", result, err)
	}
}

// The real PostgreSQL conversation keeps the question and answer without a
// second durable copy in the priority judgment journal.
func TestBoundSystemOneConsultationKeepsConversationWithoutReceipts(t *testing.T) {
	ctx := context.Background()
	admin, ok := store.TestFixtureDSN()
	if !ok {
		t.Skip("SODA_PG fixture unavailable")
	}
	st, dsn, cleanup, err := store.OpenEphemeral(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	capability := fakeCatalogCapability()
	capability.ID = strings.Repeat("c", 32)
	capability.AssignmentID = strings.Repeat("d", 32)
	capability.State = factory.MCPCapabilityStaged
	capability.DeadlineUnix = time.Now().Add(time.Hour).Unix()
	if err := st.StageMCPCapability(ctx, capability); err != nil {
		t.Fatal(err)
	}
	if err := st.ActivateMCPCapability(ctx, capability.ID, capability.Generation, capability.RunID, capability.AssignmentID, capability.Authority, time.Now().Unix(), false); err != nil {
		t.Fatal(err)
	}
	qualified := newFakeCatalogSink()
	assignment := qualified.assignments[qualified.capability.AssignmentID]
	assignment.ID = capability.AssignmentID
	assignment.Repository, assignment.Issue, assignment.Run = capability.Repository, 42, capability.RunID
	// The real journal also attaches its actual question/result to this
	// canonical work post; the PostgreSQL fixture must own that assignment.
	seedLiveMCPAssignment(t, ctx, dsn, assignment)
	qualified.capability = capability
	qualified.assignments = map[string]factory.Assignment{assignment.ID: assignment}
	capability.State = factory.MCPCapabilityActive
	qualified.capability = capability
	provider := &fakeSystemOne{}
	key := []byte(strings.Repeat("k", 32))
	secret, err := factory.DeriveMCPCapabilitySecret(key, capability)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: qualified, Submissions: qualified, Assignments: qualified, PriorityJudgments: st, Runs: qualified, SystemOne: provider}, key))
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "consultation-pg-test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{capability.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	state := map[string]any{"task": "demo"}
	const instructions = "Judge the supplied task"
	args := map[string]any{"state": state, "instructions": instructions}
	first, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.noul", Arguments: args})
	if catalogRefused(first, err) {
		t.Fatalf("first call: %v %v", first, err)
	}
	repeat, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.noul", Arguments: args})
	if catalogRefused(repeat, err) {
		t.Fatalf("replay: %v %v", repeat, err)
	}
	if provider.count() != 2 {
		t.Fatalf("repeated question did not call the provider: calls=%d", provider.count())
	}
	snapshot, err := factory.SystemOneWorkerSnapshot(capability.ID, factory.MCPToolSystemOneNoul, state, instructions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.PriorityJudgment(ctx, snapshot); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("consultation wrote an unnecessary judgment receipt: %v", err)
	}
	comment, err := st.FeedPost(ctx, factory.FeedConsultationPostID(snapshot))
	if err != nil || comment.Consultation == nil || comment.Consultation.Status != "returned" || comment.Consultation.Instructions != instructions || comment.Consultation.Answer == nil {
		t.Fatalf("actual MCP question/result did not reach its work thread: %v", err)
	}
	thread, _, err := st.FeedReplyIDs(ctx, capability.Repository, "record", capability.AssignmentID, 0)
	if err != nil || len(thread) != 1 || thread[0] != comment.Record.ID {
		t.Fatalf("consultation replay duplicated or lost its thread: %v %v", thread, err)
	}
}

// A batch uses one provider call. Invalid batches refuse before dispatch.
func TestBoundSystemOneBatchQuestionsSingleRoundTrip(t *testing.T) {
	ctx := context.Background()
	sink := newFakeCatalogSink()
	provider := &fakeSystemOne{}
	key := []byte(strings.Repeat("k", 32))
	secret, _ := factory.DeriveMCPCapabilitySecret(key, sink.capability)
	srv := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink, PriorityJudgments: sink, SystemOne: provider}, key))
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "batch-test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{sink.capability.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	batch := map[string]any{
		"state": map[string]any{"task": "demo"},
		"questions": map[string]any{
			"q1": map[string]any{"instructions": "Pick one", "criteria": map[string]any{"a": "A", "b": "B"}},
			"q2": map[string]any{"instructions": "Pick another", "criteria": map[string]any{"c": "C", "d": "D"}},
		},
	}
	first, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.choice", Arguments: batch})
	if catalogRefused(first, err) {
		t.Fatalf("batch: %v %v", first, err)
	}
	if provider.count() != 1 {
		t.Fatalf("batch dispatched %d provider calls, want 1", provider.count())
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(catalogText(t, first)), &envelope); err != nil {
		t.Fatal(err)
	}
	answers, _ := envelope["answers"].(map[string]any)
	q1, _ := answers["q1"].(map[string]any)
	q2, _ := answers["q2"].(map[string]any)
	if q1["choice"] != "a" || q2["choice"] != "a" {
		t.Fatalf("batch answers: %+v", envelope)
	}
	repeat, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.choice", Arguments: batch})
	if catalogRefused(repeat, err) {
		t.Fatalf("batch replay: %v %v", repeat, err)
	}
	if provider.count() != 2 {
		t.Fatalf("repeated batch did not call the provider: calls=%d", provider.count())
	}
	if catalogText(t, repeat) != catalogText(t, first) {
		t.Fatal("batch replay returned different bytes")
	}
	mixed := map[string]any{"state": map[string]any{"task": "demo"}, "instructions": "Pick", "questions": batch["questions"]}
	if res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.choice", Arguments: mixed}); !catalogRefused(res, err) {
		t.Fatal("mixed single and batch questions admitted")
	}
	over := map[string]any{"state": map[string]any{"task": "demo"}, "questions": map[string]any{}}
	for i := 0; i < factory.SystemOneMaxBatchQuestions+1; i++ {
		over["questions"].(map[string]any)["q"+strings.Repeat("x", i+1)] = map[string]any{"instructions": "Pick"}
	}
	if res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.noul", Arguments: over}); !catalogRefused(res, err) {
		t.Fatal("over-bound batch admitted")
	}
	if provider.count() != 2 {
		t.Fatalf("refused batches spent consultations: calls=%d", provider.count())
	}
}
