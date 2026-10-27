package control

import (
	"context"
	"errors"
	"strings"
	"testing"

	sdk "forgejo.org/extension-sdk"
	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/store"
)

func recordedGenerationFixture(t *testing.T) (*Coordinator, *generatedGraphFixture, dispatchFixture, factory.IntentRevision) {
	t.Helper()
	ctx := context.Background()
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	draft := choicePlanDraft(fx.repo)
	if _, ok := settlePlanAnswerWithOperation(t, ctx, db, fx, draft, factory.IntentPlanningOptions, choicePlanAnswer(t, draft, "completed", "cand-a")); !ok {
		t.Fatal("recorded plan did not settle")
	}
	c := choicePlanCoordinator(t, db, fx, &candidateChoiceStub{selected: "cand-a", confidence: 1})
	seedChoiceWalkthroughReview(t, ctx, c, draft, choicePlanWalkthroughs(draft))
	r := choicePlanAccepted(t, fx.repo)
	if err := db.SaveIntentRevision(ctx, r); err != nil {
		t.Fatal(err)
	}
	f := &generatedGraphFixture{revision: 10, actor: 91, owner: true, records: map[string]sdk.OperationRecord{}, issues: map[int64]sdk.NativeSnapshot{}}
	c.Graph, c.GraphActorID, c.AcceptanceReads = f, 91, f
	return c, f, fx, r
}

func TestGenerationRequiresOriginalPlanBinding(t *testing.T) {
	for _, mode := range []string{"fresh", "pending", "committed"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			c, f, _, r := recordedGenerationFixture(t)
			if mode == "fresh" {
				if _, err := c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner); err == nil {
					t.Fatal("resume entry created a fresh graph")
				}
				if _, err := c.GenerateAcceptedIntentFromChoice(ctx, r.Intent, r.Owner, chosenFor(r)); err == nil {
					t.Fatal("unbound choice created a fresh graph")
				}
				if _, err := c.Store.IntentGeneration(ctx, r.ID); !errors.Is(err, store.ErrNotFound) {
					t.Fatalf("refusal created durable operation: %v", err)
				}
				return
			}
			g := seedUnboundGeneration(t, c, r)
			if mode == "committed" {
				if _, err := f.SubmitGraph(ctx, g, r.Repository); err != nil {
					t.Fatal(err)
				}
			}
			before := f.submissions
			if _, _, err := c.GenerateAcceptedIntentFromRecordedPlan(ctx, r.Intent, r.Owner); err == nil {
				t.Fatal("existing unbound operation acquired today's plan")
			}
			got, err := c.Store.IntentGeneration(ctx, r.ID)
			if err != nil || got.PlanAssignment != "" || f.submissions != before {
				t.Fatalf("retroactive binding or duplicate native effect: %+v, %v", got, err)
			}
			if mode == "committed" && (got.State != "committed" || len(got.Receipt) == 0) {
				t.Fatal("lost attributable committed receipt")
			}
			got.PlanAssignment = factory.NewID()
			if err := c.Store.RecordIntentGeneration(ctx, got); !errors.Is(err, store.ErrCommandConflict) {
				t.Fatalf("store allowed retrospective plan binding: %v", err)
			}
		})
	}
}

func TestGenerationRejectsTasksOutsideRecordedCandidate(t *testing.T) {
	ctx := context.Background()
	c, f, _, r := recordedGenerationFixture(t)
	chosen, wait, err := c.ResolveAcceptedPlanChoice(ctx, r.Owner, r)
	if err != nil || wait != nil {
		t.Fatalf("resolve: %v %+v", err, wait)
	}
	// These tasks fit the human scope, but differ from the recorded candidate.
	chosen.Tasks = append([]factory.IntentTask(nil), chosen.Tasks...)
	chosen.Tasks[0].Objective += " with an unrecorded variation"
	work := r
	work.Tasks = chosen.Tasks
	if err := work.Validate(); err != nil {
		t.Fatalf("fixture must remain inside accepted scope: %v", err)
	}
	if _, err := c.GenerateAcceptedIntentFromChoice(ctx, r.Intent, r.Owner, chosen); err == nil || f.submissions != 0 {
		t.Fatalf("unrecorded tasks submitted: %d, %v", f.submissions, err)
	}
}

type movingAdoptionGraph struct {
	*generatedGraphFixture
	move func()
}

func (g *movingAdoptionGraph) ReadGeneratedIssue(ctx context.Context, repo, issue, actor int64) (sdk.NativeSnapshot, int64, error) {
	if g.move != nil {
		g.move()
		g.move = nil
	}
	return g.generatedGraphFixture.ReadGeneratedIssue(ctx, repo, issue, actor)
}

func TestGenerationRechecksPlanAfterNativeAdoptionReads(t *testing.T) {
	ctx := context.Background()
	c, f, fx, r := recordedGenerationFixture(t)
	c.Graph = &movingAdoptionGraph{generatedGraphFixture: f, move: func() {
		rotateChoicePlanSource(t, ctx, c.Store, fx, strings.Repeat("d", 40))
	}}
	g, _, err := c.GenerateAcceptedIntentFromRecordedPlan(ctx, r.Intent, r.Owner)
	if err == nil || g.State != "committed" || f.submissions != 1 || len(g.Receipt) == 0 {
		t.Fatalf("stale adoption did not retain committed receipt: state=%s submits=%d err=%v", g.State, f.submissions, err)
	}
	bindings, err := c.Store.IntentTaskBindings(ctx, r.ID)
	if err != nil || len(bindings) != 0 {
		t.Fatalf("stale native read adopted authority: %+v %v", bindings, err)
	}
	if _, _, err := c.GenerateAcceptedIntentFromRecordedPlan(ctx, r.Intent, r.Owner); err == nil || f.submissions != 1 {
		t.Fatal("retry duplicated native work or admitted stale plan")
	}
}
