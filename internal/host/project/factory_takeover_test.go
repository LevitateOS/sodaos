package project

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

func takeoverReceipt(run domain.FactoryRun, phase string, binding *identity.Binding) factoryReceipt {
	return factoryReceipt{Run: run, Phase: phase, Binding: binding}
}

func TestFactoryTakeoverRefusesUnretiredRuns(t *testing.T) {
	f := openTestFactory(t, "")
	ctx := context.Background()
	run := factoryTestRun()
	req := domain.FactoryTakeover{Project: run.Project, ID: run.ID, Member: "alice"}
	if _, err := f.Takeover(ctx, domain.FactoryTakeover{Project: "bad", ID: run.ID, Member: "alice"}); err == nil {
		t.Fatal("invalid takeover address admitted")
	}
	if _, err := f.Takeover(ctx, req); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("unknown run: %v", err)
	}
	if err := f.storeReceipt(takeoverReceipt(run, domain.FactoryApproved, nil)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Takeover(ctx, req); err == nil {
		t.Fatal("approved run admitted to takeover")
	}
}

func TestFactoryTakeoverRefusesTombstonesAndUnboundRuns(t *testing.T) {
	f := openTestFactory(t, "")
	ctx := context.Background()
	run := factoryTestRun()
	stopped, err := f.Stop(ctx, domain.FactoryStop{Project: run.Project, ID: run.ID})
	if err != nil || stopped.Phase != domain.FactoryStopped {
		t.Fatalf("tombstone: %+v %v", stopped, err)
	}
	// The tombstone carries no validated run record.
	if _, err = f.Takeover(ctx, domain.FactoryTakeover{Project: run.Project, ID: run.ID, Member: "alice"}); err == nil {
		t.Fatal("stop-before-start tombstone admitted to takeover")
	}
	second := factoryTestRun()
	second.ID = strings.Repeat("d", 32)
	if err = f.storeReceipt(takeoverReceipt(second, domain.FactoryStopped, nil)); err != nil {
		t.Fatal(err)
	}
	if _, err = f.Takeover(ctx, domain.FactoryTakeover{Project: second.Project, ID: second.ID, Member: "alice"}); err == nil {
		t.Fatal("binding-less run admitted to takeover")
	}
}

func TestFactoryTakeoverVerifiesRecordedIncarnation(t *testing.T) {
	f := openTestFactory(t, "")
	ctx := context.Background()
	run := factoryTestRun()
	recorded := strings.Repeat("c", 64)
	binding := &identity.Binding{Kind: identity.Factory, ID: run.ID, Project: recorded}
	if err := f.storeReceipt(takeoverReceipt(run, domain.FactoryStopped, binding)); err != nil {
		t.Fatal(err)
	}
	fake := f.terminal.Exec.(*factoryFakeExec)
	inspect := func(id string) {
		fake.run = func(context.Context, []byte, string, ...string) ([]byte, error) {
			return []byte(`{"id":"` + id + `","running":true,"project":"` + run.Project + `","owner":"7","privileged":false,"userns":"private","mappings":{}}`), nil
		}
	}
	inspect(strings.Repeat("e", 64))
	req := domain.FactoryTakeover{Project: run.Project, ID: run.ID, Member: "alice"}
	if _, err := f.Takeover(ctx, req); !errors.Is(err, identity.ErrStale) {
		t.Fatalf("replaced container: %v", err)
	}
	inspect(recorded)
	calls := fake.calls
	fake.run = func(ctx context.Context, in []byte, command string, args ...string) ([]byte, error) {
		for _, arg := range args {
			if arg == "inspect" {
				return []byte(`{"id":"` + recorded + `","running":true,"project":"` + run.Project + `","owner":"7","privileged":false,"userns":"private","mappings":{}}`), nil
			}
		}
		return nil, errors.New("native execution unavailable in unit test")
	}
	if _, err := f.Takeover(ctx, req); err == nil {
		t.Fatal("copy failure reported success")
	}
	if fake.calls == calls {
		t.Fatal("verified incarnation never reached the copy")
	}
}
