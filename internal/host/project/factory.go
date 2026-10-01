package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/levitateos/sodaos/internal/filelock"
	"github.com/levitateos/sodaos/internal/host/terminal"
	"github.com/levitateos/sodaos/internal/identity"
	identityclient "github.com/levitateos/sodaos/internal/identity/client"
	domain "github.com/levitateos/sodaos/internal/project"
	"golang.org/x/sys/unix"
)

// FactoryStateRoot is the daemon's protected receipt directory for
// supervised factory runs. tmpfiles owns its parent; the daemon ensures the
// leaf. Tests and drivers inject their own state directory instead.
const FactoryStateRoot = "/var/lib/soda/host/factory"

// Factory orchestrates supervised factory runs: durable run receipts, broker
// lease acquisition with execution identity, and managed native execution
// through host/terminal. Launch and stop never hold the per-run lock across
// broker or native calls; late-stop fencing works through receipt re-checks
// and broker close, so the Register-to-Validate callback cannot deadlock
// against them. Takeover is the exception: it makes no broker calls, so it
// holds the lock across the copy to serialize duplicate takeovers.
type Factory struct {
	terminal *terminal.Service
	broker   *identityclient.Client
	stateDir string
}

// OpenFactory wires run orchestration over a private receipt directory. The
// directory must already exist with private permissions; receipts are
// protected mechanical records, not factory policy.
func OpenFactory(stateDir string, term *terminal.Service, broker *identityclient.Client) (*Factory, error) {
	if term == nil || broker == nil {
		return nil, errors.New("factory run dependencies required")
	}
	if !filepath.IsAbs(stateDir) {
		return nil, errors.New("factory receipt directory must be absolute")
	}
	info, err := os.Lstat(stateDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("factory receipts must live in a private directory")
	}
	return &Factory{terminal: term, broker: broker, stateDir: stateDir}, nil
}

// factoryReceipt is the durable per-run record. The broker execution identity
// (factory, run ID) is the cross-service source of truth for the lease link;
// this receipt additionally carries the native binding and outcome.
type factoryReceipt struct {
	Run                domain.FactoryRun `json:"run"`
	Lease              *identity.Lease   `json:"lease,omitempty"`
	Binding            *identity.Binding `json:"binding,omitempty"`
	ExitCode           *int              `json:"exit_code,omitempty"`
	Generation         int64             `json:"generation,omitempty"`
	Phase              string            `json:"phase"`
	Started            bool              `json:"started"`
	Delivered          bool              `json:"delivered"`
	CredentialReturned bool              `json:"credential_returned"`
	Output             string            `json:"output,omitempty"`
	Retirement         string            `json:"retirement,omitempty"`
	Reason             string            `json:"reason,omitempty"`
}

func (r factoryReceipt) validate() error {
	if err := r.Run.Validate(); err != nil {
		return err
	}
	if !domain.ValidFactoryPhase(r.Phase) {
		return errors.New("invalid run phase")
	}
	if r.Lease != nil && (r.Lease.ExecutionID != r.Run.ID || r.Lease.Kind != identity.Factory) {
		return errors.New("run lease identity mismatch")
	}
	if r.Binding != nil && (r.Binding.Kind != identity.Factory || r.Binding.ID != r.Run.ID) {
		return errors.New("run binding identity mismatch")
	}
	if len(r.Output) > domain.MaxFactoryOutput+1 || len(r.Reason) > 256 || len(r.Retirement) > 32 {
		return errors.New("run record exceeds bounds")
	}
	return nil
}

func (f *Factory) receiptPath(project, run string) string {
	return filepath.Join(f.stateDir, project+"-"+run+".json")
}

