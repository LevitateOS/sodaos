package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFactoryBoundMCPIdentityDiscoveryAndRevocation(t *testing.T) {
	ctx := context.Background()
	sink := newFakeCatalogSink()
	assignment := sink.assignments[sink.capability.AssignmentID]
	assignment.Issue, assignment.ActorID = 42, 7
	sink.assignments[assignment.ID] = assignment
	key := []byte(strings.Repeat("k", 32))
	secret, err := factory.DeriveMCPCapabilitySecret(key, sink.capability)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink}, key))
	defer srv.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "bound-worker-test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + FactoryMCPRoute, HTTPClient: &http.Client{Transport: workerMCPTransport{sink.capability.ID, secret}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	list, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, tool := range list.Tools {
		seen[tool.Name] = true
		if !factory.MCPToolAdmitted(sink.capability.Profile, factory.MCPTool(tool.Name)) {
			t.Fatalf("foreign tool advertised: %s", tool.Name)
		}
		raw, _ := json.Marshal(tool.InputSchema)
		if strings.Contains(string(raw), "capability_id") {
			t.Fatalf("identity leaked into model arguments: %s", tool.Name)
		}
	}
	if !seen["factory.partial"] || !seen["factory.record_decision"] || !seen[string(factory.MCPToolFinishImplement)] {
		t.Fatalf("incomplete worker catalog: %v", seen)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{}})
	if catalogRefused(result, err) {
		t.Fatalf("bound context: %v %v", result, err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(catalogText(t, result)), &body); err != nil {
		t.Fatal(err)
	}
	if body["project"] != sink.capability.Project || body["run"] != sink.capability.RunID || body["issue"] != float64(42) || body["actor_id"] != float64(7) {
		t.Fatalf("wrong bound metadata: %v", body)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "factory.context", Arguments: map[string]any{"capability_id": sink.capability.ID}})
	if !catalogRefused(result, err) {
		t.Fatal("model could supply identity")
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: string(factory.MCPToolFinishImplement), Arguments: map[string]any{"submission_id": strings.Repeat("d", 32), "payload": map[string]any{"unchanged": true}}})
	if catalogRefused(result, err) {
		t.Fatalf("default bound revision refused: %v %v", result, err)
	}
	sink.mu.Lock()
	sink.capability.State = factory.MCPCapabilityRevoked
	sink.mu.Unlock()
	if _, err = session.ListTools(ctx, nil); err == nil {
		t.Fatal("revoked worker still discovers tools")
	}
}

func TestFactoryBoundMCPRejectsForeignSecret(t *testing.T) {
	sink := newFakeCatalogSink()
	handler := factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink}, []byte(strings.Repeat("k", 32)))
	request := httptest.NewRequest(http.MethodPost, FactoryMCPRoute, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	request.Header.Set("X-Soda-Factory-Capability", sink.capability.ID)
	request.Header.Set("Authorization", "Bearer "+strings.Repeat("a", 64))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("foreign secret: %d", response.Code)
	}
	body, _ := io.ReadAll(response.Result().Body)
	if strings.Contains(string(body), sink.capability.AssignmentID) {
		t.Fatal("refusal disclosed worker context")
	}
}
