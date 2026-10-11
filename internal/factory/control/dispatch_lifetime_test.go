package control

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
)

type observationDispatchHost struct {
	*fakeDispatchHost
	observe func(context.Context, project.FactoryLaunch) (project.FactoryState, error)
}

func (h observationDispatchHost) FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
	return h.observe(ctx, in)
}

func TestLaunchObservationDoesNotUseExecutionLifetime(t *testing.T) {
	now := time.Now()
	run := factory.Run{ID: "run", ProjectID: "project", Deadline: now.Add(time.Hour)}
	var observed context.Context
	host := observationDispatchHost{observe: func(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error) {
		observed = ctx
		if in.Run.Deadline != run.Deadline {
			t.Fatal("launch changed the immutable admitted deadline")
		}
		return project.FactoryState{Phase: project.FactoryRunning}, nil
	}}
	caller, cancel := context.WithCancel(context.Background())
	defer cancel()
	launchTail(caller, DispatchDeps{Host: host}, factory.Assignment{}, run, project.FactoryLaunch{Run: project.FactoryRun{Deadline: run.Deadline}}, &DispatchReport{})
	deadline, ok := observed.Deadline()
	if !ok || deadline.After(now.Add(6*time.Second)) {
		t.Fatalf("launch observation retained the execution lifetime: %v", deadline)
	}
	// Capture cancellation while the observation is in progress; cancelling
	// the request releases its scheduler without changing the native lifetime.
	host.observe = func(ctx context.Context, _ project.FactoryLaunch) (project.FactoryState, error) {
		cancel()
		if ctx.Err() == nil {
			t.Error("launch observation detached caller cancellation")
		}
		return project.FactoryState{Phase: project.FactoryRunning}, nil
	}
	launchTail(caller, DispatchDeps{Host: host}, factory.Assignment{}, run, project.FactoryLaunch{}, &DispatchReport{})
}

func TestConfirmUnusedFencesLateLaunchBeforeRelease(t *testing.T) {
	for _, scenario := range []string{"unused", "late-delivery", "close-unconfirmed"} {
		t.Run(scenario, func(t *testing.T) {
			sealed, closed := false, false
			host := &fakeDispatchHost{stop: func(in project.FactoryStop) (project.FactoryState, error) {
				sealed = true
				state := project.FactoryState{ID: in.ID, Project: in.Project, Phase: project.FactoryStopped, Retirement: "confirmed"}
				if scenario == "late-delivery" {
					state.Delivered, state.CredentialReturned, state.LeaseID = true, true, "lease"
				}
				return state, nil
			}}
			broker := &fakeDispatchBroker{
				get: func(kind, id string) (identity.Execution, error) {
					if !closed {
						return identity.Execution{}, identity.ErrNotFound
					}
					return identity.Execution{Kind: kind, ExecutionID: id, State: identity.ExecutionTerminal}, nil
				},
				close: func(string, string) error {
					if scenario == "close-unconfirmed" {
						return errors.New("reply lost")
					}
					closed = true
					return nil
				},
			}
			unused, _ := confirmUnused(context.Background(), DispatchDeps{Host: host, Broker: broker}, factory.Assignment{ProjectID: "project"}, factory.Run{ID: "run"})
			if scenario == "unused" {
				if !unused || !sealed || !closed {
					t.Fatalf("released without both durable fences: unused=%t host=%t broker=%t", unused, sealed, closed)
				}
			} else if unused {
				t.Fatal("released uncertain or delivered capacity")
			}
		})
	}
}
