package api

import (
	"context"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) workJourneys(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.WorkJourneys {
	return factoryservice.WorkJourneys{Store: s.Store, Visibility: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) error {
		if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
			return auth.ErrNativeSessionChanged
		}
		_, err := s.factoryVisibleRepository(r.WithContext(ctx), v, int64(repository))
		return err
	}}
}
