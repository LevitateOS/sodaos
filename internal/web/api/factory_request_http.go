package api

import (
	"net/http"
	"strings"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// apiFactoryRequestsHTTP supplies current native ports to shared request HTTP.
func (s *API) apiFactoryRequestsHTTP(w http.ResponseWriter, r *http.Request, v store.Session) {
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	admitted := collaboration.ActorID(actor)
	requests := s.requestReads()
	requests.WriteAccess = s.requestWriteAccess(r, v, admitted)
	transport := factoryhttpapi.Requests{Service: requests,
		Current: func(w http.ResponseWriter, current *http.Request) bool {
			return s.checkLifecycleSession(w, current, v)
		},
		AuthorityError: factoryOwnerError, VisibilityError: auth.ProviderError,
	}
	if strings.HasPrefix(r.URL.Path, "/api/factory/feed/posts/") {
		transport.Posts = s.posts(r, v, admitted)
	}
	if r.Method == http.MethodPost {
		switch {
		case strings.HasSuffix(r.URL.Path, "/plan"), strings.HasSuffix(r.URL.Path, "/generate"):
			transport.Planning = s.requestPlanning(r, v, admitted)
		case strings.HasSuffix(r.URL.Path, "/plan/review"), strings.HasSuffix(r.URL.Path, "/explore"):
			transport.Exploration = s.requestExploration(r, v, admitted)
		}
	}
	product := s.factoryProduct(v, admitted, transport.Current)
	product.Requests = transport
	product.Handler().ServeHTTP(w, r)
}
