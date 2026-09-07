package forgejo

import (
	"context"
	"fmt"
)

// MyRepositories supplies the existing environment-creation picker, not a second
// repository browser or repository-creation authority.
func (c *Client) MyRepositories(ctx context.Context, token string, page int) ([]Repository, Pagination, error) {
	result := []Repository{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("/user/repos?page=%d&limit=50", page), token, nil, &result)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return result, metadata, err
}
