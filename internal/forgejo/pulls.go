package forgejo

import (
	"context"
	"fmt"
	"net/url"
)

type PullBranch struct {
	Ref   string     `json:"ref"`
	SHA   string     `json:"sha"`
	Label string     `json:"label"`
	Repo  Repository `json:"repo"`
}
type Pull struct {
	Issue
	Head               PullBranch `json:"head"`
	Base               PullBranch `json:"base"`
	MergeBase          string     `json:"merge_base"`
	Merged             bool       `json:"merged"`
	Mergeable          bool       `json:"mergeable"`
	Draft              bool       `json:"draft"`
	RequestedReviewers []User     `json:"requested_reviewers"`
}

func (c *Client) Pulls(ctx context.Context, token, owner, repo, state string, page int) ([]Pull, Pagination, error) {
	items := []Pull{}
	headers, err := c.requestHeaders(ctx, "GET", repoAPI(owner, repo)+"/pulls?"+url.Values{"state": {state}, "page": {fmt.Sprint(page)}, "limit": {"30"}}.Encode(), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}

type CreatePull struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Base  string `json:"base"`
	Head  string `json:"head"`
}

func (c *Client) CreatePull(ctx context.Context, token, owner, repo string, input CreatePull) (Pull, error) {
	var result Pull
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/pulls", token, input, &result)
	return result, err
}
func (c *Client) Pull(ctx context.Context, token, owner, repo string, index int64) (Pull, error) {
	var result Pull
	err := c.request(ctx, "GET", fmt.Sprintf("%s/pulls/%d", repoAPI(owner, repo), index), token, nil, &result)
	return result, err
}
func (c *Client) PullCommits(ctx context.Context, token, owner, repo string, index int64, page int) ([]Commit, Pagination, error) {
	items := []Commit{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/pulls/%d/commits?page=%d&limit=30", repoAPI(owner, repo), index, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}

type ChangedFile struct {
	Name         string `json:"filename"`
	PreviousName string `json:"previous_filename"`
	Status       string `json:"status"`
	Additions    int64  `json:"additions"`
	Deletions    int64  `json:"deletions"`
}

func (c *Client) PullFiles(ctx context.Context, token, owner, repo string, index int64, page int) ([]ChangedFile, Pagination, error) {
	items := []ChangedFile{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/pulls/%d/files?page=%d&limit=30", repoAPI(owner, repo), index, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) PullDiff(ctx context.Context, token, owner, repo string, index int64) ([]byte, error) {
	return c.readBytes(ctx, fmt.Sprintf("%s/pulls/%d.diff", repoAPI(owner, repo), index), token, 1<<20)
}

type Review struct {
	ID        int64  `json:"id"`
	User      User   `json:"user"`
	Body      string `json:"body"`
	Commit    string `json:"commit_id"`
	State     string `json:"state"`
	Stale     bool   `json:"stale"`
	Dismissed bool   `json:"dismissed"`
	Submitted string `json:"submitted_at"`
}

func (c *Client) PullReviews(ctx context.Context, token, owner, repo string, index int64, page int) ([]Review, Pagination, error) {
	items := []Review{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/pulls/%d/reviews?page=%d&limit=30", repoAPI(owner, repo), index, page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}

type ReviewComment struct {
	Path    string `json:"path"`
	Body    string `json:"body"`
	NewLine int64  `json:"new_position"`
}
type CreateReview struct {
	Body     string          `json:"body"`
	Commit   string          `json:"commit_id"`
	Event    string          `json:"event"`
	Comments []ReviewComment `json:"comments,omitempty"`
}

func (c *Client) CreatePullReview(ctx context.Context, token, owner, repo string, index int64, input CreateReview) (Review, error) {
	var result Review
	err := c.request(ctx, "POST", fmt.Sprintf("%s/pulls/%d/reviews", repoAPI(owner, repo), index), token, input, &result)
	return result, err
}
func (c *Client) RequestPullReviewers(ctx context.Context, token, owner, repo string, index int64, reviewers []string, remove bool) error {
	method := "POST"
	if remove {
		method = "DELETE"
	}
	return c.request(ctx, method, fmt.Sprintf("%s/pulls/%d/requested_reviewers", repoAPI(owner, repo), index), token, struct {
		Reviewers []string `json:"reviewers"`
	}{reviewers}, nil)
}
func (c *Client) MergePull(ctx context.Context, token, owner, repo string, index int64, head, strategy, title, message string) error {
	return c.request(ctx, "POST", fmt.Sprintf("%s/pulls/%d/merge", repoAPI(owner, repo), index), token, struct {
		Strategy     string `json:"Do"`
		Head         string `json:"head_commit_id"`
		Title        string `json:"MergeTitleField"`
		Message      string `json:"MergeMessageField"`
		Force        bool   `json:"force_merge"`
		DeleteBranch bool   `json:"delete_branch_after_merge"`
		Auto         bool   `json:"merge_when_checks_succeed"`
	}{Strategy: strategy, Head: head, Title: title, Message: message}, nil)
}

type CommitStatus struct {
	ID          int64  `json:"id"`
	Context     string `json:"context"`
	Description string `json:"description"`
	State       string `json:"status"`
	TargetURL   string `json:"target_url"`
}
type CombinedStatus struct {
	SHA      string         `json:"sha"`
	State    string         `json:"state"`
	Statuses []CommitStatus `json:"statuses"`
	Total    int64          `json:"total_count"`
}

func (c *Client) Status(ctx context.Context, token, owner, repo, sha string, page int) (CombinedStatus, Pagination, error) {
	var result CombinedStatus
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/commits/%s/status?page=%d&limit=50", repoAPI(owner, repo), url.PathEscape(sha), page), token, nil, &result)
	if err != nil {
		return result, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return result, metadata, err
}
