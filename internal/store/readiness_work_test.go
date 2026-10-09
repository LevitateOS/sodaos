package store

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestReadinessWorkCheckpointAndAtomicDeliveryFinish(t *testing.T) {
	s, ctx := readinessTestStore(t)
	now := time.Unix(1700000000, 0)
	root := factory.DependenceRef{Repository: 7, Issue: 3}
	if err := s.EnqueueReadinessWork(ctx, "delivery:delivery-1", "delivery-1", root); err != nil {
		t.Fatal(err)
	}
	if err := s.EnqueueReadinessWork(ctx, "delivery:delivery-1", "delivery-1", root); err != nil {
		t.Fatal("identical enqueue must be idempotent:", err)
	}
	if err := s.EnqueueReadinessWork(ctx, "delivery:delivery-1", "delivery-2", root); !errors.Is(err, ErrCommandConflict) {
		t.Fatal("source identity rewrite must conflict:", err)
	}
	work, err := s.NextReadinessWork(ctx, now)
	if err != nil || work.ID != "delivery:delivery-1" {
		t.Fatal("next source:", work, err)
	}
	work, err = s.BeginReadinessWork(ctx, work.ID, now)
	if err != nil {
		t.Fatal("begin source:", work, err)
	}
	node, err := s.NextReadinessWorkNode(ctx, work.ID)
	if err != nil || node.Ref != root || node.State != "queued" {
		t.Fatal("root node:", node, err)
	}
	if err := s.CheckpointReadinessAssessment(ctx, work, node, false, true); err != nil {
		t.Fatal("checkpoint root assessment:", err)
	}
	node, err = s.NextReadinessWorkNode(ctx, work.ID)
	if err != nil || node.State != "scanning" {
		t.Fatal("scanning root:", node, err)
	}
	more, err := s.CheckpointReadinessPage(ctx, work, node)
	if err != nil || more {
		t.Fatal("empty dependant scan:", more, err)
	}
	work, err = s.ReadinessWork(ctx, work.ID)
	if err != nil || !work.RootChanged {
		t.Fatal("root change flag was not retained:", work, err)
	}
	complete, err := s.CompleteReadinessWork(ctx, work, now)
	if err != nil || !complete {
		t.Fatal("complete source:", complete, err)
	}
	seen, err := s.IntakeDeliverySeen(ctx, "delivery-1")
	if err != nil || !seen {
		t.Fatal("completion must atomically acknowledge delivery:", seen, err)
	}
	if _, err = s.ReadinessWork(ctx, work.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("completed source retained:", err)
	}
}

