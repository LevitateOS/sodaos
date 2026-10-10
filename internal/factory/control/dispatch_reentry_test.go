package control

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestMergeCompletionPersistenceFailureDoesNotReenterDispatch(t *testing.T) {
	fx, p, exec := completedOpenMerge(t)
	ctx := context.Background()
	dependant := factory.Acceptance{
		ID: "d" + strings.Repeat("f", 24), Repository: p.Repository, IssueIndex: "4", Approver: 5, NativeRev: 41,
		TitleDigest: strings.Repeat("a", 64), ContentDigest: strings.Repeat("b", 64), ContentVersion: 2,
		Sources: []factory.SelectedSource{{ID: "11", ContentVersion: 0, Digest: strings.Repeat("c", 64)}},
		Prerequisites: []factory.AcceptedPrerequisite{{
			Occurrence: "21", DependsOn: "9", EndpointRepo: p.Repository, EndpointIssue: p.Issue,
			Outcome: factory.PrereqCode, PrereqAcceptance: fx.publish.seed.decision[p.Issue].ID,
		}},
	}
	if err := fx.publish.db.AdmitAcceptanceDecision(ctx, dependant); err != nil {
		t.Fatal(err)
	}
	fx.publish.reads.evidence["7/4"] = AcceptanceEvidence{
		Revision: 9,
		Issue: AcceptanceIssueView{
			Index: "4", TitleDigest: dependant.TitleDigest, ContentDigest: dependant.ContentDigest,
			ContentVer: dependant.ContentVersion, Visible: true,
		},
		Comments:     []AcceptanceComment{{ID: "11", Digest: dependant.Sources[0].Digest, ContentVer: 0, Visible: true}},
		Dependencies: []AcceptanceEdge{{Occurrence: "21", DependsOn: "9", Visible: true}},
	}
	before, err := fx.publish.coord.assessOne(ctx, p.Repository, 4)
	if err != nil || before.control.Readiness != factory.ReadinessBlocked {
		t.Fatalf("dependant precondition: %+v, %v", before.control, err)
	}

	// Clear the seed's real acceptance roots before filling the source table.
	if _, err := fx.publish.coord.drainReadinessWork(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.publish.db.ReadinessWork(ctx, "root:7/3"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("seed readiness root was not completed: %v", err)
	}
	if _, err := fx.publish.db.ReadinessWork(ctx, "root:7/4"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("dependant readiness root was not completed: %v", err)
	}
	var fillerIDs []string
	for i := 0; ; i++ {
		id := fmt.Sprintf("reentry-capacity:%d", i)
		issue := int64(i + 100)
		err = fx.publish.db.EnqueueReadinessWork(ctx, id, "", factory.DependenceRef{Repository: p.Repository, Issue: issue})
		if errors.Is(err, store.ErrReadinessCapacity) {
			break
		}
		if err != nil {
			t.Fatalf("fill readiness source capacity: %v", err)
		}
		fillerIDs = append(fillerIDs, id)
		index := fmt.Sprint(issue)
		fx.publish.reads.evidence[fmt.Sprintf("%d/%s", p.Repository, index)] = AcceptanceEvidence{
			Revision: 9,
			Issue:    AcceptanceIssueView{Index: index, Visible: true, IsPull: true},
		}
	}

	confirmCalls := 0
	capacityFull := true
	confirmed := exec.confirm
	exec.confirm = func(work factory.MergeWork) (factory.MergeConfirmation, error) {
		confirmCalls++
		if capacityFull && confirmCalls > 1 {
			return factory.MergeConfirmation{}, errors.New("completion confirmation trapped recursive retry")
		}
		return confirmed(work)
	}
	fx.publish.coord.DispatchReads = fx.publish.seed.reads
	fx.publish.seed.reads.err = &AcceptanceRefusal{Reason: RefusalSnapshotUnavailable}
	_ = fx.publish.coord.Dispatch(ctx)

	if confirmCalls != 1 {
		t.Fatalf("failed merge persistence reentered dispatch and repeated confirmation %d times", confirmCalls)
	}
	merge := fx.merge(t, p)
	if merge.Stage != factory.MergeOpen || merge.Operation.Completion != factory.OpCompletionComplete {
		t.Fatalf("capacity rollback did not retain the completed open merge: %+v", merge)
	}
	if _, err := fx.publish.db.IssueMergeCompletion(ctx, p.Repository, p.Issue); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("failed completion persistence recorded a completion: %v", err)
	}
	after, err := fx.publish.db.IssueControl(ctx, dependant.Repository, 4)
	if err != nil || after.Readiness != factory.ReadinessBlocked {
		t.Fatalf("failed persistence released a code dependant: %+v, %v", after, err)
	}
	if len(fillerIDs) == 0 {
		t.Fatal("readiness table had no filler source to release")
	}
	// Let the real readiness worker retire known visible pull-request fillers.
	// It may already own a node slot after the bounded dispatch drain, so
	// direct BeginReadinessWork would correctly reject it as pending.
	if _, err := fx.publish.coord.drainReadinessWork(ctx, ""); err != nil {
		t.Fatalf("drain readiness work before capacity-release retry: %v", err)
	}
	released := false
	for _, id := range fillerIDs {
		if _, err := fx.publish.db.ReadinessWork(ctx, id); errors.Is(err, store.ErrNotFound) {
			released = true
			break
		} else if err != nil {
			t.Fatalf("inspect synthetic readiness source %q: %v", id, err)
		}
	}
	if !released {
		t.Fatal("normal readiness drain did not release a synthetic filler source")
	}
	capacityFull = false
	_ = fx.publish.coord.Dispatch(ctx)
	if confirmCalls != 2 {
		t.Fatalf("capacity-release retry confirmations = %d, want one retry after the failed persistence", confirmCalls)
	}
	merge = fx.merge(t, p)
	if merge.Stage != factory.MergeMerged {
		t.Fatalf("capacity-release retry did not finish the merge: %+v", merge)
	}
	if _, err := fx.publish.db.IssueMergeCompletion(ctx, p.Repository, p.Issue); err != nil {
		t.Fatalf("successful retry did not persist completion: %v", err)
	}
	// The readiness pass is intentionally bounded. A second ordinary dispatch
	// drains the completion root if the first retry pass was spent on fillers.
	_ = fx.publish.coord.Dispatch(ctx)
	after, err = fx.publish.db.IssueControl(ctx, dependant.Repository, 4)
	if err != nil || after.Readiness != factory.ReadinessQueued {
		t.Fatalf("successful retry did not release the code dependant: %+v, %v", after, err)
	}
}

