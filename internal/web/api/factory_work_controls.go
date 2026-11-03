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

func (s *API) workControls(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.WorkControls {
	return factoryservice.WorkControls{Store: s.Store, Visibility: s.workJourneys(r, v, admitted).Visibility,
		Authority: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (factoryservice.WorkAuthority, error) {
			if actor != admitted {
				return factoryservice.WorkAuthority{}, auth.ErrNativeSessionChanged
			}
			request := r.WithContext(ctx)
			current := factoryservice.WorkAuthority{SessionCurrent: s.extensionSessionCurrent(ctx, request, v)}
			if _, err := s.factoryExecutionRepository(request, v, int64(repository)); err != nil {
				switch {
				case errors.Is(err, errRepositoryWriteRequired):
					current.Reason = factoryservice.WorkWriteRequired
				case errors.Is(err, errRepositoryDenied):
					current.Reason = factoryservice.WorkNotVisible
				default:
					current.Reason = factoryservice.WorkAuthorityUnavailable
				}
			}
			return current, nil
		},
		Membership: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, project string) (bool, error) {
			if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
				return false, auth.ErrNativeSessionChanged
			}
			// Native Project membership retains its native user key. Other
			// runtimes supply their own membership port to the shared reader.
			_, err := s.Store.MemberLogin(ctx, project, v.User.ID)
			if errors.Is(err, store.ErrNotFound) {
				return false, nil
			}
			return err == nil, err
		},
	}
}
