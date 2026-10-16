package host

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/project"
)

// T13 transport scope: the task-events page is read-only and cursor-bound.
// An unbounded or misaddressed request never reaches the host; a stale or
// foreign page never advances the caller.
func TestFactoryTaskEventsClientBindsPage(t *testing.T) {
	in := project.FactoryTaskEvents{
		Project: "p" + strings.Repeat("b", 24), ID: strings.Repeat("a", 32),
		AfterSequence: 1, ByteLimit: 4096,
	}
	for _, tc := range []struct {
		name   string
		mutate func(*project.FactoryTaskEvents)
		fail   bool
	}{
		{"run", func(r *project.FactoryTaskEvents) { r.ID = strings.Repeat("b", 31) }, true},
		{"project", func(r *project.FactoryTaskEvents) { r.Project = "p" + strings.Repeat("z", 24) }, true},
		{"stale cursor", func(r *project.FactoryTaskEvents) { r.AfterSequence = -1 }, true},
		{"no bound", func(r *project.FactoryTaskEvents) { r.ByteLimit = 0 }, true},
		{"oversized bound", func(r *project.FactoryTaskEvents) { r.ByteLimit = project.MaxTaskEventsBytes + 1 }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient("unused")
			c.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				t.Fatal("invalid request reached host")
				return nil, nil
			})
			req := in
			tc.mutate(&req)
			if _, err := c.FactoryTaskEvents(context.Background(), req); err == nil {
				t.Fatalf("invalid events request sent: %s", tc.name)
			}
		})
	}
}

func TestFactoryTaskEventsClientConfirmsIdentity(t *testing.T) {
	in := project.FactoryTaskEvents{
		Project: "p" + strings.Repeat("b", 24), ID: strings.Repeat("a", 32),
		AfterSequence: 1, ByteLimit: 4096,
	}
	event := project.TaskEvent{
		Sequence: 2, PrevDigest: strings.Repeat("3", 64),
		Type: project.TaskEventChildStarted, TaskKey: "code-task",
		ChildID:    strings.Repeat("c", 32),
		Invocation: strings.Repeat("1", 32), PlanDigest: strings.Repeat("2", 64),
	}
	good := project.FactoryTaskEventsState{
		ID: in.ID, Project: in.Project, NextSequence: 3,
		Events: []project.TaskEvent{event},
	}
	for _, tc := range []struct {
		name   string
		mutate func(*project.FactoryTaskEventsState)
		fail   bool
	}{
		{"page", func(*project.FactoryTaskEventsState) {}, false},
		{"empty page", func(s *project.FactoryTaskEventsState) {
			s.Events, s.NextSequence = nil, 2
		}, false},
		{"foreign run", func(s *project.FactoryTaskEventsState) {
			s.ID = strings.Repeat("b", 32)
		}, true},
		{"foreign project", func(s *project.FactoryTaskEventsState) {
			s.Project = "p" + strings.Repeat("c", 24)
		}, true},
		{"no advance", func(s *project.FactoryTaskEventsState) {
			s.Events, s.NextSequence = nil, 1
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient("unused")
			c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/factory-task-events" {
					t.Fatal("wrong events route")
				}
				out := good
				tc.mutate(&out)
				return jsonResponse(out), nil
			})}
			got, err := c.FactoryTaskEvents(context.Background(), in)
			if (err != nil) != tc.fail {
				t.Fatalf("page error: %v", err)
			}
			if !tc.fail && (got.ID != in.ID || got.NextSequence < in.AfterSequence+1) {
				t.Fatalf("page did not bind its cursor: %+v", got)
			}
		})
	}
}