func TestReadinessWorkCapacityGenerationAndDefer(t *testing.T) {
	s, ctx := readinessTestStore(t)
	now := time.Unix(1700000100, 0)
	for i := 0; i < maxReadinessNodeWorkItems; i++ {
		id := fmt.Sprintf("best-effort:%d", i)
		if err := s.EnqueueReadinessWork(ctx, id, "", factory.DependenceRef{Repository: 7, Issue: int64(i + 1)}); err != nil {
			t.Fatal("enqueue:", err)
		}
		if _, err := s.BeginReadinessWork(ctx, id, now); err != nil {
			t.Fatal("allocate bounded root:", err)
		}
	}
	if err := s.EnqueueReadinessWork(ctx, "best-effort:overflow", "", factory.DependenceRef{Repository: 7, Issue: 99}); err != nil {
		t.Fatal("header admission is separately bounded:", err)
	}
	if _, err := s.BeginReadinessWork(ctx, "best-effort:overflow", now); !errors.Is(err, ErrReadinessPending) {
		t.Fatal("fifth node-bearing source must wait:", err)
	}
	if err := s.DeferReadinessWork(ctx, "best-effort:0", now); err != nil {
		t.Fatal("defer source:", err)
	}
	if _, err := s.BeginReadinessWork(ctx, "best-effort:0", now); !errors.Is(err, ErrReadinessPending) {
		t.Fatal("deferred source must respect retry time:", err)
	}
	var resetNodes int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_readiness_nodes WHERE source=$1`, "best-effort:0").Scan(&resetNodes); err != nil || resetNodes != 0 {
		t.Fatal("defer must release all node capacity:", resetNodes, err)
	}
	if _, err := s.BeginReadinessWork(ctx, "best-effort:overflow", now); err != nil {
		t.Fatal("released node capacity:", err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE factory_readiness_budget SET generation=generation+1 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	work, err := s.BeginReadinessWork(ctx, "best-effort:1", now)
	if err != nil || work.RootChanged {
		t.Fatal("generation restart must not invent a changed assessment:", work, err)
	}
	var nodes int
	if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_readiness_nodes WHERE source=$1`, work.ID).Scan(&nodes); err != nil || nodes != 1 {
		t.Fatal("generation restart must retain only root:", nodes, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM factory_readiness_sources`); err != nil {
		t.Fatal(err)
	}
}

func TestReadinessWorkSourceAdmissionBound(t *testing.T) {
	s, ctx := readinessTestStore(t)
	for i := 0; i < maxReadinessSources; i++ {
		if err := s.EnqueueReadinessWork(ctx, fmt.Sprintf("source:%02d", i), "", factory.DependenceRef{Repository: 7, Issue: int64(i + 1)}); err != nil {
			t.Fatal("admit within source cap:", i, err)
		}
	}
	if err := s.EnqueueReadinessWork(ctx, "source:overflow", "", factory.DependenceRef{Repository: 7, Issue: 1000}); !errors.Is(err, ErrReadinessCapacity) {
		t.Fatal("source cap must refuse before allocation:", err)
	}
}

func TestReadinessWorkFairRotationAndStaleCheckpoint(t *testing.T) {
	s, ctx := readinessTestStore(t)
	now := time.Unix(1700000200, 0)
	for _, id := range []string{"fair:a", "fair:b", "fair:c"} {
		if err := s.EnqueueReadinessWork(ctx, id, "", factory.DependenceRef{Repository: 7, Issue: int64(len(id))}); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []string{"fair:a", "fair:b", "fair:c", "fair:a"} {
		work, err := s.NextReadinessWork(ctx, now)
		if err != nil || work.ID != want {
			t.Fatalf("round-robin source: got=%q want=%q err=%v", work.ID, want, err)
		}
	}
	work, err := s.BeginReadinessWork(ctx, "fair:a", now)
	if err != nil {
		t.Fatal(err)
	}
	node, err := s.NextReadinessWorkNode(ctx, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE factory_readiness_budget SET generation=generation+1 WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	if err = s.CheckpointReadinessAssessment(ctx, work, node, false, false); !errors.Is(err, ErrReadinessPending) {
		t.Fatal("stale generation checkpoint must be rejected:", err)
	}
}

func TestReadinessWorkPageCursorDedupAndCapacityRollback(t *testing.T) {
	s, ctx := readinessTestStore(t)
	now := time.Unix(1700000300, 0)
	root := factory.DependenceRef{Repository: 7, Issue: 9}
	decision := factory.Acceptance{
		ID: "d" + fmt.Sprintf("%024d", 1), Repository: 7, IssueIndex: "3", Approver: 5,
		NativeRev: 9, TitleDigest: strings.Repeat("d", 64), ContentDigest: strings.Repeat("d", 64),
		Prerequisites: []factory.AcceptedPrerequisite{{Occurrence: "21", DependsOn: "8", EndpointRepo: 7, EndpointIssue: 9, Outcome: factory.PrereqResult}},
	}
	if err := s.AdmitAcceptanceDecision(ctx, decision); err != nil {
		t.Fatal(err)
	}
	if err := s.EnqueueReadinessWork(ctx, "page:normal", "", root); err != nil {
		t.Fatal(err)
	}
	work, err := s.BeginReadinessWork(ctx, "page:normal", now)
	if err != nil {
		t.Fatal(err)
	}
	node, err := s.NextReadinessWorkNode(ctx, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CheckpointReadinessAssessment(ctx, work, node, false, false); err != nil {
		t.Fatal(err)
	}
	node, err = s.NextReadinessWorkNode(ctx, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	more, err := s.CheckpointReadinessPage(ctx, work, node)
	if err != nil || more {
		t.Fatal("bounded page:", more, err)
	}
	if _, err = s.CheckpointReadinessPage(ctx, work, node); !errors.Is(err, ErrReadinessPending) {
		t.Fatal("replayed stale cursor must not advance twice:", err)
	}
	var storedCursor int64
	if err = s.db.QueryRowContext(ctx, `SELECT cursor_issue FROM factory_readiness_nodes WHERE source=$1 AND repository=7 AND issue=9`, work.ID).Scan(&storedCursor); err != nil || storedCursor != 3 {
		t.Fatal("final page cursor should advance over scanned heads:", storedCursor, err)
	}
	var matches int
	if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_readiness_nodes WHERE source=$1 AND repository=7 AND issue=3`, work.ID).Scan(&matches); err != nil || matches != 1 {
		t.Fatal("match was not enqueued exactly once:", matches, err)
	}

	if err := s.EnqueueReadinessWork(ctx, "page:full", "", root); err != nil {
		t.Fatal(err)
	}
	full, err := s.BeginReadinessWork(ctx, "page:full", now)
	if err != nil {
		t.Fatal(err)
	}
	fullNode, err := s.NextReadinessWorkNode(ctx, full.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CheckpointReadinessAssessment(ctx, full, fullNode, false, false); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO factory_readiness_nodes(source,repository,issue) SELECT $1,8,n FROM generate_series(1,$2) n`, full.ID, maxReadinessNodesPerWork-1); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CheckpointReadinessPage(ctx, full, fullNode); !errors.Is(err, ErrReadinessCapacity) {
		t.Fatal("full source must reject a page before checkpoint:", err)
	}
	var count, cursor int
	if err = s.db.QueryRowContext(ctx, `SELECT count(*),max(cursor_issue) FILTER (WHERE repository=7 AND issue=9) FROM factory_readiness_nodes WHERE source=$1`, full.ID).Scan(&count, &cursor); err != nil || count != maxReadinessNodesPerWork || cursor != 0 {
		t.Fatal("capacity rejection must roll back nodes and cursor:", count, cursor, err)
	}
}

func TestReadinessWorkFinishPruneFailureRollsBackAckAndCleanup(t *testing.T) {
	s, ctx := readinessTestStore(t)
	now := time.Unix(1700000400, 0)
	if err := s.EnqueueReadinessWork(ctx, "delivery:prune-fail", "delivery-prune-fail", factory.DependenceRef{Repository: 7, Issue: 3}); err != nil {
		t.Fatal(err)
	}
	work, err := s.BeginReadinessWork(ctx, "delivery:prune-fail", now)
	if err != nil {
		t.Fatal(err)
	}
	node, err := s.NextReadinessWorkNode(ctx, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CheckpointReadinessAssessment(ctx, work, node, true, false); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO intake_deliveries(delivery,repository,issue,kind,received_at) SELECT 'old-'||n,7,3,'event',n FROM generate_series(1,$1) n`, MaxIntakeDeliveries+1); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `CREATE FUNCTION fail_readiness_prune() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced prune failure'; END $$`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, `CREATE TRIGGER fail_readiness_prune BEFORE DELETE ON intake_deliveries FOR EACH ROW EXECUTE FUNCTION fail_readiness_prune()`); err != nil {
		t.Fatal(err)
	}
	if complete, err := s.CompleteReadinessWork(ctx, work, now); err == nil || complete {
		t.Fatal("forced prune failure must abort completion:", complete, err)
	}
	seen, err := s.IntakeDeliverySeen(ctx, "delivery-prune-fail")
	if err != nil || seen {
		t.Fatal("failed completion must leave delivery unseen:", seen, err)
	}
	if _, err = s.ReadinessWork(ctx, work.ID); err != nil {
		t.Fatal("failed completion must retain source:", err)
	}
	var count int
	if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM factory_readiness_nodes WHERE source=$1`, work.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("failed completion must retain node progress:", count, err)
	}
}
