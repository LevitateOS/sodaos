package forgejo

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// RepositoryByName resolves a user-selected native Git remote before its stable
// repository ID is bound to a broker lease.
func (c *Client) RepositoryByName(ctx context.Context, token, owner, name string) (Repository, error) {
	if !repositoryPart(owner) || !repositoryPart(name) {
		return Repository{}, ErrInvalidResponse
	}
	var repo Repository
	path := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name)
	if err := c.request(ctx, "GET", path, token, nil, &repo); err != nil {
		return Repository{}, err
	}
	if !namedRepositoryValid(repo, owner, name) {
		return Repository{}, ErrInvalidResponse
	}
	return repo, nil
}

func namedRepositoryValid(repo Repository, owner, name string) bool {
	return repo.ID > 0 && repo.Owner.ID > 0 && repositoryPart(repo.Owner.Login) && repositoryPart(repo.Name) &&
		repo.FullName == repo.Owner.Login+"/"+repo.Name && strings.EqualFold(repo.Owner.Login, owner) && strings.EqualFold(repo.Name, name)
}

// RepositoryByID follows native identity across repository renames/transfers.
func (c *Client) RepositoryByID(ctx context.Context, token string, id int64) (Repository, error) {
	if id <= 0 {
		return Repository{}, ErrInvalidResponse
	}
	var repo Repository
	if err := c.request(ctx, "GET", "/repositories/"+strconv.FormatInt(id, 10), token, nil, &repo); err != nil {
		return Repository{}, err
	}
	if repo.ID != id || repo.Owner.ID <= 0 || repo.Owner.Login == "" || repo.Name == "" || repo.FullName == "" {
		return Repository{}, ErrInvalidResponse
	}
	return repo, nil
}

// OrganizationOwner asks Forgejo for the acting user's native ownership, not
// repository administration or the broader organization is_admin capability.
func (c *Client) OrganizationOwner(ctx context.Context, token, login, organization string) (bool, error) {
	for _, part := range []string{login, organization} {
		if part == "" || part == "." || part == ".." || strings.ContainsAny(part, "/\\\x00\r\n") {
			return false, ErrInvalidResponse
		}
	}
	var permissions struct {
		Owner *bool `json:"is_owner"`
	}
	err := c.request(ctx, "GET", "/users/"+url.PathEscape(login)+"/orgs/"+url.PathEscape(organization)+"/permissions", token, nil, &permissions)
	if err != nil {
		return false, err
	}
	if permissions.Owner == nil {
		return false, ErrInvalidResponse
	}
	return *permissions.Owner, nil
}
