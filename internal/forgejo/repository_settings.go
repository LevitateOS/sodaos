package forgejo

import (
	"context"
	"fmt"
	"net/url"
)

type RepositorySettings struct {
	Repository
	Website     string `json:"website"`
	HasIssues   bool   `json:"has_issues"`
	HasPulls    bool   `json:"has_pull_requests"`
	HasWiki     bool   `json:"has_wiki"`
	HasActions  bool   `json:"has_actions"`
	HasReleases bool   `json:"has_releases"`
	HasPackages bool   `json:"has_packages"`
}

// Ownership/identity/destructive options are intentionally not part of this DTO.
type EditRepositorySettings struct {
	Description         *string `json:"description,omitempty"`
	Website             *string `json:"website,omitempty"`
	Private             *bool   `json:"private,omitempty"`
	DefaultBranch       *string `json:"default_branch,omitempty"`
	HasIssues           *bool   `json:"has_issues,omitempty"`
	HasPulls            *bool   `json:"has_pull_requests,omitempty"`
	HasWiki             *bool   `json:"has_wiki,omitempty"`
	HasActions          *bool   `json:"has_actions,omitempty"`
	HasReleases         *bool   `json:"has_releases,omitempty"`
	HasPackages         *bool   `json:"has_packages,omitempty"`
	AllowMerge          *bool   `json:"allow_merge_commits,omitempty"`
	AllowSquash         *bool   `json:"allow_squash_merge,omitempty"`
	AllowRebase         *bool   `json:"allow_rebase,omitempty"`
	AllowRebaseExplicit *bool   `json:"allow_rebase_explicit,omitempty"`
	AllowFastForward    *bool   `json:"allow_fast_forward_only_merge,omitempty"`
}

func (c *Client) RepositorySettings(ctx context.Context, token, owner, repo string) (RepositorySettings, error) {
	var result RepositorySettings
	err := c.request(ctx, "GET", repoAPI(owner, repo), token, nil, &result)
	return result, err
}
func (c *Client) EditRepositorySettings(ctx context.Context, token, owner, repo string, input EditRepositorySettings) (RepositorySettings, error) {
	var result RepositorySettings
	err := c.request(ctx, "PATCH", repoAPI(owner, repo), token, input, &result)
	return result, err
}
func (c *Client) Collaborators(ctx context.Context, token, owner, repo string, page int) ([]User, Pagination, error) {
	items := []User{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/collaborators?page=%d&limit=30", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}

type CollaboratorPermission struct {
	Permission string `json:"permission"`
	Role       string `json:"role_name"`
	User       User   `json:"user"`
}

func (c *Client) CollaboratorPermission(ctx context.Context, token, owner, repo, login string) (CollaboratorPermission, error) {
	var result CollaboratorPermission
	err := c.request(ctx, "GET", repoAPI(owner, repo)+"/collaborators/"+url.PathEscape(login)+"/permission", token, nil, &result)
	return result, err
}
func (c *Client) SetCollaborator(ctx context.Context, token, owner, repo, login, permission string, remove bool) error {
	path := repoAPI(owner, repo) + "/collaborators/" + url.PathEscape(login)
	if remove {
		return c.request(ctx, "DELETE", path, token, nil, nil)
	}
	return c.request(ctx, "PUT", path, token, struct {
		Permission string `json:"permission"`
	}{permission}, nil)
}

type DeployKey struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint"`
	ReadOnly    bool   `json:"read_only"`
}

func (c *Client) DeployKeys(ctx context.Context, token, owner, repo string, page int) ([]DeployKey, Pagination, error) {
	items := []DeployKey{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/keys?page=%d&limit=30", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) CreateDeployKey(ctx context.Context, token, owner, repo, title, key string, readOnly bool) (DeployKey, error) {
	var result DeployKey
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/keys", token, struct {
		Title    string `json:"title"`
		Key      string `json:"key"`
		ReadOnly bool   `json:"read_only"`
	}{title, key, readOnly}, &result)
	return result, err
}
func (c *Client) DeleteDeployKey(ctx context.Context, token, owner, repo string, id int64) error {
	return c.request(ctx, "DELETE", fmt.Sprintf("%s/keys/%d", repoAPI(owner, repo), id), token, nil, nil)
}
