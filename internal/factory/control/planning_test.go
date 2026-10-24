package control

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

type planningChooser struct{ calls int }

func (f *planningChooser) Select(_ context.Context, _ any, _ string, choices map[string]any) (factory.PriorityAnswer, error) {
	f.calls++
	selected := "incremental_acceptance"
	if _, ok := choices["plan_stable_scope"]; ok {
		selected = "plan_stable_scope"
	}
	a := factory.PriorityAnswer{Selected: selected, Model: "synthetic-model", Confidence: 1, Probabilities: map[string]float64{}}
	for id := range choices {
		a.Probabilities[id] = 0
	}
	a.Probabilities[selected] = 1
	return a, nil
}

func planningCoordinator(t *testing.T, body ...string) (*Coordinator, *planningChooser, dispatchFixture) {
	t.Helper()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	decision := fx.accept(t, 3, "d333333333333333333333333", body...)
	fx.queue(t, 3, decision.ID)
	input := fx.reads.inputs[fmt.Sprintf("%d/3", fx.repo)]
	evidence := AcceptanceEvidence{Revision: input.Revision, Issue: AcceptanceIssueView{Index: "3", TitleOccurrence: input.Issue.TitleOccurrence, TitleDigest: input.Issue.TitleDigest, ContentDigest: input.Issue.ContentDigest, ContentVer: input.Issue.ContentVer, Visible: true}, Comments: []AcceptanceComment{{ID: "11", Digest: input.Comments[0].Digest, Visible: true}}}
	source := &fakeEvidenceSource{evidence: map[string]AcceptanceEvidence{fmt.Sprintf("%d/3", fx.repo): evidence}}
	chooser := &planningChooser{}
	c := NewCoordinator(db, fx.host, fx.broker)
	c.AcceptanceReads, c.DispatchReads = source, fx.reads
	c.Planning = &PlanningJudgments{Journal: db, Funding: db, Chooser: chooser}
	return c, chooser, fx
}

func TestPlanningRecordsOnceAndFeedsFreshLaunch(t *testing.T) {
	// Generate preserves product-scoped directions in the accepted task body.
	// The Architect still plans this issue with source-inspected local checks;
	// Product Lead's whole-app catalog does not become its missing prerequisite.
	const productDirection = "For the normal product plan, use the current approved verification catalog and propose its canonical project.check check for every effective outcome through the typed verification-manifest proposal."
	c, chooser, fx := planningCoordinator(t, "Update USAGE.md and README.md with the requested usage example.\n"+productDirection)
	ctx := context.Background()
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	if chooser.calls != 1 {
		t.Fatalf("unchanged plan spent %d calls", chooser.calls)
	}
	r, err := c.Store.PlanningRecord(ctx, fx.repo, 3)
	if err != nil || r.Provisional || r.Outcome != "planned" {
		t.Fatalf("plan=%+v %v", r, err)
	}
	report := c.Dispatch(ctx)
	if len(report.Launched) != 1 {
		t.Fatalf("dispatch=%+v", report)
	}
	// FW12: the selected investigation carries an inspection directive
	// bound to the exact source and preparation, never generic fixed
	// steps. The recorded investigation plan stages on implementation.
	prompt := string(fx.host.launches[0].Prompt)
	for _, want := range []string{"# Soda factory investigation assignment", "## Investigation", fx.tip, "f111111111111111111111111", productDirection, "direct executable argv justified by inspected prepared source", `argv may be ["/bin/sh","check.sh"]`, "ProductLead-scoped directions embedded in accepted task text remain with that owning role", "Architect context intentionally omits verification_catalog", "preserve all accepted task outcomes and source/check constraints", "native CI status context names, not executable commands"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("investigation prompt lacks %q", want)
		}
	}
	for _, banned := range []string{"## Implementation approach", "direct_small_change", "incremental_acceptance", "Implement the accepted requirements below in the prepared checkout"} {
		if strings.Contains(prompt, banned) {
			t.Fatalf("investigation prompt carries generic fixed steps: %q", banned)
		}
	}
}

func TestPlanningProvisionalRecordCannotLaunch(t *testing.T) {
	c, chooser, fx := planningCoordinator(t)
	ctx := context.Background()
	control, err := c.Store.IssueControl(ctx, fx.repo, 3)
	if err != nil {
		t.Fatal(err)
	}
	control.Readiness = factory.ReadinessBlocked
	control.Reason = factory.BlockerCodePending
	control.Blockers = []factory.Blocker{{Code: factory.BlockerCodePending, Resolution: factory.BlockerResolution(factory.BlockerCodePending), EndpointRepo: fx.repo, EndpointIssue: 9}}
	control.Fingerprint = strings.Repeat("5", 64)
	if _, _, err := c.Store.RecordIssueAssessment(ctx, control, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := c.PlanningPass(ctx); err != nil {
		t.Fatal(err)
	}
	r, err := c.Store.PlanningRecord(ctx, fx.repo, 3)
	if err != nil || !r.Provisional || r.Strategy != "plan_stable_scope" || chooser.calls != 1 {
		t.Fatalf("plan=%+v calls=%d %v", r, chooser.calls, err)
	}
	if report := c.Dispatch(ctx); len(report.Launched) != 0 || len(fx.host.launches) != 0 {
		t.Fatalf("blocked plan launched %+v", report)
	}
	fx.queue(t, 3, control.Acceptance)
	if report := DispatchPass(ctx, c.dispatchDeps()); len(report.Launched) != 0 {
		t.Fatalf("provisional plan launched after unblock %+v", report)
	}
	if report := c.Dispatch(ctx); len(report.Launched) != 1 || chooser.calls != 2 {
		t.Fatalf("plan did not refresh %+v calls=%d", report, chooser.calls)
	}
}
