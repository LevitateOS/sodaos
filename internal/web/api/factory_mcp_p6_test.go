// P6/T47 protocol proof (FW33).
//
// For each exact admitted harness the pinned SDK server is driven over
// a real native child on stdio: initialize, discovery, allowed read,
// typed finish, lost-response receipt recovery and forbidden-call
// refusal. Unsupported harnesses stay unavailable. The private HTTP
// leg proves the fixed serving route: POST /mcp initializes through
// the SDK handler while GET returns 405 with no SSE stream.

package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestFactoryMCPServerProcess is the stdio helper: the test client
// below re-executes the test binary with SODA_FACTORY_MCP_P6 set and
// speaks MCP to this server over the child's stdin/stdout.
func TestFactoryMCPServerProcess(t *testing.T) {
	if os.Getenv("SODA_FACTORY_MCP_P6") == "" {
		return
	}
	server := newFactoryMCPServer(newP6Store())
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		t.Logf("p6 helper server exited: %v", err)
	}
}

func p6Dial(t *testing.T, ctx context.Context) *mcp.ClientSession {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestFactoryMCPServerProcess")
	cmd.Env = append(os.Environ(), "SODA_FACTORY_MCP_P6=1")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "p6-proof"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("initialize over stdio: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func p6Call(t *testing.T, ctx context.Context, session *mcp.ClientSession, name string, args map[string]any) (*mcp.CallToolResult, error) {
	t.Helper()
	return session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
}

// p6Refused reports a refused call: a protocol error or an MCP
// isError result. Handler errors arrive as isError results, not as
// Go errors, so both shapes must count as refusal.
func p6Refused(res *mcp.CallToolResult, err error) bool {
	return err != nil || (res != nil && res.IsError)
}

func p6Text(t *testing.T, res *mcp.CallToolResult) string {
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

// TestFactoryMCPTransportPerHarness proves T47 per exact admitted
// harness: serving init, allowed read/finish and receipt recovery
// work; forbidden discovery/direct calls refuse.
func TestFactoryMCPTransportPerHarness(t *testing.T) {
	harnesses := []struct{ name, vers string }{
		{"muse", "1.4.9"},
		{"codex", "0.161.0"},
	}
	for _, h := range harnesses {
		t.Run(h.name+"@"+h.vers, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			session := p6Dial(t, ctx)

			tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
			if err != nil {
				t.Fatalf("discovery: %v", err)
			}
			seen := map[string]bool{}
			for _, tool := range tools.Tools {
				seen[tool.Name] = true
			}
			for _, want := range []string{"factory.context", "factory.finish", "factory.receipt"} {
				if !seen[want] {
					t.Fatalf("admitted tool %s missing from discovery", want)
				}
			}
			// Forbidden discovery: no merge, retry, policy or
			// provider control may be advertised.
			for name := range seen {
				if strings.HasPrefix(name, "factory.merge") || strings.HasPrefix(name, "factory.retry") ||
					name == "exec" || name == "shell" {
					t.Fatalf("forbidden tool %s advertised", name)
				}
			}

			id := strings.Repeat("a", 31) + "0"
			res, err := p6Call(t, ctx, session, "factory.context", map[string]any{
				"harness": h.name, "vers": h.vers,
			})
			if err != nil {
				t.Fatalf("allowed read: %v", err)
			}
			if body := p6Text(t, res); !strings.Contains(body, h.name) || !strings.Contains(body, FactoryMCPRoute) {
				t.Fatalf("allowed read body %q lacks harness/route", body)
			}

			finish := map[string]any{"harness": h.name, "vers": h.vers, "submission_id": id, "outcome": "done"}
			first, err := p6Call(t, ctx, session, "factory.finish", finish)
			if err != nil {
				t.Fatalf("typed finish: %v", err)
			}
			if body := p6Text(t, first); !strings.Contains(body, `"state":"recorded"`) {
				t.Fatalf("finish receipt %q not recorded", body)
			}
			// Exact replay returns the existing receipt.
			replay, err := p6Call(t, ctx, session, "factory.finish", finish)
			if err != nil {
				t.Fatalf("exact replay: %v", err)
			}
			if p6Text(t, replay) != p6Text(t, first) {
				t.Fatalf("replay receipt differs from recorded receipt")
			}
			// Conflicting bytes under the same ID refuse.
			clash := map[string]any{"harness": h.name, "vers": h.vers, "submission_id": id, "outcome": "blocked"}
			if res, err := p6Call(t, ctx, session, "factory.finish", clash); !p6Refused(res, err) {
				t.Fatal("conflicting terminal submission admitted")
			}
			// Lost-response recovery through factory.receipt.
			lost, err := p6Call(t, ctx, session, "factory.receipt", map[string]any{"submission_id": id})
			if err != nil {
				t.Fatalf("receipt recovery: %v", err)
			}
			if p6Text(t, lost) != p6Text(t, first) {
				t.Fatal("recovered receipt differs from recorded receipt")
			}
			// Forbidden direct call refuses.
			if res, err := p6Call(t, ctx, session, "factory.merge", map[string]any{}); !p6Refused(res, err) {
				t.Fatal("forbidden direct call admitted")
			}
			// Unsupported harness stays unavailable.
			if res, err := p6Call(t, ctx, session, "factory.context", map[string]any{
				"harness": "shell", "vers": "9.9.9",
			}); !p6Refused(res, err) {
				t.Fatal("unsupported harness admitted")
			}
		})
	}
}

// TestFactoryMCPUnsupportedHarness pins the unavailable set without a
// child: anything outside the exact admitted harness versions refuses.
func TestFactoryMCPUnsupportedHarness(t *testing.T) {
	for _, h := range [][2]string{
		{"muse", "1.4.2"},
		{"muse", "9.9.9"},
		{"codex", "0.160.0"},
		{"codex", "0.162.0"},
		{"shell", "1.4.9"},
		{"muse", ""},
	} {
		if p6AdmittedHarness(h[0], h[1]) {
			t.Fatalf("%s@%s: unsupported harness admitted", h[0], h[1])
		}
	}
}

// TestFactoryMCPHTTPRoute proves the fixed serving route: POST /mcp
// initializes through the SDK handler with a JSON response, while GET
// returns 405 and establishes no SSE stream.
func TestFactoryMCPHTTPRoute(t *testing.T) {
	srv := httptest.NewServer(factoryMCPMux(newFactoryMCPServer(newP6Store())))
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

	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"p6-proof","version":"1.0"}}}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+FactoryMCPRoute, strings.NewReader(initBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Spec transport negotiation: the client offers both framings
	// while the stateless JSON server answers application/json.
	req.Header.Set("Accept", "application/json, text/event-stream")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /mcp initialize = %d (%s), want 200", res.StatusCode, raw)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("POST /mcp content-type = %q, want JSON", ct)
	}
}
