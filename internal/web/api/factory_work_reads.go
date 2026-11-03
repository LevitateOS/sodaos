package api

import (
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

func (s *API) recordedWorkReads(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.WorkReads {
	return factoryservice.WorkReads{Store: s.Store, Visibility: s.workJourneys(r, v, admitted).Visibility}
}
