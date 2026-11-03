package api

import (
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

func (s *API) requestPlanning(r *http.Request, v store.Session, actor collaboration.ActorID) factoryservice.RequestPlanning {
	requests := s.requestReads()
	requests.WriteAccess = s.requestWriteAccess(r, v, actor)
	planning := factoryservice.RequestPlanning{Requests: requests}
	if s.Coordinator != nil {
		if s.Coordinator.Host != nil {
			planning.Planner = s.Coordinator
		}
		if s.Coordinator.Graph != nil {
			planning.Generator = s.Coordinator
		}
	}
	return planning
}
