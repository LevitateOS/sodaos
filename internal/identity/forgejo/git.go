package forgejo

import (
	"context"

	upstream "github.com/levitateos/sodaos/internal/forgejo"
	"github.com/levitateos/sodaos/internal/identity"
)

// ResolveGitRepository resolves a native URL to a stable repository identity.
func (p *Provider) ResolveGitRepository(ctx context.Context, ownerID int64, data []byte, owner, name string) (upstream.Repository, error) {
	credential, err := Decode(data, ownerID)
	if err != nil {
		return upstream.Repository{}, identity.ErrDenied
	}
	repo, err := p.client.RepositoryByName(ctx, credential.Access, owner, name)
	if err != nil {
		return upstream.Repository{}, identity.ErrDenied
	}
	return repo, nil
}

// GitAuthority checks current native attribution and repository permission for
// each mediated operation. No account token leaves provider custody.
func (p *Provider) GitAuthority(ctx context.Context, ownerID int64, data []byte, repositoryID int64, write bool) (upstream.Repository, upstream.User, string, error) {
	credential, err := Decode(data, ownerID)
	if err != nil || repositoryID <= 0 {
		return upstream.Repository{}, upstream.User{}, "", identity.ErrDenied
	}
	user, email, err := p.client.GitIdentity(ctx, credential.Access, ownerID)
	if err != nil {
		return upstream.Repository{}, upstream.User{}, "", identity.ErrDenied
	}
	repo, err := p.client.RepositoryByID(ctx, credential.Access, repositoryID)
	if err != nil || repo.ID != repositoryID || !gitPermission(repo.Permissions, write) {
		return upstream.Repository{}, upstream.User{}, "", identity.ErrDenied
	}
	return repo, user, email, nil
}

func gitPermission(permissions *upstream.RepositoryPermissions, write bool) bool {
	if permissions == nil {
		return false
	}
	if write {
		return permissions.Push
	}
	return permissions.Pull
}
