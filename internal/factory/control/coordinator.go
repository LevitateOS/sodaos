// Package control supervises recorded factory runs: status reads recorded
// state, dispatch starts bounded recorded assignments, and reconcile settles
// outstanding runs against host and broker truth. Publication advances finished
// candidates through persisted conditional operations. The host owns native
// execution and the broker owns credential custody.
package control

import (
	"context"
	"errors"
	"os"
	"time"

	"golang.org/x/sys/unix"

	"github.com/levitateos/sodaos/internal/factory"
	"github.com/levitateos/sodaos/internal/filelock"
	"github.com/levitateos/sodaos/internal/identity"
	"github.com/levitateos/sodaos/internal/project"
	"github.com/levitateos/sodaos/internal/store"
)

var (
	// ErrNotFound reports an unrecorded run. Reconciliation never adopts
	// host state it did not record.
	ErrNotFound = errors.New("factory run not found")
	// ErrCommandRunning reports a duplicate command whose first execution
	// has not finished. The caller refreshes instead of executing twice.
	ErrCommandRunning = errors.New("factory command already running")
)

// HostFactory is the coordinator's entire host surface: launch starts one
// recorded run under supervision, stop retires one run, inspect observes
// one run without mutating it, takeover copies a retired run's retained
// work, export reads a settled run's exact candidate bundle, and harness
// reports the staged-harness pin dispatch requires.
type HostFactory interface {
	FactoryLaunch(ctx context.Context, in project.FactoryLaunch) (project.FactoryState, error)
	FactoryStop(ctx context.Context, in project.FactoryStop) (project.FactoryState, error)
	FactoryInspect(ctx context.Context, in project.FactoryInspect) (project.FactoryState, error)
	FactoryTakeover(ctx context.Context, in project.FactoryTakeover) (project.TakeoverResult, error)
	FactoryExport(ctx context.Context, in project.FactoryExport) (project.FactoryExportState, error)
	FactoryHarness(ctx context.Context) (project.FactoryHarnessPin, error)
}

// BrokerExecution is the coordinator's entire broker surface: close seals
// one execution and get reads its nonsecret metadata. The coordinator never
// acquires, registers or handles credential delivery.
type BrokerExecution interface {
	GetExecution(ctx context.Context, kind, executionID string) (identity.Execution, error)
	CloseExecution(ctx context.Context, kind, executionID string) error
}

// Coordinator supervises the factory ledger in the dashboard store. One
// coordinator owns the database at a time; startup settles outstanding runs
// before serving operator commands. AcceptanceReads brackets native issue
// evidence for acceptance decisions; while no source is wired, admission
// and validity checks refuse as unavailable instead of guessing.
// DispatchReads brackets accepted-input content plus the target tip for
// dispatch; while no source is wired, dispatch waits instead of guessing.
// Publication executes conditional publication operations; while no
// executor is wired, finished candidates wait instead of publishing.
// Merges executes conditional merge operations; while no executor is
// wired, published candidates wait instead of merging.
// Checks observes native check evidence for published candidates; while
// no assessor is wired, checks report unavailable instead of guessing.
type Coordinator struct {
	Store           *store.Store
	Host            HostFactory
	Broker          BrokerExecution
	AcceptanceReads AcceptanceSource
	Readiness       ReadinessObservation
	DispatchReads   DispatchReads
	Publication     PublicationExecutor
	Reviews         ReviewExecutor
	Checks          CheckObserver
	Merges          MergeExecutor
	traversal       traversalState
	queue           DispatchQueueCursor
	lock            *os.File
}

func NewCoordinator(db *store.Store, host HostFactory, broker BrokerExecution) *Coordinator {
	return &Coordinator{Store: db, Host: host, Broker: broker}
}

// Start takes exclusive coordinator ownership of the database and settles
// outstanding runs. A second coordinator fails fast instead of reconciling
// concurrently.
func (c *Coordinator) Start(ctx context.Context, lockPath string) error {
	if c.Store == nil || c.Host == nil || c.Broker == nil {
		return errors.New("factory coordinator dependencies required")
	}
	if c.lock != nil {
		return errors.New("factory coordinator already started")
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	bounded, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	if err = filelock.Acquire(bounded, file, unix.LOCK_EX); err != nil {
		_ = file.Close()
		return errors.New("another factory coordinator owns this database")
	}
	c.lock = file
	settle, err := c.reconcileRuns(ctx)
	if err != nil {
		_ = c.Close()
		return err
	}
	if report := RecoverDispatch(ctx, c.dispatchDeps()); len(report.Errors) != 0 {
		_ = c.Close()
		return errors.New("factory dispatch recovery failed")
	}
	if err = c.settleAbandonedCommands(ctx, settle); err != nil {
		_ = c.Close()
		return err
	}
	if c.Publication != nil {
		c.PublishPass(ctx)
	}
	if c.Checks != nil {
		c.CheckPass(ctx)
	}
	if c.Merges != nil {
		c.MergePass(ctx)
	}
	return nil
}

// Close releases coordinator ownership. Recorded runs keep their settled or
// fenced state; a later Start reconciles what remains.
func (c *Coordinator) Close() error {
	if c.lock == nil {
		return nil
	}
	err := c.lock.Close()
	c.lock = nil
	return err
}

// Status returns bounded recorded run state with credential-adjacent paths
// redacted. It never calls the host or broker; stop and reconcile refresh
// the recorded observations.
func (c *Coordinator) Status(ctx context.Context, target string) ([]factory.Run, error) {
	bounded, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if target != "" {
		if !factory.ValidID(target) {
			return nil, ErrNotFound
		}
		r, err := c.Store.FactoryRun(bounded, target)
		if err != nil {
			return nil, ErrNotFound
		}
		return []factory.Run{redactRun(r)}, nil
	}
	runs, err := c.Store.FactoryRuns(bounded, 100)
	if err != nil {
		return nil, err
	}
	for i := range runs {
		runs[i] = redactRun(runs[i])
	}
	return runs, nil
}

func redactRun(r factory.Run) factory.Run {
	if r.IdentityBinding == nil {
		return r
	}
	redacted := *r.IdentityBinding
	redacted.CredentialRoot = ""
	r.IdentityBinding = &redacted
	return r
}
