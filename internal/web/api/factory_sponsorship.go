package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/levitateos/factory/pkg/collaboration"

	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/identity"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

var errSponsorshipProjectRequired = errors.New("native sponsorship requires a repository Project")

func (s *API) sponsorships(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.Sponsorships {
	application := factoryservice.Sponsorships{
		Store:      s.Store,
		Visibility: s.workJourneys(r, v, admitted).Visibility,
		Custody: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID, connectionID string) (factoryservice.SponsorshipCustody, error) {
			var empty factoryservice.SponsorshipCustody
			if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
				return empty, auth.ErrNativeSessionChanged
			}
			if s.Identity == nil {
				return empty, factoryservice.ErrSponsorshipBrokerUnavailable
			}
			// The broker retains native user ownership. The shared service only
			// receives this verified connection's generation and scoped grants.
			connections, err := s.Identity.Connections(ctx, v.User.ID)
			if err != nil {
				if errors.Is(err, identity.ErrDenied) {
					return empty, factoryservice.ErrSponsorshipOwnerRequired
				}
				return empty, err
			}
			var connection identity.Connection
			found := false
			for _, candidate := range connections {
				if candidate.ID == connectionID {
					connection, found = candidate, true
					break
				}
			}
			if !found {
				return empty, factoryservice.ErrSponsorshipConnectionMissing
			}
			if v.User.ID <= 0 || connection.OwnerID != v.User.ID {
				return empty, factoryservice.ErrSponsorshipOwnerRequired
			}
			project, err := s.Store.NativeProjectForCollaborationRepository(ctx, repository)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return empty, errSponsorshipProjectRequired
				}
				return empty, err
			}
			grants, err := s.Identity.Grants(ctx, v.User.ID, connectionID)
			if err != nil {
				if errors.Is(err, identity.ErrDenied) {
					return empty, factoryservice.ErrSponsorshipOwnerRequired
				}
				if !errors.Is(err, identity.ErrNotFound) {
					return empty, err
				}
				grants = nil
			}
			if !s.extensionSessionCurrent(ctx, r, v) {
				return empty, auth.ErrNativeSessionChanged
			}
			return factoryservice.SponsorshipCustody{Repository: repository, Connection: connectionID,
				Project: project.ID, Generation: connection.Generation, Grants: grants}, nil
		},
	}
	if s.Coordinator != nil {
		application.Writer = s.Coordinator
	}
	return application
}
