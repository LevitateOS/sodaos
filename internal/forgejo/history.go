package forgejo

import (
	"context"
	"fmt"
	"net/url"
)

type CommitFile struct {
	Name   string `json:"filename"`
	Status string `json:"status"`
}
type Commit struct {
	SHA     string `json:"sha"`
	Created string `json:"created"`
	Commit  struct {
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	Files []CommitFile `json:"files"`
}

func repoAPI(owner, repo string) string {
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
}
func (c *Client) Commits(ctx context.Context, token, owner, repo, ref, file string, page int) ([]Commit, Pagination, error) {
	query := url.Values{"sha": {ref}, "page": {fmt.Sprint(page)}, "limit": {"30"}, "stat": {"false"}, "verification": {"false"}, "files": {"false"}}
	if file != "" {
		query.Set("path", file)
	}
	items := []Commit{}
	headers, err := c.requestHeaders(ctx, "GET", repoAPI(owner, repo)+"/commits?"+query.Encode(), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) Commit(ctx context.Context, token, owner, repo, sha string) (Commit, error) {
	var result Commit
	err := c.request(ctx, "GET", repoAPI(owner, repo)+"/git/commits/"+url.PathEscape(sha), token, nil, &result)
	return result, err
}

// ResolveRef uses the native single-commit interface's ref support. Request only
// identity metadata; dependent reads use the returned full SHA, not a moving ref.
func (c *Client) ResolveRef(ctx context.Context, token, owner, repo, ref string) (string, error) {
	var result Commit
	err := c.request(ctx, "GET", repoAPI(owner, repo)+"/git/commits/"+url.PathEscape(ref)+"?stat=false&verification=false&files=false", token, nil, &result)
	return result.SHA, err
}

func (c *Client) CommitDiff(ctx context.Context, token, owner, repo, sha string) ([]byte, error) {
	return c.readBytes(ctx, repoAPI(owner, repo)+"/git/commits/"+url.PathEscape(sha)+".diff", token, 1<<20)
}

type Branch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	CanPush   bool   `json:"user_can_push"`
}

func (c *Client) Branches(ctx context.Context, token, owner, repo string, page int) ([]Branch, Pagination, error) {
	items := []Branch{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/branches?page=%d&limit=50", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) CreateBranch(ctx context.Context, token, owner, repo, name, oldRef string) (Branch, error) {
	var result Branch
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/branches", token, struct {
		Name string `json:"new_branch_name"`
		Ref  string `json:"old_ref_name"`
	}{name, oldRef}, &result)
	return result, err
}

type Tag struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Commit  struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

func (c *Client) Tags(ctx context.Context, token, owner, repo string, page int) ([]Tag, Pagination, error) {
	items := []Tag{}
	headers, err := c.requestHeaders(ctx, "GET", fmt.Sprintf("%s/tags?page=%d&limit=50", repoAPI(owner, repo), page), token, nil, &items)
	if err != nil {
		return nil, Pagination{}, err
	}
	metadata, err := pagination(headers, page)
	return items, metadata, err
}
func (c *Client) CreateTag(ctx context.Context, token, owner, repo, name, target, message string) (Tag, error) {
	var result Tag
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/tags", token, struct {
		Name    string `json:"tag_name"`
		Target  string `json:"target"`
		Message string `json:"message"`
	}{name, target, message}, &result)
	return result, err
}

type Comparison struct {
	Commits []Commit     `json:"commits"`
	Files   []CommitFile `json:"files"`
	Total   int64        `json:"total_commits"`
}

func (c *Client) Compare(ctx context.Context, token, owner, repo, base, head string) (Comparison, error) {
	var result Comparison
	err := c.request(ctx, "GET", repoAPI(owner, repo)+"/compare/"+url.PathEscape(base+"..."+head)+"?verification=false", token, nil, &result)
	return result, err
}
func (c *Client) Fork(ctx context.Context, token, owner, repo, name string) (Repository, error) {
	var result Repository
	err := c.request(ctx, "POST", repoAPI(owner, repo)+"/forks", token, struct {
		Name string `json:"name,omitempty"`
	}{name}, &result)
	return result, err
}
