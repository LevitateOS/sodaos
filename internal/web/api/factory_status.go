package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/levitateos/factory/pkg/collaboration"
	"github.com/levitateos/factory/pkg/factory"
	"github.com/levitateos/factory/pkg/factory/control"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/project"
	"github.com/levitateos/factory/pkg/store"
	"github.com/levitateos/sodaos/internal/web/auth"
)

func (s *API) repositoryStatus(r *http.Request, v store.Session, admitted collaboration.ActorID) factoryservice.RepositoryStatus {
	current := func(ctx context.Context, actor collaboration.ActorID) error {
		if actor != admitted || !s.extensionSessionCurrent(ctx, r, v) {
			return auth.ErrNativeSessionChanged
		}
		return nil
	}
	status := factoryservice.RepositoryStatus{Store: s.Store, Visibility: s.workJourneys(r, v, admitted).Visibility,
		Bindings: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) ([]factory.ActorBindingRef, error) {
			if err := current(ctx, actor); err != nil {
				return nil, err
			}
			native, err := s.FountainIdentities.NativeRepository(ctx, repository)
			if err != nil {
				return nil, err
			}
			refs := []factory.ActorBindingRef{}
			if s.Config == nil {
				return nil, factoryservice.ErrRepositoryStatusBindings
			}
			for _, binding := range s.Config.FactoryNativeBindings {
				if strconv.FormatInt(binding.RepositoryID, 10) != native {
					continue
				}
				local, err := s.FountainIdentities.LocalActor(ctx, strconv.FormatInt(binding.ActorID, 10))
				if err != nil {
					return nil, err
				}
				refs = append(refs, factory.ActorBindingRef{TokenID: binding.TokenID, ActorID: int64(local), Kind: binding.Kind})
			}
			return refs, nil
		},
		Preparation: func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (*factoryservice.RepositoryPreparation, error) {
			if err := current(ctx, actor); err != nil {
				return nil, err
			}
			return s.factoryPreparation(ctx, int64(repository))
		},
	}
	if s.Coordinator != nil {
		status.Authority = func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (factory.EffectiveAuthority, error) {
			if err := current(ctx, actor); err != nil {
				return factory.EffectiveAuthority{}, err
			}
			return s.Coordinator.EffectiveAuthority(ctx, int64(repository))
		}
		status.Diagnostic = func(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (*control.DispatchObservation, error) {
			if err := current(ctx, actor); err != nil {
				return nil, err
			}
			request := r.WithContext(ctx)
			access, err := s.factoryVisibleRepository(request, v, int64(repository))
			if err != nil {
				return nil, err
			}
			admin := access.repository.Permissions != nil && access.repository.Permissions.Admin
			if !admin {
				admin, err = s.environmentAdministrator(request, access)
			}
			if err != nil || !admin {
				return nil, err
			}
			return s.Coordinator.LatestDispatchObservation(int64(repository)), nil
		}
	}
	return status
}

func (s *API) factoryPreparation(ctx context.Context, repository int64) (*factoryservice.RepositoryPreparation, error) {
	p, err := s.Store.NativeProjectForCollaborationRepository(ctx, collaboration.RepositoryID(repository))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	view := &factoryservice.RepositoryPreparation{Project: p.ID}
	if head, err := s.Store.RequirementHead(ctx, p.ID); err == nil {
		view.Requirements = head
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	if head, err := s.Store.ApprovalHead(ctx, p.ID); err == nil {
		view.Approval = head
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	records, err := s.Store.ProjectPreparations(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	ready := map[string]bool{}
	for _, record := range records {
		if record.State.Ready {
			ready[record.Preparation.Role] = true
		}
	}
	view.Ready = ready[project.RoleCoder] && ready[project.RoleReviewer]
	return view, nil
}