func TestDispatchMergeCompletionDoesNotReenterActiveDispatch(t *testing.T) {
	fx, p, exec, host := dispatchReentryFixture(t)
	ctx := context.Background()

	report := fx.publish.coord.Dispatch(ctx)

	if waitReason(report, 4) != WaitHarness {
		t.Fatalf("outer dispatch did not reach the bounded harness boundary: %+v", report)
	}
	if len(host.families) != 1 {
		t.Fatalf("staged harness probes = %d, want only the outer dispatch pass", len(host.families))
	}
	merge := fx.merge(t, p)
	if merge.Stage != factory.MergeMerged {
		t.Fatalf("outer dispatch did not complete the merge: %+v", merge)
	}
	if len(exec.confirms) != 1 {
		t.Fatalf("native completion confirmations = %d, want 1", len(exec.confirms))
	}
	if _, err := fx.publish.db.IssueMergeCompletion(ctx, p.Repository, p.Issue); err != nil {
		t.Fatalf("confirmed completion was not durable: %v", err)
	}
	if _, err := fx.publish.db.ReadinessWork(ctx, "root:7/3"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("outer dispatch did not consume the durable completion continuation: %v", err)
	}
}

func TestMergePassCompletionStillDispatchesStandalone(t *testing.T) {
	fx, p, exec, host := dispatchReentryFixture(t)
	ctx := context.Background()

	fx.publish.coord.MergePass(ctx)

	if len(host.families) != 1 {
		t.Fatalf("standalone completion staged harness probes = %d, want one dispatch", len(host.families))
	}
	if merge := fx.merge(t, p); merge.Stage != factory.MergeMerged {
		t.Fatalf("standalone merge pass did not complete the merge: %+v", merge)
	}
	if len(exec.confirms) != 1 {
		t.Fatalf("native completion confirmations = %d, want 1", len(exec.confirms))
	}
	if _, err := fx.publish.db.IssueMergeCompletion(ctx, p.Repository, p.Issue); err != nil {
		t.Fatalf("confirmed completion was not durable: %v", err)
	}
	if _, err := fx.publish.db.ReadinessWork(ctx, "root:7/3"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("standalone merge pass did not consume the durable completion continuation: %v", err)
	}
}

func dispatchReentryFixture(t *testing.T) (*mergeFixture, factory.Publication, *fakeMerger, *fakeDispatchHost) {
	t.Helper()
	fx, p, exec := completedOpenMerge(t)
	decision := fx.publish.seed.accept(t, 4, "d"+strings.Repeat("e", 23)+"4")
	fx.publish.seed.queue(t, 4, decision.ID)
	host := fx.publish.seed.host
	host.harness = func(string) (project.FactoryHarnessPin, error) {
		return project.FactoryHarnessPin{}, errors.New("stop at the staged harness boundary")
	}
	fx.publish.coord.Host = host
	fx.publish.coord.Broker = fx.publish.seed.broker
	fx.publish.coord.DispatchReads = fx.publish.seed.reads
	return fx, p, exec, host
}

func completedOpenMerge(t *testing.T) (*mergeFixture, factory.Publication, *fakeMerger) {
	t.Helper()
	fx, p := mergeSeed(t, 3)
	exec := happyMerger()
	exec.submit = func(factory.MergeWork) (factory.OperationOutcome, error) {
		return factory.OperationOutcome{
			Effect: factory.OpEffectCommitted, Cancellation: factory.OpCancelNone,
			Completion: factory.OpCompletionPending, Receipt: []byte(`{"pr_number":9}`),
		}, nil
	}
	fx.wire(exec)
	fx.pass(t)
	for id, outcome := range exec.ledger {
		outcome.Completion = factory.OpCompletionComplete
		exec.ledger[id] = outcome
	}
	merge := fx.merge(t, p)
	if merge.Stage != factory.MergeOpen || merge.Operation.Completion != factory.OpCompletionPending {
		t.Fatalf("completed-open merge precondition: %+v", merge)
	}
	return fx, p, exec
}
