package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) repositoryAccess(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.RepositoryAccess {
	runtime := s.repositoryStatus(r, v, admitted)
	current := func(ctx context.Context, actor collaboration.ActorID) error {
		if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
			return auth.ErrNativeSessionChanged
		}
		return nil
	}
	return factoryservice.RepositoryAccess{Store: s.Store, Visibility: runtime.Visibility, Preparation: runtime.Preparation, Authority: runtime.Authority,
		Viewer: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (factoryservice.RepositoryViewer, error) {
			var viewer factoryservice.RepositoryViewer
			if err := current(ctx, actor); err != nil {
				return viewer, err
			}
			if s.Config == nil {
				return viewer, factoryservice.ErrRepositoryAccessUnavailable
			}
			request := r.WithContext(ctx)
			access, err := s.factoryVisibleRepository(request, v, int64(repository))
			if err != nil {
				return viewer, err
			}
			viewer.Operator = v.User.ID > 0 && v.User.ID == s.Config.OperatorID
			viewer.CodeWrite = repositoryExecutionAllowed(access)
			viewer.OwnerOrAdmin = access.repository.Permissions != nil && access.repository.Permissions.Admin
			owner, err := s.environmentAdministrator(request, access)
			if err != nil {
				return viewer, err
			}
			viewer.EnvironmentOwner, viewer.OwnerOrAdmin = owner, viewer.OwnerOrAdmin || owner
			project, err := s.Store.NativeProjectForCollaborationRepository(ctx, repository)
			if err == nil {
				// Native membership stays keyed by the native session user.
				_, err = s.Store.MemberLogin(ctx, project.ID, v.User.ID)
				viewer.Member = err == nil
			}
			if err != nil && !errors.Is(err, store.ErrNotFound) {
				return viewer, errors.Join(factoryservice.ErrRepositoryAccessUnavailable, err)
			}
			viewer.SessionCurrent = s.extensionSessionCurrent(ctx, r, v)
			return viewer, nil
		},
		Connections: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (map[string]bool, error) {
			if err := current(ctx, actor); err != nil {
				return nil, err
			}
			if s.Identity == nil {
				return nil, factoryservice.ErrSponsorshipBrokerUnavailable
			}
			connections, err := s.Identity.Connections(ctx, v.User.ID)
			if err != nil {
				return nil, err
			}
			owned := map[string]bool{}
			for _, connection := range connections {
				if v.User.ID > 0 && connection.OwnerID == v.User.ID {
					owned[connection.ID] = true
				}
			}
			if err := current(ctx, actor); err != nil {
				return nil, err
			}
			return owned, nil
		},
	}
}
