package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
)

func TestRetryStaysQueuedAfterReconciledRun(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	c := coordinatorFixture(t, host, broker)
	lifecycleProjectFixture(t, c, 42)
	grantFullAuthority(t, c)
	readyPreparations(t, c, lifecycleProject)
	settleStubs(host, broker)
	r := recordRun(t, c, nil)
	if _, err := c.RetryRun(context.Background(), factory.NewID(), "native:7", r.ID); err == nil {
		t.Fatal("retry admitted for an unsettled run")
	} else if _, ok := err.(ErrPendingRuns); !ok {
		t.Fatalf("retry pending: %v", err)
	}
	if _, err := c.Stop(context.Background(), stopCommand(r.ID)); err != nil {
		t.Fatal(err)
	}
	decision, err := c.RetryRun(context.Background(), factory.NewID(), "native:7", r.ID)
	if err != nil || !decision.Queued || decision.Prior != r.ID {
		t.Fatal("reconciled retry refused", decision, err)
	}
	if err = decision.Validate(); err != nil {
		t.Fatal(err)
	}
	replay, err := c.RetryRun(context.Background(), decision.CommandID, "native:7", r.ID)
	if err != nil || replay != decision {
		t.Fatal("retry replay diverged", replay, err)
	}
	if _, err = c.RetryRun(context.Background(), factory.NewID(), "native:7", factory.NewID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown retry: %v", err)
	}
}
