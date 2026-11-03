package api

import (
	"errors"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) apiFactoryRunHTTP(w http.ResponseWriter, r *http.Request, v store.Session) {
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	admitted := collaboration.ActorID(actor)
	transport := factoryhttpapi.Runs{Store: s.Store,
		Current: func(w http.ResponseWriter, current *http.Request) bool {
			return s.checkLifecycleSession(w, current, v)
		},
		AuthorityError: reportExecutionAuthorityError, VisibilityError: auth.ProviderError,
		HostError: func(w http.ResponseWriter, err error) bool {
			if errors.Is(err, auth.ErrNativeSessionChanged) {
				auth.ProviderError(w, err)
				return true
			}
			return false
		},
	}
	if r.Method == http.MethodGet {
		transport.Status = s.runStatus(r, v, admitted)
	} else {
		transport.Commands = s.interventions(r, v, admitted)
		transport.Visibility = s.workJourneys(r, v, admitted).Visibility
	}
	product := s.factoryProduct(v, admitted, transport.Current)
	product.Runs = transport
	product.Handler().ServeHTTP(w, r)
}
