package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

type bundleChooser struct {
	selected   string
	confidence float64
	calls      int
	fail       bool
	force      bool
}

func (f *bundleChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	f.calls++
	if f.fail {
		return factory.PriorityAnswer{}, errors.New("consultation down")
	}
	selected := f.selected
	if _, ok := choices[selected]; !ok && !f.force {
		selected = factory.ContextBundleSnapshotHold
	}
	share := (1 + f.confidence) / 2
	a := factory.PriorityAnswer{Selected: selected, Model: "synthetic-model", Confidence: f.confidence, Probabilities: map[string]float64{}}
	for id := range choices {
		a.Probabilities[id] = (1 - share) / float64(len(choices)-1)
	}
	a.Probabilities[selected] = share
	return a, nil
}

func contextBundleFixture(t *testing.T, chooser *bundleChooser) (*Coordinator, factory.MCPCapability, factory.Assignment) {
	t.Helper()
	c := coordinatorFixture(t, nil, nil)
	c.Planning = &PlanningJudgments{Journal: c.Store, Funding: c.Store, Chooser: chooser}
	capability := testMCPCapability()
	assignment := factory.Assignment{
		ID: capability.AssignmentID, SourceCommit: "abc123", PromptSHA: strings.Repeat("p", 64),
	}
	return c, capability, assignment
}

func onceReserve(spent *map[string]bool) func(context.Context, string) (bool, error) {
	return func(_ context.Context, snapshot string) (bool, error) {
		if (*spent)[snapshot] {
			return false, nil
		}
		(*spent)[snapshot] = true
		return true, nil
	}
}

func TestSelectContextBundleServesConfidentCurrent(t *testing.T) {
	chooser := &bundleChooser{selected: factory.ContextBundleCurrent, confidence: 0.9}
	c, capability, assignment := contextBundleFixture(t, chooser)
	spent := map[string]bool{}
	bundle, reason, err := c.SelectContextBundle(context.Background(), capability, assignment, false, false, onceReserve(&spent))
	if err != nil || bundle != factory.ContextBundleCurrent || reason != "" {
		t.Fatalf("bundle=%q reason=%q err=%v", bundle, reason, err)
	}
	again, _, err := c.SelectContextBundle(context.Background(), capability, assignment, false, false, onceReserve(&spent))
	if err != nil || again != factory.ContextBundleCurrent || chooser.calls != 1 {
		t.Fatalf("reuse bundle=%q calls=%d err=%v", again, chooser.calls, err)
	}
}

func TestSelectContextBundleServesConfidentAttempt(t *testing.T) {
	chooser := &bundleChooser{selected: factory.ContextBundleAttempt, confidence: 0.9}
	c, capability, assignment := contextBundleFixture(t, chooser)
	assignment.RunHistory = []string{"run-1", "run-2"}
	spent := map[string]bool{}
	bundle, reason, err := c.SelectContextBundle(context.Background(), capability, assignment, true, false, onceReserve(&spent))
	if err != nil || bundle != factory.ContextBundleAttempt || reason != "" {
		t.Fatalf("bundle=%q reason=%q err=%v", bundle, reason, err)
	}
}

func TestSelectContextBundleHoldsLowConfidence(t *testing.T) {
	chooser := &bundleChooser{selected: factory.ContextBundleCurrent, confidence: 0.2}
	c, capability, assignment := contextBundleFixture(t, chooser)
	spent := map[string]bool{}
	bundle, reason, err := c.SelectContextBundle(context.Background(), capability, assignment, false, false, onceReserve(&spent))
	if err != nil || bundle != factory.ContextBundleSnapshotHold || reason != factory.ContextBundleHoldLowConfidence {
		t.Fatalf("bundle=%q reason=%q err=%v", bundle, reason, err)
	}
}

func TestSelectContextBundleHoldsUnofferedLineage(t *testing.T) {
	chooser := &bundleChooser{selected: factory.ContextBundleLineage, confidence: 0.95, force: true}
	c, capability, assignment := contextBundleFixture(t, chooser)
	spent := map[string]bool{}
	bundle, reason, err := c.SelectContextBundle(context.Background(), capability, assignment, true, false, onceReserve(&spent))
	if err == nil || bundle != factory.ContextBundleSnapshotHold || reason != "unavailable" {
		t.Fatalf("bundle=%q reason=%q err=%v", bundle, reason, err)
	}
}

func TestSelectContextBundleHoldsWhenChooserDown(t *testing.T) {
	chooser := &bundleChooser{fail: true}
	c, capability, assignment := contextBundleFixture(t, chooser)
	spent := map[string]bool{}
	bundle, _, err := c.SelectContextBundle(context.Background(), capability, assignment, false, false, onceReserve(&spent))
	if bundle != factory.ContextBundleSnapshotHold || err == nil {
		t.Fatalf("bundle=%q err=%v", bundle, err)
	}
}

func TestSelectContextBundleExhaustedHoldsUnfunded(t *testing.T) {
	chooser := &bundleChooser{selected: factory.ContextBundleCurrent, confidence: 0.9}
	c, capability, assignment := contextBundleFixture(t, chooser)
	refuse := func(context.Context, string) (bool, error) { return false, nil }
	if _, _, err := c.SelectContextBundle(context.Background(), capability, assignment, false, false, refuse); err == nil {
		t.Fatal("exhausted funding served a bundle")
	}
	if chooser.calls != 0 {
		t.Fatal("unreserved consultation was attempted")
	}
}

func TestContextBundleOptionsOfferCurrentAndHold(t *testing.T) {
	options := contextBundleOptions(false, false)
	if len(options) != 2 {
		t.Fatalf("options=%d, want current plus snapshot_hold", len(options))
	}
	if _, ok := options[factory.ContextBundleCurrent]; !ok {
		t.Fatal("current bundle missing from options")
	}
	if _, ok := options[factory.ContextBundleSnapshotHold]; !ok {
		t.Fatal("snapshot_hold missing from options")
	}
	if _, ok := options[factory.ContextBundleAttempt]; ok {
		t.Fatal("attempt offered before its history content lands")
	}
	if _, ok := options[factory.ContextBundleLineage]; ok {
		t.Fatal("lineage offered before its history content lands")
	}
}

func TestContextBundleStateBindsScope(t *testing.T) {
	capability := testMCPCapability()
	assignment := factory.Assignment{
		ID: capability.AssignmentID, SourceCommit: "abc123", PromptSHA: strings.Repeat("p", 64),
	}
	state := contextBundleStateFor(capability, assignment)
	if state.Scope.ScopeDigest != capability.ScopeDigest() || state.Scope.Profile != string(capability.Profile) || state.Scope.PromptSHA != assignment.PromptSHA {
		t.Fatalf("state=%+v", state.Scope)
	}
}
