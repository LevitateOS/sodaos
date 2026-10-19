// P6 MCP transport fixture (FW33).
//
// Smallest SDK integration for D14/P6: the pinned upstream Go MCP SDK
// (github.com/modelcontextprotocol/go-sdk v1.8.0, see go.mod/go.sum)
// serves a three-tool P6 catalog over stdio and over the fixed Factory
// route. The private HTTP leg runs stateless with JSON responses: GET
// returns 405 and there is no server-initiated SSE stream. Capability
// admission, persistence and the remaining eleven tools ride FW38; the
// in-memory receipt store below only proves replay/recovery identity.

package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FactoryMCPRoute is the fixed Factory-only listener route. Clients
// choose no socket, URL or listener; the relay forwards here.
const FactoryMCPRoute = "/mcp"

// p6AdmittedHarness selects a supported interface, without version admission.
func p6AdmittedHarness(harness string) bool {
	return harness == "muse" || harness == "codex"
}

// p6Receipt is the durable terminal receipt for one typed finish.
type p6Receipt struct {
	SubmissionID string `json:"submission_id"`
	Outcome      string `json:"outcome"`
	State        string `json:"state"`
}

// p6Store records one receipt per submission ID. Exact replay returns
// the stored receipt; changed bytes under the same ID conflict.
type p6Store struct {
	mu       sync.Mutex
	receipts map[string]p6Receipt
}

func newP6Store() *p6Store { return &p6Store{receipts: map[string]p6Receipt{}} }

func isHex32(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, c := range []byte(s) {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

type p6ContextArgs struct {
	Harness string `json:"harness"`
	Vers    string `json:"vers,omitempty"`
}

type p6FinishArgs struct {
	Harness      string `json:"harness"`
	Vers         string `json:"vers,omitempty"`
	SubmissionID string `json:"submission_id"`
	Outcome      string `json:"outcome"`
}

type p6ReceiptArgs struct {
	SubmissionID string `json:"submission_id"`
}

// newFactoryMCPServer builds the P6 catalog: one allowed read, one
// typed finish and receipt recovery. No merge, retry, policy or
// provider control exists here, so discovery cannot advertise one.
func newFactoryMCPServer(store *p6Store) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "soda-factory-p6"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "factory.context",
		Description: "P6 allowed read: immutable assignment summary",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args p6ContextArgs) (*mcp.CallToolResult, any, error) {
		if !p6AdmittedHarness(args.Harness) {
			return nil, nil, fmt.Errorf("unsupported factory harness")
		}
		return &mcp.CallToolResult{Content: []mcp.Content{
			&mcp.TextContent{Text: `{"assignment":"p6","harness":"` + args.Harness + `","route":"` + FactoryMCPRoute + `"}`},
		}}, nil, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "factory.finish",
		Description: "P6 typed finish: exactly-once terminal receipt",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args p6FinishArgs) (*mcp.CallToolResult, any, error) {
		if !p6AdmittedHarness(args.Harness) {
			return nil, nil, fmt.Errorf("unsupported factory harness")
		}
		if !isHex32(args.SubmissionID) {
			return nil, nil, fmt.Errorf("unsupported submission identity")
		}
		if args.Outcome != "done" && args.Outcome != "blocked" {
			return nil, nil, fmt.Errorf("unsupported finish outcome")
		}
		store.mu.Lock()
		defer store.mu.Unlock()
		if prev, ok := store.receipts[args.SubmissionID]; ok {
			if prev.Outcome != args.Outcome {
				return nil, nil, fmt.Errorf("submission conflict")
			}
			return &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.TextContent{Text: `{"submission_id":"` + prev.SubmissionID + `","state":"` + prev.State + `"}`},
			}}, nil, nil
		}
		got := p6Receipt{SubmissionID: args.SubmissionID, Outcome: args.Outcome, State: "recorded"}
		store.receipts[args.SubmissionID] = got
		return &mcp.CallToolResult{Content: []mcp.Content{
			&mcp.TextContent{Text: `{"submission_id":"` + got.SubmissionID + `","state":"recorded"}`},
		}}, nil, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "factory.receipt",
		Description: "P6 receipt recovery: no new submission",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args p6ReceiptArgs) (*mcp.CallToolResult, any, error) {
		store.mu.Lock()
		defer store.mu.Unlock()
		got, ok := store.receipts[args.SubmissionID]
		if !ok {
			return nil, nil, fmt.Errorf("receipt not found")
		}
		return &mcp.CallToolResult{Content: []mcp.Content{
			&mcp.TextContent{Text: `{"submission_id":"` + got.SubmissionID + `","state":"` + got.State + `"}`},
		}}, nil, nil
	})
	return server
}

// factoryMCPMux serves the P6 server on the fixed route only: POST
// reaches the SDK handler (stateless JSON, no SSE); every other method
// — including the SSE-establishing GET — returns 405.
func factoryMCPMux(server *mcp.Server) *http.ServeMux {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	mux := http.NewServeMux()
	mux.HandleFunc(FactoryMCPRoute, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handler.ServeHTTP(w, r)
	})
	return mux
}
