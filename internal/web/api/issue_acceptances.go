package api

import (
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"

	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

func (s *API) acceptances(r *http.Request, v store.Session, actor collaboration.ActorID) factoryservice.Acceptances {
	commands := s.interventions(r, v, actor)
	application := factoryservice.Acceptances{WriteAccess: commands.WriteAccess}
	if s.Coordinator != nil {
		application.Writer = s.Coordinator
		application.Evidence = s.Coordinator.AcceptanceReads
	}
	return application
}
