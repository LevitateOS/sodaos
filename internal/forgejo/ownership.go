package forgejo

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

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
