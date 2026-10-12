package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

// joinEnvelopeCases pins the Soda side of the L07 native join: every
// envelope a real hooked pass submits must carry the operation-ID shape,
// native kind and payload keys the native writer consumes (captured in
// .coord/l07-consumer/captured-work.json, sha256 bd4b0a98; native join
// green at Fountain 17fbd56ab3). Fixture IDs are the test's own; the
// contract is shape, kinds, keys, role split and work-index visibility.
var joinEnvelopeCases = []struct {
	event string
	kind  string
}{
	{factory.LifecycleEventIssueCreated, factory.LifecycleKindIssue},
	{factory.LifecycleEventPRCreated, factory.LifecycleKindReview},
	{factory.LifecycleEventVerifiedMerge, factory.LifecycleKindNote},
}

func checkJoinSubmit(t *testing.T, work factory.LifecycleWork, event, kind string) {
	t.Helper()
	prefix := "lifecycle:" + event + ":"
	if !strings.HasPrefix(work.OperationID, prefix) || !strings.HasSuffix(work.OperationID, ":attempt-1") || len(work.OperationID) <= len(prefix)+len(":attempt-1") {
		t.Fatalf("operation id shape: %q", work.OperationID)
	}
	if factory.LifecycleEventKind(work.Event) != kind {
		t.Fatalf("event %q kind %q, want %q", work.Event, factory.LifecycleEventKind(work.Event), kind)
	}
	if work.ActorID != 7 {
		t.Fatalf("submitter actor %d, want enrolled executor 7", work.ActorID)
	}
	if work.RepositoryID <= 0 || work.IssueID <= 0 {
		t.Fatalf("submit scope: %+v", work)
	}
}

func checkJoinVisible(t *testing.T, ctx context.Context, db interface {
	ListActions(context.Context, int64, int64, string, int) ([]factory.LifecycleAction, error)
}, repo, workIndex int64, event string, target int64) {
	t.Helper()
	visible, err := db.ListActions(ctx, repo, workIndex, event, 64)
	if err != nil || len(visible) != 1 {
		t.Fatalf("work-index visibility: %+v %v", visible, err)
	}
	got := visible[0]
	if got.IssueID != workIndex || got.TargetID != target {
		t.Fatalf("read-key split: issue %d target %d", got.IssueID, got.TargetID)
	}
	if got.ActorID != 7 || got.ConfigOwner != 99 {
		t.Fatalf("role split: submitter %d authorizer %d", got.ActorID, got.ConfigOwner)
	}
}

// TestLifecycleEnvelopeJoin proves the three real hooked passes emit the
// native join contract: operation-ID shape, exact native kinds, enrolled
// submitter identity, and journal rows readable under the original work
// index with the native target split out.
func TestLifecycleEnvelopeJoin(t *testing.T) {
	ctx := context.Background()

	t.Run("issue_created", func(t *testing.T) {
		c, _, r := acceptedGenerationFixture(t)
		ex := callbackLifecycleExecutor()
		c.Lifecycle = ex
		if _, err := c.Store.RegisterCapability(ctx, factory.LifecycleEventIssueCreated, "gen-1", "install-1"); err != nil {
			t.Fatal(err)
		}
		cfg := factory.LifecycleConfig{Event: factory.LifecycleEventIssueCreated, Enabled: true, ActorID: 99, RepositoryID: 42, LabelID: 3}
		if _, err := c.Store.SaveConfig(ctx, cfg); err != nil {
			t.Fatal(err)
		}
		if _, err := c.GenerateAcceptedIntent(ctx, r.Intent, r.Owner); err != nil {
			t.Fatal(err)
		}
		if len(ex.submits) != 2 {
			t.Fatalf("submits: %+v", ex.submits)
		}
		for _, work := range ex.submits {
			checkJoinSubmit(t, work, factory.LifecycleEventIssueCreated, factory.LifecycleKindIssue)
			if work.LabelID != 3 {
				t.Fatalf("label: %+v", work)
			}
		}
		checkJoinVisible(t, ctx, c.Store, 42, 1, factory.LifecycleEventIssueCreated, ex.submits[0].IssueID)
	})

	t.Run("pr_created", func(t *testing.T) {
		fx := publishSeed(t)
		fx.finishReported(t, 3)
		enableCallbackConfig(t, fx, factory.LifecycleEventPRCreated, nil)
		ex := callbackLifecycleExecutor()
		fx.coord.Lifecycle = ex
		fx.wire(happyPublisher())
		if first := fx.pass(t); len(first.Published) != 1 {
			t.Fatalf("default report: %+v", first)
		}
		if len(ex.submits) != 1 {
			t.Fatalf("submits: %+v", ex.submits)
		}
		work := ex.submits[0]
		checkJoinSubmit(t, work, factory.LifecycleEventPRCreated, factory.LifecycleKindReview)
		if len(work.ReviewerIDs) != 1 || work.ReviewerIDs[0] != 15 {
			t.Fatalf("reviewers: %+v", work)
		}
		checkJoinVisible(t, ctx, fx.db, 7, 3, factory.LifecycleEventPRCreated, work.IssueID)
	})

	t.Run("verified_merge", func(t *testing.T) {
		fx, _ := mergeSeed(t, 3)
		enableCallbackConfig(t, fx.publish, factory.LifecycleEventVerifiedMerge, nil)
		ex := callbackLifecycleExecutor()
		fx.publish.coord.Lifecycle = ex
		fx.wire(happyMerger())
		if first := fx.pass(t); len(first.Merged) != 1 {
			t.Fatalf("default report: %+v", first)
		}
		if len(ex.submits) != 1 {
			t.Fatalf("submits: %+v", ex.submits)
		}
		work := ex.submits[0]
		checkJoinSubmit(t, work, factory.LifecycleEventVerifiedMerge, factory.LifecycleKindNote)
		if !work.Closed || work.Note == "" || !strings.HasPrefix(work.MergeRef, "pr/") || !strings.Contains(work.MergeRef, "@") {
			t.Fatalf("note: %+v", work)
		}
		checkJoinVisible(t, ctx, fx.publish.db, 7, 3, factory.LifecycleEventVerifiedMerge, work.IssueID)
	})
}
