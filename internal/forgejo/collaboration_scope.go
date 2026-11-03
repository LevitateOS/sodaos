package forgejo

import (
	"context"
	extensions "forgejo.org/extension-sdk"
	"github.com/levitateos/factory/pkg/collaboration"
)

// Resolve the adopted namespace and current human before selecting a resource.
func (s HumanReads) scope(ctx context.Context, actor collaboration.ActorID, repository collaboration.RepositoryID) (string, string, error) {
	if s.Identities == nil {
		return "", "", ErrUnavailable
	}
	nativeRepository, err := s.Identities.NativeRepository(ctx, repository)
	if err != nil {
		return "", "", err
	}
	nativeActor, err := s.Identities.NativeActor(ctx, actor)
	if err != nil {
		return "", "", err
	}
	current, err := s.Authority.Native().CurrentActor(ctx)
	if err != nil {
		return "", "", err
	}
	if current.ID != nativeActor || current.ID != s.Authority.Actor.ID {
		return "", "", extensions.ErrRepositoryNotVisible
	}
	repo, err := s.Authority.Native().Repository(ctx, nativeRepository)
	if err != nil {
		return "", "", err
	}
	name := repo.Owner + "/" + repo.Name
	if repo.ID != nativeRepository || !collaboration.ValidText(name, 512) {
		return "", "", ErrInvalidResponse
	}
	if _, err = s.Identities.registry.ObserveCollaborationRepository(ctx, s.Identities.host, nativeRepository, name); err != nil {
		return "", "", err
	}
	return nativeRepository, nativeActor, nil
}
