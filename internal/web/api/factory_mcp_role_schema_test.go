package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFactoryRoleDiscoverySchemasFitRelayAndExposeResults(t *testing.T) {
	for _, profile := range factory.MCPProfiles {
		t.Run(string(profile), func(t *testing.T) {
			ctx := context.Background()
			sink := newFakeCatalogSink()
			sink.capability.Profile = profile
			server := newFactoryCatalogServer(FactoryMCPDeps{Capabilities: sink, Submissions: sink, Assignments: sink, BoundCapability: &sink.capability})
			clientTransport, serverTransport := mcp.NewInMemoryTransports()
			serverSession, err := server.Connect(ctx, serverTransport, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer serverSession.Close()
			client := mcp.NewClient(&mcp.Implementation{Name: "role-schema-test"}, nil)
			session, err := client.Connect(ctx, clientTransport, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			listed, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(listed)
			if err != nil {
				t.Fatal(err)
			}
			if len(encoded) > 256*1024 {
				t.Fatalf("discovery exceeds relay response bound: %d", len(encoded))
			}
			if strings.Contains(string(encoded), `"$ref"`) {
				t.Fatal("embedded result schemas require unbound root references")
			}
			foundFinish := false
			for _, tool := range listed.Tools {
				if !factory.MCPIsTerminal(factory.MCPTool(tool.Name)) || tool.Name == string(factory.MCPToolFinishIncomplete) {
					continue
				}
				foundFinish = true
				raw, _ := json.Marshal(tool.InputSchema)
				var envelope struct {
					Properties map[string]struct {
						Properties map[string]json.RawMessage `json:"properties"`
					} `json:"properties"`
				}
				if err := json.Unmarshal(raw, &envelope); err != nil {
					t.Fatal(err)
				}
				if len(envelope.Properties["payload"].Properties) == 0 {
					t.Fatalf("%s hides its result fields", tool.Name)
				}
			}
			if !foundFinish {
				t.Fatal("role finish absent")
			}
			t.Logf("role discovery: %d bytes", len(encoded))
		})
	}
}
