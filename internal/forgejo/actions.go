package forgejo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Event payloads, runner credentials and task tokens are intentionally omitted.
type ActionRun struct {
	ID           int64  `json:"id"`
	Number       int64  `json:"index_in_repo"`
	Title        string `json:"title"`
	Workflow     string `json:"workflow_id"`
	SHA          string `json:"commit_sha"`
	Ref          string `json:"prettyref"`
	Status       string `json:"status"`
	Event        string `json:"event"`
	Created      string `json:"created"`
	Updated      string `json:"updated"`
	NeedApproval bool   `json:"need_approval"`
}
type ActionRunQuery struct {
	Ref, Workflow, Status string
	Page                  int
}

// These two native endpoints expose a body count but no pagination headers.
// Read the actual public API cap rather than guessing it from a partial page.
func (c *Client) actionPageSize(ctx context.Context, token string) (int, error) {
	var settings struct {
		Max int `json:"max_response_items"`
	}
	if err := c.request(ctx, "GET", "/settings/api", token, nil, &settings); err != nil {
		return 0, err
	}
	if settings.Max < 1 {
		return 0, ErrInvalidResponse
	}
	return min(30, settings.Max), nil
}
func actionPagination(page, size int, total int64) (Pagination, error) {
	if total < 0 {
		return Pagination{}, ErrInvalidResponse
	}
	count := strconv.FormatInt(total, 10)
	result := Pagination{Total: &count}
	if int64(page)*int64(size) < total {
		next := page + 1
		result.NextPage = &next
	}
	return result, nil
}
func (c *Client) ActionRuns(ctx context.Context, token, owner, repo string, q ActionRunQuery) ([]ActionRun, Pagination, error) {
	size, err := c.actionPageSize(ctx, token)
	if err != nil {
		return nil, Pagination{}, err
	}
	query := url.Values{"ref": {q.Ref}, "workflow_id": {q.Workflow}, "page": {fmt.Sprint(q.Page)}, "limit": {fmt.Sprint(size)}}
	if q.Status != "" {
		query.Set("status", q.Status)
	}
	var result struct {
		Runs  []ActionRun `json:"workflow_runs"`
		Total *int64      `json:"total_count"`
	}
	err = c.request(ctx, "GET", repoAPI(owner, repo)+"/actions/runs?"+query.Encode(), token, nil, &result)
	if err != nil {
		return nil, Pagination{}, err
	}
	if result.Total == nil || result.Runs == nil || *result.Total < int64(len(result.Runs)) {
		return nil, Pagination{}, ErrInvalidResponse
	}
	metadata, err := actionPagination(q.Page, size, *result.Total)
	return result.Runs, metadata, err
}
func (c *Client) ActionRun(ctx context.Context, token, owner, repo string, id int64) (ActionRun, error) {
	var result ActionRun
	err := c.request(ctx, "GET", fmt.Sprintf("%s/actions/runs/%d", repoAPI(owner, repo), id), token, nil, &result)
	return result, err
}

type ActionTask struct {
	ID        int64  `json:"id"`
	RunNumber int64  `json:"run_number"`
	Name      string `json:"name"`
	Title     string `json:"display_title"`
	Status    string `json:"status"`
	SHA       string `json:"head_sha"`
	Ref       string `json:"head_branch"`
	Workflow  string `json:"workflow_id"`
}

func (c *Client) ActionTasks(ctx context.Context, token, owner, repo string, page int) ([]ActionTask, Pagination, error) {
	size, err := c.actionPageSize(ctx, token)
	if err != nil {
		return nil, Pagination{}, err
	}
	var result struct {
		Tasks []ActionTask `json:"workflow_runs"`
		Total *int64       `json:"total_count"`
	}
	err = c.request(ctx, "GET", fmt.Sprintf("%s/actions/tasks?page=%d&limit=%d", repoAPI(owner, repo), page, size), token, nil, &result)
	if err != nil {
		return nil, Pagination{}, err
	}
	if result.Total == nil || result.Tasks == nil || *result.Total < int64(len(result.Tasks)) {
		return nil, Pagination{}, ErrInvalidResponse
	}
	metadata, err := actionPagination(page, size, *result.Total)
	return result.Tasks, metadata, err
}

type WorkflowFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	SHA  string `json:"sha"`
}

