package control

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
)

// bundleGapChooser records whether the lineage bundle was offered and
// selects a configured eligible bundle with full confidence.
type bundleGapChooser struct {
	calls      int
	sawLineage bool
	pick       string
}

func (f *bundleGapChooser) Select(_ context.Context, _ any, _ string, options map[string]any) (factory.PriorityAnswer, error) {
	f.calls++
	if _, ok := options[factory.ContextBundleLineage]; ok {
		f.sawLineage = true
	}
	selected := f.pick
	if _, ok := options[selected]; !ok {
		selected = factory.ContextBundleCurrent
	}
	answer := factory.PriorityAnswer{Selected: selected, Model: "synthetic-model", Confidence: 1, Probabilities: map[string]float64{}}
	for id := range options {
		answer.Probabilities[id] = 0
	}
	answer.Probabilities[selected] = 1
	return answer, nil
}

func bundleGapCapability(t *testing.T, assignment factory.Assignment, run factory.Run) factory.MCPCapability {
	t.Helper()
	digest := strings.Repeat("a", 64)
	preparation := project.StoredPreparation{Preparation: project.Preparation{SetupDigest: digest}}
	capability, err := BuildMCPCapability(assignment, run, preparation)
	if err != nil {
		t.Fatalf("capability: %v", err)
	}
	return capability
}

func bundleGapAssignment(runID string) (factory.Assignment, factory.Run) {
	assignment := factory.Assignment{
		ID: factory.NewID(), Repository: 7, ProjectID: "p" + strings.Repeat("d", 24),
		Phase: project.ExecutionImplement, Issue: 7, RunHistory: []string{runID},
		ParentAssignment: factory.NewID(), SourceCommit: strings.Repeat("c", 40),
	}
	run := factory.Run{ID: runID, Deadline: time.Now().Add(time.Hour)}
	return assignment, run
}

// TestContextBundleMissingParentWithholdsLineage proves the R31.2
// source-gap rule: a parent that fails to load keeps the lineage
// bundle out of the offered options, so even a lineage-favoring
// chooser cannot serve it — the eligible attempt bundle serves
// instead, and the recorded selection reuses without a second
// consultation.
func TestContextBundleMissingParentWithholdsLineage(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	runID := factory.NewID()
	assignment, run := bundleGapAssignment(runID)
	capability := bundleGapCapability(t, assignment, run)
	chooser := &bundleGapChooser{pick: factory.ContextBundleAttempt}
	coord := &Coordinator{Store: db, Planning: &PlanningJudgments{Chooser: chooser}}
	reserve := func(ctx context.Context, snapshot string) (bool, error) {
		return db.ReservePriorityJudgment(ctx, snapshot, 1)
	}
	attemptEligible, lineageEligible := factory.BundleEligibility(assignment, false)
	bundle, reason, err := coord.SelectContextBundle(ctx, capability, assignment, attemptEligible, lineageEligible, reserve)
	if err != nil || bundle != factory.ContextBundleAttempt || reason != "" {
		t.Fatalf("bundle=%q reason=%q err=%v", bundle, reason, err)
	}
	if chooser.sawLineage {
		t.Fatal("missing parent still offered the lineage bundle")
	}
	again, _, err := coord.SelectContextBundle(ctx, capability, assignment, attemptEligible, lineageEligible, reserve)
	if err != nil || again != bundle || chooser.calls != 1 {
		t.Fatalf("reselection=%q calls=%d err=%v", again, chooser.calls, err)
	}
}

// TestContextBundleBareAssignmentServesCurrent proves the other side
// of the gap rule: with no attempt continuity and no parent, only the
// current bundle is offered and served.
func TestContextBundleBareAssignmentServesCurrent(t *testing.T) {
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	runID := factory.NewID()
	assignment, run := bundleGapAssignment(runID)
	assignment.RunHistory = nil
	assignment.ParentAssignment = ""
	capability := bundleGapCapability(t, assignment, run)
	chooser := &bundleGapChooser{pick: factory.ContextBundleCurrent}
	coord := &Coordinator{Store: db, Planning: &PlanningJudgments{Chooser: chooser}}
	reserve := func(ctx context.Context, snapshot string) (bool, error) {
		return db.ReservePriorityJudgment(ctx, snapshot, 1)
	}
	attemptEligible, lineageEligible := factory.BundleEligibility(assignment, false)
	bundle, reason, err := coord.SelectContextBundle(ctx, capability, assignment, attemptEligible, lineageEligible, reserve)
	if err != nil || bundle != factory.ContextBundleCurrent || reason != "" {
		t.Fatalf("bundle=%q reason=%q err=%v", bundle, reason, err)
	}
}