func (f *Factory) lockRun(ctx context.Context, project, run string) (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(f.stateDir, project+"-"+run+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err = filelock.Acquire(ctx, file, unix.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func (f *Factory) loadReceipt(project, run string) (factoryReceipt, bool, error) {
	var receipt factoryReceipt
	data, err := os.ReadFile(f.receiptPath(project, run))
	if errors.Is(err, os.ErrNotExist) {
		return receipt, false, nil
	}
	if err != nil {
		return receipt, false, err
	}
	if len(data) > 128<<10 {
		return receipt, false, errors.New("run receipt exceeds bounds")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&receipt); err != nil {
		return receipt, false, errors.New("invalid run receipt")
	}
	if receipt.Run.ID != run || receipt.Run.Project != project {
		return receipt, false, errors.New("run receipt identity mismatch")
	}
	// A stop tombstone for a never-admitted run carries no validated run
	// record; its only job is to refuse future work under that identity.
	if receipt.Phase == domain.FactoryStopped && receipt.Run.Validate() != nil {
		if receipt.Lease != nil || receipt.Binding != nil || receipt.Started || receipt.Delivered {
			return receipt, false, errors.New("invalid run tombstone")
		}
		return receipt, true, nil
	}
	return receipt, true, receipt.validate()
}

func (f *Factory) storeReceipt(receipt factoryReceipt) error {
	if err := receipt.validate(); err != nil {
		return err
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	path := f.receiptPath(receipt.Run.Project, receipt.Run.ID)
	tmp, err := os.CreateTemp(f.stateDir, ".receipt-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Chmod(0o600)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpName, path)
	}
	if err != nil {
		_ = os.Remove(tmpName)
	}
	return err
}

func receiptState(receipt factoryReceipt, live bool) domain.FactoryState {
	state := domain.FactoryState{
		Generation:         receipt.Generation,
		CredentialReturned: receipt.CredentialReturned,
		Live:               live,
		Delivered:          receipt.Delivered,
		ID:                 receipt.Run.ID,
		Project:            receipt.Run.Project,
		Role:               receipt.Run.Role,
		Phase:              receipt.Phase,
		LeaseID:            receipt.LeaseID(),
		Output:             receipt.Output,
		Retirement:         receipt.Retirement,
		Reason:             receipt.Reason,
		ExitCode:           receipt.ExitCode,
		Unit:               domain.FactoryUnitName(receipt.Run.ID),
	}
	if receipt.Binding != nil {
		state.Container = receipt.Binding.Project
		state.Invocation = receipt.Binding.InvocationID
		state.Login = receipt.Binding.Login
		state.UID = receipt.Binding.UID
		state.GID = receipt.Binding.GID
	}
	if len(state.Output) > domain.MaxFactoryOutput {
		state.Output = state.Output[:domain.MaxFactoryOutput]
	}
	return state
}

func (r factoryReceipt) LeaseID() string {
	if r.Lease == nil {
		return ""
	}
	return r.Lease.ID
}

func receiptTerminal(phase string) bool {
	switch phase {
	case domain.FactoryCompleted, domain.FactoryFailed, domain.FactoryStopped:
		return true
	default:
		return false
	}
}

// Launch admits one supervised run and drives it to completion: broker
// acquisition by execution identity, native reservation, attested credential
// delivery, single-use start, bounded wait and credential return. An existing
// receipt is authoritative: duplicates return its state instead of starting
// another conversation, and recovery runs through Stop with a fresh identity
// for the retry. Only unconfirmed operations return an error.
func (f *Factory) Launch(ctx context.Context, req domain.FactoryLaunch) (domain.FactoryState, error) {
	var empty domain.FactoryState
	if err := req.Validate(); err != nil {
		return empty, err
	}
	if req.HarnessSHA256 != f.terminal.CodexHarnessSHA256 || req.HarnessSHA256 == "" {
		return empty, errors.New("factory harness is unavailable until its own proof passes")
	}
	now := time.Now()
	if !req.Run.Deadline.After(now) || req.Run.Deadline.After(now.Add(3*time.Hour)) {
		return empty, errors.New("run deadline is outside the supervised bound")
	}
	file, err := f.lockRun(ctx, req.Run.Project, req.Run.ID)
	if err != nil {
		return empty, err
	}
	receipt, exists, err := f.loadReceipt(req.Run.Project, req.Run.ID)
	if err == nil && !exists {
		receipt = factoryReceipt{Run: req.Run, Phase: domain.FactoryApproved}
		err = f.storeReceipt(receipt)
	}
	_ = file.Close()
	if err != nil {
		return empty, err
	}
	if exists {
		return receiptState(receipt, false), nil
	}
	return f.drive(ctx, req, receipt)
}

func (f *Factory) drive(ctx context.Context, req domain.FactoryLaunch, receipt factoryReceipt) (domain.FactoryState, error) {
	deadlineCtx, cancel := context.WithDeadline(ctx, req.Run.Deadline)
	defer cancel()
	lease, err := f.acquireRunLease(deadlineCtx, req.Run)
	if err != nil {
		return f.failRun(ctx, receipt, err)
	}
	receipt.Lease, receipt.Generation = &lease, lease.Generation
	stopped, err := f.refreshStopped(ctx, &receipt)
	if err != nil {
		return domain.FactoryState{}, err
	}
	if stopped != nil {
		_ = f.broker.CloseExecution(ctx, identity.Factory, req.Run.ID)
		return *stopped, nil
	}
	binding, _, err := f.terminal.FactoryCodexReserve(deadlineCtx, req.Run, lease, req.HarnessSHA256, int64(time.Until(req.Run.Deadline).Seconds()))
	if err != nil {
		return f.abandonRun(ctx, receipt, "reserve-refused")
	}
	receipt.Binding = &binding
	stopped, err = f.refreshStopped(ctx, &receipt)
	if err != nil {
		return domain.FactoryState{}, err
	}
	if stopped != nil {
		_ = f.terminal.FactoryCodexStop(deadlineCtx, leaseWithBinding(lease, binding))
		_ = f.broker.CloseExecution(ctx, identity.Factory, req.Run.ID)
		return *stopped, nil
	}
	delivery, err := f.broker.Register(deadlineCtx, lease.ID, binding)
	if err != nil {
		_ = f.terminal.FactoryCodexStop(deadlineCtx, leaseWithBinding(lease, binding))
		_ = f.broker.CloseExecution(ctx, identity.Factory, req.Run.ID)
		return f.abandonRun(ctx, receipt, "register-refused")
	}
	defer clear(delivery.Credential)
	stopped, err = f.consumeStart(ctx, &receipt)
	if err != nil {
		return domain.FactoryState{}, err
	}
	if stopped != nil {
		_ = f.terminal.FactoryCodexStop(deadlineCtx, leaseWithBinding(lease, binding))
		_ = f.broker.CloseExecution(ctx, identity.Factory, req.Run.ID)
		return *stopped, nil
	}
	if err = f.terminal.FactoryCodexStart(deadlineCtx, leaseWithBinding(lease, binding), delivery.Credential, req.Prompt); err != nil {
		return f.stopFailedStart(ctx, receipt, lease, binding)
	}
	clear(delivery.Credential)
	receipt.Delivered = true
	if err = f.updateReceipt(ctx, &receipt); err != nil {
		return domain.FactoryState{}, err
	}
	exit, output, err := f.terminal.FactoryCodexWait(deadlineCtx, leaseWithBinding(lease, binding))
	if err != nil {
		return f.stopTimedOut(ctx, receipt, lease, binding)
	}
	return f.finishRun(ctx, receipt, lease, binding, exit, output)
}

func leaseWithBinding(lease identity.Lease, binding identity.Binding) identity.Lease {
	lease.Binding = &binding
	return lease
}

func (f *Factory) acquireRunLease(ctx context.Context, run domain.FactoryRun) (identity.Lease, error) {
	return f.broker.Acquire(ctx, identity.AcquireRequest{
		ProviderID: identity.Codex, ExecutionID: run.ID, ActorID: run.Actor,
		ConnectionID: run.Connection, ProjectID: run.Project, Kind: identity.Factory,
		Deadline: run.Deadline, Role: run.Role,
	})
}

// failRun records a refusal that never acquired native or broker resources.
// A concurrent stop still wins: the tombstone is authoritative.
func (f *Factory) failRun(ctx context.Context, receipt factoryReceipt, cause error) (domain.FactoryState, error) {
	file, err := f.lockRun(ctx, receipt.Run.Project, receipt.Run.ID)
	if err != nil {
		return domain.FactoryState{}, err
	}
	defer func() { _ = file.Close() }()
	current, exists, err := f.loadReceipt(receipt.Run.Project, receipt.Run.ID)
	if err != nil || !exists {
		return domain.FactoryState{}, err
	}
	if current.Phase == domain.FactoryStopped {
		return receiptState(current, false), nil
	}
	if errors.Is(cause, identity.ErrBusy) {
		current.Reason = "broker-busy"
		if err = f.storeReceipt(current); err != nil {
			return domain.FactoryState{}, err
		}
		return receiptState(current, false), nil
	}
	if errors.Is(cause, identity.ErrDenied) || errors.Is(cause, identity.ErrUncertain) || errors.Is(cause, context.DeadlineExceeded) {
		current.Phase, current.Reason = domain.FactoryFailed, runReason(cause)
		if err = f.storeReceipt(current); err != nil {
			return domain.FactoryState{}, err
		}
		return receiptState(current, false), nil
	}
	return domain.FactoryState{}, cause
}

func runReason(err error) string {
	switch {
	case errors.Is(err, identity.ErrDenied):
		return "broker-denied"
	case errors.Is(err, identity.ErrUncertain):
		return "broker-unavailable"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline-exceeded"
	default:
		return "launch-refused"
	}
}

// abandonRun closes the broker execution and records failure after a refused
// local step. Transport failures stay errors: the step may have landed.
func (f *Factory) abandonRun(ctx context.Context, receipt factoryReceipt, reason string) (domain.FactoryState, error) {
	_ = f.broker.CloseExecution(ctx, identity.Factory, receipt.Run.ID)
	file, err := f.lockRun(ctx, receipt.Run.Project, receipt.Run.ID)
	if err != nil {
		return domain.FactoryState{}, err
	}
	defer func() { _ = file.Close() }()
	current, exists, err := f.loadReceipt(receipt.Run.Project, receipt.Run.ID)
	if err != nil || !exists {
		return domain.FactoryState{}, err
	}
	if current.Phase == domain.FactoryStopped {
		return receiptState(current, false), nil
	}
	current.Phase, current.Reason = domain.FactoryFailed, reason
	if err = f.storeReceipt(current); err != nil {
		return domain.FactoryState{}, err
	}
	return receiptState(current, false), nil
}

// refreshStopped re-reads the receipt and stores progress. A stop tombstone
// aborts the launch; otherwise the receipt carries the recorded progress.
func (f *Factory) refreshStopped(ctx context.Context, receipt *factoryReceipt) (*domain.FactoryState, error) {
	file, err := f.lockRun(ctx, receipt.Run.Project, receipt.Run.ID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	current, exists, err := f.loadReceipt(receipt.Run.Project, receipt.Run.ID)
	if err != nil || !exists {
		return nil, err
	}
	if current.Phase == domain.FactoryStopped {
		state := receiptState(current, false)
		return &state, nil
	}
	current.Lease, current.Binding, current.Generation = receipt.Lease, receipt.Binding, receipt.Generation
	if err = f.storeReceipt(current); err != nil {
		return nil, err
	}
	*receipt = current
	return nil, nil
}

// consumeStart consumes the single-use start marker. A second start, or a
// start after the stop tombstone, never reaches the native boundary.
func (f *Factory) consumeStart(ctx context.Context, receipt *factoryReceipt) (*domain.FactoryState, error) {
	file, err := f.lockRun(ctx, receipt.Run.Project, receipt.Run.ID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	current, exists, err := f.loadReceipt(receipt.Run.Project, receipt.Run.ID)
	if err != nil || !exists {
		return nil, err
	}
	if current.Phase == domain.FactoryStopped || current.Started {
		state := receiptState(current, false)
		return &state, nil
	}
	current.Started, current.Phase = true, domain.FactoryRunning
	if err = f.storeReceipt(current); err != nil {
		return nil, err
	}
	*receipt = current
	return nil, nil
}

func (f *Factory) updateReceipt(ctx context.Context, receipt *factoryReceipt) error {
	file, err := f.lockRun(ctx, receipt.Run.Project, receipt.Run.ID)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	current, exists, err := f.loadReceipt(receipt.Run.Project, receipt.Run.ID)
	if err != nil || !exists {
		return err
	}
	current.Delivered, current.Output, current.ExitCode = receipt.Delivered, receipt.Output, receipt.ExitCode
	current.CredentialReturned, current.Retirement, current.Reason = receipt.CredentialReturned, receipt.Retirement, receipt.Reason
	if receipt.Phase != "" {
		current.Phase = receipt.Phase
	}
	if err = f.storeReceipt(current); err != nil {
		return err
	}
	*receipt = current
	return nil
}

// launchYielded reports whether a concurrent stop tombstoned the run. Native
// stops are idempotent and always run, but credential work yields to the
// stop owner so exactly one path returns or reconciles the lease.
func (f *Factory) launchYielded(ctx context.Context, receipt factoryReceipt) bool {
	file, err := f.lockRun(ctx, receipt.Run.Project, receipt.Run.ID)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	current, exists, err := f.loadReceipt(receipt.Run.Project, receipt.Run.ID)
	return err == nil && exists && current.Phase == domain.FactoryStopped
}

// stopFailedStart retires a run whose start never confirmed: native stop,
// credential capture attempt, broker return or reconcile, then close.
func (f *Factory) stopFailedStart(ctx context.Context, receipt factoryReceipt, lease identity.Lease, binding identity.Binding) (domain.FactoryState, error) {
	l := leaseWithBinding(lease, binding)
	_ = f.terminal.FactoryCodexStop(ctx, l)
	if f.launchYielded(ctx, receipt) {
		return f.recordStopOutcome(ctx, receipt, false, "stopped", false)
	}
	uncertain := false
	if auth, err := f.terminal.FactoryCodexCapture(ctx, l); err == nil {
		if err = f.broker.Return(ctx, lease.ID, binding, auth); err == nil {
			receipt.CredentialReturned = true
		} else {
			uncertain = true
		}
		clear(auth)
	} else {
		_ = f.broker.ReconcileLease(ctx, lease.ID)
		uncertain = true
	}
	if err := f.broker.CloseExecution(ctx, identity.Factory, receipt.Run.ID); err != nil {
		uncertain = true
	}
	return f.recordStopOutcome(ctx, receipt, uncertain, "start-unconfirmed", false)
}

func (f *Factory) stopTimedOut(ctx context.Context, receipt factoryReceipt, lease identity.Lease, binding identity.Binding) (domain.FactoryState, error) {
	l := leaseWithBinding(lease, binding)
	uncertain := false
	if err := f.terminal.FactoryCodexStop(ctx, l); err != nil {
		uncertain = true
	}
	if f.launchYielded(ctx, receipt) {
		return f.recordStopOutcome(ctx, receipt, uncertain, "stopped", false)
	}
	if auth, err := f.terminal.FactoryCodexCapture(ctx, l); err == nil {
		if err = f.broker.Return(ctx, lease.ID, binding, auth); err == nil {
			receipt.CredentialReturned = true
		} else {
			uncertain = true
		}
		clear(auth)
	} else {
		_ = f.broker.ReconcileLease(ctx, lease.ID)
		uncertain = true
	}
	if err := f.broker.CloseExecution(ctx, identity.Factory, receipt.Run.ID); err != nil {
		uncertain = true
	}
	return f.recordStopOutcome(ctx, receipt, uncertain, "deadline-exceeded", false)
}

func (f *Factory) finishRun(ctx context.Context, receipt factoryReceipt, lease identity.Lease, binding identity.Binding, exit int, output string) (domain.FactoryState, error) {
	l := leaseWithBinding(lease, binding)
	if exit >= 0 {
		receipt.ExitCode = &exit
	}
	receipt.Output = output
	// The CLI exited, but descendants may linger: retire the boundary
	// before capturing, so the captured bytes are final.
	if err := f.terminal.FactoryCodexStop(ctx, l); err != nil {
		_ = f.broker.ReconcileLease(ctx, lease.ID)
		_ = f.broker.CloseExecution(ctx, identity.Factory, receipt.Run.ID)
		return f.recordStopOutcome(ctx, receipt, true, "retirement-unconfirmed", false)
	}
	if f.launchYielded(ctx, receipt) {
		return f.recordStopOutcome(ctx, receipt, false, "stopped", false)
	}
	auth, err := f.terminal.FactoryCodexCapture(ctx, l)
	if err != nil {
		_ = f.broker.ReconcileLease(ctx, lease.ID)
		_ = f.broker.CloseExecution(ctx, identity.Factory, receipt.Run.ID)
		return f.recordStopOutcome(ctx, receipt, true, "credential-capture-unconfirmed", false)
	}
	defer clear(auth)
	if err = f.broker.Return(ctx, lease.ID, binding, auth); err != nil {
		_ = f.broker.CloseExecution(ctx, identity.Factory, receipt.Run.ID)
		return f.recordStopOutcome(ctx, receipt, true, "credential-return-unconfirmed", false)
	}
	receipt.CredentialReturned = true
	_ = f.broker.CloseExecution(ctx, identity.Factory, receipt.Run.ID)
	receipt.Phase, receipt.Retirement, receipt.Reason = domain.FactoryCompleted, "confirmed", ""
	if exit != 0 {
		receipt.Phase, receipt.Reason = domain.FactoryFailed, "execution-failed"
	}
	if err = f.updateReceipt(ctx, &receipt); err != nil {
		return domain.FactoryState{}, err
	}
	return receiptState(receipt, false), nil
}

// Stop persists the stop tombstone for one run identity and retires its
// native boundary and broker execution. It works before the run is ever
// observed, refuses all subsequent work under that ID, and reports
// requested/confirmed/uncertain retirement. An uncertain run stays fenced;
// stopping again retries reconciliation.
func (f *Factory) Stop(ctx context.Context, req domain.FactoryStop) (domain.FactoryState, error) {
	var empty domain.FactoryState
	if err := req.Validate(); err != nil {
		return empty, err
	}
	file, err := f.lockRun(ctx, req.Project, req.ID)
	if err != nil {
		return empty, err
	}
	receipt, exists, err := f.loadReceipt(req.Project, req.ID)
	if err != nil {
		_ = file.Close()
		return empty, err
	}
	if !exists {
		receipt = factoryReceipt{Run: domain.FactoryRun{ID: req.ID, Project: req.Project}, Phase: domain.FactoryStopped, Retirement: "confirmed", Reason: "stop-before-start"}
		// A tombstone for an unknown run carries no validated run record;
		// store it directly so later launches refuse without a full identity.
		_ = file.Close()
		if err = f.storeTombstone(receipt); err != nil {
			return empty, err
		}
		if err = f.broker.CloseExecution(ctx, identity.Factory, req.ID); err != nil {
			receipt.Reason = "broker-close-uncertain"
			_ = f.storeTombstone(receipt)
			return receiptState(receipt, false), nil
		}
		return receiptState(receipt, false), nil
	}
	if receiptTerminal(receipt.Phase) {
		// A tombstone whose broker fence never confirmed retries it; every
		// other terminal receipt is final.
		retryClose := receipt.Phase == domain.FactoryStopped && receipt.Reason == "broker-close-uncertain" && receipt.Run.Validate() != nil
		_ = file.Close()
		if retryClose {
			if err = f.broker.CloseExecution(ctx, identity.Factory, req.ID); err == nil {
				receipt.Reason = "stop-before-start"
				_ = f.storeTombstone(receipt)
			}
		}
		return receiptState(receipt, false), nil
	}
	receipt.Phase = domain.FactoryStopped
	if err = f.storeReceipt(receipt); err != nil {
		_ = file.Close()
		return empty, err
	}
	_ = file.Close()
	uncertain := false
	if receipt.Binding != nil && receipt.Lease != nil {
		if err = f.terminal.FactoryCodexStop(ctx, leaseWithBinding(*receipt.Lease, *receipt.Binding)); err != nil {
			uncertain = true
		}
	} else if receipt.Run.Validate() == nil {
		if err = f.terminal.FactoryCodexStopUnbound(ctx, receipt.Run); err != nil {
			uncertain = true
		}
	}
	if !f.reconcileRunCredential(ctx, &receipt) {
		uncertain = true
	}
	if err = f.broker.CloseExecution(ctx, identity.Factory, req.ID); err != nil {
		uncertain = true
	}
	reason := "stopped"
	if uncertain {
		reason = "stop-uncertain"
	}
	return f.recordStopOutcome(ctx, receipt, uncertain, reason, true)
}

// storeTombstone persists a stop marker for a run that was never admitted.
// It skips run validation: the marker's only job is to refuse future work.
func (f *Factory) storeTombstone(receipt factoryReceipt) error {
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	path := f.receiptPath(receipt.Run.Project, receipt.Run.ID)
	tmp, err := os.CreateTemp(f.stateDir, ".receipt-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Chmod(0o600)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpName, path)
	}
	if err != nil {
		_ = os.Remove(tmpName)
	}
	return err
}

// reconcileRunCredential returns or reconciles the run's broker lease after
// native retirement. It reports whether credential custody is settled.
func (f *Factory) reconcileRunCredential(ctx context.Context, receipt *factoryReceipt) bool {
	if receipt.Lease == nil || receipt.CredentialReturned {
		return true
	}
	if receipt.Delivered && receipt.Binding != nil {
		l := leaseWithBinding(*receipt.Lease, *receipt.Binding)
		auth, err := f.terminal.FactoryCodexCapture(ctx, l)
		if err != nil {
			_ = f.broker.ReconcileLease(ctx, receipt.Lease.ID)
			return false
		}
		defer clear(auth)
		if err = f.broker.Return(ctx, receipt.Lease.ID, *receipt.Binding, auth); err != nil {
			// The racing launch may have returned first: a terminal
			// execution means custody settled without this stop, so a
			// repeated stop converges instead of staying uncertain.
			// CredentialReturned stays false: this stop did not return.
			if executed, gerr := f.broker.GetExecution(ctx, identity.Factory, receipt.Run.ID); gerr == nil && executed.State == identity.ExecutionTerminal {
				return true
			}
			return false
		}
		receipt.CredentialReturned = true
		return true
	}
	_ = f.broker.ReconcileLease(ctx, receipt.Lease.ID)
	return true
}

// Inspect reports the authoritative recorded state for one run identity and
// whether its unit is currently live. It never starts, resumes or mutates.
func (f *Factory) Inspect(ctx context.Context, req domain.FactoryInspect) (domain.FactoryState, error) {
	var empty domain.FactoryState
	if err := req.Validate(); err != nil {
		return empty, err
	}
	file, err := f.lockRun(ctx, req.Project, req.ID)
	if err != nil {
		return empty, err
	}
	receipt, exists, err := f.loadReceipt(req.Project, req.ID)
	_ = file.Close()
	if err != nil {
		return empty, err
	}
	if !exists {
		return empty, identity.ErrNotFound
	}
	live := false
	if receipt.Binding != nil && (receipt.Phase == domain.FactoryApproved || receipt.Phase == domain.FactoryRunning) {
		liveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		live = f.terminal.FactoryCodexLive(liveCtx, *receipt.Binding)
		cancel()
	}
	return receiptState(receipt, live), nil
}

// Takeover copies one retired run's retained work into the admitted
// member's own derived checkout destination. The receipt must be terminal
// with a recorded binding; the copy verifies the recorded container
// incarnation against the current running Project container and excludes
// role Git state plus the private run homes. Uncertain, running and
// never-admitted runs refuse: takeover cannot finish while retirement or
// credential accounting is unresolved.
func (f *Factory) Takeover(ctx context.Context, req domain.FactoryTakeover) (domain.TakeoverResult, error) {
	var empty domain.TakeoverResult
	if err := req.Validate(); err != nil {
		return empty, err
	}
	file, err := f.lockRun(ctx, req.Project, req.ID)
	if err != nil {
		return empty, err
	}
	defer func() { _ = file.Close() }()
	receipt, exists, err := f.loadReceipt(req.Project, req.ID)
	if err != nil {
		return empty, err
	}
	if !exists {
		return empty, identity.ErrNotFound
	}
	if !receiptTerminal(receipt.Phase) || receipt.Run.Validate() != nil || receipt.Binding == nil {
		return empty, errors.New("factory run is not retired for takeover")
	}
	if !domain.TakeoverSource("/home/"+receipt.Run.Role+"/checkouts/"+receipt.Run.Preparation, receipt.Run.Role, receipt.Run.Preparation) {
		return empty, errors.New("invalid takeover source")
	}
	dest, reused, err := f.terminal.FactoryTakeoverCopy(ctx, req.Project, receipt.Binding.Project, receipt.Run.Role, receipt.Run.Preparation, req.Member, req.ID)
	if err != nil {
		return empty, err
	}
	result := domain.TakeoverResult{ID: req.ID, Project: req.Project, Member: req.Member, Destination: dest, Reused: reused}
	if err = result.Validate(); err != nil {
		return empty, err
	}
	return result, nil
}

func (f *Factory) recordStopOutcome(ctx context.Context, receipt factoryReceipt, uncertain bool, reason string, overwrite bool) (domain.FactoryState, error) {
	file, err := f.lockRun(ctx, receipt.Run.Project, receipt.Run.ID)
	if err != nil {
		return domain.FactoryState{}, err
	}
	defer func() { _ = file.Close() }()
	current, exists, err := f.loadReceipt(receipt.Run.Project, receipt.Run.ID)
	if err != nil || !exists {
		return domain.FactoryState{}, err
	}
	// A concurrent operator stop owns the outcome of a launch; only the
	// stop path itself overwrites the tombstone it just persisted.
	if current.Phase == domain.FactoryStopped && !overwrite {
		return receiptState(current, false), nil
	}
	current.CredentialReturned, current.Output, current.ExitCode = receipt.CredentialReturned, receipt.Output, receipt.ExitCode
	if uncertain {
		current.Phase, current.Retirement, current.Reason = domain.FactoryUncertain, "uncertain", reason
	} else if overwrite {
		current.Phase, current.Retirement, current.Reason = domain.FactoryStopped, "confirmed", reason
	} else {
		current.Phase, current.Retirement, current.Reason = domain.FactoryFailed, "confirmed", reason
	}
	if err = f.storeReceipt(current); err != nil {
		return domain.FactoryState{}, err
	}
	return receiptState(current, false), nil
}
