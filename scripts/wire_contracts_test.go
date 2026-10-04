// Wire-contract pins for the language port: byte-exact client request bodies,
// response mapping, and strict JSON admission rules that Rust replacements
// must reproduce. Fixtures under fixtures/portcontracts are frozen shared
// contracts; port lanes may change this harness to invoke a new binary but
// must keep the fixture bytes identical.
package scripts

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/strictjson"
)

func portContractFixture(t *testing.T, name string, out any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("fixtures", "portcontracts", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatal(err)
	}
}

// vectorEnvelope mirrors the operator command envelope shape so the strict
// vectors exercise admission through a real request type.
type vectorEnvelope struct {
	CommandID string `json:"command_id,omitempty"`
	Type      string `json:"type"`
	Target    string `json:"target,omitempty"`
}

type strictVector struct {
	Name          string `json:"name"`
	Input         string `json:"input"`
	OK            bool   `json:"ok"`
	ErrorContains string `json:"error_contains"`
}

func TestStrictjsonWireVectors(t *testing.T) {
	var fixture struct {
		Vectors []strictVector `json:"vectors"`
	}
	portContractFixture(t, "strictjson_vectors.json", &fixture)
	if len(fixture.Vectors) == 0 {
		t.Fatal("strictjson vectors fixture is empty")
	}
	for _, v := range fixture.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			var decoded vectorEnvelope
			err := strictjson.Decode(strings.NewReader(v.Input), &decoded)
			if v.OK {
				if err != nil {
					t.Fatalf("valid input rejected: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("invalid input accepted: %q", v.Input)
			}
			if v.ErrorContains != "" && !strings.Contains(err.Error(), v.ErrorContains) {
				t.Fatalf("error %q lacks %q", err, v.ErrorContains)
			}
		})
	}
}

func TestStrictjsonWireLimits(t *testing.T) {
	t.Run("oversized", func(t *testing.T) {
		var decoded vectorEnvelope
		input := `{"type":"` + strings.Repeat("a", 1<<20) + `"}`
		err := strictjson.Decode(strings.NewReader(input), &decoded)
		if err == nil || !strings.Contains(err.Error(), "exceeds 1 MiB") {
			t.Fatalf("oversized input not refused: %v", err)
		}
	})
	t.Run("invalid utf8", func(t *testing.T) {
		var decoded vectorEnvelope
		input := append([]byte(`{"type":"`), 0xff)
		input = append(input, []byte(`"}`)...)
		err := strictjson.Decode(bytes.NewReader(input), &decoded)
		if err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
			t.Fatalf("invalid UTF-8 not refused: %v", err)
		}
	})
	t.Run("deep nesting", func(t *testing.T) {
		var decoded vectorEnvelope
		input := `{"type":` + strings.Repeat(`{"k":`, 150) + `1` + strings.Repeat(`}`, 150) + `}`
		err := strictjson.Decode(strings.NewReader(input), &decoded)
		if err == nil || !strings.Contains(err.Error(), "nested too deeply") {
			t.Fatalf("deep nesting not refused: %v", err)
		}
	})
}

type operatorWire struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	ClientHost  string `json:"client_host"`
	ContentType string `json:"content_type"`
	CommandID   string `json:"command_id"`
	RunID       string `json:"run_id"`
	Requests    []struct {
		Name string   `json:"name"`
		Argv []string `json:"argv"`
		Body string   `json:"body"`
	} `json:"requests"`
	Responses []struct {
		Name          string `json:"name"`
		Code          int    `json:"code"`
		Body          string `json:"body"`
		Stdout        string `json:"stdout"`
		StderrContain string `json:"stderr_contains"`
	} `json:"responses"`
	Misuse []struct {
		Name          string   `json:"name"`
		Argv          []string `json:"argv"`
		StderrContain string   `json:"stderr_contains"`
	} `json:"misuse"`
}

func buildPortBinary(t *testing.T, pkg string) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "soda-wire-probe")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, pkg)
	cmd.Dir = root
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, combined)
	}
	return out
}

