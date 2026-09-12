package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/tailnet"
)

type tailnetRoundTrip func(*http.Request) (*http.Response, error)

func (f tailnetRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestTailnetNativeDisabledAndStrictRouting(t *testing.T) {
	calls := 0
	d := &Daemon{Exec: managementExec(func(context.Context, []byte, string, ...string) ([]byte, error) { calls++; return nil, nil })}
	for _, action := range []string{"settings", "host", "enrollment", "options", "project"} {
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest("POST", "/tailnet/"+action, strings.NewReader(`{}`)))
		if w.Code != 503 || calls != 0 {
			t.Fatal(w.Code, calls)
		}
	}
	d.Tailnet = tailnet.NewManagement()
	for _, tc := range []struct {
		path, body string
		status     int
	}{
		{"/tailnet/settings?", `{}`, 400},
		{"/tailnet/settings", `null`, 400},
		{"/tailnet/options", `{"socket":"/private"}`, 400},
		{"/tailnet/enrollment", `{"action":"save","secret":"synthetic-private"}`, 400},
		{"/tailnet/host", `{"action":"logout","revision":"a","confirm":"logout"}`, 400},
		{"/tailnet/project", `{"project":"../other","action":"inspect"}`, 400},
		{"/tailnet/unknown", `{}`, 404},
	} {
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body)))
		if w.Code != tc.status || calls != 0 || strings.Contains(w.Body.String(), "synthetic-private") {
			t.Fatal(tc.path, w.Code, w.Body.String(), calls)
		}
	}
}
func TestTailnetNativeDoesNotTakeProjectGateOrEnableRuntime(t *testing.T) {
	const id = "p0123456789abcdef01234567"
	calls := 0
	d := &Daemon{Tailnet: tailnet.NewManagement(), Exec: managementExec(func(ctx context.Context, in []byte, command string, args ...string) ([]byte, error) {
		calls++
		if command != "/usr/bin/podman" || len(args) != 5 || args[1] != "inspect" {
			t.Error("unexpected native call", command, args)
		}
		return []byte(fmt.Sprintf(`{"id":%q,"running":true,"project":%q,"owner":"1","privileged":false,"userns":"private","mappings":{"UidMap":["0:1000000:262144"],"GidMap":["0:1000000:262144"]}}`, strings.Repeat("a", 64), id)), nil
	})}
	if e := d.acquireAdmission(t.Context()); e != nil {
		t.Fatal(e)
	}
	defer func() { <-d.admission }()
	in := tailnet.ProjectRequest{Project: id, Action: "enable", Revision: "0", Binding: strings.Repeat("b", 32), ConfirmID: id}
	body, _ := json.Marshal(in)
	done := make(chan int, 1)
	go func() {
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest("POST", "/tailnet/project", strings.NewReader(string(body))))
		done <- w.Code
	}()
	select {
	case status := <-done:
		if status != 422 || calls != 1 {
			t.Fatal(status, calls)
		}
	case <-time.After(time.Second):
		t.Fatal("Tailnet blocked on unrelated project gate")
	}
}
func TestTailnetNativeRejectsProjectIsolationDrift(t *testing.T) {
	d := &Daemon{Tailnet: tailnet.NewManagement(), Exec: managementExec(func(context.Context, []byte, string, ...string) ([]byte, error) {
		return []byte(`{"id":"bad","running":true,"project":"p0123456789abcdef01234567","owner":"1","privileged":true,"userns":"host"}`), nil
	})}
	body := `{"project":"p0123456789abcdef01234567","action":"enable","revision":"0","binding":"` + strings.Repeat("a", 32) + `","confirm_id":"p0123456789abcdef01234567"}`
	w := httptest.NewRecorder()
	d.ServeHTTP(w, httptest.NewRequest("POST", "/tailnet/project", strings.NewReader(body)))
	if w.Code != 502 {
		t.Fatal(w.Code)
	}
}
func TestTailnetClientRejectsMalformedAndSecretBearingErrorResponses(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   error
	}{
		{200, `{}`, tailnet.ErrUnavailable},
		{200, `null`, tailnet.ErrUnconfirmed},
		{200, strings.Repeat(" ", 65537), tailnet.ErrUnconfirmed},
		{409, `synthetic-private`, tailnet.ErrConflict},
		{422, `synthetic-private`, tailnet.ErrUnsupported},
		{503, `synthetic-private`, tailnet.ErrUnavailable},
		{502, `synthetic-private`, tailnet.ErrUnconfirmed},
	} {
		c := &Client{HTTP: &http.Client{Transport: tailnetRoundTrip(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
		})}}
		_, e := c.TailnetSettings(t.Context())
		if !errors.Is(e, tc.want) || strings.Contains(e.Error(), "synthetic") {
			t.Fatal(tc.status, e)
		}
	}
}
func TestTailnetHostConfigurationDefaultsOff(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.json")
	for _, enabled := range []bool{false, true} {
		field := ""
		if enabled {
			field = `,"tailnet_management":true`
		}
		if e := os.WriteFile(path, []byte(`{"image":"image","network":"soda","bridge":"soda0","subnet":"10.89.0.0/16"`+field+`}`), 0600); e != nil {
			t.Fatal(e)
		}
		v, e := LoadConfig(path)
		if e != nil || v.TailnetManagement != enabled {
			t.Fatal(v, e)
		}
	}
}
