package api

import (
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	"github.com/levitateos/factory/pkg/store"
)

// factoryProduct binds the same shared product router used by native hosts to
// this request's admitted Fountain account. Route adapters supply their existing
// current-session check and only the service ports required for that request.
// No unrelated collaborators, provider calls or background work are constructed.
func (s *API) factoryProduct(v store.Session, actor collaboration.ActorID, current func(http.ResponseWriter, *http.Request) bool) factoryhttpapi.Product {
	return factoryhttpapi.Product{Store: s.Store, Host: s.FountainIdentities.HostID(), Actor: actor,
		Principal: factoryPrincipal(v), AuthorName: v.User.Login, Current: current}
}
