package forgejo

import (
	"context"
	"fmt"
	"net/url"
)

type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}
type Milestone struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	DueOn       string `json:"due_on"`
}
type Issue struct {
	ID        int64      `json:"id"`
	Number    int64      `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	State     string     `json:"state"`
	User      User       `json:"user"`
	Assignees []User     `json:"assignees"`
	Labels    []Label    `json:"labels"`
	Milestone *Milestone `json:"milestone"`
	Created   string     `json:"created_at"`
	Updated   string     `json:"updated_at"`
}
type IssueQuery struct {
	Page                                        int
	State, Search, Labels, Milestones, Assignee string
}

func (c *Client) Issues(ctx context.Context, token, owner, repo string, query IssueQuery) ([]Issue, Pagination, error) {
	q := url.Values{"page": {fmt.Sprint(query.Page)}, "limit": {"30"}, "type": {"issues"}, "state": {query.State}, "q": {query.Search}, "labels": {query.Labels}, "milestones": {query.Milestones}, "assigned_by": {query.Assignee}}
	items := []Issue{}
	headers, err := c.requestHeaders(ctx, "GET", repoAPI(owner, repo)+"/issues?"+q.Encode(), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, query.Page)
	return items, metadata, err
}

type CreateIssue struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Assignees []string `json:"assignees,omitempty"`
	Labels    []int64  `json:"labels,omitempty"`
	Milestone int64    `json:"milestone,omitempty"`
}
type EditIssue struct {
	Title     *string   `json:"title,omitempty"`
	Body      *string   `json:"body,omitempty"`
	State     *string   `json:"state,omitempty"`
	Assignees *[]string `json:"assignees,omitempty"`
	Milestone *int64    `json:"milestone,omitempty"`
}

func (c *Client) CreateIssue(ctx context.Context, token, owner, repo string, input CreateIssue) (Issue, error) {
	var result Issue
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/issues", token, input, &result)
	return result, err
}
func (c *Client) Issue(ctx context.Context, token, owner, repo string, index int64) (Issue, error) {
	var result Issue
	err := c.request(ctx, "GET", fmt.Sprintf("%s/issues/%d", repoAPI(owner, repo), index), token, nil, &result)
	return result, err
}
func (c *Client) EditIssue(ctx context.Context, token, owner, repo string, index int64, input EditIssue) (Issue, error) {
	var result Issue
	err := c.request(ctx, "PATCH", fmt.Sprintf("%s/issues/%d", repoAPI(owner, repo), index), token, input, &result)
	return result, err
}

type Comment struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	User    User   `json:"user"`
	Created string `json:"created_at"`
	Updated string `json:"updated_at"`
}

func (c *Client) IssueComments(ctx context.Context, token, owner, repo string, index int64, page int) ([]Comment, Pagination, error) {
	items := []Comment{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/issues/%d/comments?page=%d&limit=30", repoAPI(owner, repo), index, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) AddIssueComment(ctx context.Context, token, owner, repo string, index int64, body string) (Comment, error) {
	var result Comment
	err := c.request(ctx, "POST", fmt.Sprintf("%s/issues/%d/comments", repoAPI(owner, repo), index), token, struct {
		Body string `json:"body"`
	}{body}, &result)
	return result, err
}
func (c *Client) EditIssueComment(ctx context.Context, token, owner, repo string, id int64, body string) (Comment, error) {
	var result Comment
	err := c.request(ctx, "PATCH", fmt.Sprintf("%s/issues/comments/%d", repoAPI(owner, repo), id), token, struct {
		Body string `json:"body"`
	}{body}, &result)
	return result, err
}
func (c *Client) ReplaceIssueLabels(ctx context.Context, token, owner, repo string, index int64, labels []int64) ([]Label, error) {
	items := []Label{}
	err := c.request(ctx, "PUT", fmt.Sprintf("%s/issues/%d/labels", repoAPI(owner, repo), index), token, struct {
		Labels []int64 `json:"labels"`
	}{labels}, &items)
	return items, err
}

type WriteLabel struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func (c *Client) Labels(ctx context.Context, token, owner, repo string, page int) ([]Label, Pagination, error) {
	items := []Label{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/labels?page=%d&limit=50", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) CreateLabel(ctx context.Context, token, owner, repo string, input WriteLabel) (Label, error) {
	var result Label
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/labels", token, input, &result)
	return result, err
}
func (c *Client) EditLabel(ctx context.Context, token, owner, repo string, id int64, input WriteLabel) (Label, error) {
	var result Label
	err := c.request(ctx, "PATCH", fmt.Sprintf("%s/labels/%d", repoAPI(owner, repo), id), token, input, &result)
	return result, err
}

type WriteMilestone struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	DueOn       string `json:"due_on,omitempty"`
}

func (c *Client) Milestones(ctx context.Context, token, owner, repo string, page int) ([]Milestone, Pagination, error) {
	items := []Milestone{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/milestones?page=%d&limit=50&state=all", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) CreateMilestone(ctx context.Context, token, owner, repo string, input WriteMilestone) (Milestone, error) {
	var result Milestone
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/milestones", token, input, &result)
	return result, err
}
func (c *Client) EditMilestone(ctx context.Context, token, owner, repo string, id int64, input WriteMilestone) (Milestone, error) {
	var result Milestone
	err := c.request(ctx, "PATCH", fmt.Sprintf("%s/milestones/%d", repoAPI(owner, repo), id), token, input, &result)
	return result, err
}
