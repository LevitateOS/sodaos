package api

import (
	"context"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

// requestReads wires the native setup read into the shared owner-private views.
func (s *API) requestReads() factoryservice.Requests {
	return factoryservice.Requests{Store: s.Store, Setup: s.requestSetup}
}

func (s *API) requestWriteAccess(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.RequestWriteAccess {
	return s.factoryRepositoryAuthority(r, v, admitted).Configure
}

func (s *API) requestSetup(ctx context.Context, repository int64) (factoryservice.RequestSetup, error) {
	preparation, err := s.factoryPreparation(ctx, repository)
	if err != nil {
		return factoryservice.RequestSetup{}, err
	}
	if preparation == nil {
		return factoryservice.RequestSetup{State: factoryservice.RequestSetupMissing}, nil
	}
	state := factoryservice.RequestSetupPreparing
	if preparation.Ready {
		state = factoryservice.RequestSetupReady
	}
	return factoryservice.RequestSetup{Project: preparation.Project, State: state}, nil
}
