package forgejo

import (
	"context"
	"fmt"
)

// Native hook responses contain decrypted authorization and possibly credentials
// in their URL. Never serialize this transport model to dashboard responses.
type Hook struct {
	ID            int64    `json:"id"`
	Type          string   `json:"type"`
	Active        bool     `json:"active"`
	Events        []string `json:"events"`
	BranchFilter  string   `json:"branch_filter"`
	URL           string   `json:"url"`
	Authorization string   `json:"authorization_header"`
}

func (c *Client) Hooks(ctx context.Context, token, owner, repo string, page int) ([]Hook, Pagination, error) {
	items := []Hook{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/hooks?page=%d&limit=30", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) Hook(ctx context.Context, token, owner, repo string, id int64) (Hook, error) {
	var result Hook
	err := c.request(ctx, "GET", fmt.Sprintf("%s/hooks/%d", repoAPI(owner, repo), id), token, nil, &result)
	return result, err
}

type HookInput struct {
	URL           string
	Secret        string
	Authorization string
	Events        []string
	BranchFilter  string
	Active        bool
}

func (c *Client) CreateHook(ctx context.Context, token, owner, repo string, input HookInput) (Hook, error) {
	var result Hook
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/hooks", token, struct {
		Type          string            `json:"type"`
		Config        map[string]string `json:"config"`
		Events        []string          `json:"events"`
		Active        bool              `json:"active"`
		BranchFilter  string            `json:"branch_filter"`
		Authorization string            `json:"authorization_header"`
	}{"forgejo", map[string]string{"url": input.URL, "content_type": "json", "secret": input.Secret}, input.Events, input.Active, input.BranchFilter, input.Authorization}, &result)
	return result, err
}

// Forgejo 15's PATCH replaces events, branch filter and authorization even when
// omitted; callers must explicitly supply the retained or replacement values.
// Native PATCH does not rotate the signing secret, so no such option is exposed.
func (c *Client) EditHook(ctx context.Context, token, owner, repo string, id int64, input HookInput) (Hook, error) {
	var result Hook
	err := c.request(ctx, "PATCH", fmt.Sprintf("%s/hooks/%d", repoAPI(owner, repo), id), token, struct {
		Config        map[string]string `json:"config"`
		Events        []string          `json:"events"`
		Active        bool              `json:"active"`
		BranchFilter  string            `json:"branch_filter"`
		Authorization string            `json:"authorization_header"`
	}{map[string]string{"url": input.URL}, input.Events, input.Active, input.BranchFilter, input.Authorization}, &result)
	return result, err
}
