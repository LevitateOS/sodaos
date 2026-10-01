package forgejo

import (
	"context"
	"net/url"
	"strings"
)

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
