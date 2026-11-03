package api

import (
	"net/http"
	"strings"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryhttpapi "github.com/levitateos/factory/pkg/factory/httpapi"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

// apiFactoryWorkHTTP supplies this host's current work read ports and principal.
func (s *API) apiFactoryWorkHTTP(w http.ResponseWriter, r *http.Request, v store.Session) {
	actor, ok := s.factoryCurrentActor(w, r, v)
	if !ok {
		return
	}
	admitted := collaboration.ActorID(actor)
	transport := factoryhttpapi.Work{Reads: s.recordedWorkReads(r, v, admitted),
		Current: func(w http.ResponseWriter, current *http.Request) bool {
			if !s.extensionSessionCurrent(current.Context(), current, v) {
				auth.ProviderError(w, auth.ErrNativeSessionChanged)
				return false
			}
			return true
		},
		VisibilityError: auth.ProviderError, StatusError: factoryhttpapi.AcceptanceDecisionError,
	}
	if !strings.HasSuffix(r.URL.Path, "/assignment") && !strings.HasSuffix(r.URL.Path, "/plan") {
		if s.Coordinator == nil {
			auth.JSONError(w, 500, "coordinator_unavailable", "Factory coordination is unavailable.")
			return
		}
		transport.Issues = s.issueReads(r, v, admitted)
	}
	product := s.factoryProduct(v, admitted, transport.Current)
	product.Work = transport
	product.Handler().ServeHTTP(w, r)
}
