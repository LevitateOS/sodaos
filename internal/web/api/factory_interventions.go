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

func (s *API) interventions(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.Interventions {
	current := func(ctx context.Context, actor collaboration.ActorID) error {
		if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
			return auth.ErrNativeSessionChanged
		}
		return nil
	}
	application := factoryservice.Interventions{Store: s.Store,
		WriteAccess: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) error {
			if err := current(ctx, actor); err != nil {
				return err
			}
			_, err := s.factoryExecutionRepository(r.WithContext(ctx), v, int64(repository))
			return err
		},
		RunRepository: func(ctx context.Context, actor collaboration.ActorID, run factory.Run) (collaboration.RepositoryID, error) {
			if err := current(ctx, actor); err != nil {
				return 0, err
			}
			// The native Project remains a host binding. The shared command
			// receives its canonical repository, never a native numeric alias.
			ref, err := s.Store.CollaborationRepositoryForNativeProject(ctx, run.ProjectID)
			if errors.Is(err, store.ErrNotFound) {
				return 0, factory.ErrNotFound
			}
			return ref.ID, err
		},
		RecoveryTarget: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, run factory.Run) (execution.RecoveryTarget, error) {
			if err := current(ctx, actor); err != nil {
				return execution.RecoveryTarget{}, err
			}
			member, err := s.Store.MemberLogin(ctx, run.ProjectID, v.User.ID)
			if errors.Is(err, store.ErrNotFound) {
				return execution.RecoveryTarget{}, factoryservice.ErrInterventionRecoveryTarget
			}
			return execution.RecoveryTarget{Member: member}, err
		},
	}
	if s.Coordinator != nil {
		application.Writer = s.Coordinator
	}
	return application
}
