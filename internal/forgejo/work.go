package forgejo

import (
	"context"
	"net/url"
	"strconv"
)

// Issue and Pull contain only the Forgejo-owned inputs needed by factory work.
type Issue struct {
	Number      int64     `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	User        User      `json:"user"`
	PullRequest *struct{} `json:"pull_request"`
}

type Pull struct {
	User   User       `json:"user"`
	Number int64      `json:"number"`
	State  string     `json:"state"`
	Merged bool       `json:"merged"`
	Head   PullBranch `json:"head"`
	Base   PullBranch `json:"base"`
}

type PullBranch struct {
	Ref  string     `json:"ref"`
	SHA  string     `json:"sha"`
	Repo Repository `json:"repo"`
}

func workPath(repo Repository) (string, error) {
	if repo.ID <= 0 || !repositoryPart(repo.Owner.Login) || !repositoryPart(repo.Name) {
		return "", ErrInvalidResponse
	}
	return "/repos/" + url.PathEscape(repo.Owner.Login) + "/" + url.PathEscape(repo.Name), nil
}

func (c *Client) WorkIssue(ctx context.Context, token string, repo Repository, number int64) (Issue, error) {
	path, err := workPath(repo)
	if err != nil || number <= 0 {
		return Issue{}, ErrInvalidResponse
	}
	var issue Issue
	err = c.request(ctx, "GET", path+"/issues/"+strconv.FormatInt(number, 10), token, nil, &issue)
	if err == nil && issue.Number != number {
		err = ErrInvalidResponse
	}
	return issue, err
}

func (c *Client) WorkPull(ctx context.Context, token string, repo Repository, number int64) (Pull, error) {
	path, err := workPath(repo)
	if err != nil || number <= 0 {
		return Pull{}, ErrInvalidResponse
	}
	var pull Pull
	err = c.request(ctx, "GET", path+"/pulls/"+strconv.FormatInt(number, 10), token, nil, &pull)
	if err == nil && pull.Number != number {
		err = ErrInvalidResponse
	}
	return pull, err
}

func (c *Client) CreateWorkPull(ctx context.Context, token string, repo Repository, head, base, title, body string) (Pull, error) {
	path, err := workPath(repo)
	if err != nil {
		return Pull{}, err
	}
	var pull Pull
	err = c.request(ctx, "POST", path+"/pulls", token, map[string]any{"head": head, "base": base, "title": title, "body": body, "allow_maintainer_edit": false}, &pull)
	return pull, err
}

// SubmitWorkReview deliberately exposes review submission, not merge or status writes.
// The caller enforces live run authority and the assigned PR/commit before invoking it.
func (c *Client) SubmitWorkReview(ctx context.Context, token string, repo Repository, number int64, commit, event, body string) error {
	path, err := workPath(repo)
	if err != nil || number <= 0 || (event != "APPROVED" && event != "REQUEST_CHANGES") {
		return ErrInvalidResponse
	}
	return c.request(ctx, "POST", path+"/pulls/"+strconv.FormatInt(number, 10)+"/reviews", token, map[string]string{"commit_id": commit, "event": event, "body": body}, nil)
}

type ActionRun struct {
	ID       int64  `json:"id"`
	Commit   string `json:"commit_sha"`
	Status   string `json:"status"`
	Workflow string `json:"workflow_id"`
	Event    string `json:"event"`
	Ref      string `json:"prettyref"`
}

// WorkActions uses the pinned Forgejo API's SHA filter and native run fields.
// It does not manufacture CI evidence from agent-supplied commit statuses.
func (c *Client) WorkActions(ctx context.Context, token string, repo Repository, sha, workflow string, page int) ([]ActionRun, error) {
	path, err := workPath(repo)
	if err != nil || page < 1 {
		return nil, ErrInvalidResponse
	}
	query := url.Values{"head_sha": {sha}, "workflow_id": {workflow}, "page": {strconv.Itoa(page)}, "limit": {"50"}}
	var result struct {
		Runs []ActionRun `json:"workflow_runs"`
	}
	err = c.request(ctx, "GET", path+"/actions/runs?"+query.Encode(), token, nil, &result)
	if err != nil {
		return nil, err
	}
	if len(result.Runs) > 50 {
		return nil, ErrInvalidResponse
	}
	for _, run := range result.Runs {
		if run.ID <= 0 || run.Commit != sha || run.Workflow != workflow {
			return nil, ErrInvalidResponse
		}
	}
	return result.Runs, nil
}

type WorkReview struct {
	ID     int64  `json:"id"`
	Commit string `json:"commit_id"`
	State  string `json:"state"`
	Body   string `json:"body"`
	User   User   `json:"user"`
}

func (c *Client) WorkReviews(ctx context.Context, token string, repo Repository, number int64) ([]WorkReview, error) {
	path, err := workPath(repo)
	if err != nil || number <= 0 {
		return nil, ErrInvalidResponse
	}
	var reviews []WorkReview
	err = c.request(ctx, "GET", path+"/pulls/"+strconv.FormatInt(number, 10)+"/reviews?limit=50", token, nil, &reviews)
	return reviews, err
}

func (c *Client) WorkComment(ctx context.Context, token string, repo Repository, issue int64, body string) error {
	path, err := workPath(repo)
	if err != nil || issue <= 0 {
		return ErrInvalidResponse
	}
	return c.request(ctx, "POST", path+"/issues/"+strconv.FormatInt(issue, 10)+"/comments", token, map[string]string{"body": body}, nil)
}

func (c *Client) WorkActor(ctx context.Context, token, login string) (User, error) {
	if !repositoryPart(login) {
		return User{}, ErrInvalidResponse
	}
	var user User
	err := c.request(ctx, "GET", "/users/"+url.PathEscape(login), token, nil, &user)
	if err == nil && user.Login != login {
		err = ErrInvalidResponse
	}
	return user, err
}
