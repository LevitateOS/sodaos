package api

import (
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

func (s *API) requestExploration(r *http.Request, v store.Session, actor collaboration.ActorID) factoryservice.RequestExploration {
	requests := s.requestReads()
	requests.WriteAccess = s.requestWriteAccess(r, v, actor)
	judge := s.IntentExploreChooser
	if judge == nil {
		judge = &control.TypeSafePriority{APIKeyFile: s.Config.TypeSafeAPIKeyFile, Model: s.Config.TypeSafeModel}
	}
	return factoryservice.RequestExploration{Requests: requests, Chooser: judge}
}
