package forgejo

import (
	"context"
	"fmt"
	"net/url"
)

type Organization struct {
	ID                  int64  `json:"id"`
	Name                string `json:"name"`
	Username            string `json:"username"`
	FullName            string `json:"full_name"`
	Description         string `json:"description"`
	Email               string `json:"email"`
	Website             string `json:"website"`
	Location            string `json:"location"`
	Visibility          string `json:"visibility"`
	RepoAdminTeamAccess bool   `json:"repo_admin_change_team_access"`
}
type OrganizationFields struct {
	FullName            string  `json:"full_name"`
	Description         string  `json:"description"`
	Email               *string `json:"email,omitempty"`
	Website             string  `json:"website"`
	Location            string  `json:"location"`
	Visibility          string  `json:"visibility"`
	RepoAdminTeamAccess *bool   `json:"repo_admin_change_team_access,omitempty"`
}
type CreateOrganization struct {
	Username string `json:"username"`
	OrganizationFields
}

func (c *Client) Organizations(ctx context.Context, token string, page int, mine bool) ([]Organization, Pagination, error) {
	path := "/orgs"
	if mine {
		path = "/user/orgs"
	}
	items := []Organization{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s?page=%d&limit=30", path, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) Organization(ctx context.Context, token, name string) (Organization, error) {
	var result Organization
	err := c.request(ctx, "GET", "/orgs/"+url.PathEscape(name), token, nil, &result)
	return result, err
}
func (c *Client) CreateOrganization(ctx context.Context, token string, input CreateOrganization) (Organization, error) {
	var result Organization
	err := c.request(ctx, "POST", "/orgs", token, input, &result)
	return result, err
}

// Pinned native PATCH clears omitted profile strings. Callers retain values
// within the request, without a Soda organization/profile authority or store.
func (c *Client) EditOrganization(ctx context.Context, token, name string, input OrganizationFields) (Organization, error) {
	var result Organization
	err := c.request(ctx, "PATCH", "/orgs/"+url.PathEscape(name), token, input, &result)
	return result, err
}
func (c *Client) OrganizationMembers(ctx context.Context, token, name string, page int) ([]User, Pagination, error) {
	items := []User{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("/orgs/%s/members?page=%d&limit=30", url.PathEscape(name), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}

type Team struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Organization   Organization      `json:"organization"`
	Permission     string            `json:"permission"`
	IncludesAll    bool              `json:"includes_all_repositories"`
	CanCreateRepos bool              `json:"can_create_org_repo"`
	Units          map[string]string `json:"units_map"`
}
type TeamFields struct {
	Name           string            `json:"name,omitempty"`
	Description    *string           `json:"description,omitempty"`
	Permission     string            `json:"permission,omitempty"`
	IncludesAll    *bool             `json:"includes_all_repositories,omitempty"`
	CanCreateRepos *bool             `json:"can_create_org_repo,omitempty"`
	Units          map[string]string `json:"units_map,omitempty"`
}

func (c *Client) OrganizationTeams(ctx context.Context, token, name string, page int) ([]Team, Pagination, error) {
	items := []Team{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("/orgs/%s/teams?page=%d&limit=30", url.PathEscape(name), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) CreateTeam(ctx context.Context, token, org string, input TeamFields) (Team, error) {
	var result Team
	err := c.request(ctx, "POST", "/orgs/"+url.PathEscape(org)+"/teams", token, input, &result)
	return result, err
}
func (c *Client) Team(ctx context.Context, token string, id int64) (Team, error) {
	var result Team
	err := c.request(ctx, "GET", fmt.Sprintf("/teams/%d", id), token, nil, &result)
	return result, err
}
func (c *Client) EditTeam(ctx context.Context, token string, id int64, input TeamFields) (Team, error) {
	var result Team
	err := c.request(ctx, "PATCH", fmt.Sprintf("/teams/%d", id), token, input, &result)
	return result, err
}
func (c *Client) TeamMembers(ctx context.Context, token string, id int64, page int) ([]User, Pagination, error) {
	items := []User{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("/teams/%d/members?page=%d&limit=30", id, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) SetTeamMember(ctx context.Context, token string, id int64, login string, remove bool) error {
	method := "PUT"
	if remove {
		method = "DELETE"
	}
	return c.request(ctx, method, fmt.Sprintf("/teams/%d/members/%s", id, url.PathEscape(login)), token, nil, nil)
}
func (c *Client) TeamRepositories(ctx context.Context, token string, id int64, page int) ([]Repository, Pagination, error) {
	items := []Repository{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("/teams/%d/repos?page=%d&limit=30", id, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) SetTeamRepository(ctx context.Context, token string, id int64, owner, repo string, remove bool) error {
	method := "PUT"
	if remove {
		method = "DELETE"
	}
	return c.request(ctx, method, fmt.Sprintf("/teams/%d/repos/%s/%s", id, url.PathEscape(owner), url.PathEscape(repo)), token, nil, nil)
}
