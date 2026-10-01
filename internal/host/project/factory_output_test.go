package project

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/levitateos/sodaos/internal/identity"
	domain "github.com/levitateos/sodaos/internal/project"
)

func TestFactoryOutputRequiresItsRun(t *testing.T) {
	f := openTestFactory(t, strings.Repeat("9", 64))
	ctx := context.Background()
	run := factoryTestRun()
	_, err := f.Output(ctx, domain.FactoryOutput{Project: run.Project, ID: run.ID, Limit: 1024})
	if !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("unknown run: %v", err)
	}
	if _, err = f.Output(ctx, domain.FactoryOutput{Project: run.Project, ID: "short", Limit: 1024}); err == nil {
		t.Fatal("invalid run address admitted")
	}
}

func TestFactoryOutputBeforeBindingReportsPhase(t *testing.T) {
	f := openTestFactory(t, strings.Repeat("9", 64))
	ctx := context.Background()
	run := factoryTestRun()
	if _, err := f.Stop(ctx, domain.FactoryStop{Project: run.Project, ID: run.ID}); err != nil {
		t.Fatalf("stop tombstone: %v", err)
	}
	state, err := f.Output(ctx, domain.FactoryOutput{Project: run.Project, ID: run.ID, Limit: 1024})
	if err != nil {
		t.Fatalf("tombstone output: %v", err)
	}
	if state.ID != run.ID || state.Project != run.Project || state.Phase != domain.FactoryStopped {
		t.Fatalf("tombstone identity: %+v", state)
	}
	if !state.Terminal || state.Live || state.Container != "" || state.Unit != "" || state.Data != "" || state.Next != 0 {
		t.Fatalf("tombstone carries execution bytes: %+v", state)
	}
}
