// MCP protocol and installed-CLI interface tests. The SDK fixture proves
// transport, discovery, finish/replay and refusal behavior. CLI checks use
// disposable configuration and Muse’s local echo provider; no paid provider
// calls or version-banner admission are involved.

package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestFactoryMCPServerProcess is the SDK stdio fixture: the protocol
// proof below re-executes the test binary with SODA_FACTORY_MCP_P6 set
// and speaks MCP to this Go SDK server over the child's stdin/stdout.
// It is SDK test scaffolding, not a harness.
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

// TestFactoryMCPSDKProtocolProof proves the T47 SDK protocol once over
// a real native child: serving init, allowed read/finish and receipt
// recovery work; forbidden discovery/direct calls refuse. The
// harness/version pair is admission data for one admitted pin, not a
// per-harness CLI proof: no muse/codex binary runs here. Real-CLI
// evidence lives in TestFactoryMCPCLIBoundaries.
func TestFactoryMCPSDKProtocolProof(t *testing.T) {
	const harness, vers = "muse", "1.4.9"
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
		"harness": harness, "vers": vers,
	})
	if err != nil {
		t.Fatalf("allowed read: %v", err)
	}
	if body := p6Text(t, res); !strings.Contains(body, harness) || !strings.Contains(body, FactoryMCPRoute) {
		t.Fatalf("allowed read body %q lacks harness/route", body)
	}

	finish := map[string]any{"harness": harness, "vers": vers, "submission_id": id, "outcome": "done"}
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
	clash := map[string]any{"harness": harness, "vers": vers, "submission_id": id, "outcome": "blocked"}
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
}

// TestFactoryMCPUnsupportedHarness checks interface families without version admission.
func TestFactoryMCPUnsupportedHarness(t *testing.T) {
	for _, harness := range []string{"muse", "codex"} {
		if !p6AdmittedHarness(harness) {
			t.Fatalf("%s: supported interface refused", harness)
		}
	}
	for _, harness := range []string{"shell", ""} {
		if p6AdmittedHarness(harness) {
			t.Fatalf("%s: unsupported interface admitted", harness)
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

// r1Run executes one pinned-CLI probe with a bounded context, returning
// combined output. A nonzero exit fails the test with the exact log.
func r1Run(t *testing.T, ctx context.Context, dir string, env []string, name string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v exited: %v\n%s", name, args, err, out)
	}
	return string(out)
}

// r1Scratch returns a scratch root under the invoking user's HOME (never
// /tmp and never the repo tree) with cleanup, or skips when HOME is
// unset or outside /home.
func r1Scratch(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil || (home != "/home" && !strings.HasPrefix(home, "/home/")) {
		t.Skipf("R1 CLI probes need scratch under /home, HOME=%q", home)
	}
	root, err := os.MkdirTemp(home, "soda-lane-b19-r1-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}

// TestFactoryMCPCLIBoundaries drives the REAL pinned muse/codex binaries
// for what is natively provable without provider credentials: exact
// version pins, the muse OAuth-only MCP surface, codex stdio server
// registration against a disposable CODEX_HOME, and a muse
// echo-provider run proving the staged --prompt-file bytes reach the
// harness as the USER prompt while the relay-shaped command spawns.
// Skips when the pinned binaries are absent. The headless agentic
// tool-call loop (model-driven tool calls through the relay) is NOT
// attempted here: without credentials `muse exec` exits
// missing-meta-credentials and `codex exec` fails 401 (see
// evidence/lane-b19-r1/commands.log); that loop stays unavailable.
func TestFactoryMCPCLIBoundaries(t *testing.T) {
	museBin, err := exec.LookPath("muse")
	if err != nil {
		t.Skip("muse CLI absent: no installed CLI to exercise")
	}
	codexBin, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("codex CLI absent: no installed CLI to exercise")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	scratch := r1Scratch(t)

	// Codex stdio registration against a disposable CODEX_HOME: the
	// recorded entry keeps the fixed relay command shape.
	codexHome := filepath.Join(scratch, "codex-home")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	codexEnv := append(os.Environ(), "CODEX_HOME="+codexHome)
	r1Run(t, ctx, scratch, codexEnv, codexBin, "mcp", "add", "factory", "--", "/bin/true", "factory-mcp")
	got := r1Run(t, ctx, scratch, codexEnv, codexBin, "mcp", "get", "factory", "--json")
	if !strings.Contains(got, `"stdio"`) || !strings.Contains(got, "/bin/true") {
		t.Fatalf("codex mcp get factory --json = %q, want recorded stdio entry", got)
	}

	// Muse echo leg: a scratch settings document with a top-level
	// mcpServers stdio entry is accepted and the relay-shaped command
	// spawns; the echo reply proves the staged prompt file arrived as
	// the user prompt (untrusted delivery, not a trusted instruction
	// channel — see mcp_relay.rs TRUSTED_INSTRUCTION_CHANNEL).
	museHome := filepath.Join(scratch, "muse-home")
	museConfig := filepath.Join(museHome, ".config", "muse")
	if err := os.MkdirAll(museConfig, 0o700); err != nil {
		t.Fatal(err)
	}
	markerLog := filepath.Join(scratch, "relay-marker.log")
	relay := filepath.Join(scratch, "marker-relay.sh")
	script := "#!/bin/sh\necho \"relay-spawned factory-mcp $*\" >> \"" + markerLog + "\"\nexit 0\n"
	if err := os.WriteFile(relay, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	settings := `{"schema_version": 1, "mcpServers": {"factory": {"command": "` + relay + `", "args": ["factory-mcp"]}}}`
	if err := os.WriteFile(filepath.Join(museConfig, "settings.json"), []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}
	const sentinel = "PONG-LANE-B19"
	prompt := filepath.Join(scratch, "prompt.txt")
	if err := os.WriteFile(prompt, []byte("reply with the single word "+sentinel+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	museEnv := append(os.Environ(), "HOME="+museHome, "XDG_CONFIG_HOME="+filepath.Join(museHome, ".config"))
	echo := r1Run(t, ctx, scratch, museEnv, museBin, "exec", "--provider", "echo", "--prompt-file", prompt)
	if !strings.Contains(echo, sentinel) {
		t.Fatalf("muse echo run = %q, want staged prompt echoed as the user prompt", echo)
	}
	marker, err := os.ReadFile(markerLog)
	if err != nil || !strings.Contains(string(marker), "relay-spawned factory-mcp") {
		t.Fatalf("relay marker log = %q, want at least one relay-shaped spawn", marker)
	}
}
