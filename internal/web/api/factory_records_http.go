package api

import (
	"context"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

func (s *API) apiFactoryRecordsHTTP(w http.ResponseWriter, r *http.Request, v store.Session) {
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	admitted := collaboration.ActorID(actor)
	visible := s.factoryRepositoryVisibility(r, v, admitted)
	visibility := func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) error {
		_, err := visible(ctx, actor, repository)
		return err
	}
	transport := factoryhttpapi.Records{
		Current: func(w http.ResponseWriter, current *http.Request) bool {
			return s.checkLifecycleSession(w, current, v)
		},
	}
	if r.Pattern == "GET /api/factory/artifacts/{id}" {
		transport.Artifacts = factoryservice.Artifacts{Source: s.Store, Visibility: visibility}
	} else {
		transport.Library = factoryservice.Library{Source: s.Store, Visibility: visibility}
	}
	product := s.factoryProduct(v, admitted, transport.Current)
	product.Records = transport
	product.Handler().ServeHTTP(w, r)
}
