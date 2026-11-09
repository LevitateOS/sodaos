package api

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"

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
		nativeActor, err := s.FountainIdentities.NativeActor(ctx, collaboration.ActorID(ref.ActorID))
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return factoryservice.ErrPolicyBindingInvalid
			}
			return errors.Join(factoryservice.ErrPolicyBindingUnavailable, err)
		}
		if ref.ControllerGrantID != "" {
			authority, ok := requestExtensionAuthority(r)
			if !ok {
				return factoryservice.ErrPolicyBindingUnavailable
			}
			nativeRepo, err := s.FountainIdentities.NativeRepository(ctx, repository)
			if err != nil {
				return err
			}
			g, err := authority.RuntimeOperationGrant(ctx, nativeRepo, ref.ControllerGrantID)
			if err != nil {
				return errors.Join(factoryservice.ErrPolicyBindingUnavailable, err)
			}
			if !g.Active || g.ActorID != nativeActor || g.TokenID != strconv.FormatInt(ref.TokenID, 10) || !slices.Contains(g.Operations, ref.Kind) {
				return factoryservice.ErrPolicyBindingInvalid
			}
		}
		return nil
	}}
	if s.Coordinator != nil {
		policies.Writer = s.Coordinator
	}
	return policies
}
