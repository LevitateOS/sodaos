package observation

import (
	"context"
	"strconv"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// ServiceReadinessSource adapts the unattended service observation client
// to the coordinator's acceptance evidence and readiness observation
// interfaces: bracketed snapshot reads and cheap revision observations
// through the background channel.
type ServiceReadinessSource struct {
	Evidence   AcceptanceSnapshotSource
	Dispatch   DispatchSnapshotSource
	Observer   *forgejo.ServiceObserver
	Identities *forgejo.FountainIdentities
}

var (
	_ control.AcceptanceSource     = (*ServiceReadinessSource)(nil)
	_ control.ReadinessObservation = (*ServiceReadinessSource)(nil)
	_ control.DispatchReads        = (*ServiceReadinessSource)(nil)
)

// NewServiceReadinessSource binds one service observer to both
// coordinator observation interfaces.
func NewServiceReadinessSource(observer *forgejo.ServiceObserver, identities *forgejo.FountainIdentities) *ServiceReadinessSource {
	if observer == nil {
		return &ServiceReadinessSource{Identities: identities}
	}
	return &ServiceReadinessSource{
		Evidence: AcceptanceSnapshotSource{
			Reader: observer.SnapshotReader(), Credential: observer.Credential(),
		},
		Dispatch: DispatchSnapshotSource{
			Reader: observer.SnapshotReader(), Credential: observer.Credential(),
		},
		Observer:   observer,
		Identities: identities,
	}
}

// ReadAcceptanceEvidence brackets one native evidence read at a single
// idle revision through the service background channel.
func (s *ServiceReadinessSource) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, commentIDs []string) (control.AcceptanceEvidence, error) {
	if s == nil || s.Identities == nil {
		return control.AcceptanceEvidence{}, &control.AcceptanceRefusal{Reason: control.RefusalSnapshotUnavailable}
	}
	local, err := strconv.ParseInt(repository, 10, 64)
	if err != nil || local < 1 || strconv.FormatInt(local, 10) != repository {
		return control.AcceptanceEvidence{}, forgejo.ErrInvalidResponse
	}
	native, err := s.Identities.NativeRepository(ctx, collaboration.RepositoryID(local))
	if err != nil {
		return control.AcceptanceEvidence{}, err
	}
	evidence, err := s.Evidence.ReadAcceptanceEvidence(ctx, native, issue, commentIDs)
	if err != nil {
		return control.AcceptanceEvidence{}, err
	}
	if evidence.Issue.PosterID != "" {
		actor, err := s.Identities.LocalActor(ctx, evidence.Issue.PosterID)
		if err != nil {
			return control.AcceptanceEvidence{}, err
		}
		evidence.Issue.PosterID = strconv.FormatInt(int64(actor), 10)
	}
	return evidence, nil
}

// ReadDispatchInputs brackets one accepted-input read at a single idle
// revision through the service background channel.
func (s *ServiceReadinessSource) ReadDispatchInputs(ctx context.Context, repository, issue string, commentIDs []string, targetRef string) (control.DispatchInputs, error) {
	if s == nil || s.Identities == nil {
		return control.DispatchInputs{}, &control.AcceptanceRefusal{Reason: control.RefusalSnapshotUnavailable}
	}
	local, err := strconv.ParseInt(repository, 10, 64)
	if err != nil || local < 1 || strconv.FormatInt(local, 10) != repository {
		return control.DispatchInputs{}, forgejo.ErrInvalidResponse
	}
	native, err := s.Identities.NativeRepository(ctx, collaboration.RepositoryID(local))
	if err != nil {
		return control.DispatchInputs{}, err
	}
	return s.Dispatch.ReadDispatchInputs(ctx, native, issue, commentIDs, targetRef)
}

// ObserveNativeRevision observes the atomic native revision and idle/busy
// state through owning-installation admission.
func (s *ServiceReadinessSource) ObserveNativeRevision(ctx context.Context) (int64, bool, error) {
	if s == nil || s.Observer == nil {
		return 0, false, forgejo.ErrUnavailable
	}
	observation, err := s.Observer.SnapshotReader().ReadNativeRevision(ctx)
	if err != nil {
		return 0, false, err
	}
	return observation.Revision, observation.Idle, nil
}
