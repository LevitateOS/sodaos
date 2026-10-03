package control

import (
	"context"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

// settleFixture returns a coordinator whose host confirms retirement and
// whose broker closes cleanly, so reconcile settles every reachable run.
func settleFixture(t *testing.T) (*Coordinator, *stubHost) {
	t.Helper()
	host := &stubHost{}
	c := coordinatorFixture(t, host, &stubBroker{
		get:   func(string, string) (identity.Execution, error) { return identity.Execution{}, identity.ErrNotFound },
		close: func(string, string) error { return nil },
	})
	host.stop = func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{Phase: project.FactoryCompleted, Retirement: "confirmed"}, nil
	}
	return c, host
}

// recordSettledHistory records limit settled runs in another project behind
// the live run, so bounded newest-first readers must skip them to stay
// correct.
func recordSettledHistory(t *testing.T, c *Coordinator, live factory.Run, limit int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < limit; i++ {
		r := factory.Run{ID: factory.NewID(), ProjectID: "p999999999999999999999999", Role: "coder",
			InputSHA: strings.Repeat("a", 40), Started: live.Started, Deadline: live.Deadline,
			Image: live.Image, Harness: live.Harness, Model: live.Model}
		if err := c.Store.RecordFactoryRun(ctx, r); err != nil {
			t.Fatal(err)
		}
		r.Outcome, r.Summary, r.Reconciled = factory.Cancelled, "settled history", true
		if err := c.Store.SaveFactoryRun(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReconcileFindsUnresolvedBehindSettledHistory(t *testing.T) {
	c, _ := settleFixture(t)
	live := recordRun(t, c, nil)
	recordSettledHistory(t, c, live, 1000)
	cmd := factory.Command{ID: factory.NewID(), Type: factory.CommandReconcile, Principal: "os-uid:0", Digest: factory.CommandDigest(factory.CommandReconcile, "")}
	receipt, err := c.Reconcile(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Settled) != 1 || receipt.Settled[0] != live.ID {
		t.Fatalf("reconcile missed unresolved work behind settled history: %+v", receipt)
	}
}

func TestProjectRunsFindsUnresolvedBehindSettledHistory(t *testing.T) {
	c, _ := settleFixture(t)
	live := recordRun(t, c, nil)
	recordSettledHistory(t, c, live, 1000)
	runs, err := c.projectRuns(context.Background(), live.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range runs {
		if run.ID == live.ID {
			return
		}
	}
	t.Fatalf("project inventory missed %d live runs behind settled history", 1)
}
