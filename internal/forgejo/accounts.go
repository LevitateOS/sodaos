package forgejo

import (
	"context"
	"fmt"
)

// People supports the retained, native-authorized HTML onboarding page.
func (c *Client) People(ctx context.Context, token string, page int) ([]User, Pagination, error) {
	result := []User{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("/admin/users?page=%d&limit=50", page), token, nil, &result)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return result, metadata, err
}
