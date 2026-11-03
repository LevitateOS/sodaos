package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"

	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) repositoryPolicies(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.Policies {
	policies := factoryservice.Policies{WriteAccess: s.requestWriteAccess(r, v, admitted), BindingAccess: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, ref factory.ActorBindingRef) error {
		if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
			return auth.ErrNativeSessionChanged
		}
		if _, err := s.FountainIdentities.NativeActor(ctx, collaboration.ActorID(ref.ActorID)); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return factoryservice.ErrPolicyBindingInvalid
			}
			return errors.Join(factoryservice.ErrPolicyBindingUnavailable, err)
		}
		return nil
	}}
	if s.Coordinator != nil {
		policies.Writer = s.Coordinator
	}
	return policies
}
