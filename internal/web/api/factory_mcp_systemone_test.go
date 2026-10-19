package api

import (
	"context"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeSystemOne struct {
	calls int
	kinds []string
}

func (f *fakeSystemOne) Evaluate(_ context.Context, k string, state, instructions, criteria any) (map[string]any, error) {
	f.calls++
	f.kinds = append(f.kinds, k)
	return map[string]any{"answer": map[string]any{"type": k}, "model": "fixture"}, nil
}
func TestBoundGenericSystemOneTools(t *testing.T) {
	ctx := context.Background()
	sink := newFakeCatalogSink()
	provider := &fakeSystemOne{}
	key := []byte(strings.Repeat("k", 32))
	secret, _ := factory.DeriveMCPCapabilitySecret(key, sink.capability)
	srv := httptest.NewServer(factoryBoundCatalogHandler(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink, SystemOne: provider}, key))
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
	if provider.calls != 3 {
		t.Fatalf("calls=%d", provider.calls)
	}
	bad, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "systemone.score", Arguments: map[string]any{"state": "task", "instructions": "Score", "criteria": []string{"only"}}})
	if !catalogRefused(bad, err) || provider.calls != 3 {
		t.Fatal("invalid score called provider")
	}
}
