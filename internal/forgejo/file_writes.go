package forgejo

import (
	"context"
	"net/url"
	"strings"
)

type WriteFile struct {
	Branch  string `json:"branch"`
	Content string `json:"content"`
	Message string `json:"message"`
	SHA     string `json:"sha,omitempty"`
}
type FileResult struct {
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
	Content struct {
		SHA string `json:"sha"`
	} `json:"content"`
}

func (c *Client) WriteFile(ctx context.Context, token, owner, repo, file string, input WriteFile, update bool) (FileResult, error) {
	method := "POST"
	if update {
		method = "PUT"
	}
	parts := strings.Split(file, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	var result FileResult
	err := c.request(ctx, method, repoAPI(owner, repo)+"/contents/"+strings.Join(parts, "/"), token, input, &result)
	return result, err
}

type ImportRepository struct {
	CloneURL string `json:"clone_addr"`
	Name     string `json:"repo_name"`
	Owner    string `json:"repo_owner"`
	Private  bool   `json:"private"`
	Service  string `json:"service"`
	Username string `json:"auth_username,omitempty"`
	Password string `json:"auth_password,omitempty"`
	Token    string `json:"auth_token,omitempty"`
}

func (c *Client) ImportRepository(ctx context.Context, token string, input ImportRepository) (Repository, error) {
	var result Repository
	err := c.request(ctx, "POST", "/repos/migrate", token, input, &result)
	return result, err
}
