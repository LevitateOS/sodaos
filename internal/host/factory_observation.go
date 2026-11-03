package host

import (
	"context"
	"errors"
	"fmt"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory/execution"
	"github.com/levitateos/factory/pkg/identity"
	"github.com/levitateos/factory/pkg/project"
)

type FactoryObservationHost interface {
	FactoryInspect(context.Context, project.FactoryInspect) (project.FactoryState, error)
	FactoryStop(context.Context, project.FactoryStop) (project.FactoryState, error)
}

type FactoryCustodyBroker interface {
	GetExecution(context.Context, string, string) (identity.Execution, error)
	CloseExecution(context.Context, string, string) error
}

// FactoryObserver adapts the existing appliance transports. Launch consumes
// the captured semantic packet; native admission remains with the host. It
// acquires no credentials and enrolls no placement. A stop acknowledgement and
// a broker seal are separate observations.
type FactoryObserver struct {
	host   FactoryObservationHost
	broker FactoryCustodyBroker
}

var (
	_ execution.RunObserver     = (*FactoryObserver)(nil)
	_ execution.CustodyObserver = (*FactoryObserver)(nil)
)

func NewFactoryObserver(host FactoryObservationHost, broker FactoryCustodyBroker) *FactoryObserver {
	return &FactoryObserver{host: host, broker: broker}
}

func (c *Client) InspectRun(ctx context.Context, address execution.RunAddress) (execution.RunObservation, error) {
	return NewFactoryObserver(c, nil).InspectRun(ctx, address)
}

func (c *Client) RetireRun(ctx context.Context, address execution.RunAddress) (execution.RunObservation, error) {
	return NewFactoryObserver(c, nil).RetireRun(ctx, address)
}

func (c *Client) LaunchRun(ctx context.Context, launch execution.LaunchRequest) (execution.RunObservation, error) {
	return NewFactoryObserver(c, nil).LaunchRun(ctx, launch)
}

func (o *FactoryObserver) LaunchRun(ctx context.Context, request execution.LaunchRequest) (execution.RunObservation, error) {
	if o == nil || request.Validate() != nil || request.Kind != collaboration.Appliance {
		return execution.RunObservation{}, execution.ErrLaunch
	}
	launch := request.Input
	bounded, cancel := context.WithDeadline(ctx, launch.Run.Deadline)
	defer cancel()
	if err := bounded.Err(); err != nil {
		return execution.RunObservation{}, err
	}
	launcher, ok := o.host.(interface {
		FactoryLaunch(context.Context, project.FactoryLaunch) (project.FactoryState, error)
	})
	if !ok {
		return execution.RunObservation{}, errors.New("appliance run launcher is unavailable")
	}
	state, err := launcher.FactoryLaunch(bounded, launch)
	if err != nil {
		return execution.RunObservation{}, runObservationError(err)
	}
	if err := bounded.Err(); err != nil {
		return execution.RunObservation{}, err
	}
	observed, err := RunObservationFromNative(state, request.Address())
	if err != nil || !request.MatchesObservation(observed) {
		return execution.RunObservation{}, execution.ErrLaunch
	}
	return observed, nil
}

func nativeRunAddress(address execution.RunAddress) error {
	if address.Validate() != nil {
		return execution.ErrObservation
	}
	return (project.FactoryInspect{Project: address.Workspace, ID: address.Run}).Validate()
}

func runObservationError(err error) error {
	if errors.Is(err, ErrRunNotFound) {
		return fmt.Errorf("%w: %w", execution.ErrRunNotObserved, err)
	}
	if errors.Is(err, ErrRunStale) {
		return fmt.Errorf("%w: %w", execution.ErrRunChanged, err)
	}
	return err
}

func (o *FactoryObserver) InspectRun(ctx context.Context, address execution.RunAddress) (execution.RunObservation, error) {
	if err := nativeRunAddress(address); err != nil {
		return execution.RunObservation{}, err
	}
	if o == nil || o.host == nil {
		return execution.RunObservation{}, errors.New("appliance run observer is unavailable")
	}
	state, err := o.host.FactoryInspect(ctx, project.FactoryInspect{Project: address.Workspace, ID: address.Run})
	if err != nil {
		return execution.RunObservation{}, runObservationError(err)
	}
	return RunObservationFromNative(state, address)
}

func (o *FactoryObserver) RetireRun(ctx context.Context, address execution.RunAddress) (execution.RunObservation, error) {
	if err := nativeRunAddress(address); err != nil {
		return execution.RunObservation{}, err
	}
	if o == nil || o.host == nil {
		return execution.RunObservation{}, errors.New("appliance run observer is unavailable")
	}
	state, err := o.host.FactoryStop(ctx, project.FactoryStop{Project: address.Workspace, ID: address.Run})
	if err != nil {
		return execution.RunObservation{}, runObservationError(err)
	}
	return RunObservationFromNative(state, address)
}