type capturedRequest struct {
	method, path, host, contentType, body string
}

func wireStubServer(t *testing.T, code int, body string) (string, <-chan capturedRequest) {
	t.Helper()
	socket := filepath.Join(t.TempDir(), "operator.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	got := make(chan capturedRequest, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r.Body)
		got <- capturedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			host:        r.Host,
			contentType: r.Header.Get("Content-Type"),
			body:        buf.String(),
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return socket, got
}

func runPortBinary(t *testing.T, binary string, argv ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, argv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %v: %v", argv, err)
		}
		code = exit.ExitCode()
	}
	return stdout.String(), stderr.String(), code
}

func TestOperatorWireBytes(t *testing.T) {
	var wire operatorWire
	portContractFixture(t, "operator_wire.json", &wire)
	if wire.Path == "" || len(wire.Requests) == 0 {
		t.Fatal("operator wire fixture is empty")
	}
	binary := buildPortBinary(t, "./cmd/soda-factory")

	t.Run("requests", func(t *testing.T) {
		for _, v := range wire.Requests {
			t.Run(v.Name, func(t *testing.T) {
				socket, got := wireStubServer(t, http.StatusOK, `{"runs":[]}`)
				argv := append([]string{"--socket", socket}, v.Argv...)
				stdout, stderr, code := runPortBinary(t, binary, argv...)
				if code != 0 {
					t.Fatalf("exit %d: stdout %q stderr %q", code, stdout, stderr)
				}
				var req capturedRequest
				select {
				case req = <-got:
				case <-time.After(15 * time.Second):
					t.Fatal("client sent no request")
				}
				if req.method != wire.Method || req.path != wire.Path {
					t.Fatalf("got %s %s, want %s %s", req.method, req.path, wire.Method, wire.Path)
				}
				if req.host != wire.ClientHost {
					t.Fatalf("host %q, want %q", req.host, wire.ClientHost)
				}
				if req.contentType != wire.ContentType {
					t.Fatalf("content type %q, want %q", req.contentType, wire.ContentType)
				}
				if req.body != v.Body {
					t.Fatalf("body %q, want %q", req.body, v.Body)
				}
			})
		}
	})

	t.Run("responses", func(t *testing.T) {
		for _, v := range wire.Responses {
			t.Run(v.Name, func(t *testing.T) {
				socket, _ := wireStubServer(t, v.Code, v.Body)
				stdout, stderr, code := runPortBinary(t, binary, "--socket", socket, "status")
				if v.StderrContain == "" {
					if code != 0 {
						t.Fatalf("exit %d: stdout %q stderr %q", code, stdout, stderr)
					}
					if stdout != v.Stdout {
						t.Fatalf("stdout %q, want %q", stdout, v.Stdout)
					}
					return
				}
				if code == 0 {
					t.Fatalf("error response exited 0 with stdout %q", stdout)
				}
				if !strings.Contains(stderr, v.StderrContain) {
					t.Fatalf("stderr %q lacks %q", stderr, v.StderrContain)
				}
			})
		}
	})

	t.Run("misuse", func(t *testing.T) {
		for _, v := range wire.Misuse {
			t.Run(v.Name, func(t *testing.T) {
				argv := make([]string, len(v.Argv))
				for i, arg := range v.Argv {
					switch arg {
					case "SOCK":
						// Nonexistent on purpose: any dial attempt surfaces
						// as "operator endpoint unavailable", not the pinned
						// usage error.
						arg = filepath.Join(t.TempDir(), "unused.sock")
					case "CID":
						arg = wire.CommandID
					case "RUN":
						arg = wire.RunID
					}
					argv[i] = arg
				}
				_, stderr, code := runPortBinary(t, binary, argv...)
				if code == 0 {
					t.Fatalf("misuse exited 0: %v", argv)
				}
				if !strings.Contains(stderr, v.StderrContain) {
					t.Fatalf("stderr %q lacks %q", stderr, v.StderrContain)
				}
			})
		}
	})
}
