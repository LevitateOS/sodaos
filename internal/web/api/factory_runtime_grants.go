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

var (
	errFactoryOperatorRequired         = errors.New("current host operator is required")
	errFactoryEnvironmentOwnerRequired = errors.New("current repository environment owner is required")
)

func (s *API) runtimeGrants(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.RuntimeGrants {
	grants := factoryservice.RuntimeGrants{
		Visibility: s.workJourneys(r, v, admitted).Visibility,
		OperatorAccess: func(ctx context.Context, actor collaboration.ActorID) error {
			if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
				return auth.ErrNativeSessionChanged
			}
			if s.Config == nil {
				return factoryservice.ErrRuntimeGrantUnavailable
			}
			// Native configuration holds a native user key, never a canonical ID.
			if v.User.ID <= 0 || v.User.ID != s.Config.OperatorID {
				return errFactoryOperatorRequired
			}
			return nil
		},
		EnvironmentAccess: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) error {
			if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
				return auth.ErrNativeSessionChanged
			}
			current := r.WithContext(ctx)
			access, err := s.factoryVisibleRepository(current, v, int64(repository))
			if err != nil {
				return err
			}
			owner, err := s.environmentAdministrator(current, access)
			if err != nil {
				return err
			}
			if !owner {
				return errFactoryEnvironmentOwnerRequired
			}
			if !s.extensionSessionCurrent(ctx, r, v) {
				return auth.ErrNativeSessionChanged
			}
			return nil
		},
	}
	if s.Coordinator != nil {
		grants.Writer = s.Coordinator
	}
	return grants
}