// Pinned ListWorkflows selects the first existing standard directory; it does
// not merge directories or parse workflow YAML in Soda.
func (c *Client) WorkflowFiles(ctx context.Context, token, owner, repo, ref string) ([]WorkflowFile, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// Establish repository/ref visibility before interpreting a directory 404.
	if _, err := c.Contents(ctx, token, owner, repo, ref, ""); err != nil {
		return nil, err
	}
	for _, parent := range []string{".forgejo", ".gitea", ".github"} {
		entries, err := c.Contents(ctx, token, owner, repo, ref, parent)
		if err != nil {
			var status *HTTPError
			if errors.As(err, &status) && status.Status == 404 {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.Name != "workflows" || entry.Type != "dir" {
				continue
			}
			result := []WorkflowFile{}
			count := 0
			for page := 1; page <= 20; page++ {
				var tree struct {
					Page      int  `json:"page"`
					Truncated bool `json:"truncated"`
					Entries   []struct {
						Path string `json:"path"`
						Type string `json:"type"`
						SHA  string `json:"sha"`
					} `json:"tree"`
				}
				err := c.request(ctx, "GET", fmt.Sprintf("%s/git/trees/%s?recursive=true&page=%d&per_page=500", repoAPI(owner, repo), url.PathEscape(entry.SHA), page), token, nil, &tree)
				if err != nil {
					return nil, err
				}
				if tree.Page != page {
					return nil, ErrInvalidResponse
				}
				count += len(tree.Entries)
				if count > 10000 {
					return nil, ErrResponseTooLarge
				}
				for _, file := range tree.Entries {
					if file.Type == "blob" && (strings.HasSuffix(file.Path, ".yaml") || strings.HasSuffix(file.Path, ".yml")) {
						result = append(result, WorkflowFile{file.Path, parent + "/workflows/" + file.Path, file.SHA})
					}
				}
				if !tree.Truncated {
					return result, nil
				}
				if len(tree.Entries) == 0 {
					return nil, ErrInvalidResponse
				}
			}
			return nil, ErrResponseTooLarge
		}
	}
	return []WorkflowFile{}, nil
}

type DispatchResult struct {
	ID     int64    `json:"id"`
	Number int64    `json:"run_number"`
	Jobs   []string `json:"jobs"`
}

func (c *Client) DispatchWorkflow(ctx context.Context, token, owner, repo, name, ref string, inputs map[string]string) (DispatchResult, error) {
	var result DispatchResult
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/actions/workflows/"+url.PathEscape(name)+"/dispatches", token, struct {
		Ref        string            `json:"ref"`
		Inputs     map[string]string `json:"inputs"`
		ReturnInfo bool              `json:"return_run_info"`
	}{ref, inputs, true}, &result)
	if err == nil && result.ID <= 0 {
		return result, ErrInvalidResponse
	}
	return result, err
}

type ActionSecret struct {
	Name    string `json:"name"`
	Created string `json:"created_at"`
}
type ActionVariable struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// Only repository and organization configuration roots can be constructed.
// This value is server-created, never a browser-supplied URL or role selector.
type ActionsConfig struct {
	client *Client
	path   string
}

func (c *Client) RepositoryActions(owner, repo string) ActionsConfig {
	return ActionsConfig{c, repoAPI(owner, repo) + "/actions"}
}
func (c *Client) OrganizationActions(org string) ActionsConfig {
	return ActionsConfig{c, "/orgs/" + url.PathEscape(org) + "/actions"}
}
func (a ActionsConfig) Secrets(ctx context.Context, token string, page int) ([]ActionSecret, Pagination, error) {
	items := []ActionSecret{}
	headers, err := a.client.requestHeaders(ctx, "GET", fmt.Sprintf("%s/secrets?page=%d&limit=30", a.path, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (a ActionsConfig) SetSecret(ctx context.Context, token, name, data string, remove bool) error {
	path := a.path + "/secrets/" + url.PathEscape(name)
	if remove {
		return a.client.request(ctx, "DELETE", path, token, nil, nil)
	}
	return a.client.request(ctx, "PUT", path, token, struct {
		Data string `json:"data"`
	}{data}, nil)
}
func (a ActionsConfig) Variables(ctx context.Context, token string, page int) ([]ActionVariable, Pagination, error) {
	items := []ActionVariable{}
	headers, err := a.client.requestHeaders(ctx, "GET", fmt.Sprintf("%s/variables?page=%d&limit=30", a.path, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (a ActionsConfig) SetVariable(ctx context.Context, token, name, value string, create, remove bool) error {
	path := a.path + "/variables/" + url.PathEscape(name)
	if remove {
		return a.client.request(ctx, "DELETE", path, token, nil, nil)
	}
	method := "PUT"
	if create {
		method = "POST"
	}
	return a.client.request(ctx, method, path, token, struct {
		Value string `json:"value"`
	}{value}, nil)
}
