package host

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

func TestPrepareClientConfirmsIdentity(t *testing.T) {
	c := NewClient("unused")
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		state := project.PrepareState{ID: "f99999999999999999999999", Project: "p0123456789abcdef01234567", Role: project.RoleCoder, Phase: project.PrepareRunning, Container: strings.Repeat("c", 64)}
		return jsonResponse(state), nil
	})}
	prep := project.Preparation{ID: "f0123456789abcdef01234567", Project: "p0123456789abcdef01234567", Role: project.RoleCoder}
	if _, err := c.Prepare(context.Background(), project.Prepare{Preparation: prep}); err == nil {
		t.Fatal("mismatched preparation identity accepted")
	}
}

func TestStopClientRequiresConfirmedStop(t *testing.T) {
	c := NewClient("unused")
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		state := project.PrepareState{ID: "f0123456789abcdef01234567", Project: "p0123456789abcdef01234567", Phase: project.PrepareRunning}
		return jsonResponse(state), nil
	})}
	if _, err := c.StopPreparation(context.Background(), project.PrepareStop{Project: "p0123456789abcdef01234567", ID: "f0123456789abcdef01234567"}); err == nil {
		t.Fatal("unconfirmed stop accepted")
	}
}

func TestProjectAccessClientRequiresBoundCompleteObservation(t *testing.T) {
	in := project.ProjectAccessRequest{Project: "p0123456789abcdef01234567", Login: "alice", Identity: 42}
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   bool
		ok     bool
	}{
		{name: "confirmed false", status: 200, body: `{"project":"p0123456789abcdef01234567","login":"alice","identity":42,"administrator":false}`, ok: true},
		{name: "missing administrator", status: 200, body: `{"project":"p0123456789abcdef01234567","login":"alice","identity":42}`},
		{name: "null administrator", status: 200, body: `{"project":"p0123456789abcdef01234567","login":"alice","identity":42,"administrator":null}`},
		{name: "wrong project", status: 200, body: `{"project":"p1123456789abcdef01234567","login":"alice","identity":42,"administrator":true}`},
		{name: "wrong login", status: 200, body: `{"project":"p0123456789abcdef01234567","login":"bob","identity":42,"administrator":true}`},
		{name: "wrong identity", status: 200, body: `{"project":"p0123456789abcdef01234567","login":"alice","identity":43,"administrator":true}`},
		{name: "malformed", status: 200, body: `{`},
		{name: "oversized response", status: 200, body: `{"project":"p0123456789abcdef01234567","login":"alice","identity":42,"administrator":false}` + strings.Repeat(" ", 4097)},
		{name: "native failure", status: 500, body: "native operation failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient("unused")
			c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || r.URL.Path != "/project-access" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				var got project.ProjectAccessRequest
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil || got != in {
					t.Fatalf("request=%+v err=%v", got, err)
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}
			got, err := c.ProjectAccess(context.Background(), in)
			if tc.ok {
				if err != nil || got.Administrator == nil || *got.Administrator != tc.want {
					t.Fatalf("status=%+v err=%v", got, err)
				}
			} else if err == nil {
				t.Fatalf("accepted unconfirmed observation: %+v", got)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(value any) *http.Response {
	body, _ := json.Marshal(value)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}
}
