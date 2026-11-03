package forgejo

import (
	"context"
	"errors"
	"time"

	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
	factoryservice "github.com/levitateos/factory/pkg/factory/service"
	"github.com/levitateos/factory/pkg/store"
)

var _ factoryservice.CollaborationRepositorySource = HumanReads{}

func (s HumanReads) ReadRepositories(ctx context.Context, actor collaboration.ActorID, q factoryservice.RepositoryCatalogQuery) (collaboration.Page[collaboration.RepositoryRef], error) {
	var empty collaboration.Page[collaboration.RepositoryRef]
	if actor < 1 || q.Validate() != nil {
		return empty, factoryservice.ErrRepositoryCatalogQuery
	}
	if s.Identities == nil {
		return empty, factoryservice.ErrRepositoryCatalogUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	nativeActor, err := s.Identities.NativeActor(ctx, actor)
	if err != nil {
		return empty, humanCatalogError(err)
	}
	current, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, err
	}
	if current.ID != nativeActor || current.ID != s.Authority.Actor.ID {
		return empty, factoryservice.ErrRepositoryCatalogMissing
	}
	p, err := s.Authority.ReadableRepositories(ctx, extensions.RepositoryCatalogRequest{Query: q.Query, Cursor: q.Cursor, Limit: q.Limit})
	if err != nil {
		return empty, humanCatalogError(err)
	}
	host, err := s.Identities.registry.FountainLedgerHost(ctx)
	if err != nil {
		return empty, err
	}
	if host.ID != s.Identities.host || host.Provider != collaboration.Fountain {
		return empty, ErrInvalidResponse
	}
	result := collaboration.Page[collaboration.RepositoryRef]{Observation: collaboration.Observation{State: collaboration.ReadAvailable, Revision: issueReadRevision(p), ObservedAt: time.Now().UTC()}, Items: []collaboration.RepositoryRef{}, Next: p.NextCursor, Complete: p.NextCursor == ""}
	for _, native := range p.Items {
		name := native.Owner + "/" + native.Name
		if !collaboration.ValidText(name, 512) {
			return empty, ErrInvalidResponse
		}
		ref, err := s.Identities.registry.ObserveCollaborationRepository(ctx, host.ID, native.ID, name)
		if err != nil {
			return empty, err
		}
		result.Items = append(result.Items, ref)
	}
	after, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return empty, err
	}
	if after.ID != nativeActor {
		return empty, factoryservice.ErrRepositoryCatalogMissing
	}
	return result, nil
}
func humanCatalogError(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return factoryservice.ErrRepositoryCatalogMissing
	case errors.Is(err, extensions.ErrRepositoryCatalogQuery):
		return factoryservice.ErrRepositoryCatalogQuery
	case errors.Is(err, extensions.ErrRepositoryCatalogChanged):
		return factoryservice.ErrRepositoryCatalogChanged
	default:
		return err
	}
}
