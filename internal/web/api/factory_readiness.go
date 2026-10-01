package api

import (
	"context"

	"github.com/levitateos/sodaos/internal/factory/control"
	"github.com/levitateos/sodaos/internal/forgejo"
)

// ServiceReadinessSource adapts the unattended service observation client
// to the coordinator's acceptance evidence and readiness observation
// interfaces: bracketed snapshot reads through the background channel,
// cheap revision observations, and bounded issue enumeration over the
// same service credential.
type ServiceReadinessSource struct {
	Evidence AcceptanceSnapshotSource
	Dispatch DispatchSnapshotSource
	Observer *forgejo.ServiceObserver
}

var (
	_ control.AcceptanceSource     = (*ServiceReadinessSource)(nil)
	_ control.ReadinessObservation = (*ServiceReadinessSource)(nil)
	_ control.DispatchReads        = (*ServiceReadinessSource)(nil)
)

// NewServiceReadinessSource binds one service observer to both
// coordinator observation interfaces.
func NewServiceReadinessSource(observer *forgejo.ServiceObserver) *ServiceReadinessSource {
	return &ServiceReadinessSource{
		Evidence: AcceptanceSnapshotSource{
			Reader: observer.SnapshotReader(), Credential: observer.Credential(),
		},
		Dispatch: DispatchSnapshotSource{
			Reader: observer.SnapshotReader(), Credential: observer.Credential(),
		},
		Observer: observer,
	}
}

// ReadAcceptanceEvidence brackets one native evidence read at a single
// idle revision through the service background channel.
func (s *ServiceReadinessSource) ReadAcceptanceEvidence(ctx context.Context, repository, issue string, commentIDs []string) (control.AcceptanceEvidence, error) {
	if s == nil || s.Observer == nil {
		return control.AcceptanceEvidence{}, &control.AcceptanceRefusal{Reason: control.RefusalSnapshotUnavailable}
	}
	return s.Evidence.ReadAcceptanceEvidence(ctx, repository, issue, commentIDs)
}

// ReadDispatchInputs brackets one accepted-input read at a single idle
// revision through the service background channel.
func (s *ServiceReadinessSource) ReadDispatchInputs(ctx context.Context, repository, issue string, commentIDs []string, targetRef string) (control.DispatchInputs, error) {
	if s == nil || s.Observer == nil {
		return control.DispatchInputs{}, &control.AcceptanceRefusal{Reason: control.RefusalSnapshotUnavailable}
	}
	return s.Dispatch.ReadDispatchInputs(ctx, repository, issue, commentIDs, targetRef)
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

// ListRepositoryIssues enumerates one bounded page of a repository's
// native issue indexes, oldest first.
func (s *ServiceReadinessSource) ListRepositoryIssues(ctx context.Context, repository int64, page int) ([]int64, bool, error) {
	if s == nil || s.Observer == nil {
		return nil, false, forgejo.ErrUnavailable
	}
	return s.Observer.ListIssuesPage(ctx, repository, page)
}
