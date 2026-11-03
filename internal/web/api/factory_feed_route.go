package api

import (
	"net/http"
	"strings"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// apiFactoryFeedHTTP supplies native host admission to the shared feed protocol.
func (s *API) apiFactoryFeedHTTP(w http.ResponseWriter, r *http.Request, v store.Session) {
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	admitted := collaboration.ActorID(actor)
	feed := factoryhttpapi.Feed{
		Service:    factoryservice.Feed{Store: s.Store},
		Visibility: s.factoryRepositoryVisibility(r, v, admitted),
		Current: func(w http.ResponseWriter, current *http.Request) bool {
			if current.Method == http.MethodPost && strings.HasSuffix(current.URL.Path, "/retry") {
				return s.checkLifecycleSession(w, current, v)
			}
			if !s.extensionSessionCurrent(current.Context(), current, v) {
				auth.ProviderError(w, auth.ErrNativeSessionChanged)
				return false
			}
			return true
		},
	}
	if r.Method == http.MethodPost || strings.Contains(r.URL.Path, "/retry/") {
		feed.Commands = s.postCommands(r, v, admitted)
		feed.Posts = feed.Commands.Posts
	} else if r.URL.Path == "/api/factory/feed" || strings.HasPrefix(r.URL.Path, "/api/factory/feed/posts") {
		feed.Posts = s.posts(r, v, admitted)
	}
	product := s.factoryProduct(v, admitted, feed.Current)
	product.Feed = feed
	product.Handler().ServeHTTP(w, r)
}
