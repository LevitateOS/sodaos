package control

import (
	"context"
	"errors"
	"testing"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

// A transient broker close heals inside settle: the stopped run still
// reconciles as cancelled instead of fencing on "broker closure
// unconfirmed".
func TestSettleRunRetriesTransientBrokerClose(t *testing.T) {
	closes := 0
	host := &stubHost{stop: func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{Phase: project.FactoryStopped, Reason: "stopped", Retirement: "confirmed"}, nil
	}}
	broker := &stubBroker{
		get: func(string, string) (identity.Execution, error) {
			return identity.Execution{}, nil
		},
		close: func(string, string) error {
			closes++
			if closes == 1 {
				return errors.New("blip")
			}
			return nil
		},
	}
	c := coordinatorFixture(t, host, broker)
	run := recordRun(t, c, nil)
	receipt := c.settleRun(context.Background(), run)
	if !receipt.Confirmed || receipt.Uncertain {
		t.Fatalf("receipt=%+v", receipt)
	}
	if receipt.Outcome != string(factory.Cancelled) {
		t.Fatalf("outcome=%q", receipt.Outcome)
	}
	if closes != 2 {
		t.Fatalf("closes=%d", closes)
	}
	stored, err := c.Store.FactoryRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.Reconciled {
		t.Fatal("run never reconciled")
	}
}

// A fence that never confirms still fences the run.
func TestSettleRunRefusesUnconfirmedBrokerClose(t *testing.T) {
	closes := 0
	host := &stubHost{stop: func(project.FactoryStop) (project.FactoryState, error) {
		return project.FactoryState{Phase: project.FactoryStopped, Reason: "stopped", Retirement: "confirmed"}, nil
	}}
	broker := &stubBroker{
		get: func(string, string) (identity.Execution, error) {
			return identity.Execution{}, nil
		},
		close: func(string, string) error {
			closes++
			return errors.New("down")
		},
	}
	c := coordinatorFixture(t, host, broker)
	run := recordRun(t, c, nil)
	receipt := c.settleRun(context.Background(), run)
	if receipt.Confirmed || !receipt.Uncertain {
		t.Fatalf("receipt=%+v", receipt)
	}
	if closes != 3 {
		t.Fatalf("closes=%d", closes)
	}
}
