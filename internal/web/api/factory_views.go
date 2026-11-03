package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/execution"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) runStatus(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.RunStatus {
	commands := s.interventions(r, v, admitted)
	reader := factoryservice.RunStatus{Store: s.Store, WriteAccess: commands.WriteAccess,
		RunRepository: func(ctx context.Context, actor collaboration.ActorID, run factory.Run) (collaboration.RepositoryID, error) {
			if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
				return 0, auth.ErrNativeSessionChanged
			}
			project, err := s.Store.Project(ctx, run.ProjectID)
			if errors.Is(err, store.ErrNotFound) {
				return 0, factory.ErrNotFound
			}
			if err != nil {
				return 0, err
			}
			// Native visibility precedes disclosure of an unmapped binding.
			// An equal numeric Project repository key cannot adopt a ledger scope.
			if _, err := s.executionRepository(r.WithContext(ctx), v, project.RepositoryID); err != nil {
				return 0, errors.Join(factoryservice.ErrRunStatusAccess, err)
			}
			ref, err := s.Store.CollaborationRepositoryForNativeProject(ctx, project.ID)
			if errors.Is(err, store.ErrNotFound) {
				return 0, factoryservice.ErrRunStatusScope
			}
			return ref.ID, err
		},
		Inspect: func(ctx context.Context, actor collaboration.ActorID, run factory.Run) (execution.RunObservation, error) {
			if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
				return execution.RunObservation{}, auth.ErrNativeSessionChanged
			}
			if s.Host == nil {
				return execution.RunObservation{}, factoryservice.ErrRunStatusObservation
			}
			return s.Host.InspectRun(ctx, execution.RunAddress{Run: run.ID, Workspace: run.ProjectID})
		},
	}
	if s.Coordinator != nil {
		reader.Availability = s.Coordinator
	}
	return reader
}
