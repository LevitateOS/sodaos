package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

func TestRetryRequiresLinkedTerminalAttempt(t *testing.T) {
	host, broker := &stubHost{}, &stubBroker{}
	launches := 0
	host.launch = func(project.FactoryLaunch) (project.FactoryState, error) {
		launches++
		return project.FactoryState{}, nil
	}
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
	commandID := factory.NewID()
	if _, err := c.RetryRun(context.Background(), commandID, "native:7", r.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("standalone reconciled run became a retry command: %v", err)
	}
	if _, err := c.Store.FactoryCommand(context.Background(), commandID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unlinked retry persisted a command: %v", err)
	}
	if _, err := c.RetryRun(context.Background(), factory.NewID(), "native:7", factory.NewID()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown retry: %v", err)
	}
	if launches != 0 {
		t.Fatalf("unlinked retry launched %d runs", launches)
	}
}
