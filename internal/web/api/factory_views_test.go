package api

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestFactoryHostTerminalMapping(t *testing.T) {
	for phase, terminal := range map[string]bool{
		project.FactoryApproved: false, project.FactoryRunning: false,
		project.FactoryCompleted: true, project.FactoryFailed: true, project.FactoryStopped: true,
		project.FactoryUncertain: false,
	} {
		if got := factoryHostTerminal(phase); got != terminal {
			t.Fatalf("phase %s terminal=%v", phase, got)
		}
	}
}

func TestFactoryRunLiveExcerptIsBounded(t *testing.T) {
	exit := 3
	out := project.FactoryState{ID: strings.Repeat("a", 32), Project: "p" + strings.Repeat("b", 24), Phase: project.FactoryFailed, Output: strings.Repeat("x", factoryRunOutputExcerpt+100), ExitCode: &exit}
	live := factoryRunLiveDTO(out)
	if !live.OutputTruncated || len(live.Output) != factoryRunOutputExcerpt || live.ExitCode == nil || *live.ExitCode != 3 {
		t.Fatalf("excerpt: %+v", live)
	}
	short := factoryRunLiveDTO(project.FactoryState{ID: out.ID, Project: out.Project, Phase: project.FactoryCompleted, Output: "done"})
	if short.OutputTruncated || short.Output != "done" || !short.Terminal {
		t.Fatalf("short: %+v", short)
	}
}

func TestFactoryRunBindingRendersDecimalIDs(t *testing.T) {
	bound := factoryRunBindingDTO(factory.RunView{RunID: factory.NewID(), Repository: 7, Issue: 42, Attempt: "attempt-1"})
	if bound.Repository != "7" || bound.Issue != "42" || bound.Attempt != "attempt-1" {
		t.Fatalf("bound: %+v", bound)
	}
	plain := factoryRunBindingDTO(factory.RunView{RunID: factory.NewID(), Repository: 7})
	if plain.Repository != "7" || plain.Issue != "" || plain.Attempt != "" {
		t.Fatalf("plain: %+v", plain)
	}
}

func TestFactoryOutputHandshakeBindsRunRepositoryAndGeneration(t *testing.T) {
	identity := factoryViewerIdentity{
		authority: extensions.Authority{SessionGeneration: "gen-1"},
		project:   store.Project{ID: "p" + strings.Repeat("b", 24), RepositoryID: 7},
		runID:     strings.Repeat("a", 32),
		actorID:   1,
	}
	valid := factoryOutputHandshake{RunID: identity.runID, RepositoryID: "7", SessionGeneration: "gen-1", Cursor: 128}
	if !validFactoryOutputHandshake(valid, identity) {
		t.Fatal("valid handshake refused")
	}
	for name, mutate := range map[string]func(*factoryOutputHandshake){
		"run":        func(h *factoryOutputHandshake) { h.RunID = strings.Repeat("b", 32) },
		"repository": func(h *factoryOutputHandshake) { h.RepositoryID = "8" },
		"generation": func(h *factoryOutputHandshake) { h.SessionGeneration = "gen-2" },
		"cursor":     func(h *factoryOutputHandshake) { h.Cursor = -1 },
		"far":        func(h *factoryOutputHandshake) { h.Cursor = project.MaxFactoryOutputOffset + 1 },
	} {
		broken := valid
		mutate(&broken)
		if validFactoryOutputHandshake(broken, identity) {
			t.Fatalf("accepted invalid %s", name)
		}
	}
}

func frameKeys(t *testing.T, frame any) string {
	t.Helper()
	body, err := json.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for key := range decoded {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func TestFactoryFramesUseFixedKeySets(t *testing.T) {
	exit := 0
	status := factoryOutputStatus(strings.Repeat("a", 32), project.FactoryOutputState{Phase: project.FactoryRunning, Live: true, ExitCode: &exit})
	if keys := frameKeys(t, status); keys != "container,exit_code,invocation,live,phase,reason,run_id,terminal,type,unit" {
		t.Fatalf("status keys: %s", keys)
	}
	output := factoryOutputFrame{Type: "output", Data: "eA==", Cursor: 0, Next: 1}
	if keys := frameKeys(t, output); keys != "cursor,data,gap,next,truncated,type" {
		t.Fatalf("output keys: %s", keys)
	}
	closed := factoryClosedFrame{Type: "closed", Reason: "eof"}
	if keys := frameKeys(t, closed); keys != "reason,type" {
		t.Fatalf("closed keys: %s", keys)
	}
}

func TestSameFactoryStatusComparesExitByValue(t *testing.T) {
	zero, one := 0, 1
	a := factoryStatusFrame{Type: "status", Phase: project.FactoryCompleted, Terminal: true, ExitCode: &zero}
	b := factoryStatusFrame{Type: "status", Phase: project.FactoryCompleted, Terminal: true, ExitCode: &zero}
	if !sameFactoryStatus(a, b) {
		t.Fatal("equal statuses differ")
	}
	c := factoryStatusFrame{Type: "status", Phase: project.FactoryCompleted, Terminal: true, ExitCode: &one}
	if sameFactoryStatus(a, c) {
		t.Fatal("exit change ignored")
	}
	d := factoryStatusFrame{Type: "status", Phase: project.FactoryFailed, Terminal: true}
	if sameFactoryStatus(a, d) {
		t.Fatal("phase/exit change ignored")
	}
}
