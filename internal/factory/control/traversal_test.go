package control

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

// TestReadinessSweepReachesDeepPages proves repeated bounded sweeps
// continue past the first page window instead of revisiting its prefix.
func TestReadinessSweepReachesDeepPages(t *testing.T) {
	evidence := map[string]AcceptanceEvidence{}
	pages := map[int]fakeIssuePage{}
	for page := 1; page <= 5; page++ {
		issue := int64(10 + page)
		evidence[fmt.Sprintf("42/%d", issue)] = readinessEvidence(12, readinessView(fmt.Sprint(issue)), nil, nil)
		pages[page] = fakeIssuePage{indexes: []int64{issue}, hasMore: page < 5}
	}
	observer := &fakeObserver{revision: 12, idle: true, pages: pages}
	c := readinessCoordinator(t, &fakeEvidenceSource{evidence: evidence}, observer)
	grantFullAuthority(t, c)
	for range 3 {
		if _, err := c.ReconcileReadiness(context.Background(), 42); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.Store.IssueControl(context.Background(), 42, 15); err != nil {
		t.Fatalf("repeated sweeps never reached page 5: %v", err)
	}
}

// TestDispatchPassReachesRunnableBehindWaitingPrefix proves repeated
// bounded dispatch passes rotate past a permanently waiting prefix to
// later runnable work.
func TestDispatchPassReachesRunnableBehindWaitingPrefix(t *testing.T) {
	db, _ := dispatchTestDB(t)
	fx := dispatchSeed(t, db)
	ctx := context.Background()
	const waiting = MaxDispatchVisits
	base := time.Now().Add(-time.Hour)
	for i := int64(1); i <= waiting; i++ {
		issue := 100 + i
		fx.accept(t, issue, fmt.Sprintf("d%024x", issue))
		fx.queueAt(t, issue, fx.decision[issue].ID, base.Add(time.Duration(i)*time.Second))
		// The accepted objective changed after admission: every pass
		// must wait on these without consuming or finishing them.
		inputs := fx.reads.inputs[fmt.Sprintf("%d/%d", fx.repo, issue)]
		inputs.Issue.TitleDigest = strings.Repeat("f", 64)
		fx.reads.inputs[fmt.Sprintf("%d/%d", fx.repo, issue)] = inputs
	}
	runnable := int64(500)
	fx.accept(t, runnable, fmt.Sprintf("d%024x", runnable))
	fx.queueAt(t, runnable, fx.decision[runnable].ID, base.Add(time.Duration(waiting+1)*time.Second))
	deps := fx.deps()
	// The shared traversal cursor is owned by the coordinator in
	// production; the pass advances whatever cursor it is given.
	deps.Queue = NewDispatchQueueCursor()
	for pass := 0; pass < 2; pass++ {
		report := DispatchPass(ctx, deps)
		if len(report.Errors) != 0 {
			t.Fatalf("pass %d errored: %+v", pass, report.Errors)
		}
	}
	if len(fx.host.launches) != 1 {
		t.Fatalf("repeated passes launched %d runs, want the 1 runnable issue", len(fx.host.launches))
	}
	if _, err := db.AssignmentByRun(ctx, fx.host.launches[0].Run.ID); err != nil {
		t.Fatal("launched run has no dispatch packet:", err)
	}
	_ = project.RoleCoder
	_ = store.ErrNotFound
}
