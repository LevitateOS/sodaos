package api

import (
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) apiFactoryAcceptanceHTTP(w http.ResponseWriter, r *http.Request, v store.Session) {
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	admitted := collaboration.ActorID(actor)
	transport := factoryhttpapi.Acceptances{Service: s.acceptances(r, v, admitted),
		Visibility: s.workJourneys(r, v, admitted).Visibility,
		Current: func(w http.ResponseWriter, current *http.Request) bool {
			return s.checkLifecycleSession(w, current, v)
		},
		AuthorityError: reportExecutionAuthorityError, VisibilityError: auth.ProviderError,
	}
	product := s.factoryProduct(v, admitted, transport.Current)
	product.Acceptances = transport
	product.Handler().ServeHTTP(w, r)
}
