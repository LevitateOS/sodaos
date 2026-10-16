package control

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

type servingPlanChooser struct {
	mu     sync.Mutex
	inputs []factory.PlanningInput
}

func TestServingCoordinatorConsultsCompleteEligibleSetAndLaunchesFreshSelection(t *testing.T) {
	c, _, fx := planningCoordinator(t)
	// The D02 cursor needs the observer joint production always wires; the
	// stable revision mirrors the seeded evidence bracket.
	c.Readiness = &fakeObserver{revision: 41, idle: true}
	young := fx.accept(t, 5, "d555555555555555555555555")
	fx.queue(t, 5, young.ID)
	input := fx.reads.inputs[fmt.Sprintf("%d/5", fx.repo)]
	source := c.AcceptanceReads.(*fakeEvidenceSource)
	source.evidence[fmt.Sprintf("%d/5", fx.repo)] = AcceptanceEvidence{Revision: input.Revision, Issue: AcceptanceIssueView{Index: "5", TitleOccurrence: input.Issue.TitleOccurrence, TitleDigest: input.Issue.TitleDigest, ContentDigest: input.Issue.ContentDigest, ContentVer: input.Issue.ContentVer, Visible: true}, Comments: []AcceptanceComment{{ID: "11", Digest: input.Comments[0].Digest, Visible: true}}}
	chooser := &priorityFixture{choose: priorityLast}
	c.Priority = &PriorityJudgments{Journal: c.Store, Chooser: chooser, MaxCalls: 16}
	started := make(chan project.FactoryLaunch, 2)
	launch := fx.host.launch
	fx.host.launch = func(in project.FactoryLaunch) (project.FactoryState, error) {
		state, err := launch(in)
		started <- in
		return state, err
	}
	if err := c.Start(context.Background(), filepath.Join(t.TempDir(), "coordinator.lock")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	select {
	case run := <-started:
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
		assignment, err := c.Store.AssignmentByRun(context.Background(), run.Run.ID)
		if err != nil || assignment.Issue != 5 || len(chooser.calls) != 1 || len(chooser.calls[0].Candidates) != 2 {
			t.Fatalf("assignment=%+v error=%v consultations=%+v", assignment, err, chooser.calls)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("serving owner did not consult and dispatch eligible work")
	}
}

func (f *servingPlanChooser) Select(_ context.Context, state any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	input := state.(factory.PlanningInput)
	f.mu.Lock()
	f.inputs = append(f.inputs, input)
	f.mu.Unlock()
	selected := "incremental_acceptance"
	if len(input.Missing) > 0 {
		selected = "plan_stable_scope"
	}
	a := factory.PriorityAnswer{Selected: selected, Model: "synthetic-model", Confidence: 1, Probabilities: map[string]float64{}}
	for id := range choices {
		a.Probabilities[id] = 0
	}
	a.Probabilities[selected] = 1
	return a, nil
}

func TestServingCoordinatorRefreshesProvisionalPlanningForActualMergedOutput(t *testing.T) {
	chooser := &servingPlanChooser{}
	var coordinator *Coordinator
	servingCorrectionAndDependent(t, "", func(c *Coordinator) {
		coordinator = c
		c.Store.PriorityCallBudget = 20
		c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
		c.Priority = &PriorityJudgments{Journal: c.Store, Chooser: &priorityFixture{choose: priorityLast}, MaxCalls: 16}
	})
	chooser.mu.Lock()
	defer chooser.mu.Unlock()
	provisional, finalized := 0, 0
	for _, input := range chooser.inputs {
		if input.Candidate.Control.Issue != 32 {
			continue
		}
		if len(input.Missing) > 0 {
			provisional++
		} else if input.Candidate.Source == strings.Repeat("3", 40) {
			finalized++
		}
	}
	if provisional != 1 || finalized != 1 {
		t.Fatalf("dependent planning: provisional=%d finalized=%d inputs=%+v", provisional, finalized, chooser.inputs)
	}
	record, err := coordinator.Store.PlanningRecord(context.Background(), 7, 32)
	if err != nil || record.Provisional || record.Input.Candidate.Source != strings.Repeat("3", 40) {
		t.Fatalf("dependent finalized plan=%+v err=%v", record, err)
	}
}