// RunObservationFromNative translates scoped native facts; it neither performs
// a read nor authenticates the supplied wire record. Production uses it after
// the native client returns, and then validates against the captured address.
func RunObservationFromNative(state project.FactoryState, address execution.RunAddress) (execution.RunObservation, error) {
	retirement := execution.Uncertain
	switch {
	case state.Retirement == "confirmed":
		retirement = execution.Retired
	case state.Retirement != "":
		// Unknown future evidence is not retirement confirmation.
	case state.Live:
		retirement = execution.Running
	case state.Invocation != "" && state.ExitCode != nil:
		retirement = execution.Exited
	case state.Phase == project.FactoryApproved && state.Invocation == "" && !state.Delivered:
		retirement = execution.NotStarted
	}
	observation := execution.RunObservation{
		Run: state.ID, Workspace: state.Project, Role: state.Role, Phase: state.Phase, Output: state.Output, Reason: state.Reason,
		Live: state.Live, Retirement: retirement, Kind: collaboration.Appliance,
		Appliance: &execution.ApplianceRunFacts{
			Invocation: execution.ApplianceInvocation{Project: state.Project, Generation: state.Generation, Container: state.Container, Unit: state.Unit, Invocation: state.Invocation, UID: state.UID, GID: state.GID, Login: state.Login},
			LeaseID:    state.LeaseID, CredentialDelivered: state.Delivered, CredentialReturned: state.CredentialReturned,
		},
	}
	if state.ExitCode != nil {
		value := *state.ExitCode
		observation.ExitCode = &value
	}
	if state.VerificationChecks != nil {
		checks := *state.VerificationChecks
		if checks.ExitCode != nil {
			value := *checks.ExitCode
			checks.ExitCode = &value
		}
		observation.VerificationChecks = &checks
	}
	if err := observation.ValidateFor(address); err != nil {
		return execution.RunObservation{}, err
	}
	return observation, nil
}

func (o *FactoryObserver) ReadCustody(ctx context.Context, address execution.RunAddress) (execution.CustodyObservation, error) {
	if err := nativeRunAddress(address); err != nil {
		return execution.CustodyObservation{}, err
	}
	if o == nil || o.broker == nil {
		return execution.CustodyObservation{}, errors.New("appliance custody observer is unavailable")
	}
	metadata, err := o.broker.GetExecution(ctx, identity.Factory, address.Run)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			return execution.CustodyObservation{}, fmt.Errorf("%w: %w", execution.ErrCustodyNotObserved, err)
		}
		return execution.CustodyObservation{}, err
	}
	if metadata.Validate() != nil || metadata.Kind != identity.Factory || metadata.ExecutionID != address.Run {
		return execution.CustodyObservation{}, execution.ErrObservation
	}
	state := execution.CustodyUncertain
	switch metadata.State {
	case identity.ExecutionTerminal:
		if metadata.Binding == nil && metadata.LeaseID == "" {
			state = execution.Released
		}
	case identity.ExecutionLive:
		if metadata.Binding != nil && metadata.LeaseID != "" {
			state = execution.Held
		}
	}
	broker := &execution.BrokerObservation{ExecutionID: metadata.ExecutionID, State: state, Digest: metadata.Digest, LeaseID: metadata.LeaseID}
	observation := execution.CustodyObservation{Run: metadata.ExecutionID, Kind: collaboration.Appliance, Sealed: metadata.State == identity.ExecutionTerminal, Appliance: broker}
	if b := metadata.Binding; b != nil {
		if b.ID != address.Run {
			return execution.CustodyObservation{}, execution.ErrObservation
		}
		broker.Binding = &execution.BrokerBinding{Kind: b.Kind, CredentialRoot: b.CredentialRoot, ID: b.ID, Generation: b.Generation, Invocation: b.InvocationID, Boundary: b.Project, ChildID: b.ChildID, Scope: b.Scope, UID: b.UID, GID: b.GID, Login: b.Login}
	}
	if err := observation.ValidateFor(address); err != nil {
		return execution.CustodyObservation{}, err
	}
	return observation, nil
}

func (o *FactoryObserver) SealCustody(ctx context.Context, address execution.RunAddress) error {
	if err := nativeRunAddress(address); err != nil {
		return err
	}
	if o == nil || o.broker == nil {
		return errors.New("appliance custody observer is unavailable")
	}
	return o.broker.CloseExecution(ctx, identity.Factory, address.Run)
}
