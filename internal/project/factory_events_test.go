package project

import (
	"encoding/json"
	"strings"
	"testing"
)

// T13 wire scope: typed events carry sequence/hash binding, task/child/
// invocation/plan identity and byte bounds; request/response cursors page
// whole events with a terminal marker.
func TestTaskEventChainDigestAndValidation(t *testing.T) {
	first := TaskEvent{
		Sequence: 1, Type: TaskEventChildStartIntent, TaskKey: "design-api",
		Invocation: strings.Repeat("1", 32), PlanDigest: strings.Repeat("2", 64),
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("valid open event refuses: %v", err)
	}
	digest := TaskEventDigest(first)
	if !ValidDigest(digest) {
		t.Fatalf("chain digest is not a digest: %q", digest)
	}
	second := TaskEvent{
		Sequence: 2, PrevDigest: digest, Type: TaskEventChildStarted,
		TaskKey: "design-api", ChildID: strings.Repeat("b", 32),
		Invocation: strings.Repeat("1", 32), PlanDigest: strings.Repeat("2", 64),
	}
	if err := second.Validate(); err != nil {
		t.Fatalf("valid chained event refuses: %v", err)
	}
	if TaskEventDigest(second) == digest {
		t.Fatal("distinct events share a chain digest")
	}
}

func TestTaskEventRefusals(t *testing.T) {
	good := func() TaskEvent {
		return TaskEvent{
			Sequence: 2, PrevDigest: strings.Repeat("0", 64),
			Type: TaskEventChildReported, TaskKey: "code-task",
			ChildID:    strings.Repeat("b", 32),
			Invocation: strings.Repeat("1", 32), PlanDigest: strings.Repeat("2", 64),
			Detail: "exit=0",
		}
	}
	cases := map[string]func(*TaskEvent){
		"zero sequence":    func(e *TaskEvent) { e.Sequence = 0 },
		"origin with prev": func(e *TaskEvent) { e.Sequence, e.PrevDigest = 1, strings.Repeat("0", 64) },
		"chain gap":        func(e *TaskEvent) { e.PrevDigest = "" },
		"unknown type":     func(e *TaskEvent) { e.Type = "child_whispered" },
		"empty task":       func(e *TaskEvent) { e.TaskKey = "" },
		"leading digit":    func(e *TaskEvent) { e.TaskKey = "1task" },
		"bad child":        func(e *TaskEvent) { e.ChildID = "short" },
		"bad invocation":   func(e *TaskEvent) { e.Invocation = "" },
		"bad plan digest":  func(e *TaskEvent) { e.PlanDigest = "x" },
		"detail too large": func(e *TaskEvent) { e.Detail = strings.Repeat("d", MaxTaskEventBytes+1) },
	}
	for name, mutate := range cases {
		e := good()
		mutate(&e)
		if err := e.Validate(); err == nil {
			t.Fatalf("%s validated", name)
		}
	}
	if !TaskEventTerminalType(TaskEventRunStopped) {
		t.Fatal("run_stopped is not terminal")
	}
	for _, terminal := range []string{TaskEventTaskFailed, TaskEventTaskCommitted, TaskEventCheckpointReady} {
		if TaskEventTerminalType(terminal) {
			t.Fatalf("%s is terminal", terminal)
		}
	}
}

func TestFactoryTaskEventsRequestAndState(t *testing.T) {
	req := FactoryTaskEvents{
		Project: "p" + strings.Repeat("b", 24), ID: strings.Repeat("a", 32),
		AfterSequence: 7, ByteLimit: 4096,
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseFactoryTaskEvents(raw)
	if err != nil {
		t.Fatalf("valid events request refuses: %v", err)
	}
	if parsed != req {
		t.Fatalf("events request did not round-trip: %+v", parsed)
	}
	for _, mutate := range []func(*FactoryTaskEvents){
		func(r *FactoryTaskEvents) { r.AfterSequence = -1 },
		func(r *FactoryTaskEvents) { r.ByteLimit = 0 },
		func(r *FactoryTaskEvents) { r.ByteLimit = MaxTaskEventsBytes + 1 },
		func(r *FactoryTaskEvents) { r.ID = "short" },
	} {
		bad := req
		mutate(&bad)
		raw, _ := json.Marshal(bad)
		if _, err := ParseFactoryTaskEvents(raw); err == nil {
			t.Fatalf("invalid events request parsed: %+v", bad)
		}
	}
	state := FactoryTaskEventsState{
		ID: req.ID, Project: req.Project, NextSequence: 8, Terminal: false,
		Events: []TaskEvent{{
			Sequence: 8, PrevDigest: strings.Repeat("3", 64),
			Type: TaskEventTaskCommitted, TaskKey: "code-task",
			Invocation: strings.Repeat("1", 32), PlanDigest: strings.Repeat("2", 64),
		}},
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("valid events state refuses: %v", err)
	}
	tooMany := state
	tooMany.Events = make([]TaskEvent, MaxTaskEventsPerRead+1)
	for i := range tooMany.Events {
		tooMany.Events[i] = state.Events[0]
		tooMany.Events[i].Sequence = int64(8 + i)
	}
	if err := tooMany.Validate(); err == nil {
		t.Fatal("oversized events page validated")
	}
}
