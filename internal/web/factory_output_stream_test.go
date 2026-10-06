package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/coder/websocket"
)

func TestFactoryOutputStreamDeliversStatusOutputAndEOF(t *testing.T) {
	script := &factoryOutputScript{offsets: make(chan float64, 16)}
	proxy, _, origin, s := factoryOutputProxyFixture(t, script)
	run := factoryViewsRun(t, s)
	script.serve = func(body map[string]any) (int, string) {
		offset, _ := body["offset"].(float64)
		if offset == 0 {
			return 200, `{"id":"` + run.ID + `","project":"` + webTerminalProject + `","phase":"running","live":true,` +
				`"container":"` + strings.Repeat("c", 64) + `","unit":"soda-factory-` + run.ID + `.service","invocation":"` + strings.Repeat("d", 32) + `",` +
				`"total":5,"offset":0,"next":5,"data":"aGVsbG8="}`
		}
		return 200, `{"id":"` + run.ID + `","project":"` + webTerminalProject + `","phase":"completed","terminal":true,"exit_code":0,` +
			`"container":"` + strings.Repeat("c", 64) + `","unit":"soda-factory-` + run.ID + `.service","invocation":"` + strings.Repeat("d", 32) + `",` +
			`"total":5,"offset":5,"next":5,"data":""}`
	}
	conn := dialFactoryOutput(t, proxy, origin, run.ID, factoryOutputHandshakeFor(run.ID))
	status := readFactoryFrame(t, conn)
	if status["type"] != "status" || status["run_id"] != run.ID || status["phase"] != "running" || status["live"] != true {
		t.Fatalf("status: %v", status)
	}
	if status["container"] != strings.Repeat("c", 64) || status["unit"] != "soda-factory-"+run.ID+".service" {
		t.Fatalf("process binding: %v", status)
	}
	output := readFactoryFrame(t, conn)
	if output["type"] != "output" || output["data"] != "aGVsbG8=" || output["cursor"] != float64(0) || output["next"] != float64(5) {
		t.Fatalf("output: %v", output)
	}
	if done := readFactoryFrame(t, conn); done["type"] != "status" || done["phase"] != "completed" || done["terminal"] != true {
		t.Fatalf("terminal status: %v", done)
	}
	if closed := readFactoryFrame(t, conn); closed["type"] != "closed" || closed["reason"] != "eof" {
		t.Fatalf("eof: %v", closed)
	}
	// Reattachment resumes by cursor through reads only; the host sees the
	// resumed offset and no launch-shaped call exists on this route.
	resume := factoryOutputHandshakeFor(run.ID)
	resume["cursor"] = 5
	reattached := dialFactoryOutput(t, proxy, origin, run.ID, resume)
	if status := readFactoryFrame(t, reattached); status["phase"] != "completed" {
		t.Fatalf("resumed status: %v", status)
	}
	if closed := readFactoryFrame(t, reattached); closed["reason"] != "eof" {
		t.Fatalf("resumed eof: %v", closed)
	}
	sawResume := false
	for {
		select {
		case offset := <-script.offsets:
			if offset == 5 {
				sawResume = true
			}
		default:
			if !sawResume {
				t.Fatal("resumed cursor never reached the host")
			}
			return
		}
	}
}

func TestFactoryOutputStreamRejectsInput(t *testing.T) {
	script := &factoryOutputScript{offsets: make(chan float64, 16)}
	proxy, _, origin, s := factoryOutputProxyFixture(t, script)
	run := factoryViewsRun(t, s)
	script.serve = func(body map[string]any) (int, string) {
		return 200, `{"id":"` + run.ID + `","project":"` + webTerminalProject + `","phase":"running","live":true,"total":0,"offset":0,"next":0,"data":""}`
	}
	conn := dialFactoryOutput(t, proxy, origin, run.ID, factoryOutputHandshakeFor(run.ID))
	if status := readFactoryFrame(t, conn); status["type"] != "status" {
		t.Fatalf("status: %v", status)
	}
	ctx, done := context.WithTimeout(t.Context(), 5*time.Second)
	defer done()
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"input","data":"ZWNobyBoaQ=="}`)); err != nil {
		t.Fatal(err)
	}
	_, _, err := conn.Read(ctx)
	var closed websocket.CloseError
	if !errors.As(err, &closed) || closed.Code != websocket.StatusPolicyViolation {
		t.Fatalf("input accepted: %v", err)
	}
}

func TestFactoryOutputStreamRefusesStaleHandshake(t *testing.T) {
	script := &factoryOutputScript{offsets: make(chan float64, 16)}
	script.serve = func(body map[string]any) (int, string) {
		return 200, `{"phase":"running"}`
	}
	proxy, _, origin, s := factoryOutputProxyFixture(t, script)
	run := factoryViewsRun(t, s)
	stale := factoryOutputHandshakeFor(run.ID)
	stale["session_generation"] = "stale-generation"
	conn := dialFactoryOutput(t, proxy, origin, run.ID, stale)
	ctx, done := context.WithTimeout(t.Context(), 5*time.Second)
	defer done()
	if _, _, err := conn.Read(ctx); err == nil {
		t.Fatal("stale handshake attached")
	}
	if calls := script.calls.Load(); calls != 0 {
		t.Fatalf("stale handshake reached the host %d times", calls)
	}
}

func TestFactoryOutputStreamClosesOnUnknownAndStale(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		reason string
	}{
		{"unknown", 404, "run_unknown"},
		{"stale", 409, "incarnation_changed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := &factoryOutputScript{offsets: make(chan float64, 16)}
			script.serve = func(body map[string]any) (int, string) { return tc.status, "helper refused" }
			proxy, _, origin, s := factoryOutputProxyFixture(t, script)
			run := factoryViewsRun(t, s)
			conn := dialFactoryOutput(t, proxy, origin, run.ID, factoryOutputHandshakeFor(run.ID))
			if closed := readFactoryFrame(t, conn); closed["type"] != "closed" || closed["reason"] != tc.reason {
				t.Fatalf("close: %v", closed)
			}
		})
	}
}

func TestFactoryOutputStreamRequiresWrite(t *testing.T) {
	script := &factoryOutputScript{offsets: make(chan float64, 16)}
	script.serve = func(body map[string]any) (int, string) { return 200, `{}` }
	proxy, permission, origin, s := factoryOutputProxyFixture(t, script)
	run := factoryViewsRun(t, s)
	permission.Store("read")
	wsURL := "wss" + strings.TrimPrefix(proxy.URL, "https") + "/factory/runs/" + run.ID + "/output"
	headers := nativeTerminalHeaders(http.MethodGet, origin)
	headers.Del(extensions.SessionGenerationHeader)
	_, response, err := websocket.Dial(t.Context(), wsURL, &websocket.DialOptions{HTTPClient: proxy.Client(), HTTPHeader: headers})
	if err == nil {
		t.Fatal("read-only viewer attached")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("read-only dial: %v", response)
	}
	_ = response.Body.Close()
	if calls := script.calls.Load(); calls != 0 {
		t.Fatalf("refused viewer reached the host %d times", calls)
	}
}
