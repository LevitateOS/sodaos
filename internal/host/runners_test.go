package host

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/runners"
)

type runnerDouble struct {
	calls []string
	fail  bool
}

func (f *runnerDouble) List(context.Context) ([]runners.RunnerView, error) {
	if err := f.action("list"); err != nil {
		return nil, err
	}
	return []runners.RunnerView{}, nil
}
func (f *runnerDouble) Create(_ context.Context, in runners.CreateRequest) error {
	return f.action("create:" + in.ID)
}
func (f *runnerDouble) Start(_ context.Context, id string) error   { return f.action("start:" + id) }
func (f *runnerDouble) Stop(_ context.Context, id string) error    { return f.action("stop:" + id) }
func (f *runnerDouble) Restart(_ context.Context, id string) error { return f.action("restart:" + id) }
func (f *runnerDouble) Remove(_ context.Context, id string) error  { return f.action("remove:" + id) }
func (f *runnerDouble) action(value string) error {
	f.calls = append(f.calls, value)
	if f.fail {
		return errors.New("private-registration-secret")
	}
	return nil
}
func TestRunnerSocketStrictFixedOperations(t *testing.T) {
	f := &runnerDouble{}
	d := &Daemon{Runners: &runners.Operations{Local: f, Lifecycle: f}}
	for _, tc := range []struct {
		path, method, body string
		status             int
	}{
		{"/runners/list", "POST", "{}", 200},
		{"/runners/start", "POST", `{"id":"one"}`, 200},
		{"/runners/stop", "POST", `{"id":"one"}`, 200},
		{"/runners/restart", "POST", `{"id":"one"}`, 200},
		{"/runners/remove", "POST", `{"id":"one"}`, 200},
		{"/runners/create", "POST", `{"id":"one","provider":"forgejo","registration_url":"http://forgejo:3000","registration_id":"33834eef-e758-48c4-a676-1745426747aa","labels":"native:host","registration_token":"fixture"}`, 200},
		{"/runners/list", "GET", "{}", 400},
		{"/runners/list?x=1", "POST", "{}", 400},
		{"/runners/list?", "POST", "{}", 400},
		{"/runners/exec", "POST", "{}", 404},
		{"/runners/start", "POST", `{"id":"../one"}`, 502},
		{"/runners/start", "POST", `{"id":"one","id":"two"}`, 502},
		{"/runners/start", "POST", `{"id":"one","unit":"sshd"}`, 502},
		{"/runners/list", "POST", `{} {}`, 502},
		{"/runners/remove", "POST", `{"id":"one","account":"root"}`, 502},
		{"/runners/create", "POST", `{"id":"one","provider":"github","registration_token":"fixture"}`, 502},
		{"/runners/create", "POST", `{"id":"one","provider":"unknown","registration_token":"fixture"}`, 502},
		{"/runners/create", "POST", `{"id":"one","registration_token":"` + strings.Repeat("s", 65536) + `"}`, 502},
	} {
		before := len(f.calls)
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
		if w.Code != tc.status || (w.Code != 200 && len(f.calls) != before) {
			t.Fatal(tc.path, w.Code, len(f.calls)-before)
		}
	}
	if got := strings.Join(f.calls, ","); got != "list,start:one,stop:one,restart:one,remove:one,create:one" {
		t.Fatal("wrong native dispatch", got)
	}
	f.fail = true
	for _, action := range []string{"list", "start", "stop", "restart", "remove"} {
		body := `{"id":"one"}`
		if action == "list" {
			body = `{}`
		}
		before := len(f.calls)
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/runners/"+action, strings.NewReader(body)))
		if w.Code != 502 || strings.Contains(w.Body.String(), "private-registration-secret") || w.Body.String() == "[]" || len(f.calls) != before+1 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(action, w.Code, w.Body.String())
		}
	}
	// Missing paired integration must not fall through to a fake successful action.
	d.Runners = nil
	w := httptest.NewRecorder()
	d.ServeHTTP(w, httptest.NewRequest("POST", "/runners/list", strings.NewReader("{}")))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
}
